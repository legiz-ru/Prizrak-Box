package settings

import (
	"fmt"
	"github.com/legiz-ru/prizrak-box/api"
	"sync"
)

// AppSettings holds application settings
type AppSettings struct {
	HWID bool `json:"hwid"`
}

var (
	settingsInstance = &AppSettings{
		HWID: true, // Default to enabled
	}
	settingsMutex sync.RWMutex
)

// Get returns the current settings
func Get() *AppSettings {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	// Return a copy to prevent external mutation
	return &AppSettings{
		HWID: settingsInstance.HWID,
	}
}

// SetHWIDSetting updates the HWID setting
func SetHWIDSetting(enabled bool) error {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()
	settingsInstance.HWID = enabled
	return nil
}

// GetUserAgent returns the appropriate User-Agent based on HWID setting
func GetUserAgent() string {
	settings := Get()
	if settings.HWID {
		if api.Version != "" {
			return fmt.Sprintf("prizrak-box/%s", api.Version)
		}
		// Fallback if version is not available
		return "prizrak-box/v3"
	}
	return "clash-verge/v2.3.0"
}

// ShouldIncludeDeviceHeaders returns whether device headers should be included
func ShouldIncludeDeviceHeaders() bool {
	settings := Get()
	return settings.HWID
}