//go:build !windows

package services

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
