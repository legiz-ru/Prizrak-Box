//go:build windows

package services

import (
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// windowsPipeName matches src-service/ipc/server.go.
const windowsPipeName = `\\.\pipe\prizrak-box-service`

func dialService(timeout time.Duration) (net.Conn, error) {
	t := timeout
	return winio.DialPipe(windowsPipeName, &t)
}
