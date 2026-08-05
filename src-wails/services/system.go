package services

import (
	"os"
	"os/user"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
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

// AutostartEnabled reports whether launch-at-login is currently registered AND
// will actually run. Registration comes from the built-in Wails v3 Autostart
// manager (LaunchAgent / registry Run / .desktop autostart); on Windows the
// entry can additionally be switched off in Task Manager without being removed,
// which counts as disabled here (see autostart_approval_windows.go).
func (s *SystemService) AutostartEnabled() bool {
	enabled, err := application.Get().Autostart.IsEnabled()
	if err != nil || !enabled {
		return false
	}
	return !autostartBlockedByOS()
}

// SetAutostart enables or disables launch-at-login.
func (s *SystemService) SetAutostart(enabled bool) error {
	am := application.Get().Autostart
	if !enabled {
		return am.Disable()
	}
	if err := am.Enable(); err != nil {
		return err
	}
	// Writing the Run value does not undo a Task Manager "disable", so an entry
	// switched off there would silently never launch.
	return clearAutostartOSBlock()
}
