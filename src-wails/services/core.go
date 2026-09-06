// Package services contains the Go services the Wails v3 shell binds to the
// frontend. CoreService manages the lifecycle of the px backend process.
//
// Contract with px (see src-go/prizrak/core.go and src-go/api/job/alive.go):
//   - px is spawned as:  px -addr=127.0.0.1:<cbPort> -home=<dir>
//   - px chooses its OWN control port (9686 or random) and a secret, then
//     repeatedly calls  GET http://<addr>/pxStore?port=<p>&secret=<s>
//     until it receives the body "ok".
//   - px polls  GET http://<addr>/pxAlive  every 3s and exits itself if the
//     shell stops answering "alive". This is how the backend shuts down when
//     the GUI closes.
//
// The shell therefore runs a tiny loopback HTTP server (the "callback
// server") that answers those two endpoints. This replaces the Electron
// src-electron/server.ts + admin.ts happy path.
package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/legiz-ru/prizrak-box-wails/internal/locate"
)

// ConnInfo is the connection information the frontend needs to talk to px.
type ConnInfo struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Secret string `json:"secret"`
}

// CoreService spawns and supervises the px backend.
type CoreService struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	startedBySvc bool
	cbServer    *http.Server
	cbAddr      string
	info        ConnInfo
	infoReady   chan struct{}
	readyClosed bool
	pulse       chan ConnInfo // signalled on every /pxStore callback
}

// NewCoreService creates an unstarted CoreService.
func NewCoreService() *CoreService {
	return &CoreService{infoReady: make(chan struct{})}
}

// --- Accessors used by TunService and main ---------------------------------

// CbAddr returns the callback server address ("host:port"). Empty until Start.
func (c *CoreService) CbAddr() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cbAddr
}

// PxPath returns the resolved px binary path.
func (c *CoreService) PxPath() string { return locate.PxBinary() }

// Home returns the px home directory.
func (c *CoreService) Home() string { return locate.HomeDir() }

// --- Frontend-bound methods -------------------------------------------------

// GetConnInfo is bound to the frontend; it blocks (up to a timeout) until px
// has reported its port/secret at least once, then returns the latest values.
func (c *CoreService) GetConnInfo() (ConnInfo, error) {
	c.mu.Lock()
	ready := c.infoReady
	c.mu.Unlock()
	select {
	case <-ready:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.info, nil
	case <-time.After(60 * time.Second):
		return ConnInfo{}, fmt.Errorf("backend did not become ready in time")
	}
}

// --- Lifecycle --------------------------------------------------------------

// Start launches the callback server and the px process (direct spawn), then
// waits until px has called back with its port/secret.
func (c *CoreService) Start() (ConnInfo, error) {
	if err := c.ensureCallbackServer(); err != nil {
		return ConnInfo{}, fmt.Errorf("callback server: %w", err)
	}
	return c.RestartDirect()
}

// RestartDirect kills any running px and spawns a fresh one directly (no
// elevated service). Used on first start and for non-TUN restarts.
func (c *CoreService) RestartDirect() (ConnInfo, error) {
	c.KillPx()
	c.Arm()

	cmd := exec.Command(c.PxPath(), "-addr="+c.CbAddr(), "-home="+c.Home())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return ConnInfo{}, fmt.Errorf("spawn px (%s): %w", c.PxPath(), err)
	}
	c.mu.Lock()
	c.cmd = cmd
	c.startedBySvc = false
	c.mu.Unlock()

	return c.Await(60 * time.Second)
}

// Arm prepares a fresh callback "pulse" channel before px (re)starts. Call
// this immediately before spawning px (directly or via the service).
func (c *CoreService) Arm() {
	c.mu.Lock()
	c.pulse = make(chan ConnInfo, 1)
	c.mu.Unlock()
}

// Await blocks until the next /pxStore callback arrives (or timeout), records
// the connection info and returns it.
func (c *CoreService) Await(timeout time.Duration) (ConnInfo, error) {
	c.mu.Lock()
	pulse := c.pulse
	c.mu.Unlock()
	if pulse == nil {
		return ConnInfo{}, fmt.Errorf("Await called without Arm")
	}
	select {
	case info := <-pulse:
		c.setInfo(info)
		return info, nil
	case <-time.After(timeout):
		return ConnInfo{}, fmt.Errorf("backend did not call back in time")
	}
}

// MarkStartedBySvc records that the current px was started via px-service.
func (c *CoreService) MarkStartedBySvc() {
	c.mu.Lock()
	c.startedBySvc = true
	c.mu.Unlock()
}

