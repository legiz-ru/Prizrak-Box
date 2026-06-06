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

// KillPx terminates the locally spawned px process (if any). px started via
// the service is not killed here; the caller handles that through the service.
//
// On Unix it first sends SIGINT so px can run its shutdown (which disables the
// system proxy), then force-kills after a short grace period. On Windows it
// kills directly.
func (c *CoreService) KillPx() {
	c.mu.Lock()
	cmd := c.cmd
	c.cmd = nil
	c.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
		return
	}
	// Graceful: SIGINT -> px disables proxy and exits.
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() { _, _ = cmd.Process.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
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
