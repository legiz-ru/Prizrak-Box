// Command prizrak-box-wails is the Wails v3 desktop shell for Prizrak-Box.
//
// Phase 0 (PoC): boot Wails v3, serve the existing Vue frontend, spawn/supervise
// px, hand the frontend host/port/secret, minimal tray, single-instance lock.
//
// Phase 1 (this file): TUN service management (TunService), launch-at-login
// (SystemService via Wails Autostart), deep-link handling for the
// prizrak-box:// scheme (ApplicationLaunchedWithUrl + second-instance argv),
// and bringing the window to front on a second launch.
//
// Still out of scope (later phases): full dynamic tray menu mirroring the
// Electron tray, and config-directory migration.
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

	win = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "main",
		Title:  "Prizrak-Box",
		Width:  1100,
		Height: 760,
		Hidden: true, // shown once the backend is ready
		URL:    "/",
	})

	// Close button hides to tray instead of quitting (matches Electron).
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		win.Hide()
		e.Cancel()
	})

	// Minimal native system tray (full dynamic menu lands in a later phase).
	tray := app.SystemTray.New()
	tray.SetLabel("Prizrak-Box")
	tray.SetTooltip("Prizrak-Box")
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(appIcon)
	} else {
		tray.SetIcon(appIcon)
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
