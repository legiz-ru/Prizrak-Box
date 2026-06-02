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
if (-not (Test-Path node_modules)) {
    npm install --no-audit --no-fund
}
npx vite build --outDir src-wails/frontend/dist --emptyOutDir

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
go build -trimpath -o bin/prizrak-box-wails.exe .
Write-Host '    launching bin/prizrak-box-wails.exe ...'
& ./bin/prizrak-box-wails.exe
