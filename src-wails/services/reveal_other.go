//go:build !windows

// ShowInFolder reveals a file in the OS file manager, mirroring Electron's
// shell.showItemInFolder: macOS selects the file in Finder; Linux (and other
// Unix) opens the containing directory — exactly what Electron does on Linux,
// which opens the parent folder rather than highlighting the file.
package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func (c *CoreService) ShowInFolder(path string) error {
	if path == "" {
		return nil
	}
	if runtime.GOOS == "darwin" {
		// -R reveals the file in Finder with it selected.
		return exec.Command("open", "-R", path).Start()
	}
	// Linux & other Unix: open the parent directory (Electron's behaviour).
	dir := filepath.Dir(path)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		dir = path
	}
	return exec.Command("xdg-open", dir).Start()
}
