//go:build linux

package utils

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func collectDeviceDetails() DeviceDetails {
	details := DeviceDetails{}

	if hwid := readFirstExistingFile(
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	); hwid != "" {
		details.HWID = hwid
	}

	if pretty := readKeyFromFile("/etc/os-release", "PRETTY_NAME"); pretty != "" {
		details.OS = pretty
	} else {
		details.OS = "Linux"
	}

	var uts unix.Utsname
	if err := unix.Uname(&uts); err == nil {
		details.OSVersion = trimCString(uts.Release[:])
	}

	details.Model = readFirstExistingFile(
		"/sys/devices/virtual/dmi/id/product_name",
		"/sys/devices/virtual/dmi/id/board_name",
		"/sys/class/dmi/id/product_name",
	)

	if details.Model == "" {
		details.Model = readFirstExistingFile(
			"/sys/devices/virtual/dmi/id/product_version",
			"/sys/class/dmi/id/product_version",
		)
	}

	return details
}

func readFirstExistingFile(paths ...string) string {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			trimmed := strings.TrimSpace(string(data))
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func readKeyFromFile(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "")
	prefix := key + "="
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimPrefix(line, prefix)
			return strings.Trim(value, `"`)
		}
	}
	return ""
}

func trimCString(buf []byte) string {
	n := 0
	for ; n < len(buf); n++ {
		if buf[n] == 0 {
			break
		}
	}
	return strings.TrimSpace(string(buf[:n]))
}
