//go:build !windows

package services

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	// linuxUnitName matches serviceFileName in src-service/installer_linux.go.
	linuxUnitName = "PrizrakBoxService.service"
	// darwinServiceID matches serviceName in src-service/installer_darwin.go.
	darwinServiceID = "com.prizrak-box.service"
)

// queryServiceState asks the platform service manager about px-service.
func queryServiceState() serviceState {
	switch runtime.GOOS {
	case "linux":
		return queryServiceStateLinux()
	case "darwin":
		return queryServiceStateDarwin()
	}
	return serviceState{}
}

func queryServiceStateLinux() serviceState {
	installed := false
	// The unit is written to /etc/systemd/system by the installer; distro
	// packages may ship it under /usr/lib or /lib instead.
	for _, dir := range []string{"/etc/systemd/system/", "/usr/lib/systemd/system/", "/lib/systemd/system/"} {
		if _, err := os.Stat(dir + linuxUnitName); err == nil {
			installed = true
			break
		}
	}
	if !installed {
		return serviceState{}
	}

	state := serviceState{Installed: true}
	// systemctl exits non-zero for inactive units but still prints the state,
	// so the error is deliberately ignored in favour of the output.
	out, _ := exec.Command("systemctl", "is-active", linuxUnitName).Output()
	switch strings.TrimSpace(string(out)) {
	case "active":
		state.Running = true
	case "activating", "reloading":
		state.Starting = true
	}
	out, _ = exec.Command("systemctl", "is-enabled", linuxUnitName).Output()
	switch strings.TrimSpace(string(out)) {
	case "disabled", "masked":
		state.Disabled = true
	}
	return state
}

func queryServiceStateDarwin() serviceState {
	if _, err := os.Stat("/Library/LaunchDaemons/" + darwinServiceID + ".plist"); err != nil {
		return serviceState{}
	}
	state := serviceState{Installed: true}
	// A non-zero exit means the daemon is not loaded into launchd at all.
	out, err := exec.Command("launchctl", "print", "system/"+darwinServiceID).Output()
	if err != nil {
		return state
	}
	if strings.Contains(string(out), "state = running") {
		state.Running = true
	} else {
		state.Starting = true
	}
	return state
}
