// Global "Show/Hide window" hotkey via the Wails v3 GlobalShortcut manager
// (Win32 RegisterHotKey on Windows, Carbon on macOS, X11/portal on Linux).
// This replaces the old Windows-only syscall implementation, so the hotkey now
// works on all three platforms.
//
// The frontend (MyEvent.vue) emits px:fe:shortcut:register {name, key, old} on
// toggle/rebind and px:fe:shortcut:unregister-all on disable, and listens for
// the registration outcome on shortcut:result — useful when the combination is
// already taken by another application (previously failures were silent).
package main

import (
	"runtime"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

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

	var mu sync.Mutex
	current := "" // currently registered accelerator ("" = none)

	// register swaps the active hotkey for key ("" = just unregister) and
	// reports the outcome to the frontend.
	register := func(key string) {
		mu.Lock()
		defer mu.Unlock()
		if current != "" {
			if err := app.GlobalShortcut.Unregister(current); err != nil {
				app.Logger.Error("hotkey unregister failed", "accelerator", current, "error", err)
			}
			current = ""
		}
		if key == "" {
			return
		}
		acc := normalizeAccelerator(key)
		err := app.GlobalShortcut.Register(acc, fire)
		if err != nil {
			app.Logger.Error("hotkey register failed", "accelerator", acc, "error", err)
		} else {
			current = acc
		}
		win.EmitEvent("px:be:shortcut:result", err == nil)
	}

	app.Event.On("px:fe:shortcut:register", func(e *application.CustomEvent) {
		key := ""
		if m := asMap(e.Data); m != nil {
			key = asStr(m["key"])
		}
		if key != "" {
			register(key)
		}
	})
	app.Event.On("px:fe:shortcut:unregister-all", func(_ *application.CustomEvent) {
		register("")
	})
	// The framework unregisters all shortcuts on shutdown.
}

// namedKeyAliases maps the KeyboardEvent.key names the frontend's hotkey
// recorder produces (MyHotkeyInput.vue) to the names Wails' accelerator
// parser expects. Names already matching (escape, enter, delete, home, end,
// tab, backspace, f1..f24, single characters) pass through lowercased.
var namedKeyAliases = map[string]string{
	"arrowup":    "up",
	"arrowdown":  "down",
	"arrowleft":  "left",
	"arrowright": "right",
	"pageup":     "page up",
	"pagedown":   "page down",
	" ":          "space",
	"esc":        "escape",
}

// normalizeAccelerator converts the frontend's "Ctrl+Shift+X"-style string
// (built from KeyboardEvent modifiers in MyHotkeyInput.vue) into a Wails
// accelerator. Modifier notes:
//   - "Cmd" is produced by metaKey: ⌘ on macOS, the Win key on Windows/Linux.
//     Wails maps "cmd" to CmdOrCtrl (⌘/Ctrl), so on macOS it is passed as-is
//     and elsewhere it becomes "super" to keep the Win-key semantics.
func normalizeAccelerator(key string) string {
	// A trailing "+" means the key itself is "+" (e.g. "Ctrl++"); Wails spells
	// that key "plus".
	if strings.HasSuffix(key, "+") {
		key = strings.TrimSuffix(key, "+") + "plus"
	}
	parts := strings.Split(key, "+")
	out := make([]string, 0, len(parts))
	for i, p := range parts {
		last := i == len(parts)-1
		p = strings.TrimSpace(p)
		lp := strings.ToLower(p)
		if !last {
			switch lp {
			case "cmd", "command", "meta", "win":
				if runtime.GOOS == "darwin" {
					lp = "cmd"
				} else {
					lp = "super"
				}
			case "control":
				lp = "ctrl"
			case "option":
				lp = "alt"
			}
			out = append(out, lp)
			continue
		}
		if alias, ok := namedKeyAliases[lp]; ok {
			lp = alias
		}
		out = append(out, lp)
	}
	return strings.Join(out, "+")
}
