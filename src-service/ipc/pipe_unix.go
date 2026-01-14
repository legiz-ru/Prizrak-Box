//go:build !windows

package ipc

import (
	"net"
)

// createWindowsListener не используется на Unix
func createWindowsListener() (net.Listener, error) {
	// На Unix используем Unix socket, который создаётся в Start()
	return nil, nil
}
