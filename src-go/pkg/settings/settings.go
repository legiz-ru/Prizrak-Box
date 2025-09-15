package settings

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"github.com/legiz-ru/prizrak-box/api"
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
	settingsFile  string
	initOnce      sync.Once
)

// getUserHomeDir returns the user's home directory
func getUserHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home directory is not available
		if cwd, err := os.Getwd(); err == nil {
			return cwd
		}
		// Ultimate fallback
		return "."
	}
	return homeDir
}

// initSettings initializes settings, loading from file if available
func initSettings() {
	// Get settings file path
	homeDir := getUserHomeDir()
	settingsFile = filepath.Join(homeDir, "prizrak-box-settings.json")
	log.Printf("initSettings: Using settings file: %s", settingsFile)
	
	// Try to load existing settings
	loadSettings()
	
	log.Printf("initSettings: Completed initialization - HWID=%v", settingsInstance.HWID)
}

// loadSettings loads settings from file
func loadSettings() {
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		// File doesn't exist or can't be read, use defaults
		log.Printf("loadSettings: Settings file not found or unreadable, using defaults: %v", err)
		log.Printf("loadSettings: Default settings - HWID=%v", settingsInstance.HWID)
		return
	}
	
	var settings AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		// Invalid JSON, use defaults
		log.Printf("loadSettings: Invalid settings JSON, using defaults: %v", err)
		log.Printf("loadSettings: Default settings - HWID=%v", settingsInstance.HWID)
		return
	}
	
	settingsMutex.Lock()
	defer settingsMutex.Unlock()
	oldHWID := settingsInstance.HWID
	settingsInstance = &settings
	log.Printf("loadSettings: Loaded settings from file: HWID changed from %v to %v", oldHWID, settings.HWID)
}

// saveSettings saves current settings to file
func saveSettings() error {
	data, err := json.MarshalIndent(settingsInstance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	
	log.Printf("saveSettings: Saving settings to %s: %s", settingsFile, string(data))
	
	// Ensure directory exists
	dir := filepath.Dir(settingsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}
	
	// Write to temporary file first, then rename (atomic operation)
	tmpFile := settingsFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}
	
	if err := os.Rename(tmpFile, settingsFile); err != nil {
		os.Remove(tmpFile) // Clean up on failure
		return fmt.Errorf("failed to rename settings file: %w", err)
	}
	
	// Verify the file was written correctly
	if verifyData, err := os.ReadFile(settingsFile); err != nil {
		log.Printf("saveSettings: Warning - could not verify saved file: %v", err)
	} else {
		log.Printf("saveSettings: Verification - file contents: %s", string(verifyData))
	}
	
	return nil
}

// ensureInitialized ensures settings are initialized
func ensureInitialized() {
	initOnce.Do(initSettings)
}

// Get returns the current settings
func Get() *AppSettings {
	ensureInitialized()
	
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	// Return a copy to prevent external mutation
	log.Printf("Get(): Current settings - HWID=%v", settingsInstance.HWID)
	return &AppSettings{
		HWID: settingsInstance.HWID,
	}
}

// SetHWIDSetting updates the HWID setting
func SetHWIDSetting(enabled bool) error {
	ensureInitialized()
	
	settingsMutex.Lock()
	defer settingsMutex.Unlock()
	
	oldValue := settingsInstance.HWID
	settingsInstance.HWID = enabled
	log.Printf("SetHWIDSetting: HWID setting changed from %v to %v", oldValue, enabled)
	
	// Save to file
	if err := saveSettings(); err != nil {
		log.Printf("SetHWIDSetting: Failed to save settings: %v", err)
		return err
	}
	
	log.Printf("SetHWIDSetting: Settings saved successfully to %s", settingsFile)
	
	// Verify the change was applied
	log.Printf("SetHWIDSetting: Verification - Current HWID in memory: %v", settingsInstance.HWID)
	
	return nil
}

// GetUserAgent returns the appropriate User-Agent based on HWID setting
func GetUserAgent() string {
	ensureInitialized()
	
	settings := Get()
	userAgent := ""
	if settings.HWID {
		if api.Version != "" {
			userAgent = fmt.Sprintf("prizrak-box/%s", api.Version)
		} else {
			// Fallback if version is not available
			userAgent = "prizrak-box/v3"
		}
	} else {
		userAgent = "clash-verge/v2.3.0"
	}
	
	log.Printf("GetUserAgent: HWID=%v, returning: %s", settings.HWID, userAgent)
	return userAgent
}

// ShouldIncludeDeviceHeaders returns whether device headers should be included
func ShouldIncludeDeviceHeaders() bool {
	ensureInitialized()
	
	settings := Get()
	log.Printf("ShouldIncludeDeviceHeaders: HWID=%v, returning: %v", settings.HWID, settings.HWID)
	return settings.HWID
}