// Custom-background upload/serve endpoints for the theme picker (Skin.vue).
//
// This is a port of src-electron/server.ts's Express routes of the same
// name (POST /api/custom-background, GET /user-images/*). The Wails v3
// asset server only serves the embedded frontend bundle by default and has
// no equivalent, which left the "upload a custom background image" feature
// in the theme dropdown broken on every Wails-shell build (Windows, macOS
// and Linux) — the frontend's fetch() to these paths always 404'd.
package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/legiz-ru/prizrak-box-wails/internal/locate"
)

const (
	userImagesPrefix        = "/user-images/"
	customBackgroundAPIPath = "/api/custom-background"
	customBackgroundMaxBody = 20 << 20 // 20MB, matches Electron's BODY_LIMIT
)

var (
	themeIDDisallowed = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	fileExtPattern    = regexp.MustCompile(`^\.[a-z0-9]+$`)
	dataURLPattern    = regexp.MustCompile(`(?is)^data:(.+);base64,(.+)$`)
)

type customBackgroundRequest struct {
	ThemeID      string `json:"themeId"`
	DataURL      string `json:"dataUrl"`
	FileName     string `json:"fileName"`
	PreviousPath string `json:"previousPath"`
}

type customBackgroundResponse struct {
	URL   string `json:"url,omitempty"`
	Error string `json:"error,omitempty"`
}

// CustomBackgroundMiddleware serves the two endpoints the theme picker's
// custom-background upload depends on and delegates everything else to the
// normal asset server. Wired via application.Options.Assets.Middleware.
func CustomBackgroundMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == customBackgroundAPIPath:
			handleCustomBackgroundUpload(w, r)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, userImagesPrefix):
			handleCustomBackgroundGet(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// customImagesDir returns (and creates) the directory custom background
// uploads are written to. Uses the OS temp dir, mirroring Electron's
// server.ts ensureImagesDir (electronApp.getPath('temp')).
func customImagesDir() (string, error) {
	dir := filepath.Join(os.TempDir(), locate.WorkDirName(), "images")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func sanitizeThemeID(v string) string {
	cleaned := themeIDDisallowed.ReplaceAllString(v, "")
	if len(cleaned) > 40 {
		cleaned = cleaned[:40]
	}
	if cleaned == "" {
		return "custom"
	}
	return cleaned
}

func sanitizeExtension(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" || !fileExtPattern.MatchString(ext) {
		return ".png"
	}
	return ext
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// sanitizeUserImagePath extracts a flat file name from a "/user-images/<name>"
// path, rejecting anything that could escape the images directory.
func sanitizeUserImagePath(value string) (string, bool) {
	if !strings.HasPrefix(value, userImagesPrefix) {
		return "", false
	}
	name := strings.TrimPrefix(value, userImagesPrefix)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return "", false
	}
	return name, true
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(customBackgroundResponse{Error: code})
}

func handleCustomBackgroundUpload(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, customBackgroundMaxBody))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_DATA")
		return
	}

	var req customBackgroundRequest
	if err := json.Unmarshal(body, &req); err != nil || req.DataURL == "" {
		writeJSONError(w, http.StatusBadRequest, "INVALID_DATA")
		return
	}

	match := dataURLPattern.FindStringSubmatch(req.DataURL)
	if match == nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_DATA_URL")
		return
	}

	data, err := base64.StdEncoding.DecodeString(match[2])
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_DATA_URL")
		return
	}

	dir, err := customImagesDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "WRITE_FAILED")
		return
	}

	id, err := randomID()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "WRITE_FAILED")
		return
	}

	targetName := sanitizeThemeID(req.ThemeID) + "-" + id + sanitizeExtension(req.FileName)
	if err := os.WriteFile(filepath.Join(dir, targetName), data, 0o644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "WRITE_FAILED")
		return
	}

	if fileName, ok := sanitizeUserImagePath(req.PreviousPath); ok {
		_ = os.Remove(filepath.Join(dir, fileName))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(customBackgroundResponse{URL: userImagesPrefix + targetName})
}

func handleCustomBackgroundGet(w http.ResponseWriter, r *http.Request) {
	fileName, ok := sanitizeUserImagePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	dir, err := customImagesDir()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(dir, fileName))
}
