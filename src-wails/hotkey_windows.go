//go:build windows

// Global hotkey support on Windows via the Win32 RegisterHotKey API (pure
// syscall, no CGO). Wails v3 has no cross-platform global-shortcut API, and the
// "Show/Hide window" hotkey must work while the window is hidden/unfocused, so a
// system-wide hotkey is required. This mirrors src-electron/shortcut.ts.
package main

import (
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
)

const (
	modAlt     = 0x0001
	modControl = 0x0002
	modShift   = 0x0004
	modWin     = 0x0008

	wmHotkey = 0x0312
	pmRemove = 0x0001
	hotkeyID = 1
)

type win32Msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	ptX     int32
	ptY     int32
}

type hotkeyCmd struct {
	key    string
	enable bool
}

var (
	hotkeyCh   = make(chan hotkeyCmd, 8)
	hotkeyOnce sync.Once
	hotkeyFire func()
)

// installHotkeys starts the Win32 hotkey thread and wires the frontend's
// shortcut events to it (px:fe:shortcut:register / :unregister-all, emitted by
// MyEvent.vue via window.pxTray.emit).
func installHotkeys(app *application.App, win *application.WebviewWindow) {
	fire := func() {
		application.InvokeAsync(func() {
			if win.IsVisible() {
				win.Hide()
			} else {
				win.Show()
				win.Focus()
			}
		})
	}
	hotkeyOnce.Do(func() {
		hotkeyFire = fire
		go hotkeyLoop()
	})

	app.Event.On("px:fe:shortcut:register", func(e *application.CustomEvent) {
		key := ""
		if m := asMap(e.Data); m != nil {
			key = asStr(m["key"])
		}
		if key != "" {
			setHotkey(key, true)
		}
	})
	app.Event.On("px:fe:shortcut:unregister-all", func(_ *application.CustomEvent) {
		setHotkey("", false)
	})
}

func setHotkey(key string, enable bool) {
	select {
	case hotkeyCh <- hotkeyCmd{key: key, enable: enable}:
	default:
	}
}

// hotkeyLoop owns the hotkey on a dedicated OS thread: RegisterHotKey ties the
// hotkey (and its WM_HOTKEY messages) to the calling thread, so registration and
// the message pump must share one locked thread. PeekMessage is used (non-
// blocking) so we can also process register/unregister commands on the same
// thread.
func hotkeyLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	registered := false
	var msg win32Msg

	for {
		select {
		case cmd := <-hotkeyCh:
			if registered {
				procUnregisterHotKey.Call(0, hotkeyID)
				registered = false
			}
			if cmd.enable {
				mods, vk := parseAccelerator(cmd.key)
				if vk != 0 && mods != 0 {
					r, _, _ := procRegisterHotKey.Call(0, hotkeyID, uintptr(mods), uintptr(vk))
					registered = r != 0
				}
			}
		default:
		}

		for {
			r, _, _ := procPeekMessageW.Call(
				uintptr(unsafe.Pointer(&msg)), 0, 0, 0, pmRemove)
			if r == 0 {
				break
			}
			if msg.message == wmHotkey && hotkeyFire != nil {
				hotkeyFire()
			}
		}

		time.Sleep(40 * time.Millisecond)
	}
}

// parseAccelerator turns an accelerator like "Ctrl+Shift+X" into Win32
// modifier flags and a virtual-key code. Returns vk==0 on failure.
func parseAccelerator(acc string) (mods uint32, vk uint32) {
	for _, part := range strings.Split(acc, "+") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "ctrl", "control", "cmd", "command", "cmdorctrl":
			mods |= modControl
		case "shift":
			mods |= modShift
		case "alt", "option":
			mods |= modAlt
		case "win", "super", "meta":
			mods |= modWin
		case "":
			// ignore
		default:
			vk = virtualKey(strings.TrimSpace(part))
		}
	}
	return mods, vk
}

// virtualKey maps a key token to a Windows virtual-key code (letters, digits,
// function keys). Returns 0 if unknown.
func virtualKey(k string) uint32 {
	if len(k) == 1 {
		c := k[0]
		if c >= 'a' && c <= 'z' {
			c -= 32 // to upper; VK matches ASCII uppercase
		}
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return uint32(c)
		}
	}
	u := strings.ToUpper(k)
	if len(u) >= 2 && u[0] == 'F' {
		// F1..F24 → 0x70..
		n := 0
		for _, d := range u[1:] {
			if d < '0' || d > '9' {
				return 0
			}
			n = n*10 + int(d-'0')
		}
		if n >= 1 && n <= 24 {
			return uint32(0x70 + (n - 1))
		}
	}
	return 0
}
