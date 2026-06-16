#!/usr/bin/env bash
#
# Build & run the Wails v3 shell without the `task` runner.
#
# Does everything `task frontend` + `task dev` would, using plain tools:
#   1. installs frontend deps (if needed) and builds the Vue app into
#      frontend/dist (embedded by the Go binary),
#   2. builds the px backend (../src-go/px) if it is missing,
#   3. builds and runs the Wails shell.
#
# Usage (from anywhere):
#   ./src-wails/run-dev.sh
#
# Requirements: Go, Node/npm, and on Linux the GTK4/WebKitGTK dev packages.
# On macOS you need Xcode Command Line Tools (xcode-select --install).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PX_EXE="px"
WAILS_EXE="prizrak-box-wails"
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*) PX_EXE="px.exe"; WAILS_EXE="prizrak-box-wails.exe" ;;
esac

echo "==> [1/3] Building frontend (vite -> src-wails/frontend/dist)"
cd "$REPO_ROOT"
# Always install so newly-added deps (e.g. @wailsio/runtime) are present even
# when node_modules was created by an earlier checkout.
npm install --no-audit --no-fund
npx vite build --outDir src-wails/frontend/dist --emptyOutDir

echo "==> [2/3] Ensuring px backend (src-go/$PX_EXE) and px-service"
# Rebuild a Go binary when it is missing OR any source/asset under $2 is newer
# than it. Without this, edits to the core (e.g. a new embedded asset or route)
# would be silently ignored because a stale binary already exists.
go_stale() {
  local bin="$1" dir="$2"
  [ -x "$bin" ] || return 0
  [ -n "$(find "$dir" -type f \( -name '*.go' -o -name '*.zip' -o -name '*.7z' \
      -o -name '*.dat' -o -name '*.mmdb' -o -name '*.metadb' -o -name '*.bin' \
      -o -name '*.yaml' -o -name '*.json' \) -newer "$bin" -print -quit 2>/dev/null)" ]
}
if go_stale "src-go/$PX_EXE" "src-go"; then
  echo "    building px (geo/model/zashboard assets are vendored in src-go/internal/em)..."
  ( cd src-go && CGO_ENABLED=0 go build -tags=with_gvisor -trimpath -ldflags "-X github.com/legiz-ru/prizrak-box/api.Version=v-test" -o "$PX_EXE" . )
else
  echo "    found up-to-date src-go/$PX_EXE"
fi
SERVICE_EXE="px-service"; [ "$PX_EXE" = "px.exe" ] && SERVICE_EXE="px-service.exe"
if go_stale "src-service/$SERVICE_EXE" "src-service"; then
  echo "    building px-service (TUN helper)..."
  ( cd src-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o "$SERVICE_EXE" . )
else
  echo "    found up-to-date src-service/$SERVICE_EXE"
fi

echo "==> [3/3] Building & running the Wails shell"
cd "$SCRIPT_DIR"
mkdir -p bin
go build -o "bin/$WAILS_EXE" .

# On macOS: wrap in a minimal .app bundle so the Dock and Cmd+Tab use the
# proper .icns icon (CFBundleIconFile) instead of the programmatic NSImage.
# Without a bundle, macOS scales the icon differently and it appears larger
# than other apps.
if [ "$(uname -s)" = "Darwin" ]; then
  # Keep the .icns in sync with the master (padded to Apple's icon-grid safe
  # area). Pure-Node generator, so it works without sips/iconutil; non-fatal if
  # sharp isn't available — we fall back to the committed icns.
  ( cd build && node gen-icons.mjs ) >/dev/null 2>&1 || true

  APP="bin/Prizrak-Box.app"
  mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
  cp build/darwin/appicon.icns "$APP/Contents/Resources/"
  cp build/darwin/Info.plist   "$APP/Contents/"
  # CFBundleExecutable in Info.plist is "Prizrak-Box" — binary must match exactly
  cp "bin/$WAILS_EXE" "$APP/Contents/MacOS/Prizrak-Box"
  codesign --force --deep --sign - "$APP" 2>/dev/null || true

  # macOS caches Dock / Cmd+Tab icons by bundle path, so a replaced .icns often
  # won't show until the bundle is touched, re-registered, and the icon caches
  # are flushed. Do all three so the corrected icon appears on the next launch.
  touch "$APP"
  /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister \
    -f "$APP" >/dev/null 2>&1 || true
  rm -rf "$(getconf DARWIN_USER_CACHE_DIR 2>/dev/null)/com.apple.iconservices.store" 2>/dev/null || true
  killall Dock 2>/dev/null || true

  echo "    launching $APP ..."
  exec "$APP/Contents/MacOS/Prizrak-Box"
fi

echo "    launching bin/$WAILS_EXE ..."
exec "./bin/$WAILS_EXE"
