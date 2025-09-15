package utils

import (
	"crypto/md5"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// DeviceInfo holds device-specific information for headers
type DeviceInfo struct {
	HWID        string
	OS          string
	OSVersion   string
	DeviceModel string
}

var (
	deviceInfo *DeviceInfo
	deviceOnce sync.Once
)

// generateHWID creates a unique hardware identifier based on system characteristics
func generateHWID() string {
	// Try to get system-specific identifiers
	var identifiers []string
	
	// Add hostname
	if hostname, err := os.Hostname(); err == nil {
		identifiers = append(identifiers, hostname)
	}
	
	// Add OS and architecture info
	identifiers = append(identifiers, runtime.GOOS, runtime.GOARCH)
	
	// Try to get machine-specific info
	if machineID := getMachineID(); machineID != "" {
		identifiers = append(identifiers, machineID)
	}
	
	// Create a hash of the combined identifiers
	combined := strings.Join(identifiers, "-")
	hash := md5.Sum([]byte(combined))
	
	// Convert to UUID format (8-4-4-4-12)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// getMachineID attempts to get a machine-specific identifier
func getMachineID() string {
	switch runtime.GOOS {
	case "linux":
		// Try /etc/machine-id first, then /var/lib/dbus/machine-id
		if id := readFile("/etc/machine-id"); id != "" {
			return strings.TrimSpace(id)
		}
		if id := readFile("/var/lib/dbus/machine-id"); id != "" {
			return strings.TrimSpace(id)
		}
	case "darwin":
		// macOS - try to get hardware UUID
		// This would require calling system commands, for now use a fallback
		return "darwin-machine"
	case "windows":
		// Windows - would need registry access, use fallback for now
		return "windows-machine"
	}
	return ""
}

// readFile safely reads a file and returns its content
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// getOSVersion returns the OS version string
func getOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		// Try to read from /etc/os-release
		if content := readFile("/etc/os-release"); content != "" {
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "VERSION_ID=") {
					version := strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
					return version
				}
			}
		}
		return "unknown"
	case "darwin":
		return "unknown" // Would need system call to get macOS version
	case "windows":
		return "unknown" // Would need Windows API to get version
	default:
		return "unknown"
	}
}

// getDeviceModel returns the device model
func getDeviceModel() string {
	switch runtime.GOOS {
	case "linux":
		// Try to read model from various locations
		if model := readFile("/sys/devices/virtual/dmi/id/product_name"); model != "" {
			return strings.TrimSpace(model)
		}
		if model := readFile("/proc/device-tree/model"); model != "" {
			return strings.TrimSpace(model)
		}
		return "Linux Device"
	case "darwin":
		return "Mac"
	case "windows":
		return "Windows PC"
	default:
		return "Unknown Device"
	}
}

// GetDeviceInfo returns cached device information, initializing it if necessary
func GetDeviceInfo() *DeviceInfo {
	deviceOnce.Do(func() {
		// Generate or load HWID
		hwid := generateHWID()
		
		// Ensure HWID is within 36 character limit
		if len(hwid) > 36 {
			// If longer, create a proper UUID instead
			if generatedUUID, err := uuid.NewRandom(); err == nil {
				hwid = generatedUUID.String()
			} else {
				// Fallback: truncate to 36 chars
				hwid = hwid[:36]
			}
		}
		
		// Get OS name in the format expected by the spec
		osName := runtime.GOOS
		switch osName {
		case "darwin":
			osName = "macOS"
		case "linux":
			osName = "Linux"
		case "windows":
			osName = "Windows"
		}
		
		deviceInfo = &DeviceInfo{
			HWID:        hwid,
			OS:          osName,
			OSVersion:   getOSVersion(),
			DeviceModel: getDeviceModel(),
		}
	})
	
	return deviceInfo
}

// GetDeviceHeaders returns HTTP headers map with device information
func GetDeviceHeaders() map[string]string {
	info := GetDeviceInfo()
	headers := make(map[string]string)
	
	// x-hwid is required
	headers["x-hwid"] = info.HWID
	
	// Optional headers - only add if not empty
	if info.OS != "" {
		headers["x-device-os"] = info.OS
	}
	
	if info.OSVersion != "" && info.OSVersion != "unknown" {
		headers["x-ver-os"] = info.OSVersion
	}
	
	if info.DeviceModel != "" {
		headers["x-device-model"] = info.DeviceModel
	}
	
	return headers
}