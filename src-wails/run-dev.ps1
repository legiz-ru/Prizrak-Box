# Build & run the Wails v3 shell on Windows (PowerShell), no `task` required.
#
# Mirrors run-dev.sh:
#   1. builds the Vue frontend into src-wails/frontend/dist (embedded by the exe)
#   2. builds px.exe + px-service.exe if missing
#   3. builds and runs the Wails shell
#
# Usage (from anywhere):
#   powershell -ExecutionPolicy Bypass -File .\src-wails\run-dev.ps1
#
# Requirements: Go 1.26+, Node 22+, and the WebView2 runtime (preinstalled on
# Windows 10/11). No C compiler needed (CGO is disabled).
$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot  = Split-Path -Parent $ScriptDir

Write-Host '==> [1/3] Building frontend (vite -> src-wails/frontend/dist)'
Set-Location $RepoRoot
# Always install so newly-added deps (e.g. @wailsio/runtime) are present even
# when node_modules was created by an earlier checkout.
npm install --no-audit --no-fund
if ($LASTEXITCODE -ne 0) { throw 'npm install failed' }
npx vite build --outDir src-wails/frontend/dist --emptyOutDir
if ($LASTEXITCODE -ne 0) { throw 'frontend build failed' }

Write-Host '==> [2/3] Ensuring px.exe + px-service.exe'
$env:CGO_ENABLED = '0'
if (-not (Test-Path src-go/px.exe)) {
    Write-Host '    building px.exe (geo/model files are vendored in src-go/internal/em)...'
    Push-Location src-go
    go build -tags=with_gvisor -trimpath -ldflags="-s -w" -o px.exe .
    Pop-Location
} else { Write-Host '    found existing src-go/px.exe' }
if (-not (Test-Path src-service/px-service.exe)) {
    Write-Host '    building px-service.exe...'
    Push-Location src-service
    go build -ldflags="-s -w" -o px-service.exe .
    Pop-Location
} else { Write-Host '    found existing src-service/px-service.exe' }

Write-Host '==> [3/3] Building & running the Wails shell'
Set-Location $ScriptDir
New-Item -ItemType Directory -Force -Path bin | Out-Null

# Generate multi-size .ico / tray.png from the master appicon.png using sharp
# (Lanczos downscale, BMP-in-ICO entries for maximum tool compatibility).
Write-Host '    generating icons (gen-icons.mjs)...'
Push-Location (Split-Path -Parent $ScriptDir)
if (Test-Path node_modules/sharp) {
    node src-wails/build/gen-icons.mjs
} else {
    Write-Host '    WARNING: sharp not available; skipping icon regeneration.'
    Write-Host '    (committed icons are used instead.)'
}
Pop-Location

# Embed the app icon into the .exe so the taskbar / Explorer icon is the app
# icon and crisp at every size (matches the release build). Best-effort: if
# go-winres isn't installed and can't be fetched, the build still works and
# Wails falls back to the embedded PNG for the window icon.
Remove-Item rsrc_windows_*.syso -ErrorAction SilentlyContinue
if (-not (Get-Command go-winres -ErrorAction SilentlyContinue)) {
    Write-Host '    installing go-winres (one-time)...'
    go install github.com/tc-hib/go-winres@latest 2>$null
}
$winres = Get-Command go-winres -ErrorAction SilentlyContinue
if ($winres) {
    go-winres simply --icon build/appicon.ico --manifest gui --product-name "Prizrak-Box"
} else {
    Write-Host '    go-winres unavailable; building without embedded .exe icon'
}

go build -trimpath -o bin/prizrak-box-wails.exe .
Remove-Item rsrc_windows_*.syso -ErrorAction SilentlyContinue
Write-Host '    launching bin/prizrak-box-wails.exe ...'
& ./bin/prizrak-box-wails.exe
