//go:build !windows

package services

import (
	"net"
	"runtime"
	"time"
)

// Socket paths, in the order px-service prefers them (src-service/ipc/listen_unix.go).
//
// legacySocketPath is the pre-/run location. It is still tried because the service
// is a separate binary with its own lifecycle: a machine can have an older service
// installed and running while the app has already been updated, and TUN must keep
// working across that gap.
const legacySocketPath = "/tmp/prizrak-box-service.sock"

func primarySocketPath() string {
	if runtime.GOOS == "darwin" {
		return "/var/run/prizrak-box/service.sock"
	}
	return "/run/prizrak-box/service.sock"
}

func dialService(timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("unix", primarySocketPath(), timeout)
	if err == nil {
		return conn, nil
	}
	return net.DialTimeout("unix", legacySocketPath, timeout)
}
