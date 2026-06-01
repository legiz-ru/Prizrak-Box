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
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/legiz-ru/prizrak-box-wails/services"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/tray.png
var trayIcon []byte

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

	app := application.New(application.Options{
		Name:        "Prizrak-Box",
		Description: "A Simple Mihomo GUI",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(core),
			application.NewService(system),
			application.NewService(tun),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(distFS),
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
	win = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "main",
		Title:  "Prizrak-Box",
		Width:  1100,
		Height: 760,
		Hidden: true, // shown once the backend is ready
		URL:    "/",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})

	// Window controls emitted by the Vue frontend (MyTitleBar.vue / Off.vue)
	// via window.pxTray.emit -> Wails events. This replaces the Electron
	// ipcMain handlers in src-electron/tray.ts.
	app.Event.On("close", func(_ *application.CustomEvent) { win.Hide() }) // close to tray
	app.Event.On("hide", func(_ *application.CustomEvent) { win.Hide() })
	app.Event.On("min", func(_ *application.CustomEvent) { win.Minimise() })
	app.Event.On("max", func(_ *application.CustomEvent) { win.ToggleMaximise() })
	app.Event.On("boot", func(e *application.CustomEvent) {
		if err := system.SetAutostart(asBool(e.Data)); err != nil {
			app.Logger.Error("autostart toggle failed", "error", err)
		}
	})
	app.Event.On("doQuit", func(e *application.CustomEvent) {
		if asBool(e.Data) {
			app.Quit()
		}
	})

	// Native system tray with the correct monochrome icon (no text label).
	tray := app.SystemTray.New()
	tray.SetTooltip("Prizrak-Box")
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayIconMac)
	} else {
		tray.SetIcon(trayIcon)
	}
	menu := app.NewMenu()
	menu.Add("Показать / Show").OnClick(func(_ *application.Context) { win.Show(); win.Focus() })
	menu.Add("Скрыть / Hide").OnClick(func(_ *application.Context) { win.Hide() })
	menu.AddSeparator()
	menu.Add("Выход / Quit").OnClick(func(_ *application.Context) { app.Quit() })
	tray.SetMenu(menu)

	// Start the backend, then point the window at it and reveal the window.
	go func() {
		info, err := core.Start()
		if err != nil {
			log.Printf("core start failed: %v", err)
			win.SetURL("/?error=backend")
			win.Show()
			return
		}
		win.SetURL(fmt.Sprintf("/?host=%s&port=%d&secret=%s",
			info.Host, info.Port, url.QueryEscape(info.Secret)))
		win.Show()

		// Handle a deep link passed on the very first launch via argv
		// (Windows / Linux). macOS uses ApplicationLaunchedWithUrl above.
		if u, ok := findSchemeURL(os.Args[1:]); ok {
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

// asBool coerces event payloads (which arrive as JSON) to a bool.
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
