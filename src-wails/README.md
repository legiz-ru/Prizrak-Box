# src-wails — Wails v3 desktop shell (migration, Phase 0)

This directory contains the **Wails v3** desktop shell for Prizrak-Box. It is
being built **in parallel** with the existing Electron shell (`src-electron/`)
so the app keeps working throughout the migration. See
`doc/wails-v3-migration-analysis.md` for the full plan.

## What works (Phase 0 + Phase 1)

Phase 0 (PoC):
- Wails v3 application that serves the existing Vue frontend (repo-root `/src`).
- `CoreService` spawns and supervises the `px` backend:
  - runs a loopback callback server answering `/pxStore` (captures
    `port`/`secret`) and `/pxAlive` (keep-alive; `px` self-exits when the GUI
    closes),
  - spawns `px -addr=<cb> -home=<dir>`,
  - hands `host/port/secret` to the frontend via the window URL query string
    (same mechanism the Electron shell used → zero frontend changes for the
    connection).
- Minimal native system tray (Show / Hide / Quit).
- Single-instance lock.
- Frontend compatibility shim (`/src/wails-shim.ts`) that provides the
  `window.px*` globals the Vue app expects; it is a **no-op under Electron**.

Phase 1:
- **TUN service management** — `TunService` is a Go IPC client for `px-service`
  (unix socket / Windows named pipe, same protocol as `src-service/ipc`):
  `getStatus / isRunning / install / uninstall / restartBackend /
  showInstallDialog`. Install/uninstall elevate via PowerShell RunAs (Windows),
  `osascript` (macOS) or `pkexec`/`sudo` (Linux). `restartBackend` re-launches
  `px` through the elevated service (for TUN) or directly.
- **Deep links** (`prizrak-box://install-config?...`) — delivered to the
  running instance via `ApplicationLaunchedWithUrl` (macOS) and second-instance
  argv (Windows/Linux), forwarded to the frontend as a Wails `deeplink` event.
  OS registration of the scheme happens at packaging time via `build/config.yml`.
- **Launch at login** — `SystemService.AutostartEnabled / SetAutostart`. On
  macOS/Linux it is backed by the built-in Wails v3 Autostart manager; on
  Windows it registers a Task Scheduler logon task with a 15-second delay
  (`services/autostart_windows.go`) instead of the registry Run key, so the
  tray icon doesn't race explorer.exe at logon. Enabling/disabling migrates
  (removes) any legacy Run-key entry from older builds.
- **Persistent store** — the shim's `pxStore` is backed by the Wails kvstore
  service writing the electron-store file (`<home>/px-electron.db/config.json`),
  so settings are shared with the Electron shell; legacy localStorage entries
  migrate forward on first read.
- **Native notifications** — the Wails notifications service, exposed to the
  frontend as `window.pxNotify(title, body)`; used for the "update available"
  notification (WebView2 has no Web Notification API).
- **Global hotkey** — `app.GlobalShortcut` (all platforms); registration
  results are reported back to the frontend on `shortcut:result`.
- Generated Go bindings live in `frontend/bindings/` (regenerate with
  `wails3 task bindings`, i.e. `wails3 generate bindings -d frontend/bindings
  -time-type Date` — `time.Time` values map to JS `Date`); the shim imports
  them via the `@wbind` Vite alias.
- The Wails **JS runtime is not bundled from npm**: the production build maps
  `@wailsio/runtime` to `/wails/runtime.js`, served by the Go binary itself
  (see `vite.config.ts` build.rollupOptions), so the JS and Go sides of the
  runtime can never drift apart. The npm dev-dependency exists for types and
  `vite dev` only — dev mode resolves to node_modules, where features newer
  than the published npm package (e.g. non-client regions) are unavailable.

## Not yet implemented (later phases)

Full dynamic tray menu (modes/profiles/groups/dashboards mirroring the Electron
tray) and config-directory migration.

## Known issues to verify

- **Exit cleanup**: confirm that quitting (tray "Quit" / Exit button) reliably
  removes the system proxy. `readyToQuit` → `api.exit` → px disables the proxy,
  and `KillPx` sends SIGINT as a fallback, but verify on macOS/Windows.
- **GLOBAL proxy group**: switching to Global mode from the tray does not show a
  `GLOBAL` group in the proxy-groups submenu. The submenu is built from the
  `proxyGroups` event the frontend emits; check whether the frontend re-emits
  groups (including GLOBAL) on mode change, or whether px must be queried.
- **Deep links** (`prizrak-box://`): only work from a registered `.app`
  bundle (run `make-macos-app.sh`, then launch the `.app` once); the dev
  binary does not register the scheme.

## Build & run

### Quickest: the helper script (no `task` needed)

```bash
# macOS / Linux (from the repo root):
./src-wails/run-dev.sh
```

```powershell
# Windows (from the repo root):
powershell -ExecutionPolicy Bypass -File .\src-wails\run-dev.ps1
```

It builds the Vue frontend into `frontend/dist`, builds `px` if missing
(the geo/model files are already vendored in `src-go/internal/em`), then
`go build`s and launches the Wails shell.

> macOS needs Xcode Command Line Tools (`xcode-select --install`).
> Linux needs `libgtk-4-dev` + `libwebkitgtk-6.0-dev`.

### Manual steps

```bash
# 1. frontend  (from repo root)
npm install
npx vite build --outDir src-wails/frontend/dist --emptyOutDir

# 2. px backend (files for go:embed are already in src-go/internal/em)
cd src-go && CGO_ENABLED=0 go build -tags=with_gvisor -trimpath -o px . && cd ..

# 3. the Wails shell
cd src-wails
go build -o bin/prizrak-box-wails . && ./bin/prizrak-box-wails
```

### Optional: `task` and `wails3`

If you install the [Task](https://taskfile.dev) runner (`brew install go-task`
on macOS) you can use the `Taskfile.yml` targets (`task frontend`, `task build`).
A proper macOS `.app` bundle (needed for the `prizrak-box://` scheme to be
registered with the OS) is produced by `wails3 build` / `wails3 package`.

Install the `wails3` CLI pinned to the same version as `go.mod`'s `wails/v3`
(the CLI generates bindings and build assets — a mismatch can produce
incompatible output):

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.118
```

### Environment overrides (handy for dev)

| Variable | Purpose |
|---|---|
| `PRIZRAK_PX_BIN` | explicit path to the `px` binary |
| `PRIZRAK_PX_SERVICE_BIN` | explicit path to `px-service` |
| `PRIZRAK_HOME` | data dir passed to `px -home` (defaults to a dedicated `Prizrak-Box-Wails` dir so it does not clobber the Electron build) |

> Note: a GUI build requires a desktop environment with WebView2 (Windows),
> WebKit (macOS) or WebKitGTK (Linux). On a headless CI box you can still
> `go build` the shell to type-check it, but you cannot launch the window.
