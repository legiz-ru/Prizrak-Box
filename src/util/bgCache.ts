// In-memory cache populated by initBgCache() during bootstrap (before Pinia/Vue mount).
// Backed by userData/px-bg-cache.json via IPC — port-independent, survives restarts.
let _for: string | null = null;
let _data: string | null = null;

export function initBgCache(forBg: string | null, data: string | null): void {
    _for = forBg;
    _data = data;
}

export function getCachedBg(forBackground: string): string | null {
    if (!_for || !_data || _for !== forBackground) return null;
    return _data;
}

export function setCachedBg(forBackground: string, dataUrl: string): void {
    _for = forBackground;
    _data = dataUrl;
    try {
        (window as any)['pxBgCache']?.write(forBackground, dataUrl)
            .catch((e: unknown) => console.warn('[bg-cache] IPC write failed:', e));
    } catch (e) {
        console.warn('[bg-cache] write error:', e);
    }
}

export function clearCachedBg(): void {
    _for = null;
    _data = null;
    try {
        (window as any)['pxBgCache']?.clear().catch(() => {});
    } catch {}
}
