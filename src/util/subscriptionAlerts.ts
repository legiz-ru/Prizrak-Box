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

type Translate = (key: string, params?: any) => string;

// The text of an alert as it was raised: built from the threshold that fired
// ("expires in 4 days", "80% of traffic used"). It is only the fallback now,
// for a profile whose expiry or traffic numbers are not known — everything
// shown to the user comes from the live numbers below, because by the time a
// notification is opened the threshold can be a day or a few percent behind.
export function formatAlertText(t: Translate, alert: SubscriptionAlert): string {
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

const HOUR_MS = 60 * 60 * 1000;
const MINUTE_MS = 60 * 1000;

// "05.10.2026, 14:30" in the interface language — an absolute moment, so it
// never goes stale the way "in 4 days" does.
export function formatExpireDate(ms: number, locale?: string): string {
    return new Date(ms).toLocaleString(locale, {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
    });
}

// Time left, as a sentence: "in 4 d", "in 1 d 2 h", "in 5 h 20 min", "in 12 min".
// Only meaningful for remainingMs > 0.
export function formatRemaining(t: Translate, remainingMs: number): string {
    const days = Math.floor(remainingMs / DAY_MS);
    if (days >= 3) {
        return t('subscriptionAlert.remainingDays', {days});
    }
    if (days >= 1) {
        const hours = Math.floor((remainingMs % DAY_MS) / HOUR_MS);
        return t('subscriptionAlert.remainingDaysHours', {days, hours});
    }
    const hours = Math.floor(remainingMs / HOUR_MS);
    if (hours >= 1) {
        const minutes = Math.floor((remainingMs % HOUR_MS) / MINUTE_MS);
        return t('subscriptionAlert.remainingHoursMinutes', {hours, minutes});
    }
    return t('subscriptionAlert.remainingMinutes', {minutes: Math.max(1, Math.floor(remainingMs / MINUTE_MS))});
}

// Percent of traffic used right now, or null when the quota is unknown.
// Integer arithmetic first (used * 100 / total) so 29 of 100 is 29, not 28.
export function currentTrafficPercent(profile: any): number | null {
    const total = Number(profile?.total ?? 0);
    if (!(total > 0)) {
        return null;
    }
    const used = Math.max(0, Number(profile?.used ?? 0));
    return Math.floor((used * 100) / total);
}

export interface AlertTexts {
    // The headline: what is true right now.
    message: string;
    // An extra line with the exact expiry moment, when there is one.
    detail?: string;
}

// The text shown when a notification is opened: computed from the profile as
// it is now, not from the threshold that raised the alert.
export function describeAlert(
    t: Translate,
    alert: SubscriptionAlert,
    profile: any,
    locale?: string,
): AlertTexts {
    if (alert.kind === 'traffic_used') {
        const percent = currentTrafficPercent(profile) ?? alert.percent;
        return {message: t('subscriptionAlert.trafficUsed', {percent})};
    }

    const expireAt = expireMillis(profile);
    if (!expireAt) {
        return {message: formatAlertText(t, alert)};
    }

    const remaining = expireAt - (Date.now() + clockSkewMillis(profile));
    if (remaining <= 0) {
        return {message: t('subscriptionAlert.expired')};
    }
    return {
        message: formatRemaining(t, remaining),
        detail: t('subscriptionAlert.expiresOn', {date: formatExpireDate(expireAt, locale)}),
    };
}

// The title of the native notification. A notification is frozen when it is
// posted, so it carries the exact expiry moment instead of "in N days", and
// the traffic percent measured at that moment.
export function notificationTitle(
    t: Translate,
    alert: SubscriptionAlert,
    profile: any,
    locale?: string,
): string {
    if (alert.kind === 'traffic_used') {
        const percent = currentTrafficPercent(profile) ?? alert.percent;
        return t('subscriptionAlert.trafficUsed', {percent});
    }

    const expireAt = expireMillis(profile);
    if (alert.kind === 'expires_in' && expireAt) {
        return t('subscriptionAlert.expiresAt', {date: formatExpireDate(expireAt, locale)});
    }
    return formatAlertText(t, alert);
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
