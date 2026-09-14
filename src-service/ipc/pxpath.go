package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// pxBinaryName is the only filename the service will ever execute.
func pxBinaryName() string {
	if runtime.GOOS == "windows" {
		return "px.exe"
	}
	return "px"
}

// validatePxPath decides whether the service may execute the binary a client
// asked it to start.
//
// start_px runs the requested path with the service's privileges (root /
// LocalSystem) and the IPC endpoint is reachable by any local user — the Unix
// socket was world-accessible and the Windows pipe is created with an
// "allow everyone" DACL. Without this check, asking the service to start
// /tmp/evil was a local privilege escalation to root on every installation.
//
// Accepted: the px that ships next to this service (the packaged layout), or a
// px in a location only the superuser can modify (see trustedSystemPath). Both
// mean the client cannot influence what actually gets executed.
func validatePxPath(requested string) (string, error) {
	if requested == "" {
		return "", fmt.Errorf("pxPath is required")
	}

	resolved, err := filepath.EvalSymlinks(requested)
	if err != nil {
		return "", fmt.Errorf("resolve pxPath %q: %w", requested, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat pxPath %q: %w", resolved, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("pxPath %q is not a regular file", resolved)
	}
	if filepath.Base(resolved) != pxBinaryName() {
		return "", fmt.Errorf("pxPath %q is not %s", resolved, pxBinaryName())
	}

	// The packaged layout puts px and px-service in the same directory, so this
	// is the normal path for every installed build.
	if exe, err := os.Executable(); err == nil {
		if exeResolved, err := filepath.EvalSymlinks(exe); err == nil {
			if filepath.Dir(exeResolved) == filepath.Dir(resolved) {
				return resolved, nil
			}
		}
	}

	if err := trustedSystemPath(resolved); err != nil {
		return "", fmt.Errorf("refusing to run pxPath %q: %w", resolved, err)
	}
	return resolved, nil
}
