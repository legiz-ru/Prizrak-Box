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

	"github.com/legiz-ru/prizrak-box/pkg/deeplink"
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
	ctx        context.Context
	info       ServerInfo
	env        Environment
	initOnce   sync.Once
	stopOnce   sync.Once
	launchArgs []string

	deepLinkMutex sync.Mutex
	deepLinkQueue []deepLinkRequest
	frontendReady bool
}

// New constructs a new App instance.
func New() *App {
	return &App{
		info: ServerInfo{Host: "127.0.0.1"},
	}
}

// SetLaunchArguments stores the command-line arguments the application was started with.
func (a *App) SetLaunchArguments(args []string) {
	a.deepLinkMutex.Lock()
	defer a.deepLinkMutex.Unlock()

	a.launchArgs = append([]string(nil), args...)
}

// OnStartup is called by Wails once the application context becomes available.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	a.env = Environment{Label: fmt.Sprintf("%s %s", titleCase(goruntime.GOOS), goruntime.GOARCH)}

	a.initOnce.Do(func() {
		if utils.NotSingleton("px-server.pid") {
			runtime.LogErrorf(ctx, "Another Prizrak-Box instance is already running")
			// Request application quit after logging the fatal condition.
			runtime.Quit(ctx)
			return
		}

		prizrak.Init()

		a.registerFrontendEvents(ctx)

		if err := deeplink.RegisterProtocol("prizrak-box", "Prizrak-Box"); err != nil {
			runtime.LogWarningf(ctx, "Failed to register deeplink protocol: %v", err)
		}

		a.processDeepLinkArgsLocked()

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

	runtime.BrowserOpenURL(a.ctx, uri)
	return nil
}

func (a *App) registerFrontendEvents(ctx context.Context) {
	runtime.EventsOn(ctx, "close", func(optionalData ...any) {
		runtime.Quit(ctx)
	})

	runtime.EventsOn(ctx, "doQuit", func(optionalData ...any) {
		runtime.Quit(ctx)
	})

	runtime.EventsOn(ctx, "min", func(optionalData ...any) {
		runtime.WindowMinimise(ctx)
	})

	runtime.EventsOn(ctx, "max", func(optionalData ...any) {
		runtime.WindowToggleMaximise(ctx)
	})

	runtime.EventsOn(ctx, "hide", func(optionalData ...any) {
		runtime.WindowHide(ctx)
	})

	runtime.EventsOn(ctx, "deeplink:ready", func(optionalData ...any) {
		a.handleFrontendReady()
	})
}

// HandleSecondInstanceLaunch is invoked when a secondary instance of the application is launched.
func (a *App) HandleSecondInstanceLaunch(args []string) {
	a.processDeepLinkArgs(args)

	if a.ctx == nil {
		return
	}

	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

func (a *App) processDeepLinkArgs(args []string) {
	if len(args) == 0 {
		return
	}

	for _, candidate := range args {
		a.handleDeepLinkCandidate(candidate)
	}
}

func (a *App) processDeepLinkArgsLocked() {
	a.deepLinkMutex.Lock()
	args := append([]string(nil), a.launchArgs...)
	a.launchArgs = nil
	a.deepLinkMutex.Unlock()

	a.processDeepLinkArgs(args)
}

func (a *App) handleDeepLinkCandidate(candidate string) {
	value := strings.TrimSpace(candidate)
	if value == "" {
		return
	}

	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "prizrak-box://") {
		a.enqueueDeepLink(deepLinkRequest{RawURL: value})
		return
	}

	// Allow passing direct subscription URLs for automation/debugging purposes.
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		a.enqueueDeepLink(deepLinkRequest{DirectURL: value})
	}
}

func (a *App) enqueueDeepLink(request deepLinkRequest) {
	a.deepLinkMutex.Lock()
	ready := a.frontendReady && a.ctx != nil
	if !ready {
		a.deepLinkQueue = append(a.deepLinkQueue, request)
		a.deepLinkMutex.Unlock()
		return
	}

	a.deepLinkMutex.Unlock()

	a.emitDeepLink(request)
}

func (a *App) handleFrontendReady() {
	a.deepLinkMutex.Lock()
	a.frontendReady = true
	queue := append([]deepLinkRequest(nil), a.deepLinkQueue...)
	a.deepLinkQueue = nil
	a.deepLinkMutex.Unlock()

	for _, request := range queue {
		a.emitDeepLink(request)
	}
}

func (a *App) emitDeepLink(request deepLinkRequest) {
	if a.ctx == nil {
		return
	}

	var payload any

	switch {
	case request.RawURL != "" && request.DirectURL == "" && request.Name == "":
		payload = request.RawURL
	default:
		data := map[string]string{}
		if request.RawURL != "" {
			data["rawUrl"] = request.RawURL
		}
		if request.DirectURL != "" {
			data["url"] = request.DirectURL
		}
		if request.Name != "" {
			data["name"] = request.Name
		}

		if len(data) == 0 {
			return
		}

		payload = data
	}

	runtime.EventsEmit(a.ctx, "deeplink-profile-imported", payload)
}

type deepLinkRequest struct {
	RawURL    string
	DirectURL string
	Name      string
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
