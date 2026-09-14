//go:build !windows

package ipc

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

// LegacyUnixSocketPath is the pre-/run location of the socket. It is still served
// so a freshly installed service keeps working with an app build that only knows
// the old path; it can be dropped once no such build is in use.
const LegacyUnixSocketPath = "/tmp/prizrak-box-service.sock"

// UnixSocketPath is where clients should look first.
//
// The runtime directory is root-owned and not world-writable, unlike /tmp where
// the socket used to live: there any local user could pre-create the path to keep
// the service from starting, and a systemd unit hardened with PrivateTmp= would
// have hidden the socket from the GUI entirely.
func UnixSocketPath() string {
	if runtime.GOOS == "darwin" {
		// macOS has no /run; /var/run is the equivalent.
		return "/var/run/prizrak-box/service.sock"
	}
	return "/run/prizrak-box/service.sock"
}

// socketPaths lists every path the server binds, most preferred first.
func socketPaths() []string {
	return []string{UnixSocketPath(), LegacyUnixSocketPath}
}

// createListeners binds the service's Unix sockets.
//
// The socket mode stays permissive because any desktop user's GUI has to be able
// to connect; commands are authorized individually instead (see handleRequest,
// validatePxPath and the data-directory ownership check), which is what actually
// closes the "tell the root service to run arbitrary code" hole.
func createListeners() ([]net.Listener, error) {
	var listeners []net.Listener
	for _, path := range socketPaths() {
		if dir := filepath.Dir(path); dir != "/tmp" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				log.Printf("[IPC] cannot create %s (%v); skipping %s", dir, err, path)
				continue
			}
		}
		// A stale socket from a crashed run would make bind fail.
		_ = os.Remove(path)
		ln, err := net.Listen("unix", path)
		if err != nil {
			log.Printf("[IPC] cannot listen on %s: %v", path, err)
			continue
		}
		if err := os.Chmod(path, 0o666); err != nil {
			log.Printf("[IPC] cannot chmod %s: %v", path, err)
		}
		log.Printf("[IPC] listening on %s", path)
		listeners = append(listeners, ln)
	}
	if len(listeners) == 0 {
		return nil, fmt.Errorf("could not listen on any of %v", socketPaths())
	}
	return listeners, nil
}

// removeSockets cleans up the socket files on shutdown.
func removeSockets() {
	for _, path := range socketPaths() {
		_ = os.Remove(path)
	}
}
