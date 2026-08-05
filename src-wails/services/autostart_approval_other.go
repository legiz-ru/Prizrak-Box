//go:build !windows

package services

// autostartBlockedByOS reports whether the OS has an existing autostart entry
// switched off independently of its presence. Only Windows has such a second
// switch (StartupApproved); elsewhere the registration itself is the state.
func autostartBlockedByOS() bool { return false }

// clearAutostartOSBlock is a no-op outside Windows.
func clearAutostartOSBlock() error { return nil }