// ClearStartedBySvc records that px is no longer owned by px-service (the
// service was stopped or uninstalled).
func (c *CoreService) ClearStartedBySvc() {
	c.mu.Lock()
	c.startedBySvc = false
	c.mu.Unlock()
}

// pxShutdownGrace is how long px gets to finish its own shutdown before it is
// force-killed. It has to cover DisableProxy, which shells out to `networksetup`
// on macOS (slow, and once per network service) and to `gsettings` on Linux.
const pxShutdownGrace = 5 * time.Second

// KillPx terminates the locally spawned px process (if any). px started via
// the service is not killed here; the caller handles that through the service.
//
// px undoes its OS-level state — the system proxy above all — only inside its
// own exit path, so it is always asked to shut itself down first and killed
// only if it does not. Two mechanisms feed that exit path: the /prizrak/exit
// endpoint and, on Unix, SIGINT.
//
// Windows previously had neither: Process.Kill is TerminateProcess, which
// delivers no signal, and px's /pxAlive watchdog (its other cleanup trigger)
// never noticed the shell was gone because the kill was instant. Closing the
// app therefore left ProxyEnable=1 in the registry pointing at a dead port —
// invisible while px happened to reclaim the same port next launch, and broken
// as soon as another client rewrote those keys in between.
func (c *CoreService) KillPx() {
	c.mu.Lock()
	cmd := c.cmd
	c.cmd = nil
	info := c.info
	c.mu.Unlock()

	// Sent to whoever owns px, including the elevated service (which this method
	// deliberately does not kill, and which stops px with a hard kill of its
	// own — see src-service/manager.StopPx).
	requestPxExit(info)

	if cmd == nil || cmd.Process == nil {
		return
	}

	exited := make(chan struct{})
	go func() { _, _ = cmd.Process.Wait(); close(exited) }()

	if runtime.GOOS != "windows" {
		_ = cmd.Process.Signal(os.Interrupt)
	}

	select {
	case <-exited:
		return
	case <-time.After(pxShutdownGrace):
	}

	_ = cmd.Process.Kill()
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
	}
}

// RequestExit asks the running px — whoever spawned it — to shut itself down
// cleanly, without touching the local process handle. Used where px is owned by
// the elevated service and can only be reached over its control API.
func (c *CoreService) RequestExit() {
	c.mu.Lock()
	info := c.info
	c.mu.Unlock()
	requestPxExit(info)
}

// requestPxExit asks px to shut itself down through its control API, which runs
// the same cleanup as a signal (job.Exit -> DisableProxy) before calling
// os.Exit. A no-op when px has not reported its port yet.
//
// px writes its reply and exits immediately after, so the connection is
// routinely torn down mid-response: a transport error here is expected and says
// nothing about whether the cleanup ran.
func requestPxExit(info ConnInfo) {
	if info.Port == 0 {
		return
	}
	host := info.Host
	if host == "" {
		host = "127.0.0.1"
	}
	url := fmt.Sprintf("http://%s/prizrak/exit", net.JoinHostPort(host, strconv.Itoa(info.Port)))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	if info.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+info.Secret)
	}
	client := &http.Client{Timeout: pxShutdownGrace}
	if resp, err := client.Do(req); err == nil {
		_ = resp.Body.Close()
	}
}

// Stop terminates px and the callback server (Wails lifecycle hook).
func (c *CoreService) Stop() {
	c.KillPx()
	c.mu.Lock()
	srv := c.cbServer
	c.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
}

// ServiceShutdown is the Wails lifecycle hook called on app shutdown.
func (c *CoreService) ServiceShutdown() error {
	c.Stop()
	return nil
}

// --- internals --------------------------------------------------------------

func (c *CoreService) setInfo(info ConnInfo) {
	c.mu.Lock()
	c.info = info
	if !c.readyClosed {
		c.readyClosed = true
		close(c.infoReady)
	}
	c.mu.Unlock()
}

func (c *CoreService) ensureCallbackServer() error {
	c.mu.Lock()
	if c.cbServer != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/pxStore", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		port := atoiSafe(q.Get("port"))
		secret := q.Get("secret")
		if port > 0 {
			info := ConnInfo{Host: "127.0.0.1", Port: port, Secret: secret}
			c.mu.Lock()
			c.info = info
			if !c.readyClosed {
				c.readyClosed = true
				close(c.infoReady)
			}
			p := c.pulse
			c.mu.Unlock()
			if p != nil {
				select {
				case p <- info:
				default:
				}
			}
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/pxAlive", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("alive"))
	})

	srv := &http.Server{Handler: mux}
	c.mu.Lock()
	c.cbServer = srv
	c.cbAddr = addr
	c.mu.Unlock()

	go func() { _ = srv.Serve(ln) }()
	return nil
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
