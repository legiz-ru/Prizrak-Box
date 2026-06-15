// Command prizrak-box-wails is the Wails v3 desktop shell for Prizrak-Box.
//
// Phase 0 (PoC): boot Wails v3, serve the existing Vue frontend, spawn/supervise
// px, hand the frontend host/port/secret, native tray, single-instance lock.
//
// Phase 1: TUN service management (TunService), launch-at-login (SystemService
// via Wails Autostart), deep-link handling for the prizrak-box:// scheme
// (ApplicationLaunchedWithUrl + second-instance argv).
//
// Phase 1.1 (this revision): window controls + quit wired from the frontend's
// existing pxTray events (close/min/max/hide/doQuit/boot), macOS hidden-inset
// title bar to match the Electron look, and the correct monochrome tray icon.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/legiz-ru/prizrak-box-wails/internal/locate"
	"github.com/legiz-ru/prizrak-box-wails/services"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/tray.png
var trayIcon []byte

// Windows tray icons: per-theme tiles (Wails swaps light/dark by taskbar theme)
// with an "active" variant carrying a green badge when TUN or system proxy is on.
// These are single-image PNGs (CreateIconFromResourceEx, used by Wails' Windows
// systray, accepts a PNG — not a multi-image .ico container).
//
//go:embed build/tray-win-light.png
var trayWinLight []byte

//go:embed build/tray-win-dark.png
var trayWinDark []byte

//go:embed build/tray-win-light-active.png
var trayWinLightActive []byte

//go:embed build/tray-win-dark-active.png
var trayWinDarkActive []byte

//go:embed build/tray-macos.png
var trayIconMac []byte

// deepLinkScheme is the custom URL scheme. Registration with the OS happens at
// packaging time (build/config.yml -> Info.plist / NSIS); see README.
const deepLinkScheme = "prizrak-box"

