//go:build windows

// ShowInFolder reveals a file in Explorer with it selected, mirroring Electron's
// shell.showItemInFolder on Windows.
package services

import (
	"os/exec"
	"syscall"
)

// ShowInFolder opens Explorer with the given file highlighted. The command line
// is built by hand so the path is quoted on its own (explorer needs
//
//	explorer.exe /select,"<path>"
//
// with the quotes around the PATH only) — otherwise paths containing spaces
// (e.g. C:\Program Files\…) open the default folder instead of selecting. We
// don't wait on the process: explorer returns a non-zero exit code even on
// success.
func (c *CoreService) ShowInFolder(path string) error {
	if path == "" {
		return nil
	}
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + path + `"`}
	return cmd.Start()
}
