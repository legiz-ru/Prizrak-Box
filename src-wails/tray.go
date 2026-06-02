package main

import (
	"os/exec"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// trayController builds the dynamic system-tray menu from data the Vue frontend
// pushes over Wails events, mirroring src-electron/tray.ts.
//
// Inbound events (frontend -> Go), via window.pxTray.emit:
//   translate   map[trayID]label      localised labels
//   mode        "rule"|"global"|"direct"
//   proxy       bool                  system-proxy on/off
//   tun         bool                  TUN on/off
//   profiles    [{title,selected,...}]
//   proxyGroups [{name,proxies:[{name,now}]}]
//   dashboards  [{name,url,key}]
//
// Outbound events (Go -> frontend), via window.pxTray.on:
//   switchMode <mode> | switchProfiles {profile,selected,exclusive}
//   switchProxyInGroup {group,proxy} | switchProxy | switchTun | readyToQuit
type trayController struct {
	app  *application.App
	win  *application.WebviewWindow
	tray *application.SystemTray

	mu         sync.Mutex
	labels     map[string]string
	mode       string
	proxy      bool
	tun        bool
	profiles   []any
	groups     []any
	dashboards []any
}

func setupTray(app *application.App, win *application.WebviewWindow) *trayController {
	c := &trayController{app: app, win: win, labels: map[string]string{}}

	c.tray = app.SystemTray.New()
	c.tray.SetTooltip("Prizrak-Box")
	if runtime.GOOS == "darwin" {
		c.tray.SetTemplateIcon(trayIconMac)
	} else {
		c.tray.SetIcon(trayIcon)
	}

	// Inbound state from the frontend (px:fe:* channels) → update + rebuild.
	app.Event.On("px:fe:translate", func(e *application.CustomEvent) {
		if m := asMap(e.Data); m != nil {
			c.mu.Lock()
			for k, v := range m {
				c.labels[k] = asStr(v)
			}
			c.mu.Unlock()
		}
		c.rebuild()
	})
	app.Event.On("px:fe:mode", func(e *application.CustomEvent) { c.set(func() { c.mode = asStr(e.Data) }) })
	app.Event.On("px:fe:proxy", func(e *application.CustomEvent) { c.set(func() { c.proxy = asBool(e.Data) }) })
	app.Event.On("px:fe:tun", func(e *application.CustomEvent) { c.set(func() { c.tun = asBool(e.Data) }) })
	app.Event.On("px:fe:profiles", func(e *application.CustomEvent) { c.set(func() { c.profiles = asArr(e.Data) }) })
	app.Event.On("px:fe:proxyGroups", func(e *application.CustomEvent) { c.set(func() { c.groups = asArr(e.Data) }) })
	app.Event.On("px:fe:dashboards", func(e *application.CustomEvent) { c.set(func() { c.dashboards = asArr(e.Data) }) })

	// Initial build runs on the main thread (we're still inside main(), before
	// app.Run()), so build directly — InvokeSync would dereference a nil
	// main-thread dispatcher and crash.
	c.buildMenu()
	return c
}

func (c *trayController) set(mutate func()) {
	c.mu.Lock()
	mutate()
	c.mu.Unlock()
	c.rebuild()
}

func (c *trayController) label(id, fallback string) string {
	if v := c.labels[id]; v != "" {
		return v
	}
	return fallback
}

// rebuild reassigns the menu from an event callback (a background goroutine),
// so it must hop onto the main thread.
func (c *trayController) rebuild() {
	application.InvokeSync(c.buildMenu)
}

// buildMenu constructs a fresh menu from the current state and assigns it to
// the tray. It MUST be called on the main thread.
func (c *trayController) buildMenu() {
	c.mu.Lock()
	defer c.mu.Unlock()

	menu := c.app.NewMenu()

	menu.Add(c.label("tray.show", "Show")).OnClick(func(_ *application.Context) {
		c.win.Show()
		c.win.Focus()
	})
	menu.AddSeparator()

	c.addMode(menu, "tray.rule", "Rule", "rule")
	c.addMode(menu, "tray.global", "Global", "global")
	c.addMode(menu, "tray.direct", "Direct", "direct")
	menu.AddSeparator()

	// Profiles (multi-select checkboxes).
	profiles := menu.AddSubmenu(c.label("tray.profiles", "Profiles"))
	for _, p := range c.profiles {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		title := asStr(pm["title"])
		selected := asBool(pm["selected"])
		profile := pm
		profiles.AddCheckbox(title, selected).OnClick(func(_ *application.Context) {
			c.emit("switchProfiles", map[string]any{
				"profile":   profile,
				"selected":  true,
				"exclusive": false,
			})
		})
	}

	// Proxy groups (each group is a submenu of radio items).
	groups := menu.AddSubmenu(c.label("tray.proxyGroups", "Proxy Groups"))
	for _, g := range c.groups {
		gm := asMap(g)
		if gm == nil {
			continue
		}
		gname := asStr(gm["name"])
		proxies := asArr(gm["proxies"])
		if gname == "" || len(proxies) == 0 {
			continue
		}
		sub := groups.AddSubmenu(gname)
		for _, pr := range proxies {
			prm := asMap(pr)
			if prm == nil {
				continue
			}
			pname := asStr(prm["name"])
			now := asBool(prm["now"])
			group, proxy := gname, pname
			sub.AddRadio(pname, now).OnClick(func(_ *application.Context) {
				c.emit("switchProxyInGroup", map[string]any{"group": group, "proxy": proxy})
			})
		}
	}

	// Dashboards (open external URL).
	dash := menu.AddSubmenu(c.label("tray.dashboard", "Open Dashboard"))
	for _, d := range c.dashboards {
		dm := asMap(d)
		if dm == nil {
			continue
		}
		name := asStr(dm["name"])
		url := asStr(dm["url"])
		if name == "" || url == "" {
			continue
		}
		u := url
		dash.Add(name).OnClick(func(_ *application.Context) { openExternal(u) })
	}
	menu.AddSeparator()

	menu.AddCheckbox(c.label("tray.proxy", "System Proxy"), c.proxy).OnClick(func(_ *application.Context) {
		c.emit("switchProxy", nil)
	})
	menu.AddCheckbox(c.label("tray.tun", "TUN"), c.tun).OnClick(func(_ *application.Context) {
		c.emit("switchTun", nil)
	})
	menu.AddSeparator()

	menu.Add(c.label("tray.quit", "Quit")).OnClick(func(_ *application.Context) {
		// Graceful: the frontend disables the system proxy (api.exit) before
		// the app quits via the doQuit event.
		c.emit("readyToQuit", nil)
	})

	c.tray.SetMenu(menu)
}

func (c *trayController) addMode(menu *application.Menu, id, fallback, mode string) {
	menu.AddCheckbox(c.label(id, fallback), c.mode == mode).OnClick(func(_ *application.Context) {
		c.emit("switchMode", mode)
	})
}

func (c *trayController) emit(name string, data any) {
	// px:be:* = Go -> frontend (the shim's pxTray.on listens on these).
	if data == nil {
		c.win.EmitEvent("px:be:" + name)
		return
	}
	c.win.EmitEvent("px:be:"+name, data)
}

// openExternal opens a URL in the default browser.
func openExternal(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// --- small JSON coercion helpers (event data arrives as decoded JSON) ---

func asArr(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	return nil
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func asStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	case float64:
		return t != 0
	default:
		return false
	}
}
