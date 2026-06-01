# src-wails — Wails v3 desktop shell (migration, Phase 0)

This directory contains the **Wails v3** desktop shell for Prizrak-Box. It is
being built **in parallel** with the existing Electron shell (`src-electron/`)
so the app keeps working throughout the migration. See
`doc/wails-v3-migration-analysis.md` for the full plan.

## What works in Phase 0 (PoC)

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

## Not yet implemented (later phases)

Full dynamic tray menu (modes/profiles/groups/dashboards), deep-link import
(`prizrak-box://`), TUN service management (`px-service`), autostart, and
config-directory migration. The shim stubs these so the UI still boots.

## Build & run

```bash
# 1. CLI + deps
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
npm install                 # at repo root, for the Vue frontend

# 2. Build the px backend (see repo README for full flags)
cd src-go && CGO_ENABLED=0 go build -tags=with_gvisor -trimpath -o px . && cd ..

# 3. Build / run the Wails shell (uses Taskfile here)
cd src-wails
task frontend               # vite build -> frontend/dist
task build                  # wails3 build  (or: task dev)
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
