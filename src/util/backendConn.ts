// Backend connection (host/port/secret) management.
//
// The shell hands the frontend the px control-plane host/port/secret through
// the window URL query string once, at startup (see src-wails/main.go and
// src-electron/main.ts). The axios instance is built from those values a single
// time during bootstrap.
//
// That is fine until px is restarted *while the app keeps running* — which is
// exactly what installing/uninstalling the TUN service does. The fresh px picks
// its control port again (9686 when free, otherwise a random one) and the
// elevated service may spawn it under a different account, so the values the
// frontend captured at boot can go stale. Every later request then fails
// (connection refused / 401) even though the backend is perfectly healthy,
// which is why the app only "worked after a restart".
//
// The Wails shell returns the fresh ConnInfo from pxService.restartBackend();
// applyBackendConn() feeds it back into the store and rebuilds the axios
// instance so the running app keeps talking to the right px.

import {AxiosRequest} from "@/util/axiosRequest";
import {useWebStore} from "@/store/webStore";

// Fired on window after host/port/secret changed, so long-lived consumers
// (websockets built from webStore.wsUrl) can reconnect.
export const BACKEND_CONN_UPDATED_EVENT = 'backend-conn-updated';

export interface BackendConn {
    host?: string;
    port?: number | string;
    secret?: string;
}

// app.config.globalProperties — $http lives there and is resolved at call time
// by every api wrapper, so replacing it swaps the client app-wide.
let httpTarget: any = null;

export function registerHttpTarget(globalProperties: any): void {
    httpTarget = globalProperties;
}

// Rebuilds the axios instance from the current store values.
function rebuildHttp(): void {
    if (!httpTarget) return;
    const webStore = useWebStore();
    httpTarget.$http = new AxiosRequest(webStore.baseUrl, webStore.secret);
}

/**
 * Applies connection info reported by the native shell after a px restart.
 * Returns true when the info was usable.
 */
export function applyBackendConn(info: BackendConn | null | undefined): boolean {
    if (!info || typeof info !== 'object') return false;

    const port = info.port === undefined || info.port === null ? '' : String(info.port);
    if (!port || port === '0') return false;

    const webStore = useWebStore();
    const host = info.host || webStore.host;
    const secret = info.secret || webStore.secret;
    const changed = host !== webStore.host || port !== webStore.port || secret !== webStore.secret;

    webStore.setHost(host);
    webStore.setPort(port);
    webStore.setSecret(secret);
    rebuildHttp();

    if (changed) {
        window.dispatchEvent(new CustomEvent(BACKEND_CONN_UPDATED_EVENT));
    }
    return true;
}

/**
 * Restarts px through the native shell and syncs the new connection info.
 *
 * The Wails shell resolves with the fresh ConnInfo, the Electron shell with a
 * plain boolean — both are accepted. Returns false when the shell could not
 * restart the backend (the caller should then ask the user to restart the app).
 */
export async function restartBackendAndSync(): Promise<boolean> {
    const service = (window as any).pxService;
    if (!service || typeof service.restartBackend !== 'function') {
        return false;
    }

    const result = await service.restartBackend();
    if (!result) return false;

    if (typeof result === 'object') {
        // A ConnInfo without a port means the backend never called back.
        return applyBackendConn(result);
    }
    return true;
}

/**
 * Polls the backend until it answers (or the budget runs out).
 *
 * NOTE: api.waitRunning() deliberately swallows every error (see
 * src/api/mihomo/index.ts), so it can never be used to detect that px failed to
 * come back — callers that need a real answer must use this instead.
 */
export async function waitBackendReady(api: any, timeoutMs = 10000): Promise<boolean> {
    const deadline = Date.now() + timeoutMs;
    for (;;) {
        try {
            await api.getVersion();
            return true;
        } catch (e) {
            if (Date.now() >= deadline) return false;
            await new Promise((resolve) => setTimeout(resolve, 400));
        }
    }
}
