#!/usr/bin/env bash
#
# Local macOS build of the Wails v3 shell — produces an UNSIGNED .app + .dmg.
#
# Mirrors the "macos" job of .github/workflows/wails-build.yml step by step,
# minus the parts that need Apple secrets (Developer ID signing, notarization,
# stapling). The bundle gets an ad-hoc signature instead (`codesign --sign -`),
# which is what macOS needs to run a locally built app on Apple Silicon; it is
# NOT a Developer ID signature, so Gatekeeper still flags the app as coming
# from an unidentified developer — see the quarantine hint printed at the end.
#
# Usage (from anywhere):
#   ./src-wails/build-macos-dmg.sh                 # arm64, everything
#   ./src-wails/build-macos-dmg.sh --arch amd64    # Intel build
#   ./src-wails/build-macos-dmg.sh --skip-frontend # reuse frontend/dist
#   ./src-wails/build-macos-dmg.sh --refresh-geo   # re-download geo/model data
#   ./src-wails/build-macos-dmg.sh --gen-icons     # regenerate icons (needs sharp)
#   ./src-wails/build-macos-dmg.sh --no-adhoc-sign
#   ./src-wails/build-macos-dmg.sh --help
#
# Output (both git-ignored, see src-wails/.gitignore):
#   src-wails/bin/prizrak-box-wails-macos-<arch>.dmg
#   src-wails/bin/Prizrak-Box.app
set -euo pipefail

ARCH="arm64"
REFRESH_GEO=0
GEN_ICONS=0
SKIP_FRONTEND=0
ADHOC_SIGN=1

# Prints the header comment block (everything between the shebang and the first
# line of code) as the help text.
usage() {
  awk 'NR>1 && /^#/ { sub(/^# ?/, ""); print; next } NR>1 { exit }' "${BASH_SOURCE[0]}"
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch)          ARCH="${2:-}"; shift 2 ;;
    --arch=*)        ARCH="${1#*=}"; shift ;;
    --refresh-geo)   REFRESH_GEO=1; shift ;;
    --gen-icons)     GEN_ICONS=1; shift ;;
    --skip-frontend) SKIP_FRONTEND=1; shift ;;
    --no-adhoc-sign) ADHOC_SIGN=0; shift ;;
    -h|--help)       usage ;;
    *) echo "Unknown option: $1 (try --help)" >&2; exit 2 ;;
  esac
done

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This script is macOS-only (it uses codesign / hdiutil)." >&2
  exit 1
fi
if [[ "$ARCH" != "arm64" && "$ARCH" != "amd64" ]]; then
  echo "--arch must be arm64 or amd64 (got: $ARCH)" >&2
  exit 2
fi

for tool in go node npm; do
  command -v "$tool" >/dev/null 2>&1 || { echo "Required tool not found: $tool" >&2; exit 1; }
