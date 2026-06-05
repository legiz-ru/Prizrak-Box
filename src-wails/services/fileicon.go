// File-icon extraction for the connections "Processes" view. The frontend calls
// window.electron.invoke('get-file-icon', processPath) (ConnectionTab.vue) which
// the Wails shim routes to CoreService.FileIcon. This mirrors Electron's
// app.getFileIcon(path).toDataURL() in src-electron/main.ts.
//
// The OS-specific extraction lives in fileicon_{windows,darwin,linux,other}.go,
// each providing fileIconPNG(path, size) ([]byte, error). Results (including
// empty "no icon" results) are cached per path so the list scrolls smoothly.
package services

import (
	"encoding/base64"
	"sync"
)

// fileIconSize is the requested icon edge in pixels.
const fileIconSize = 64

var (
	iconMu    sync.Mutex
	iconCache = map[string]string{}
)

// FileIcon returns the icon of the executable at path as a PNG data URL
// ("data:image/png;base64,…"), or "" if none could be resolved. Bound to the
// frontend (CoreService is registered with the Wails app).
func (c *CoreService) FileIcon(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	iconMu.Lock()
	cached, ok := iconCache[path]
	iconMu.Unlock()
	if ok {
		return cached, nil
	}

	out := ""
	if png, err := fileIconPNG(path, fileIconSize); err == nil && len(png) > 0 {
		out = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	}

	iconMu.Lock()
	iconCache[path] = out
	iconMu.Unlock()
	return out, nil
}
