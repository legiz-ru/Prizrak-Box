//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// detach re-executes the current binary with the same arguments minus
// -detach, in a new session with stdio pointed at /dev/null, and returns the
// child's pid without waiting for it.
//
// Used on macOS for privilege escalation: `do shell script ... with
// administrator privileges` runs the command via authtrampoline, which kills
// the shell's whole process group once the shell exits. A plain `&` background
// job therefore does not survive, but a process in its own session does.
func detach() (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}

	args := make([]string, 0, len(os.Args))
	for _, a := range os.Args[1:] {
		switch a {
		case "-detach", "--detach", "-detach=true", "--detach=true":
			continue
		}
		args = append(args, a)
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer func() { _ = devNull.Close() }()

	cmd := exec.Command(exe, args...)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	// Do not wait: the child is reparented to launchd/init when we exit.
	return pid, cmd.Process.Release()
}
