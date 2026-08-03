//go:build !windows

package services

import "github.com/wailsapp/wails/v3/pkg/application"

// autostartEnabled reports whether launch-at-login is registered, via the
// Wails v3 Autostart manager (LaunchAgent on macOS, .desktop autostart on
// Linux).
func autostartEnabled() bool {
	enabled, err := application.Get().Autostart.IsEnabled()
	return err == nil && enabled
}

// setAutostart enables or disables launch-at-login via the Wails v3
// Autostart manager.
func setAutostart(enabled bool) error {
	am := application.Get().Autostart
	if enabled {
		return am.Enable()
	}
	return am.Disable()
}
