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
//  2. Next to the running executable (Wails dev / portable layout).
//  3. <exeDir>/resources/ (packaged layout, matches the Electron MSI).
//  4. ../src-go/px relative to the working directory (repo dev layout).
func PxBinary() string {
	return resolveBinary("PRIZRAK_PX_BIN", pxExeName(), "src-go")
}

// ServiceBinary returns the path to the px-service binary (TUN helper).
func ServiceBinary() string {
	return resolveBinary("PRIZRAK_PX_SERVICE_BIN", serviceExeName(), "src-service")
}

// resolveBinary applies the shared lookup order for a bundled Go binary.
func resolveBinary(envVar, exeName, devDir string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		// next to the executable
		if c := filepath.Join(dir, exeName); fileExists(c) {
			return c
		}
		// packaged layout: <exeDir>/resources/<exe> (matches the Electron MSI)
		if c := filepath.Join(dir, "resources", exeName); fileExists(c) {
			return c
		}
	}
	// repo dev layout: ../<devDir>/<exe> relative to the working directory
	if wd, err := os.Getwd(); err == nil {
		if c := filepath.Join(wd, "..", devDir, exeName); fileExists(c) {
			return c
		}
	}
	// last resort: rely on PATH
	return exeName
}

// HomeDir returns the per-user data directory passed to px via -home.
//
// We match the Electron build's location ($HOME/Prizrak-Box-V3) so that the
// Wails shell reuses existing profiles/config and the frontend's
// "directory must end with Prizrak-Box-V3" check passes. Override with
// PRIZRAK_HOME to isolate the Wails build's data.
func HomeDir() string {
	if v := os.Getenv("PRIZRAK_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Prizrak-Box-V3")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
