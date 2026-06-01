package services

import (
	"os"
	"os/user"
	"runtime"
)

// SystemService exposes small OS helpers to the frontend. It is the Wails
// replacement for the misc window.px* helpers that used to be provided by
// src-electron/preload.ts (pxOs, pxUsername, ...).
//
// Phase 0 keeps this minimal; clipboard / open-external / show-in-folder are
// handled on the frontend via the Wails runtime in the PoC shim and will be
// consolidated here in a later phase.
type SystemService struct{}

// NewSystemService creates a SystemService.
func NewSystemService() *SystemService { return &SystemService{} }

// OS returns a short OS + arch string, e.g. "Linux x64".
func (s *SystemService) OS() string {
	var name string
	switch runtime.GOOS {
	case "windows":
		name = "Windows"
	case "darwin":
		name = "MacOS"
	default:
		name = "Linux"
	}
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "arm64":
		// keep arm64
	}
	return name + " " + arch
}

// Username returns the current user's username (best effort).
func (s *SystemService) Username() string {
	if u, err := user.Current(); err == nil {
		if u.Username != "" {
			return u.Username
		}
	}
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	return os.Getenv("USERNAME")
}
