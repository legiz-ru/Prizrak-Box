package utils

import (
	"regexp"
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
	expectedPrefix := "prizrak-box/1.0.1 (Desktop Build; "
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

	// Test: UA без версии — версия "unknown", слэш после имени сохраняется
	config = &HTTPClientConfig{
		EnableHWID: false,
	}
	UpdateHTTPClientConfig(config)

	ua = globalConfig.UserAgent
	if !strings.HasPrefix(ua, "prizrak-box/unknown (Desktop Build; ") {
		t.Errorf("Expected UA to start with 'prizrak-box/unknown (Desktop Build; ', got %q", ua)
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
		{"1.2.3", "Windows", []string{"prizrak-box/1.2.3 (", "Desktop Build", "Windows OS"}},
		{"2.0.0", "Linux", []string{"prizrak-box/2.0.0 (", "Desktop Build", "Linux OS"}},
		{"", "macOS", []string{"prizrak-box/unknown (", "Desktop Build", "macOS OS"}},
		{"1.0.0", "", []string{"prizrak-box/1.0.0 (", "Desktop Build", defaultOSName() + " OS"}},
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

func TestBuildUserAgentCoreVersion(t *testing.T) {
	saved := coreVersion
	defer func() { coreVersion = saved }()

	coreVersion = "v1.19.32-r1"
	if got, want := buildUserAgent("1.2.3", "Windows"), "prizrak-box/1.2.3 (Desktop Build; Windows OS; Prizrak-Core v1.19.32-r1)"; got != want {
		t.Errorf("buildUserAgent = %q, want %q", got, want)
	}

	coreVersion = ""
	if got, want := buildUserAgent("1.2.3", "Windows"), "prizrak-box/1.2.3 (Desktop Build; Windows OS)"; got != want {
		t.Errorf("buildUserAgent = %q, want %q", got, want)
	}
}

// Remnawave sends serverDescription only to "extended" clients, recognised by
// the regex /^prizrak-box\// on the User-Agent (remnawave/backend,
// extended-clients.ts). The UA must keep matching it.
func TestBuildUserAgentMatchesRemnawaveExtendedClient(t *testing.T) {
	extended := regexp.MustCompile(`^prizrak-box/`)
	for _, version := range []string{"1.0.21-beta06", ""} {
		if ua := buildUserAgent(version, "Windows"); !extended.MatchString(ua) {
			t.Errorf("buildUserAgent(%q) = %q does not match %v", version, ua, extended)
		}
	}
}
