//go:build !windows

package services

import (
	"net"
	"time"
)

// unixSocketPath matches src-service/ipc/server.go.
const unixSocketPath = "/tmp/prizrak-box-service.sock"

func dialService(timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", unixSocketPath, timeout)
}
