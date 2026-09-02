//go:build darwin

package utils

import (
	"encoding/base64"
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

	// Prefer the human-readable marketing name (e.g. "MacBook Air (13-inch, M5)")
	// which Apple Silicon Macs expose in IORegistry as the `product-name`
	// property. Fall back to the model identifier from `hw.model`
	// (e.g. "Mac17,3") on Intel / older Macs where product-name is absent.
	if name := macMarketingName(); name != "" {
		details.Model = name
	} else if modelOutput, err := exec.Command("sysctl", "-n", "hw.model").Output(); err == nil {
		details.Model = strings.TrimSpace(string(modelOutput))
	}

	return details
}

// macMarketingName reads the marketing model name from the IORegistry
// `product-name` property. Returns "" if it can't be determined.
func macMarketingName() string {
	out, err := exec.Command("ioreg", "-arc", "IOPlatformDevice", "-k", "product-name").Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	s := string(out)

	idx := strings.Index(s, "<key>product-name</key>")
	if idx < 0 {
		return ""
	}
	rest := s[idx+len("<key>product-name</key>"):]

	// product-name is stored as OSData: <data>base64(C-string)</data>.
	if data := between(rest, "<data>", "</data>"); data != "" {
		if decoded, derr := base64.StdEncoding.DecodeString(strings.TrimSpace(data)); derr == nil {
			return strings.Trim(string(decoded), "\x00\n\r\t ")
		}
	}
	// Some macOS versions may render it as a plain string.
	if str := between(rest, "<string>", "</string>"); str != "" {
		return strings.TrimSpace(str)
	}
	return ""
}

func between(s, open, close string) string {
	start := strings.Index(s, open)
	if start < 0 {
		return ""
	}
	start += len(open)
	end := strings.Index(s[start:], close)
	if end < 0 {
		return ""
	}
	return s[start : start+end]
}
