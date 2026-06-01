// Wails v3 compatibility shim.
//
// The frontend was written for the Electron shell, which exposed a set of
// `window.px*` globals via src-electron/preload.ts. Under the Wails v3 shell
// (src-wails/) those globals do not exist. This shim installs implementations
// backed by the Wails runtime + generated Go service bindings so the existing
// Vue app runs unchanged.
//
// Design notes:
//   - Connection info (host/port/secret) is delivered by the Wails shell via
//     the window URL query string (see src-wails/main.go), exactly like the
//     Electron shell did, so nothing here fetches it.
//   - The Wails runtime and the generated bindings are *dynamically imported*
//     and only on platforms where the Electron preload is absent. Under
//     Electron, window.pxOs already exists, installWailsShim() returns early,
//     and none of this code runs — the Electron build is unaffected.
//
// Phase 1 wires: TUN service (TunService), deep links (ApplicationLaunchedWithUrl
// → "deeplink" event), clipboard / open-external. Persistent storage still
// falls back to localStorage; the full StoreService + dynamic tray land later.

type AnyWindow = Window & Record<string, any>;

// Lazily-loaded modules (only fetched under the Wails shell).
let runtimePromise: Promise<any> | null = null;
const runtime = () => (runtimePromise ||= import('@wailsio/runtime'));

let servicesPromise: Promise<any> | null = null;
const services = () =>
    (servicesPromise ||= import(
        '@wbind/github.com/legiz-ru/prizrak-box-wails/services/index.js'
    ));

export function installWailsShim(): void {
    const w = window as AnyWindow;

    // If the Electron preload already provided the bridge, do nothing.
    if (typeof w.pxOs === 'function') {
        return;
    }

    // --- Synchronous OS helpers (no runtime needed) ---
    w.pxOs = (): string => {
        const ua = navigator.userAgent || '';
        let name = 'Linux';
        if (/Windows/i.test(ua)) name = 'Windows';
        else if (/Mac OS X|Macintosh/i.test(ua)) name = 'MacOS';
        const arch = /arm64|aarch64/i.test(ua) ? 'arm64' : 'x64';
        return `${name} ${arch}`;
    };
    w.pxUsername = (): string => '';

    // --- Clipboard / open-external via the Wails runtime ---
    w.pxClipboard = async (): Promise<string> => {
        try {
            const { Clipboard } = await runtime();
            return await Clipboard.Text();
        } catch {
            try {
                return await navigator.clipboard.readText();
            } catch {
                return '';
            }
        }
    };
    w.pxOpen = (url: string): void => {
        runtime()
            .then(({ Browser }) => Browser.OpenURL(url))
            .catch(() => {
                try {
                    window.open(url, '_blank');
                } catch {
                    /* ignore */
                }
            });
    };

    w.pxShowInFolder = (_path: string): void => { /* later phase */ };
    w.pxConfigDir = (_path: string): void => { /* later phase */ };
    w.pxShowBar = (): void => { /* macOS title bar only */ };

    // --- Tray event bus, backed by Wails events (Go <-> frontend) ---
    w.pxTray = {
        on: (name: string, cb: (...a: any[]) => void) => {
            runtime()
                .then(({ Events }) => Events.On(name, (e: any) => cb(e?.data)))
                .catch(() => { /* ignore */ });
        },
        off: (_name: string, _cb: (...a: any[]) => void) => { /* later phase */ },
        emit: (name: string, data: any) => {
            runtime()
                .then(({ Events }) => Events.Emit(name, data))
                .catch(() => { /* ignore */ });
        },
    };

    // --- Deep link: forward the Wails "deeplink" event to the importer ---
    w.pxDeepLink = {
        onImportProfile: (cb: (payload: any) => void) => {
            runtime()
                .then(({ Events }) =>
                    Events.On('deeplink', (e: any) => cb({ rawUrl: e?.data }))
                )
                .catch(() => { /* ignore */ });
        },
        notifyReady: () => { /* no-op: Wails delivers via event */ },
    };

    // --- TUN service via generated TunService bindings ---
    w.pxService = {
        getStatus: async () => (await services()).TunService.GetStatus(),
        isRunning: async () => (await services()).TunService.IsRunning(),
        install: async () => (await services()).TunService.Install(),
        uninstall: async () => (await services()).TunService.Uninstall(),
        restartBackend: async () => (await services()).TunService.RestartBackend(),
        showInstallDialog: async () => (await services()).TunService.ShowInstallDialog(),
    };

    // --- Persistent storage: localStorage fallback (StoreService lands later) ---
    w.pxStore = {
        get: async (key: string): Promise<any> => {
            const raw = localStorage.getItem('px:' + key);
            return raw ? JSON.parse(raw) : undefined;
        },
        set: async (key: string, value: any): Promise<void> => {
            localStorage.setItem('px:' + key, JSON.stringify(value));
        },
    };
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

    w.pxPreConfigDir = async (): Promise<string> => '';
    w.pxChangeConfigDir = async (_dir: string): Promise<void> => { /* later phase */ };

    w.electron = {
        invoke: async (_channel: string, ..._args: any[]): Promise<any> => undefined,
    };
}
