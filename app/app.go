package app

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"unicode/utf8"

	sys "github.com/legiz-ru/prizrak-box/pkg/sys/proxy"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/legiz-ru/prizrak-box/prizrak"
	"github.com/metacubex/mihomo/hub/executor"
	milog "github.com/metacubex/mihomo/log"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ServerInfo describes the local HTTP API endpoint exposed by the Mihomo core.
type ServerInfo struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Secret string `json:"secret"`
}

// Environment describes the host operating system in a user-friendly format.
type Environment struct {
	Label string `json:"label"`
}

// App orchestrates the embedded Mihomo core lifecycle inside the Wails shell.
type App struct {
	ctx      context.Context
	info     ServerInfo
	env      Environment
	initOnce sync.Once
	stopOnce sync.Once
}

// New constructs a new App instance.
func New() *App {
	return &App{
		info: ServerInfo{Host: "127.0.0.1"},
	}
}

// OnStartup is called by Wails once the application context becomes available.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	a.env = Environment{Label: fmt.Sprintf("%s %s", titleCase(goruntime.GOOS), goruntime.GOARCH)}

	a.initOnce.Do(func() {
		if utils.NotSingleton("px-server.pid") {
			runtime.LogErrorf(ctx, "Another Prizrak-Box instance is already running")
			// Request application quit after logging the fatal condition.
			_ = runtime.Quit(ctx)
			return
		}

		prizrak.Init()

		a.registerFrontendEvents(ctx)

		port, secret := prizrak.StartCore("")
		a.info.Port = port
		a.info.Secret = secret

		runtime.EventsEmit(ctx, "backend:ready", a.info)
	})
}

// OnShutdown ensures graceful release of Mihomo resources when the app exits.
func (a *App) OnShutdown(ctx context.Context) {
	a.stopOnce.Do(func() {
		prizrak.Release()
		utils.UnlockSingleton()
		executor.Shutdown()
		sys.DisableProxy()
		milog.Warnln("Prizrak-Box backend shutdown completed")
	})
}

// ServerInfo returns the currently running Mihomo endpoint configuration.
func (a *App) ServerInfo() ServerInfo {
	return a.info
}

// EnvironmentInfo exposes the detected host operating system label.
func (a *App) EnvironmentInfo() Environment {
	return a.env
}

// OpenPath asks the operating system to reveal the provided path in its file manager.
func (a *App) OpenPath(target string) error {
	if a.ctx == nil {
		return fmt.Errorf("runtime context not initialised")
	}

	if target == "" {
		return fmt.Errorf("empty path provided")
	}

	uri := target
	if !strings.HasPrefix(target, "file://") {
		fileURL := &url.URL{Scheme: "file", Path: filepath.ToSlash(target)}
		uri = fileURL.String()
	}

	return runtime.BrowserOpenURL(a.ctx, uri)
}

func (a *App) registerFrontendEvents(ctx context.Context) {
	runtime.EventsOn(ctx, "close", func(optionalData ...any) {
		if err := runtime.Quit(ctx); err != nil {
			runtime.LogErrorf(ctx, "failed to quit application: %v", err)
		}
	})

	runtime.EventsOn(ctx, "doQuit", func(optionalData ...any) {
		if err := runtime.Quit(ctx); err != nil {
			runtime.LogErrorf(ctx, "failed to quit application: %v", err)
		}
	})

	runtime.EventsOn(ctx, "min", func(optionalData ...any) {
		if err := runtime.WindowMinimise(ctx); err != nil {
			runtime.LogErrorf(ctx, "failed to minimise window: %v", err)
		}
	})

	runtime.EventsOn(ctx, "max", func(optionalData ...any) {
		if err := runtime.WindowToggleMaximise(ctx); err != nil {
			runtime.LogErrorf(ctx, "failed to toggle maximise: %v", err)
		}
	})

	runtime.EventsOn(ctx, "hide", func(optionalData ...any) {
		if err := runtime.WindowHide(ctx); err != nil {
			runtime.LogErrorf(ctx, "failed to hide window: %v", err)
		}
	})
}

func titleCase(value string) string {
	if value == "" {
		return ""
	}

	lower := strings.ToLower(value)
	r, size := utf8.DecodeRuneInString(lower)
	if r == utf8.RuneError && size == 0 {
		return lower
	}

	return strings.ToUpper(string(r)) + lower[size:]
}
