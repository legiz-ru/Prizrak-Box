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
	"io/fs"
        "fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/kvstore"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

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

	// Persistent key-value store for the frontend's pinia state (pxStore in
	// wails-shim.ts), replacing the localStorage fallback that lived in the
	// WebView2 profile (wiped by a webview cache clear, not shared with the
	// Electron shell). The file is deliberately the electron-store location
	// and format (<home>/px-electron.db/config.json, a flat JSON object), so
	// settings carry over in both directions when switching shells.
	storePath := filepath.Join(locate.HomeDir(), "px-electron.db", "config.json")
	_ = os.MkdirAll(filepath.Dir(storePath), 0o755)
	store := kvstore.NewWithConfig(&kvstore.Config{
		Filename: storePath,
		AutoSave: true,
	})

	// Native notifications (Windows toast / macOS UNUserNotification / Linux
	// D-Bus). Exposed to the frontend as window.pxNotify (wails-shim.ts).
	// On macOS delivery requires running from a signed .app bundle.
	notifier := notifications.New()

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
			application.NewService(store),
			application.NewService(notifier),
		},
		// Keep Wails' own logging quiet (px already logs plenty); the noisy
		// per-request asset logs and benign "Window #N not found" warnings on
		// shutdown are suppressed.
		LogLevel: slog.LevelError,
		// The shell runs with no console: preserve panics and internal
		// framework errors in a small log file (see shelllog.go) instead of
		// letting them vanish.
		PanicHandler: shellPanicHandler,
		ErrorHandler: shellErrorHandler,
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(distFS),
			// Serves the theme picker's custom-background upload endpoints
			// (POST /api/custom-background, GET /user-images/*) that Skin.vue
			// depends on; ported from src-electron/server.ts, which only runs
			// in the Electron shell. See services/background.go.
			Middleware:     services.CustomBackgroundMiddleware,
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
			// Pin the WebView2 profile to a stable per-user location. The
			// default (%APPDATA%\<binary-name>.exe) silently "loses" the
			// webview cache/localStorage when the executable is renamed and
			// litters APPDATA. Kept OUTSIDE the px data dir because "Change
			// config dir" moves that dir while WebView2 holds locks on its
			// profile (see locate.WebviewDataDir).
			WebviewUserDataPath: locate.WebviewDataDir(),
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
		Hidden:    true, // shown once the webview has rendered (see below)
		URL:       "/",
	}
	// Match the window's own background to the frontend's last-known theme so
	// any pre-paint frame (first reveal, resize exposure) shows the app's
	// background colour instead of a jarring black/white rectangle. The
	// frontend persists dark/light on every theme change (px:fe:darkBg below);
	// dark is the default, matching the frontend's default dark gradient.
	if locate.DarkBackground() {
		winOpts.BackgroundColour = application.NewRGB(17, 17, 17)
	} else {
		winOpts.BackgroundColour = application.NewRGB(242, 242, 242)
	}
	if runtime.GOOS == "darwin" {
		// macOS keeps the native hidden-inset title bar (traffic lights).
		winOpts.Mac = application.MacWindow{TitleBar: application.MacTitleBarHiddenInset}
	} else {
		// Windows / Linux: frameless so the web UI fills the window. The Vue
		// MyTitleBar provides min/max/close (handled via px:fe:* events) and
		// the --wails-draggable regions in the frontend provide dragging.
		winOpts.Frameless = true
		winOpts.Windows = application.WindowsWindow{
			// The window is frameless with a web titlebar; make sure no native
			// menu bar can ever be created/toggled for it.
			DisableMenu: true,
			// Native non-client hit testing for the custom titlebar. With
			// composition hosting the frontend can mark elements with
			// --wails-non-client-region: caption/minimize/maximize/close so
			// Windows treats them as real caption buttons — enabling the
			// Windows 11 Snap Layouts flyout over the HTML maximize button
			// (see MyTitleBar.vue). NonClientRegionSupport additionally lets
			// WebView2's own app-region CSS participate when the installed
			// runtime supports it. Both are no-ops on runtimes without support.
			NonClientRegionSupport:     true,
			WebView2CompositionHosting: true,
		}
	}
	win = app.Window.NewWithOptions(winOpts)

	// quitting distinguishes a genuine shutdown (set just before app.Quit) from
	// an ordinary window close, so the WindowClosing hook below knows whether to
	// hide the window or let it be destroyed.
	var quitting atomic.Bool
	quit := func() {
		quitting.Store(true)
		app.Quit()
	}

	// Closing the window must hide it to the tray, not destroy it. On macOS
	// Cmd+W (the system "Close Window" menu item) and the native red traffic
	// light fire WindowClosing; Wails' default listener then destroys the window.
	// A destroyed window leaves the tray's "Show"/"Quit" items pointing at a dead
	// window and webview — "Show" becomes a no-op and "Quit" never round-trips
	// through the frontend — so the app can only be force-quit. RegisterHook runs
	// before that default listener, and cancelling the event prevents the
	// destroy. We then hide (macOS) or minimise (Windows/Linux), mirroring the
	// Electron shell's close→hide/minimise behaviour (src-electron/tray.ts). A
	// genuine quit goes through app.Quit (native terminate, no WindowClosing), but
	// the framework's own cleanup also calls window.Close(), so honour `quitting`.
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if quitting.Load() {
			return
		}
		e.Cancel()
		if runtime.GOOS == "darwin" {
			win.Hide()
		} else {
			win.Minimise()
		}
	})

	// NOTE: the Windows blank-frame-on-first-reveal SetSize workaround that
	// lived here was removed with the Wails v3.0.0-alpha2.117 update. Since
	// alpha2.114 the framework synchronizes the WebView2 controller's
	// visibility with the window: hide()/minimise hide the controller and
	// show()/restore re-assert it, so the first reveal of a window created
	// Hidden gets a real visibility transition (the WebView2Feedback #1077
	// nudge) at reveal time instead of a stale black frame.

	// Push the real maximise state to the frontend titlebar. With native
	// non-client regions enabled (see winOpts.Windows above) a click on the
	// HTML maximize button is handled by Windows itself and never reaches the
	// Vue click handler, so MyTitleBar.vue can't infer the state from its own
	// clicks; these native events are authoritative either way (they also
	// cover Win+Up / titlebar double-click).
	if runtime.GOOS == "windows" {
		win.OnWindowEvent(events.Windows.WindowMaximise, func(_ *application.WindowEvent) {
			win.EmitEvent("px:be:maximized", true)
		})
		win.OnWindowEvent(events.Windows.WindowUnMaximise, func(_ *application.WindowEvent) {
			win.EmitEvent("px:be:maximized", false)
		})
	}

	// Window controls emitted by the Vue frontend (MyTitleBar.vue / Off.vue)
	// via window.pxTray.emit -> Wails events. This replaces the Electron
	// ipcMain handlers in src-electron/tray.ts.
	app.Event.On("px:fe:close", func(_ *application.CustomEvent) { quit() }) // custom titlebar X quits (matches Electron)
	app.Event.On("px:fe:hide", func(_ *application.CustomEvent) { win.Hide() })
	app.Event.On("px:fe:min", func(_ *application.CustomEvent) { win.Minimise() })
	app.Event.On("px:fe:max", func(_ *application.CustomEvent) { win.ToggleMaximise() })
	// Launch at login. The frontend emits this on change AND once on mount, so
	// the registration is re-applied on every launch: the setting is persisted
	// (and shared with the Electron build), and an app update can change the
	// executable path or drop the entry, leaving the toggle on with nothing
	// actually registered. Enable/Disable overwrite, so re-applying is safe.
	app.Event.On("px:fe:boot", func(e *application.CustomEvent) {
		enabled := asBool(e.Data)
		if err := system.SetAutostart(enabled); err != nil {
			// A GUI build has no console, so Logger alone would swallow this.
			appendShellLog("ERROR", fmt.Sprintf("autostart set to %v failed: %v", enabled, err))
			app.Logger.Error("autostart toggle failed", "enabled", enabled, "error", err)
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
	// Persist the frontend's dark/light theme so the next launch paints the
	// window background in the matching colour before the webview renders
	// (see winOpts.BackgroundColour above). Emitted on every theme change.
	app.Event.On("px:fe:darkBg", func(e *application.CustomEvent) {
		_ = locate.SetDarkBackground(asBool(e.Data))
	})

	// Global "Show/Hide window" hotkey, via the Wails GlobalShortcut manager
	// (see hotkeys.go) — works on all three platforms. On macOS, combos using
	// only Option/Option+Shift as modifiers are silently dropped by the OS on
	// Sequoia (Apple bug FB15168205); MyHotkeyInput.vue warns the user about
	// that case when recording the combo.
	installHotkeys(app, win)
	app.Event.On("px:fe:doQuit", func(_ *application.CustomEvent) {
		// The Exit button (Off.vue) fires this after asking px to shut down.
		// It may carry data:false when px exits before confirming over HTTP,
		// but the user's intent is always to quit, so quit unconditionally.
		quit()
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

	// Reveal the window as soon as the webview has rendered — deliberately NOT
	// gated on the backend being up.
	//
	// px startup is not instant: bringing up the TUN adapter takes seconds, and
	// at boot StartBackend may additionally wait for the still-starting
	// px-service. Revealing the window only afterwards meant nothing at all was
	// on screen for that whole time, which reads as a frozen app. The frontend
	// now fetches the connection info itself (window.pxConnInfo ->
	// CoreService.GetConnInfo, which blocks until px reports back) and shows a
	// boot placeholder meanwhile, so no navigation is needed here either.
	go func() {
		awaitWebview()
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

	// Start the backend in the background. StartBackend routes px through the
	// elevated service when available (so TUN works without elevating the GUI)
	// and otherwise spawns px directly.
	go func() {
		if _, err := tun.StartBackend(); err != nil {
			log.Printf("core start failed: %v", err)
			// Unblock the frontend's pxConnInfo wait instead of leaving it to
			// time out on the boot placeholder.
			win.EmitEvent("px:be:backendError", err.Error())
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
