package utils

import (
	"runtime"
	"strings"
	"sync"
)

type DeviceDetails struct {
	HWID      string `json:"hwid"`
	OS        string `json:"os"`
	OSVersion string `json:"osVersion"`
	Model     string `json:"model"`
}

var (
	deviceDetailsOnce   sync.Once
	cachedDeviceDetails DeviceDetails
)

func GetDeviceDetails() DeviceDetails {
	deviceDetailsOnce.Do(func() {
		cachedDeviceDetails = ensureDeviceDetails(collectDeviceDetails())
	})
	return cachedDeviceDetails
}

func generateHWID() string {
	return GetDeviceDetails().HWID
}

func ensureDeviceDetails(details DeviceDetails) DeviceDetails {
	details.HWID = strings.TrimSpace(details.HWID)
	details.OS = normalizeOSName(details.OS)
	details.OSVersion = strings.TrimSpace(details.OSVersion)
	details.Model = strings.TrimSpace(details.Model)

	if details.HWID == "" {
		details.HWID = generateRawHWID()
	}

	if details.OS == "" {
		details.OS = defaultOSName()
	}

	if details.Model == "" {
		details.Model = runtime.GOARCH
	}

	return details
}

func defaultOSName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

func normalizeOSName(value string) string {
	trimmed := strings.TrimSpace(value)
	lower := strings.ToLower(trimmed)

	switch {
	case lower == "":
		return defaultOSName()
	case strings.Contains(lower, "windows"):
		return "Windows"
	case strings.Contains(lower, "macos"), strings.Contains(lower, "mac os"), strings.Contains(lower, "darwin"):
		return "macOS"
	case strings.Contains(lower, "linux"),
		strings.Contains(lower, "ubuntu"),
		strings.Contains(lower, "debian"),
		strings.Contains(lower, "fedora"),
		strings.Contains(lower, "centos"),
		strings.Contains(lower, "red hat"),
		strings.Contains(lower, "rhel"),
		strings.Contains(lower, "suse"),
		strings.Contains(lower, "opensuse"),
		strings.Contains(lower, "arch"),
		strings.Contains(lower, "manjaro"),
		strings.Contains(lower, "mint"),
		strings.Contains(lower, "elementary"),
		strings.Contains(lower, "gentoo"),
		strings.Contains(lower, "alpine"),
		strings.Contains(lower, "void linux"),
		strings.Contains(lower, "endeavour"),
		strings.Contains(lower, "pop!_os"),
		strings.Contains(lower, "pop os"),
		strings.Contains(lower, "zorin"):
		return "Linux"
	default:
		return trimmed
	}
}

func resetCachedDeviceDetailsForTest() {
	deviceDetailsOnce = sync.Once{}
	cachedDeviceDetails = DeviceDetails{}
}
