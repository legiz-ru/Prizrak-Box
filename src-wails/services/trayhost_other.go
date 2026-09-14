//go:build !linux

package services

// TrayHostAvailable reports whether the session can display a tray icon. Windows
// and macOS always can, so the Linux StatusNotifierItem-host question (see
// trayhost_linux.go) does not arise.
func TrayHostAvailable() bool { return true }
