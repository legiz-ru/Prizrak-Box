// Config-directory management for the settings page, mirroring the Electron
// shell (src-electron preload `pxConfigDir`/`pxPreConfigDir`/`pxChangeConfigDir`
// and src-electron/change.ts doChange).
//
// These are methods on CoreService (already registered with the Wails app) so
// the frontend can reach them by name via the Wails runtime without any new
// service registration or regenerated bindings.
package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/legiz-ru/prizrak-box-wails/internal/locate"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ConfigDir returns the current px data directory (…/Prizrak-Box-V3).
// Mirrors Electron's `pre-config-dir`.
func (c *CoreService) ConfigDir() string { return locate.HomeDir() }

// OpenConfigDir opens the data directory in the OS file manager.
// Mirrors Electron's `pxConfigDir` (shell.openPath).
func (c *CoreService) OpenConfigDir() error {
	return openInFileManager(locate.HomeDir())
}

// SelectDirectory shows a native folder picker and returns the chosen path, or
// "" if the user cancelled. Mirrors Electron's `select-directory`.
func (c *CoreService) SelectDirectory() (string, error) {
	dir, err := application.Get().Dialog.OpenFile().
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		SetTitle(locate.WorkDirName()).
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return dir, nil
}

// ChangeConfigDir moves the data directory into <dir>/Prizrak-Box-V3, persists
// the new location and relaunches the app — a faithful port of Electron's
// src-electron/change.ts doChange().
func (c *CoreService) ChangeConfigDir(dir string) error {
	if dir == "" {
		return nil
	}
	dest := dir
	if filepath.Base(dest) != locate.WorkDirName() {
		dest = filepath.Join(dir, locate.WorkDirName())
	}

	cur := locate.HomeDir()
	if cur != dest {
		// Stop the locally-spawned px so the data files aren't held open while
		// we move them. A service-managed px self-exits once the callback server
		// stops answering /pxAlive (the relaunch delay below covers that).
		c.KillPx()
		time.Sleep(400 * time.Millisecond)
		if err := moveDir(cur, dest); err != nil {
			return fmt.Errorf("move %q -> %q: %w", cur, dest, err)
		}
	}

	if err := locate.SetHomeOverride(dest); err != nil {
		return fmt.Errorf("persist config dir: %w", err)
	}

	relaunchApp()
	return nil
}

// openInFileManager opens p in the OS file manager.
func openInFileManager(p string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", p)
	case "darwin":
		cmd = exec.Command("open", p)
	default:
		cmd = exec.Command("xdg-open", p)
	}
	return cmd.Start()
}

// moveDir moves src to dst. It tries an atomic rename first and falls back to a
// recursive copy for cross-device moves. The source is only removed after a
// fully successful copy, so an interrupted move never loses data.
func moveDir(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyTree(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// relaunchApp starts a fresh instance of the running executable and quits the
// current one (Wails v3 has no native Relaunch). The relaunch is delayed so the
// single-instance lock is released and any service-managed px has self-exited
// before the new instance (and its px) start.
func relaunchApp() {
	if exe, err := os.Executable(); err == nil {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "ping 127.0.0.1 -n 6 >nul & start \"\" \""+exe+"\"")
		} else {
			cmd = exec.Command("sh", "-c", "sleep 5; exec \""+exe+"\"")
		}
		_ = cmd.Start()
	}
	application.Get().Quit()
}
