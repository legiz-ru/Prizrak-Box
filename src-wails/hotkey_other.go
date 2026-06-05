//go:build !windows

// Global hotkeys are implemented only on Windows for now (RegisterHotKey via
// syscall, no CGO). On macOS/Linux a system-wide hotkey needs platform APIs
// that require CGO, which this build disables, so this is a no-op there.
package main

import "github.com/wailsapp/wails/v3/pkg/application"

func installHotkeys(_ *application.App, _ *application.WebviewWindow) {}
