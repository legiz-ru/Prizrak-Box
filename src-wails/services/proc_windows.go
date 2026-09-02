//go:build windows

package services

import (
	"os/exec"
	"syscall"
)

// hideWindow sets CREATE_NO_WINDOW on the child process so Windows does not
// open a console window when spawning a console-subsystem binary (px) from a
// windowsgui parent.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
