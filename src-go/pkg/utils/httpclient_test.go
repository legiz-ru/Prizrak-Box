package utils

import (
	"testing"
)

func TestGenerateHWID(t *testing.T) {
	hwid1 := generateHWID()
	hwid2 := generateHWID()
	
	// HWID should be unique
	if hwid1 == hwid2 {
		t.Errorf("HWID should be unique, got same HWID: %s", hwid1)
	}
	
	// HWID should be 36 characters (UUID format)
	if len(hwid1) != 36 {
		t.Errorf("HWID should be 36 characters, got %d", len(hwid1))
	}
}

func TestBuildDeviceHeaders(t *testing.T) {
	// Test with HWID disabled
	config := &HTTPClientConfig{
		EnableHWID: false,
	}
	UpdateHTTPClientConfig(config)
	
	headers := buildDeviceHeaders()
	if headers != nil {
		t.Errorf("Expected nil headers when HWID is disabled, got %v", headers)
	}
	
	// Test with HWID enabled
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
}

func TestUpdateHTTPClientConfig(t *testing.T) {
	// Test user agent with HWID enabled
	config := &HTTPClientConfig{
		EnableHWID: true,
		Version:    "1.0.1",
	}
	UpdateHTTPClientConfig(config)
	
	expected := "prizrak-box/1.0.1"
	if globalConfig.UserAgent != expected {
		t.Errorf("Expected user agent '%s', got '%s'", expected, globalConfig.UserAgent)
	}
	
	// Test user agent with HWID disabled
	config = &HTTPClientConfig{
		EnableHWID: false,
	}
	UpdateHTTPClientConfig(config)
	
	expected = "clash-verge/v2.3.0"
	if globalConfig.UserAgent != expected {
		t.Errorf("Expected user agent '%s', got '%s'", expected, globalConfig.UserAgent)
	}
}