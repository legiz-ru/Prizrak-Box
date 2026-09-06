// Synchronous localStorage cache of the active profile's custom logo + header
// title (from the Profile-Logo / Profile-Title subscription headers).
//
// The logo content itself is already persisted by the backend (profile.Logo is
// a base64 data URL stored in BoltDB). This cache only lets the custom logo and
// title render INSTANTLY on launch — before the profile list HTTP request
// returns — so there is no flash of the default Prizrak-Box logo. It is kept in
// sync on apply / refresh / rollback in App.vue and works in both the Electron
// and Wails shells (localStorage is available in both webviews).

const KEY = 'px-active-logo';

export interface CachedLogo {
    id: string;
    logo: string;
    title: string;
}

export function getCachedLogo(): CachedLogo | null {
    try {
        const raw = localStorage.getItem(KEY);
        if (!raw) return null;
        const v = JSON.parse(raw);
        if (v && typeof v.logo === 'string' && v.logo.trim() !== '') {
            return {id: String(v.id ?? ''), logo: v.logo, title: String(v.title ?? '')};
        }
    } catch {
        /* ignore */
    }
    return null;
}

export function setCachedLogo(id: string, logo: string, title: string): void {
    try {
        if (!logo || !logo.trim()) {
            clearCachedLogo();
            return;
        }
        localStorage.setItem(KEY, JSON.stringify({id: id || '', logo, title: title || ''}));
    } catch {
        /* ignore */
    }
}

export function clearCachedLogo(): void {
    try {
        localStorage.removeItem(KEY);
    } catch {
        /* ignore */
    }
}
