package internal

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/mihomo/log"
)

// ZashboardUIPath is the home-relative directory the bundled zashboard panel is
// unpacked into. core.go passes the same value to route.SetUIPath, so the
// controller serves the panel from exactly this folder at /ui/.
const ZashboardUIPath = "ui/zashboard"

// releaseZashboard unpacks the embedded zashboard.zip into <home>/ui/zashboard.
// It only re-unpacks when the embedded archive changes, tracked by a sha256
// marker, so normal startups are a cheap hash compare. The GitHub archive wraps
// everything in a single "<repo>-<branch>/" top-level folder, which is stripped
// so index.html lands at the root of the UI directory.
func releaseZashboard() {
	if len(ZashboardZip) == 0 {
		return
	}

	destDir := utils.GetUserHomeDir("ui", "zashboard")
	markerPath := utils.GetUserHomeDir("ui", ".zashboard.sha256")

	sum := sha256.Sum256(ZashboardZip)
	want := hex.EncodeToString(sum[:])

	if cur, err := os.ReadFile(markerPath); err == nil && strings.TrimSpace(string(cur)) == want {
		// Already unpacked from this exact archive.
		return
	}

	zr, err := zip.NewReader(bytes.NewReader(ZashboardZip), int64(len(ZashboardZip)))
	if err != nil {
		log.Errorln("[Zashboard] open embedded zip failed: %v", err)
		return
	}

	// Start clean so files removed in a newer build don't linger.
	_ = os.RemoveAll(destDir)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		log.Errorln("[Zashboard] mkdir failed: %v", err)
		return
	}

	cleanDest := filepath.Clean(destDir)
	for _, f := range zr.File {
		// Strip the leading "<repo>-<branch>/" segment.
		rel := f.Name
		if i := strings.IndexByte(rel, '/'); i >= 0 {
			rel = rel[i+1:]
		} else {
			rel = ""
		}
		if rel == "" {
			continue
		}

		target := filepath.Join(destDir, rel)
		// Guard against zip-slip.
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			log.Errorln("[Zashboard] mkdir failed: %v", err)
			return
		}
		if err := writeZipEntry(f, target); err != nil {
			log.Errorln("[Zashboard] extract %s failed: %v", f.Name, err)
			return
		}
	}

	if _, err := utils.SaveFile(markerPath, []byte(want)); err != nil {
		log.Errorln("[Zashboard] write marker failed: %v", err)
		return
	}
	log.Infoln("[Zashboard] UI unpacked to %s", destDir)
}

func writeZipEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
