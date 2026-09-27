package internal

import (
	"encoding/base64"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/legiz-ru/prizrak-box/api/models"
)

// Ranges match the sliders of the desktop theme dialog (Skin.vue).
const (
	themeTransparencyMin, themeTransparencyMax = 5, 85
	themeBlurMin, themeBlurMax                 = 0, 30
	themeDimMin, themeDimMax                   = 0, 80
)

var themeAccentPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// parseProfileTheme parses the pxd-theme header:
//
//	pxd-theme: image=https://…/bg.jpg; transparency=30; blur=4; dim=20; mode=dark; accent=#4f8cff
//
// The whole value may also be sent as "base64:<…>", like announce. A ';' inside
// the image URL must be percent-encoded (%3B). Numbers outside their range are
// clamped; unknown
// keys and invalid values are ignored. Returns nil when nothing usable is set,
// so an absent or empty header clears the profile's theme on refresh.
func parseProfileTheme(raw string) *models.ProfileTheme {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value[len("base64:"):]))
		if err != nil {
			return nil
		}
		value = strings.TrimSpace(string(decoded))
	}

	theme := &models.ProfileTheme{}
	set := false
	for _, part := range strings.Split(value, ";") {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		switch key {
		case "image":
			if u, err := url.Parse(val); err == nil && u.Host != "" &&
				(strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) {
				theme.Image = val
				set = true
			}
		case "transparency":
			if n, ok := themeInt(val, themeTransparencyMin, themeTransparencyMax); ok {
				theme.Transparency = &n
				set = true
			}
		case "blur":
			if n, ok := themeInt(val, themeBlurMin, themeBlurMax); ok {
				theme.Blur = &n
				set = true
			}
		case "dim":
			if n, ok := themeInt(val, themeDimMin, themeDimMax); ok {
				theme.Dim = &n
				set = true
			}
		case "mode":
			switch m := strings.ToLower(val); m {
			case "auto", "light", "dark":
				theme.Mode = m
				set = true
			}
		case "accent":
			if strings.EqualFold(val, "auto") {
				theme.Accent = "auto"
				set = true
			} else if themeAccentPattern.MatchString(val) {
				theme.Accent = strings.ToLower(val)
				set = true
			}
		}
	}
	if !set {
		return nil
	}
	return theme
}

// themeInt parses an integer (a trailing "%" or "px" is tolerated) and clamps
// it into [min, max].
func themeInt(val string, min, max int) (int, bool) {
	val = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(val), "%"), "px"))
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	if n < min {
		n = min
	} else if n > max {
		n = max
	}
	return n, true
}
