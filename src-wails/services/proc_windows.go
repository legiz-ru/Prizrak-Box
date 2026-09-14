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

// detachProcess is a no-op on Windows: the relaunch there goes through
// `cmd /c … start`, which already hands the new process off to the shell.
func detachProcess(_ *exec.Cmd) {}
