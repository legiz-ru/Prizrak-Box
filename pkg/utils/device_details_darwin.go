//go:build darwin

package utils

import (
	"os/exec"
	"strings"
)

func collectDeviceDetails() DeviceDetails {
	details := DeviceDetails{}

	if uuidOutput, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output(); err == nil {
		for _, line := range strings.Split(string(uuidOutput), "") {
			if strings.Contains(line, "IOPlatformUUID") {
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					details.HWID = strings.Trim(strings.TrimSpace(parts[1]), `"`)
					break
				}
			}
		}
	}

	details.OS = "macOS"

	if versionOutput, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		details.OSVersion = strings.TrimSpace(string(versionOutput))
	}

	if modelOutput, err := exec.Command("sysctl", "-n", "hw.model").Output(); err == nil {
		details.Model = strings.TrimSpace(string(modelOutput))
	}

	return details
}
