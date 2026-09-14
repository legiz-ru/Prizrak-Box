//go:build !windows

package services

import (
	"os/exec"
	"syscall"
)

func hideWindow(_ *exec.Cmd) {}

// detachProcess puts the child in its own session so it outlives this process.
//
// The deferred relaunch after "Change config dir" sleeps and only then execs the
// app, which means it is still alive when the current instance quits. Without a
// new session it stays in the dying process group and shares the parent's
// controlling terminal, so a group-wide signal or the terminal going away takes
// the pending relaunch down with it and the app never comes back.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