done
# The Wails shell links against Cocoa/WebKit, so cgo needs the command line tools.
xcode-select -p >/dev/null 2>&1 || {
  echo "Xcode command line tools are required (run: xcode-select --install)" >&2
  exit 1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT_DIR="$SCRIPT_DIR/bin"
APP_NAME="Prizrak-Box.app"
# create-macos-dmg.mjs expects the bundle at the repo root; it is moved into
# bin/ once the .dmg has been built.
APP_STAGE="$REPO_ROOT/$APP_NAME"
DMG_NAME="prizrak-box-wails-macos-${ARCH}.dmg"
GO_BIN="$(mktemp -t prizrak-box-wails)"

cd "$REPO_ROOT"
PKG_VERSION="$(node -p "require('./package.json').version")"
mkdir -p "$OUT_DIR"

echo "==> Prizrak-Box $PKG_VERSION — macOS/$ARCH (unsigned)"

# ── Geo/model data ────────────────────────────────────────────────────────────
# Committed in the repo; refresh only on request (same sources as the workflow).
if [[ "$REFRESH_GEO" == "1" ]]; then
  echo "==> Refreshing geo/model data"
  curl -fsSL --retry 3 --retry-delay 5 -o src-go/internal/em/Model.bin        https://github.com/vernesong/mihomo/releases/download/LightGBM-Model/Model.bin
  curl -fsSL --retry 3 --retry-delay 5 -o src-go/internal/em/geoip.metadb     https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb
  curl -fsSL --retry 3 --retry-delay 5 -o src-go/internal/em/GeoSite.dat      https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat
  curl -fsSL --retry 3 --retry-delay 5 -o src-go/internal/em/BundleMRS.7z     https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/BundleMRS.7z
  curl -fsSL --retry 3 --retry-delay 5 -o src-go/internal/em/GeoLite2-ASN.mmdb https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/GeoLite2-ASN.mmdb
fi

# ── Frontend ──────────────────────────────────────────────────────────────────
if [[ "$SKIP_FRONTEND" == "1" ]]; then
  echo "==> Skipping frontend build (--skip-frontend)"
  [[ -f "$SCRIPT_DIR/frontend/dist/index.html" ]] || {
    echo "frontend/dist is empty — run without --skip-frontend first." >&2
    exit 1
  }
else
  echo "==> Building frontend"
  [[ -d node_modules ]] || npm install --no-audit --no-fund
  npx vite build --outDir src-wails/frontend/dist --emptyOutDir
fi

# ── px + px-service ───────────────────────────────────────────────────────────
echo "==> Building px + px-service (GOARCH=$ARCH)"
( cd src-go && GOARCH="$ARCH" CGO_ENABLED=0 go build -tags=with_gvisor -trimpath \
    -ldflags="-s -w -X github.com/legiz-ru/prizrak-box/api.Version=${PKG_VERSION}" -o px . )
( cd src-service && GOARCH="$ARCH" CGO_ENABLED=0 go build -ldflags="-s -w" -o px-service . )

# ── Wails shell ───────────────────────────────────────────────────────────────
echo "==> Building Wails shell (GOARCH=$ARCH, cgo)"
( cd "$SCRIPT_DIR" && GOARCH="$ARCH" CGO_ENABLED=1 go build -trimpath -o "$GO_BIN" . )

# ── Icons ─────────────────────────────────────────────────────────────────────
# The generated icons are committed, so regenerating them would show up as a
# working-tree change — do it only on request (needs the `sharp` dev dep).
if [[ "$GEN_ICONS" == "1" ]]; then
  echo "==> Regenerating icons"
  node src-wails/build/gen-icons.mjs
fi

# ── Assemble the .app ─────────────────────────────────────────────────────────
echo "==> Assembling $APP_NAME"
rm -rf "$APP_STAGE"
mkdir -p "$APP_STAGE/Contents/MacOS" "$APP_STAGE/Contents/Resources"
cp "$GO_BIN" "$APP_STAGE/Contents/MacOS/Prizrak-Box"
# px + px-service ship next to the shell binary (internal/locate finds them there).
cp src-go/px src-service/px-service "$APP_STAGE/Contents/MacOS/"
chmod +x "$APP_STAGE/Contents/MacOS/"*
if [[ -f build/appicon.icns ]]; then
  cp build/appicon.icns "$APP_STAGE/Contents/Resources/appicon.icns"
elif [[ -f src-wails/build/darwin/appicon.icns ]]; then
  cp src-wails/build/darwin/appicon.icns "$APP_STAGE/Contents/Resources/appicon.icns"
fi
cp src-wails/build/darwin/Info.plist "$APP_STAGE/Contents/Info.plist"
rm -f "$GO_BIN"

# ── Ad-hoc signature (replaces the Developer ID signing + notarization steps) ──
# No hardened runtime and no entitlements here: those are notarization
# requirements, and enabling the hardened runtime on an ad-hoc signature only
# adds ways for a local build to be killed at launch. A plain ad-hoc signature
# is what Apple Silicon needs to execute the binaries, and it gives the bundle
# a stable code identity (macOS ties notification/network permissions to it).
if [[ "$ADHOC_SIGN" == "1" ]]; then
  echo "==> Ad-hoc signing (no Developer ID / no notarization)"
  codesign --force --sign - "$APP_STAGE/Contents/MacOS/px" "$APP_STAGE/Contents/MacOS/px-service"
  codesign --force --deep --sign - "$APP_STAGE"
  codesign --verify --deep --strict "$APP_STAGE"
fi

# ── .dmg ──────────────────────────────────────────────────────────────────────
echo "==> Packaging $DMG_NAME"
# PB_UNSIGNED switches the in-DMG READMEs to the unsigned wording (quarantine
# instructions instead of "the app is signed").
PB_UNSIGNED=1 node src-wails/build/darwin/create-macos-dmg.mjs "$ARCH" "src-wails/bin/$DMG_NAME"

# Keep the bundle around for local testing, but out of the repo root.
rm -rf "$OUT_DIR/$APP_NAME"
mv "$APP_STAGE" "$OUT_DIR/$APP_NAME"
# Register the bundle (and its prizrak-box:// scheme) with LaunchServices.
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister \
  -f "$OUT_DIR/$APP_NAME" 2>/dev/null || true

cat <<EOF

==> Done
    DMG : $OUT_DIR/$DMG_NAME
    APP : $OUT_DIR/$APP_NAME

The build is not signed with a Developer ID and not notarized, so macOS puts
anything downloaded from this DMG into quarantine and refuses to open it
("Prizrak-Box is damaged" / "cannot be opened because the developer cannot be
verified"). Remove the quarantine flag after installing:

    xattr -dr com.apple.quarantine /Applications/Prizrak-Box.app

If the DMG itself was downloaded (browser / AirDrop / Telegram), clear it there
too before opening:

    xattr -dr com.apple.quarantine ~/Downloads/$DMG_NAME

Run it straight from here without installing:

    open "$OUT_DIR/$APP_NAME"
EOF
