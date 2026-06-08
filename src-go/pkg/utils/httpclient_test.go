package utils

import (
	"runtime"
	"strings"
	"testing"
)

func TestGenerateHWID(t *testing.T) {
	resetCachedDeviceDetailsForTest()

	hwid1 := generateHWID()
	hwid2 := generateHWID()

	if hwid1 == "" {
		t.Fatal("HWID should not be empty")
	}

	if hwid1 != hwid2 {
		t.Errorf("HWID should be consistent across calls, got '%s' and '%s'", hwid1, hwid2)
	}
}

func TestBuildDeviceHeaders(t *testing.T) {
	resetCachedDeviceDetailsForTest()

	// Test with HWID disabled — no device headers expected
	config := &HTTPClientConfig{
		EnableHWID: false,
	}
	UpdateHTTPClientConfig(config)

	headers := buildDeviceHeaders()
	if headers != nil {
		t.Errorf("Expected nil headers when HWID is disabled, got %v", headers)
	}

	// Test with HWID enabled — all device headers must be present
	config = &HTTPClientConfig{
		EnableHWID:  true,
		DeviceOS:    "Linux",
		DeviceOSVer: "5.4.0",
		DeviceModel: "TestDevice",
	}
	UpdateHTTPClientConfig(config)

	headers = buildDeviceHeaders()
	if headers == nil {
		t.Fatal("Expected headers when HWID is enabled, got nil")
	}

	if _, exists := headers["x-hwid"]; !exists {
		t.Error("Expected x-hwid header")
	}

	if headers["x-device-os"] != "Linux" {
		t.Errorf("Expected x-device-os to be 'Linux', got '%s'", headers["x-device-os"])
	}

	if headers["x-ver-os"] != "5.4.0" {
		t.Errorf("Expected x-ver-os to be '5.4.0', got '%s'", headers["x-ver-os"])
	}

	if headers["x-device-model"] != "TestDevice" {
		t.Errorf("Expected x-device-model to be 'TestDevice', got '%s'", headers["x-device-model"])
	}

	config = &HTTPClientConfig{
		EnableHWID: true,
		DeviceOS:   "Windows x64",
	}
	UpdateHTTPClientConfig(config)

	headers = buildDeviceHeaders()
	if headers["x-device-os"] != "Windows" {
		t.Errorf("Expected x-device-os to be 'Windows', got '%s'", headers["x-device-os"])
	}
}

func TestUpdateHTTPClientConfig(t *testing.T) {
	// Определяем ожидаемое имя ОС для текущей платформы.
	expectedOS := defaultOSName()

	// Test: UA должен иметь единый формат с версией, HWID enabled
	config := &HTTPClientConfig{
		EnableHWID: true,
		Version:    "1.0.1",
	}
	UpdateHTTPClientConfig(config)

	ua := globalConfig.UserAgent
	expectedPrefix := "Clash-Meta/Prizrak-Box (Desktop Build 1.0.1 "
	if !strings.HasPrefix(ua, expectedPrefix) {
		t.Errorf("Expected UA to start with %q, got %q", expectedPrefix, ua)
	}
	if !strings.Contains(ua, expectedOS) {
		t.Errorf("Expected UA to contain OS %q, got %q", expectedOS, ua)
	}

	// Test: UA должен иметь единый формат с версией, HWID disabled
	config = &HTTPClientConfig{
		EnableHWID: false,
		Version:    "1.0.1",
	}
	UpdateHTTPClientConfig(config)

	ua = globalConfig.UserAgent
	if !strings.HasPrefix(ua, expectedPrefix) {
		t.Errorf("Expected UA to start with %q even when HWID disabled, got %q", expectedPrefix, ua)
	}

	// Test: UA без версии — должен содержать только ОС
	config = &HTTPClientConfig{
		EnableHWID: false,
	}
	UpdateHTTPClientConfig(config)

	ua = globalConfig.UserAgent
	if !strings.HasPrefix(ua, "Clash-Meta/Prizrak-Box (Desktop Build ") {
		t.Errorf("Expected UA to start with 'Clash-Meta/Prizrak-Box (Desktop Build ', got %q", ua)
	}
	if !strings.Contains(ua, expectedOS) {
		t.Errorf("Expected UA to contain OS %q, got %q", expectedOS, ua)
	}
}

func TestBuildUserAgent(t *testing.T) {
	cases := []struct {
		version  string
		deviceOS string
		wantContains []string
	}{
		{"1.2.3", "Windows", []string{"Clash-Meta/Prizrak-Box", "Desktop Build", "1.2.3", "Windows"}},
		{"2.0.0", "Linux", []string{"Clash-Meta/Prizrak-Box", "Desktop Build", "2.0.0", "Linux"}},
		{"", "macOS", []string{"Clash-Meta/Prizrak-Box", "Desktop Build", "macOS"}},
		{"1.0.0", "", []string{"Clash-Meta/Prizrak-Box", "Desktop Build", "1.0.0", defaultOSName()}},
	}

	for _, c := range cases {
		ua := buildUserAgent(c.version, c.deviceOS)
		for _, want := range c.wantContains {
			if !strings.Contains(ua, want) {
				t.Errorf("buildUserAgent(%q, %q) = %q, want it to contain %q", c.version, c.deviceOS, ua, want)
			}
		}
	}

	_ = runtime.GOOS // убеждаемся что пакет runtime используется
}
