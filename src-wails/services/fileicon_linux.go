//go:build linux

// Linux app-icon resolution (best-effort, no external deps). The connection's
// executable path is mapped to a freedesktop .desktop entry, whose Icon= name is
// resolved to a PNG in the icon theme / pixmaps. Apps that only ship SVG icons
// are skipped (no rasteriser), so this is partial coverage by design.
package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func fileIconPNG(path string, _ int) ([]byte, error) {
	exe := filepath.Base(path)
	if exe == "" {
		return nil, errors.New("no exe")
	}

	icon := iconNameForExe(exe)
	if icon == "" {
		// Fall back to using the executable basename as an icon name; many
		// apps name their icon after the binary.
		icon = exe
	}

	if file := resolveIconFile(icon); file != "" {
		if data, err := os.ReadFile(file); err == nil && isPNG(data) {
			return data, nil
		}
	}
	return nil, errors.New("no icon")
}

func isPNG(b []byte) bool {
	return len(b) > 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n"
}

// dataDirs returns the freedesktop data directories in priority order.
func dataDirs() []string {
	var dirs []string
	if h := os.Getenv("XDG_DATA_HOME"); h != "" {
		dirs = append(dirs, h)
	} else if home, _ := os.UserHomeDir(); home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "share"))
	}
	sys := os.Getenv("XDG_DATA_DIRS")
	if sys == "" {
		sys = "/usr/local/share:/usr/share"
	}
	dirs = append(dirs, strings.Split(sys, ":")...)
	return dirs
}

// iconNameForExe scans .desktop files for an entry whose Exec/TryExec matches
// the executable basename and returns its Icon= value.
func iconNameForExe(exe string) string {
	exeLower := strings.ToLower(exe)
	for _, base := range dataDirs() {
		appsDir := filepath.Join(base, "applications")
		entries, err := os.ReadDir(appsDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			icon, execBase := parseDesktop(filepath.Join(appsDir, e.Name()))
			if execBase != "" && strings.ToLower(execBase) == exeLower && icon != "" {
				return icon
			}
		}
	}
	return ""
}

// parseDesktop reads a .desktop file and returns (Icon, execBasename).
func parseDesktop(file string) (icon, execBase string) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", ""
	}
	inMain := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inMain = line == "[Desktop Entry]"
			continue
		}
		if !inMain {
			continue
		}
		switch {
		case strings.HasPrefix(line, "Icon="):
			icon = strings.TrimSpace(strings.TrimPrefix(line, "Icon="))
		case strings.HasPrefix(line, "TryExec="):
			if execBase == "" {
				execBase = filepath.Base(strings.TrimSpace(strings.TrimPrefix(line, "TryExec=")))
			}
		case strings.HasPrefix(line, "Exec="):
			if execBase == "" {
				execBase = filepath.Base(firstToken(strings.TrimPrefix(line, "Exec=")))
			}
		}
	}
	return icon, execBase
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i >= 0 {
		s = s[:i]
	}
	return s
}

// resolveIconFile turns an icon name (or absolute path) into a PNG file path.
func resolveIconFile(name string) string {
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		if strings.HasSuffix(name, ".png") {
			if _, err := os.Stat(name); err == nil {
				return name
			}
		}
		return ""
	}

	sizes := []string{"64x64", "48x48", "128x128", "256x256", "96x96", "72x72", "32x32", "24x24", "scalable"}
	themes := []string{"hicolor", "Adwaita", "gnome", "breeze", "Papirus"}

	for _, base := range dataDirs() {
		iconsRoot := filepath.Join(base, "icons")
		for _, theme := range themes {
			for _, sz := range sizes {
				cand := filepath.Join(iconsRoot, theme, sz, "apps", name+".png")
				if _, err := os.Stat(cand); err == nil {
					return cand
				}
			}
		}
		// Flat pixmaps directory.
		if cand := filepath.Join(base, "pixmaps", name+".png"); fileExistsLinux(cand) {
			return cand
		}
	}
	if cand := "/usr/share/pixmaps/" + name + ".png"; fileExistsLinux(cand) {
		return cand
	}
	return ""
}

func fileExistsLinux(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
