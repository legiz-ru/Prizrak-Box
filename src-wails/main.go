// Command prizrak-box-wails is the Wails v3 desktop shell for Prizrak-Box.
//
// Phase 0 (PoC) scope:
//   - Boot a Wails v3 application that serves the existing Vue frontend.
//   - Spawn and supervise the px backend via CoreService.
//   - Hand the frontend the px host/port/secret (via the window URL query,
//     exactly like the old Electron shell did).
//   - Provide a minimal native system tray (Show / Hide / Quit) and a
//     single-instance lock.
//
// Out of scope for Phase 0 (later phases): full dynamic tray menu, deep-link
// import, TUN service management, autostart, config-dir migration.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/legiz-ru/prizrak-box-wails/services"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	distFS, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatalf("embed frontend: %v", err)
	}

	core := services.NewCoreService()
	system := services.NewSystemService()

	app := application.New(application.Options{
		Name:        "Prizrak-Box",
		Description: "A Simple Mihomo GUI",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(core),
			application.NewService(system),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(distFS),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.legiz-ru.prizrak-box",
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				// Phase 1: bring the window to front and handle deep-link argv.
			},
		},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "main",
		Title:  "Prizrak-Box",
		Width:  1100,
		Height: 760,
		Hidden: true, // shown once the backend is ready
		URL:    "/",
	})

	// Close button hides to tray instead of quitting (matches Electron behaviour).
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		win.Hide()
		e.Cancel()
	})

	// Minimal native system tray.
	tray := app.SystemTray.New()
	tray.SetLabel("Prizrak-Box")
	tray.SetTooltip("Prizrak-Box")
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(appIcon)
	} else {
		tray.SetIcon(appIcon)
	}

	menu := app.NewMenu()
	menu.Add("Показать / Show").OnClick(func(_ *application.Context) { win.Show() })
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
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
