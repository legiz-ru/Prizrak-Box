// Package services contains the Go services that the Wails v3 shell binds to
// the frontend. CoreService manages the lifecycle of the px backend process.
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
// So the shell must run a tiny loopback HTTP server (the "callback server")
// that answers those two endpoints. This replaces src-electron/server.ts +
// admin.ts for the happy path.
package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	mu       sync.Mutex
	cmd      *exec.Cmd
	cbServer *http.Server
	info     ConnInfo
	ready    chan struct{}
	started  bool
}

// NewCoreService creates an unstarted CoreService.
func NewCoreService() *CoreService {
	return &CoreService{ready: make(chan struct{})}
}

// GetConnInfo is bound to the frontend; it blocks (up to a timeout) until px
// has reported its port/secret, then returns them.
func (c *CoreService) GetConnInfo() (ConnInfo, error) {
	select {
	case <-c.ready:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.info, nil
	case <-time.After(60 * time.Second):
		return ConnInfo{}, fmt.Errorf("backend did not become ready in time")
	}
}

// Start launches the callback server and the px process, then waits until px
// has called back with its port/secret. It is safe to call once.
func (c *CoreService) Start() (ConnInfo, error) {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return c.GetConnInfo()
	}
	c.started = true
	c.mu.Unlock()

	cbAddr, err := c.startCallbackServer()
	if err != nil {
		return ConnInfo{}, fmt.Errorf("callback server: %w", err)
	}

	pxPath := locate.PxBinary()
	home := locate.HomeDir()
	cmd := exec.Command(pxPath, "-addr="+cbAddr, "-home="+home)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return ConnInfo{}, fmt.Errorf("spawn px (%s): %w", pxPath, err)
	}

	c.mu.Lock()
	c.cmd = cmd
	c.mu.Unlock()

	return c.GetConnInfo()
}

// Stop terminates px and the callback server.
func (c *CoreService) Stop() {
	c.mu.Lock()
	cmd := c.cmd
	srv := c.cbServer
	c.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
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

// startCallbackServer binds a loopback HTTP server on a free port and returns
// its "host:port" address. It answers /pxStore and /pxAlive.
func (c *CoreService) startCallbackServer() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/pxStore", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		port := atoiSafe(q.Get("port"))
		secret := q.Get("secret")
		if port > 0 {
			c.mu.Lock()
			already := c.info.Port != 0
			c.info = ConnInfo{Host: "127.0.0.1", Port: port, Secret: secret}
			c.mu.Unlock()
			if !already {
				close(c.ready)
			}
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/pxAlive", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("alive"))
	})

	srv := &http.Server{Handler: mux}
	c.mu.Lock()
	c.cbServer = srv
	c.mu.Unlock()

	go func() { _ = srv.Serve(ln) }()
	return addr, nil
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