func main() {
	distFS, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatalf("embed frontend: %v", err)
	}

	core := services.NewCoreService()
	system := services.NewSystemService()
	tun := services.NewTunService(core)

	var win *application.WebviewWindow

	// On macOS the dock/Cmd+Tab icon comes from the .app bundle's appicon.icns
	// (CFBundleIconFile). Passing Icon here would call [NSApp setApplicationIconImage:]
	// with a raw PNG whose NSImage natural size makes it appear larger than other
	// apps. Leave Icon nil on darwin so the OS uses the bundle icon directly.
	var runtimeIcon []byte
	if runtime.GOOS != "darwin" {
		runtimeIcon = appIcon
	}

	app := application.New(application.Options{
		Name:        "Prizrak-Box",
		Description: "A Simple Mihomo GUI",
		Icon:        runtimeIcon,
		Services: []application.Service{
			application.NewService(core),
			application.NewService(system),
			application.NewService(tun),
		},
		// Keep Wails' own logging quiet (px already logs plenty); the noisy
		// per-request asset logs and benign "Window #N not found" warnings on
		// shutdown are suppressed.
		LogLevel: slog.LevelError,
		Assets: application.AssetOptions{
			Handler:        application.BundledAssetFileServer(distFS),
			DisableLogging: true,
		},
		// Mirror Electron's `webSecurity:false` on Windows WebView2. Theme
		// background images can be cross-origin (e.g. anime image hosts that
		// return a different image per request and serve no CORS headers).
		// Relaxing web security lets the renderer read those pixels via canvas
		// (colorthief recolor + background caching) instead of tainting the
		// canvas. The frontend still degrades gracefully where this isn't
		// available (e.g. macOS WKWebView), showing the image without recolor.
		Windows: application.WindowsOptions{
			AdditionalBrowserArgs: []string{"--disable-web-security"},
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.legiz-ru.prizrak-box",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if win == nil {
					return
				}
				win.Restore()
				win.Show()
				win.Focus()
				if u, ok := findSchemeURL(data.Args); ok {
					win.EmitEvent("deeplink", u)
				}
			},
		},
	})

	// Deep link delivered to the running instance (macOS Apple Event, and the
	// initial-launch case on all platforms).
	app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl,
		func(e *application.ApplicationEvent) {
			if win != nil {
				win.EmitEvent("deeplink", e.Context().URL())
			}
		})

	// macOS: hidden-inset title bar (native traffic lights over full-size
	// content) to match the Electron `titleBarStyle: hiddenInset` look.
	winOpts := application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Prizrak-Box",
		Width:     1100,
		Height:    760,
		MinWidth:  960, // matches the Electron window minimums
		MinHeight: 660,
		Hidden:    true, // shown once the backend is ready
		URL:       "/",
	}
	if runtime.GOOS == "darwin" {
		// macOS keeps the native hidden-inset title bar (traffic lights).
		winOpts.Mac = application.MacWindow{TitleBar: application.MacTitleBarHiddenInset}
	} else {
		// Windows / Linux: frameless so the web UI fills the window. The Vue
		// MyTitleBar provides min/max/close (handled via px:fe:* events) and
		// the --wails-draggable regions in the frontend provide dragging.
		winOpts.Frameless = true
	}
	win = app.Window.NewWithOptions(winOpts)

	// Window controls emitted by the Vue frontend (MyTitleBar.vue / Off.vue)
	// via window.pxTray.emit -> Wails events. This replaces the Electron
	// ipcMain handlers in src-electron/tray.ts.
	app.Event.On("px:fe:close", func(_ *application.CustomEvent) { app.Quit() }) // custom titlebar X quits (matches Electron)
	app.Event.On("px:fe:hide", func(_ *application.CustomEvent) { win.Hide() })
	app.Event.On("px:fe:min", func(_ *application.CustomEvent) { win.Minimise() })
	app.Event.On("px:fe:max", func(_ *application.CustomEvent) { win.ToggleMaximise() })
	app.Event.On("px:fe:boot", func(e *application.CustomEvent) {
		if err := system.SetAutostart(asBool(e.Data)); err != nil {
			app.Logger.Error("autostart toggle failed", "error", err)
		}
	})
	// Persist the "start minimized to tray" preference so the next launch can
	// honour it (read by locate.StartMinimized below). The frontend emits this
	// on change and once on mount to keep the flag in sync.
	app.Event.On("px:fe:startMinimized", func(e *application.CustomEvent) {
		_ = locate.SetStartMinimized(asBool(e.Data))
	})
	// Persist whether TUN is enabled so the next launch knows to wait for the
	// privileged service to come up before spawning px (see TunService.StartBackend).
	// The frontend emits this on TUN toggle and once on mount to keep it in sync.
	app.Event.On("px:fe:tunDesired", func(e *application.CustomEvent) {
		_ = locate.SetTunDesired(asBool(e.Data))
	})

	// Global "Show/Hide window" hotkey (Windows; no-op elsewhere).
	installHotkeys(app, win)
	app.Event.On("px:fe:doQuit", func(_ *application.CustomEvent) {
		// The Exit button (Off.vue) fires this after asking px to shut down.
		// It may carry data:false when px exits before confirming over HTTP,
		// but the user's intent is always to quit, so quit unconditionally.
		app.Quit()
	})

	// Dynamic system tray (modes / profiles / proxy groups / dashboards /
	// system-proxy / TUN), driven by data the frontend pushes over events.
	setupTray(app, win)

	// Webview-ready gate. The window is created with URL "/" and embeds the
	// WebView2 on the main thread (setupChromium -> Embed). Re-navigating with
	// SetURL before that initial embed+navigation completes dereferences a
	// not-yet-created ICoreWebView2 and crashes with a nil-pointer panic inside
	// Navigate. The frontend emits "px:fe:ready" once Vue has mounted on the
	// first "/" load, which is a reliable "the webview can be navigated" signal.
	// We wait for it (with a generous fallback timeout) before the first SetURL.
	webviewReady := make(chan struct{})
	var readyOnce sync.Once
	app.Event.On("px:fe:ready", func(_ *application.CustomEvent) {
		readyOnce.Do(func() { close(webviewReady) })
	})
	awaitWebview := func() {
		select {
		case <-webviewReady:
		case <-time.After(10 * time.Second):
			// Fallback: by now the initial embed has certainly finished even if
			// the ready signal was missed (e.g. a frontend error before mount),
			// so navigating is safe and we avoid a window that never appears.
		}
	}

	// Start the backend, then point the window at it and reveal the window.
	// StartBackend routes px through the elevated service when available (so TUN
	// works without elevating the GUI) and otherwise spawns px directly.
	go func() {
		info, err := tun.StartBackend()
		// Don't navigate until the webview has finished its initial embed.
		awaitWebview()
		if err != nil {
			log.Printf("core start failed: %v", err)
			win.SetURL("/?error=backend")
			win.Show()
			return
		}
		win.SetURL(fmt.Sprintf("/?host=%s&port=%d&secret=%s",
			info.Host, info.Port, url.QueryEscape(info.Secret)))

		// "Start minimized to tray": stay hidden unless a deep link needs the
		// import UI surfaced. The tray's "Show" item and the global hotkey both
		// reveal the window. Mirrors src-electron/main.ts startMinimized.
		u, hasDeep := findSchemeURL(os.Args[1:])
		if hasDeep || !locate.StartMinimized() {
			win.Show()
		}
		if hasDeep {
			win.EmitEvent("deeplink", u)
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func findSchemeURL(args []string) (string, bool) {
	for _, a := range args {
		if strings.HasPrefix(a, deepLinkScheme+"://") {
			return a, true
		}
	}
	return "", false
}
