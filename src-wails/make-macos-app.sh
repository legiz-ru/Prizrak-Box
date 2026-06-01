#!/usr/bin/env bash
#
# Build a minimal macOS .app bundle for the Wails shell so the
# prizrak-box:// URL scheme is registered with LaunchServices (deep links).
#
# This is a lightweight alternative to `wails3 build` (which expects the full
# Wails Taskfile template). It wraps the same `go build` binary the dev script
# uses, plus an Info.plist that declares the custom scheme, into
# bin/Prizrak-Box.app.
#
# Usage (from anywhere):
#   ./src-wails/make-macos-app.sh
#   open ./src-wails/bin/Prizrak-Box.app                 # registers the scheme
#   open 'prizrak-box://install-config?url=https://example.com/sub'
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This script is macOS-only." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
APP="$SCRIPT_DIR/bin/Prizrak-Box.app"
ID="com.legiz-ru.prizrak-box"
VERSION="1.0.1"

echo "==> Building frontend"
cd "$REPO_ROOT"
[ -d node_modules ] || npm install --no-audit --no-fund
npx vite build --outDir src-wails/frontend/dist --emptyOutDir

echo "==> Ensuring px backend + px-service"
[ -x "src-go/px" ] || ( cd src-go && CGO_ENABLED=0 go build -tags=with_gvisor -trimpath -o px . )
[ -x "src-service/px-service" ] || ( cd src-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o px-service . )

echo "==> Building Go binary"
cd "$SCRIPT_DIR"
go build -o /tmp/prizrak-box-wails-bin .

echo "==> Assembling $APP"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp /tmp/prizrak-box-wails-bin "$APP/Contents/MacOS/Prizrak-Box"
# px + px-service ship next to the binary (locate.go finds them there).
cp "$REPO_ROOT/src-go/px" "$REPO_ROOT/src-service/px-service" "$APP/Contents/MacOS/" 2>/dev/null || true
chmod +x "$APP/Contents/MacOS/"*
[ -f "$REPO_ROOT/build/appicon.icns" ] && cp "$REPO_ROOT/build/appicon.icns" "$APP/Contents/Resources/appicon.icns"
cp "$SCRIPT_DIR/build/darwin/Info.plist" "$APP/Contents/Info.plist"

# Register the bundle (and its URL scheme) with LaunchServices.
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister \
  -f "$APP" 2>/dev/null || true

echo "==> Done: $APP"
echo "    open \"$APP\"   # then test:"
echo "    open 'prizrak-box://install-config?url=https://example.com/sub'"
