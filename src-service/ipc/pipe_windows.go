//go:build windows

package ipc

import (
	"net"

	"github.com/Microsoft/go-winio"
)

// createWindowsListener создаёт Windows named pipe listener
func createWindowsListener() (net.Listener, error) {
	// Конфигурация pipe с доступом для всех пользователей
	config := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;WD)", // Allow everyone
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	return winio.ListenPipe(WindowsPipeName, config)
}
