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
if [ ! -x "src-go/$PX_EXE" ]; then
  echo "    building px (geo/model files are already vendored in src-go/internal/em)..."
  ( cd src-go && CGO_ENABLED=0 go build -tags=with_gvisor -trimpath -o "$PX_EXE" . )
else
  echo "    found existing src-go/$PX_EXE"
fi
SERVICE_EXE="px-service"; [ "$PX_EXE" = "px.exe" ] && SERVICE_EXE="px-service.exe"
if [ ! -x "src-service/$SERVICE_EXE" ]; then
  echo "    building px-service (TUN helper)..."
  ( cd src-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o "$SERVICE_EXE" . )
else
  echo "    found existing src-service/$SERVICE_EXE"
fi

echo "==> [3/3] Building & running the Wails shell"
cd "$SCRIPT_DIR"
mkdir -p bin
go build -o "bin/$WAILS_EXE" .
echo "    launching bin/$WAILS_EXE ..."
exec "./bin/$WAILS_EXE"
