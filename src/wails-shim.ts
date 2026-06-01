// Wails v3 compatibility shim.
//
// The frontend was written for the Electron shell, which exposed a set of
// `window.px*` globals via src-electron/preload.ts. Under the Wails v3 shell
// (src-wails/) those globals do not exist. This shim installs minimal
// implementations so the existing Vue app boots and talks to the px backend
// over HTTP without touching the rest of the codebase.
//
// IMPORTANT: this is a Phase 0 (PoC) shim. It is intentionally conservative:
//   - Connection info (host/port/secret) is delivered by the Wails shell via
//     the window URL query string (see src-wails/main.go), exactly like the
//     Electron shell did, so nothing here needs to fetch it.
//   - Persistent storage falls back to localStorage.
//   - Tray events, deep links and the TUN service are no-ops for now; they
//     are wired to real Go bindings in later migration phases.
//
// Under Electron, preload.ts has already defined window.pxOs, so installWailsShim()
// detects that and does nothing — the Electron build is unaffected.

type AnyWindow = Window & Record<string, any>;

export function installWailsShim(): void {
    const w = window as AnyWindow;

    // If the Electron preload already provided the bridge, do nothing.
    if (typeof w.pxOs === 'function') {
        return;
    }

    // OS string, mirroring the Electron format ("Linux x64", "MacOS arm64", ...).
    w.pxOs = (): string => {
        const ua = navigator.userAgent || '';
        let name = 'Linux';
        if (/Windows/i.test(ua)) name = 'Windows';
        else if (/Mac OS X|Macintosh/i.test(ua)) name = 'MacOS';
        const arch = /arm64|aarch64/i.test(ua) ? 'arm64' : 'x64';
        return `${name} ${arch}`;
    };

    w.pxUsername = (): string => '';

    w.pxClipboard = async (): Promise<string> => {
        try {
            return await navigator.clipboard.readText();
        } catch {
            return '';
        }
    };

    w.pxOpen = (url: string): void => {
        try {
            window.open(url, '_blank');
        } catch {
            /* ignore */
        }
    };

    w.pxShowInFolder = (_path: string): void => { /* phase 1 */ };
    w.pxConfigDir = (_path: string): void => { /* phase 1 */ };
    w.pxShowBar = (): void => { /* macOS title bar only */ };

    // Tray event bus — local no-op bus for now. Phase 2 forwards these to Go
    // via the Wails events runtime and pushes tray clicks back here.
    const listeners: Record<string, Array<(...a: any[]) => void>> = {};
    w.pxTray = {
        on: (name: string, cb: (...a: any[]) => void) => {
            (listeners[name] ||= []).push(cb);
        },
        off: (name: string, cb: (...a: any[]) => void) => {
            listeners[name] = (listeners[name] || []).filter((f) => f !== cb);
        },
        emit: (_name: string, ..._args: any[]) => { /* phase 2 */ },
    };

    // Persistent key/value store — localStorage fallback.
    w.pxStore = {
        get: async (key: string): Promise<any> => {
            const raw = localStorage.getItem('px:' + key);
            return raw ? JSON.parse(raw) : undefined;
        },
        set: async (key: string, value: any): Promise<void> => {
            localStorage.setItem('px:' + key, JSON.stringify(value));
        },
    };

    // Background image cache.
    w.pxBgCache = {
        read: async (): Promise<any> => {
            const raw = localStorage.getItem('px:bgcache');
            return raw ? JSON.parse(raw) : null;
        },
        write: async (forBg: any, dataUrl: string): Promise<void> => {
            localStorage.setItem('px:bgcache', JSON.stringify({ forBg, dataUrl }));
        },
        clear: async (): Promise<void> => {
            localStorage.removeItem('px:bgcache');
        },
    };

    // TUN service — stubbed until Phase 1 wires TunService bindings.
    w.pxService = {
        getStatus: async () => ({ installed: false, running: false }),
        isRunning: async () => false,
        install: async () => false,
        uninstall: async () => false,
        restartBackend: async () => { /* phase 1 */ },
        showInstallDialog: async () => { /* phase 1 */ },
    };

    // Deep link — stubbed until Phase 1.
    w.pxDeepLink = {
        onImportProfile: (_cb: (...a: any[]) => void) => { /* phase 1 */ },
        notifyReady: () => { /* phase 1 */ },
    };

    w.pxPreConfigDir = async (): Promise<string> => '';
    w.pxChangeConfigDir = async (_dir: string): Promise<void> => { /* phase 1 */ };

    // Generic invoke used by a few call sites (e.g. select-directory).
    w.electron = {
        invoke: async (_channel: string, ..._args: any[]): Promise<any> => undefined,
    };
}
