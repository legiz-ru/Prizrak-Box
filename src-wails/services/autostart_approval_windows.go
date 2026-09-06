//go:build windows

// Windows keeps a second, independent switch for startup entries: Task Manager
// → Startup apps (and the Settings → Apps → Startup toggle) do not delete the
// HKCU\…\Run value, they flip a flag under StartupApproved. An entry can
// therefore exist and look perfectly registered while Windows refuses to launch
// it, which is invisible to anything that only inspects the Run key — including
// the Wails autostart manager.
package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	autostartRunSubKey = `Software\Microsoft\Windows\CurrentVersion\Run`
	// StartupApproved\Run holds the approval flags for HKCU Run entries.
	// (StartupFolder / Run32 exist too, but neither applies here: the entry is
	// written to HKCU Run by a 64-bit process, and HKCU is not redirected.)
	startupApprovedSubKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
)

// autostartRunValueName returns the name of the HKCU\…\Run value pointing at
// the running executable, or "" when there is none.
//
// The name is whatever the Wails autostart manager derived from the application
// name; resolving it from the registry instead of re-deriving it keeps this
// working if that derivation ever changes.
func autostartRunValueName() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, autostartRunSubKey, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		return ""
	}
	target := strings.ToLower(exe)
	for _, name := range names {
		cmd, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(cmd), target) {
			return name
		}
	}
	return ""
}

// startupApprovalDisabled reports whether the given Run value is switched off
// under StartupApproved.
func startupApprovalDisabled(valueName string) bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupApprovedSubKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	data, _, err := key.GetBinaryValue(valueName)
	if err != nil || len(data) == 0 {
		// No approval record at all means the entry was never disabled.
		return false
	}
	// Byte 0 carries the state (2/6 = enabled, 3/7 = disabled); the low bit is
	// the "disabled" marker. The remaining bytes are the FILETIME of the change.
	return data[0]&1 == 1
}

// autostartBlockedByOS reports whether an existing autostart entry is switched
// off by Windows itself, so the app is registered but will not be launched.
func autostartBlockedByOS() bool {
	name := autostartRunValueName()
	return name != "" && startupApprovalDisabled(name)
}

// clearAutostartOSBlock re-approves an entry that was switched off in Task
// Manager, because the user has just asked for autostart through the app's own
// setting — which is the control they are expected to use to turn it back off.
// A no-op when there is no entry or it is not disabled.
func clearAutostartOSBlock() error {
	name := autostartRunValueName()
	if name == "" || !startupApprovalDisabled(name) {
		return nil
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, startupApprovedSubKey, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	defer key.Close()

	// 0x02 = enabled, followed by a zeroed FILETIME (no disable timestamp),
	// which is exactly what Windows writes when an entry is re-enabled.
	enabled := []byte{0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	return key.SetBinaryValue(name, enabled)
}
