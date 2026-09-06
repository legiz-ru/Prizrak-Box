// Subscription expiry/traffic reminders — shared logic between the "renew
// subscription" nudge button (ActiveProfile.vue) and the native-notification
// pipeline (main.ts for Electron, wails-shim.ts for Wails). Mirrors the Go
// source of truth in src-go/internal/subscriptionalerts — see that package's
// doc comment and docs/.../supported-headers.mdx for the user-facing spec.

const DAY_MS = 24 * 60 * 60 * 1000;

// Only ever applied to the renew button, and only when the panel sent no
// notify-expire-days at all — the one place in the whole feature where an
// absent header does not disable the behavior (see
// src-go/internal/subscriptionalerts.DefaultExpireDays). Never used for
// deciding whether to fire a push notification; that stays fully opt-in.
export const DEFAULT_EXPIRE_DAYS = [1, 3, 7];

const MAX_CLOCK_SKEW_AGE_SECONDS = 30 * 24 * 60 * 60;

// Mirrors models.Profile.ClockSkewMillis (Go): the Date-header correction, or
// 0 once it's older than 30 days — a stale correction is more likely to be
// wrong (device clock resynced since) than to still be accurate.
export function clockSkewMillis(profile: any): number {
    const seconds = Number(profile?.clockSkewSeconds ?? 0);
    const at = Number(profile?.clockSkewAtSeconds ?? 0);
    if (!seconds || !at) {
        return 0;
    }
    const ageSeconds = Date.now() / 1000 - at;
    if (ageSeconds < 0 || ageSeconds > MAX_CLOCK_SKEW_AGE_SECONDS) {
        return 0;
    }
    return seconds * 1000;
}

// Mirrors models.Profile.ExpireMillis (Go) — profile.expire is
// "YYYY-MM-DD HH:mm" in local time (see utils.GetDateTime on the backend).
export function expireMillis(profile: any): number {
    const raw = profile?.expire;
    if (!raw || typeof raw !== 'string') {
        return 0;
    }
    // "2026-07-16 14:30" -> "2026-07-16T14:30", parsed as local time by every
    // engine Prizrak-Box ships on (Chromium/WebView2/WebKit).
    const parsed = Date.parse(raw.replace(' ', 'T'));
    return Number.isNaN(parsed) ? 0 : parsed;
}

// Mirrors subscriptionalerts.percentReached (Go) — integer-ish arithmetic to
// avoid float surprises right at a threshold boundary.
export function percentReached(used: number, total: number, threshold: number): boolean {
    if (total <= 0) {
        return false;
    }
    const whole = Math.floor(used / total) * 100;
    const rest = Math.floor((used % total) * 100 / total);
    return whole + rest >= threshold;
}

// Whether the "renew subscription" nudge button should be visible right now
// (ActiveProfile.vue's showRenewButton) — independent of the "Subscription
// reminders" setting (this is a UI element, not a push notification) and of
// whatever the push-notification pipeline has already shown: it simply
// reflects the subscription's current state.
export function shouldShowRenewButton(profile: any): boolean {
    if (!profile || !profile.renewUrl) {
        return false;
    }

    const now = Date.now() + clockSkewMillis(profile);

    const expireAt = expireMillis(profile);
    const expireDays: number[] = profile.notifyExpireDays?.length
        ? profile.notifyExpireDays
        : DEFAULT_EXPIRE_DAYS;
    const remaining = expireAt - now;
    const expireSoon = expireAt > 0 && expireDays.length > 0 &&
        (remaining <= 0 || expireDays.some((d: number) => remaining <= d * DAY_MS));

    const total = Number(profile.total ?? 0);
    const used = Number(profile.used ?? 0);
    const trafficPercent: number[] | undefined = profile.notifyTrafficPercent;
    const trafficSoon = total > 0 && !!trafficPercent?.length &&
        trafficPercent.some((p: number) => percentReached(used, total, p));

    return expireSoon || trafficSoon;
}

export interface SubscriptionAlert {
    kind: 'expired' | 'expires_in' | 'traffic_used';
    days?: number;
    percent?: number;
}

// Builds the (title, body) pair for a native OS notification, and the same
// body text is reused as the click-response modal's message.
export function formatAlertText(t: (key: string, params?: any) => string, alert: SubscriptionAlert): string {
    switch (alert.kind) {
        case 'expired':
            return t('subscriptionAlert.expired');
        case 'expires_in':
            return t('subscriptionAlert.expiresIn', {days: alert.days});
        case 'traffic_used':
            return t('subscriptionAlert.trafficUsed', {percent: alert.percent});
        default:
            return '';
    }
}

// --- Click round-trip ------------------------------------------------------
//
// Both shells end up dispatching this same window CustomEvent once a native
// notification is clicked, so the modal (SubscriptionAlertModal.vue) only
// ever has to listen to one thing:
//   - Electron: the Notification.onclick closure calls
//     notifySubscriptionAlertClicked directly (same renderer JS context, no
//     IPC needed — see main.ts/App.vue's Electron notify path).
//   - Wails: the click round-trips through the Go NotificationService
//     (OnNotificationResponse in src-wails/main.go), which emits
//     "px:be:subscriptionAlertClicked"; SubscriptionAlertModal.vue listens
//     for it via the existing Events bridge (src/runtime) and re-dispatches
//     the same CustomEvent, same as serviceEvents.ts's pattern for
//     cross-component signaling.
export const SUBSCRIPTION_ALERT_CLICKED_EVENT = 'subscription-alert-clicked';

export interface SubscriptionAlertClickDetail {
    profileId: string;
    kind: SubscriptionAlert['kind'];
    days?: number;
    percent?: number;
}

export function notifySubscriptionAlertClicked(detail: SubscriptionAlertClickDetail): void {
    window.dispatchEvent(new CustomEvent(SUBSCRIPTION_ALERT_CLICKED_EVENT, {detail}));
}

// --- Delivery ---------------------------------------------------------------
//
// Walks the profiles just fetched (e.g. App.vue's loadProfiles) for any
// profile.pendingAlerts the backend computed since the last time we looked
// (see src-go/internal.EvaluateSubscriptionAlerts), fires a native
// notification for each — gated by the local "Subscription reminders"
// setting — and always acks them (clears pendingAlerts server-side)
// regardless of that setting: the backend's own NotifiedAlerts dedup already
// guarantees each threshold is computed at most once, so there is nothing to
// gain by holding onto a pending alert the user has opted out of seeing, and
// holding onto it would just replay it the moment they turn the setting back
// on.
export async function checkPendingSubscriptionAlerts(
    profiles: any[],
    opts: {
        enabled: boolean;
        notify: (profile: any, alert: SubscriptionAlert) => void;
        ack: (profileId: string) => Promise<void>;
    },
): Promise<void> {
    if (!Array.isArray(profiles)) {
        return;
    }

    for (const profile of profiles) {
        const alerts: SubscriptionAlert[] | undefined = profile?.pendingAlerts;
        if (!alerts || !alerts.length) {
            continue;
        }

        if (opts.enabled) {
            for (const alert of alerts) {
                opts.notify(profile, alert);
            }
        }

        try {
            await opts.ack(profile.id);
        } catch {
            // Best-effort: if the ack fails, the same alert may resurface once
            // on the next check, which is preferable to losing it silently.
        }
    }
}
