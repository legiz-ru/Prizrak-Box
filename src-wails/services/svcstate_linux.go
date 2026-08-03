//go:build linux

package services

import "os"

// systemdUnitPath is where px-service installs its systemd unit
// (see src-service/installer_linux.go).
const systemdUnitPath = "/etc/systemd/system/PrizrakBoxService.service"

// serviceInstalled reports whether the px-service is installed on this machine.
// The bundled binary always exists, so the installed systemd unit is the marker.
func serviceInstalled() bool {
	if !serviceBinaryExists() {
		return false
	}
	info, err := os.Stat(systemdUnitPath)
	return err == nil && !info.IsDir()
}
