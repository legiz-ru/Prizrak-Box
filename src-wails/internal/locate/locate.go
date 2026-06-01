// Package locate resolves filesystem paths for the bundled Go backend
// binaries (px / px-service) and the per-user home directory.
//
// This mirrors the logic that used to live in src-electron/admin.ts and
// src-electron/service.ts, but in Go for the Wails v3 shell.
package locate

import (
	"os"
	"path/filepath"
	"runtime"
)

// pxExeName returns the platform-specific filename for the px backend.
func pxExeName() string {
	if runtime.GOOS == "windows" {
		return "px.exe"
	}
	return "px"
}

// serviceExeName returns the platform-specific filename for px-service.
func serviceExeName() string {
	if runtime.GOOS == "windows" {
		return "px-service.exe"
	}
	return "px-service"
}

// PxBinary returns the path to the px backend binary.
//
// Resolution order:
//  1. PRIZRAK_PX_BIN environment variable (explicit override).
//  2. Next to the running executable (packaged layout).
//  3. ../src-go/px relative to the working directory (dev layout).
func PxBinary() string {
	if v := os.Getenv("PRIZRAK_PX_BIN"); v != "" {
		return v
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), pxExeName())
		if fileExists(candidate) {
			return candidate
		}
	}
	// dev layout: repo-root/src-go/px, shell runs from repo-root/src-wails
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "..", "src-go", pxExeName())
		if fileExists(candidate) {
			return candidate
		}
	}
	// last resort: rely on PATH
	return pxExeName()
}

// ServiceBinary returns the path to the px-service binary (TUN helper).
func ServiceBinary() string {
	if v := os.Getenv("PRIZRAK_PX_SERVICE_BIN"); v != "" {
		return v
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), serviceExeName())
		if fileExists(candidate) {
			return candidate
		}
	}
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "..", "src-service", serviceExeName())
		if fileExists(candidate) {
			return candidate
		}
	}
	return serviceExeName()
}

// HomeDir returns the per-user data directory passed to px via -home.
//
// During the migration PoC we deliberately use a dedicated directory so the
// Wails build does not clobber data created by the Electron build. Override
// with PRIZRAK_HOME if you want both shells to share the same profiles.
func HomeDir() string {
	if v := os.Getenv("PRIZRAK_HOME"); v != "" {
		return v
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base, _ = os.UserHomeDir()
	}
	dir := filepath.Join(base, "Prizrak-Box-Wails")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
