//go:build !windows

package ipc

import "os"

func isRunningAsAdmin() bool {
	return os.Geteuid() == 0
}
