package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/legiz-ru/prizrak-box/app"
	"github.com/legiz-ru/prizrak-box/pkg/constant"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	application := app.New()

	err := wails.Run(&options.App{
		Title:            "Prizrak-Box",
		Width:            1100,
		Height:           760,
		MinWidth:         960,
		MinHeight:        660,
		BackgroundColour: &options.RGBA{R: 17, G: 24, B: 39, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			dataDir := resolveDataDir()
			if mkErr := os.MkdirAll(dataDir, 0o755); mkErr != nil {
				log.Printf("failed to ensure data directory: %v", mkErr)
			}
			utils.InitHomeDir(dataDir)
			application.OnStartup(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			application.OnShutdown(ctx)
		},
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

func resolveDataDir() string {
	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, constant.DefaultWorkDir)
	}

	return filepath.Join(os.TempDir(), constant.DefaultWorkDir)
}
