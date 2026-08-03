//go:build darwin

package services

import "os"

// launchDaemonPlist is where px-service installs its launchd job
// (see src-service/installer_darwin.go).
const launchDaemonPlist = "/Library/LaunchDaemons/com.prizrak-box.service.plist"

// serviceInstalled reports whether the px-service is installed on this machine.
//
// The px-service binary is shipped *inside* the .app bundle, so its presence
// says nothing about installation state — checking it (as this used to) made
// GetStatus always report "installed" on macOS, even right after an uninstall.
// The launchd plist is the actual marker: it is created on install and removed
// on uninstall.
func serviceInstalled() bool {
	if !serviceBinaryExists() {
		return false
	}
	info, err := os.Stat(launchDaemonPlist)
	return err == nil && !info.IsDir()
}
