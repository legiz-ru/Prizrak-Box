// Latency presentation shared by MySearch and the Proxies page. Thresholds
// follow api/proxies getClass(): <=300 ms good, <=600 ms fair, above is slow;
// 99999 is the API's "no result / timeout" sentinel.

export type DelayTone = 'good' | 'fair' | 'slow' | 'none';

export function delayTone(delay: number | undefined | null): DelayTone {
    if (delay === undefined || delay === null || delay === 99999 || delay <= 0) return 'none';
    if (delay <= 300) return 'good';
    if (delay <= 600) return 'fair';
    return 'slow';
}

export function delayColor(delay: number | undefined | null): string {
    switch (delayTone(delay)) {
        case 'good':
            return 'var(--success)';
        case 'fair':
            return 'var(--warning)';
        case 'slow':
            return 'var(--error)';
        default:
            return 'var(--text-3)';
    }
}

// 99999 covers both "never tested" and "failed", so it gets the neutral dash
// rather than a red "timeout" (dev_21 hid the label entirely).
export function delayLabel(delay: number | undefined | null): string {
    if (delay === undefined || delay === null || delay <= 0 || delay === 99999) return '—';
    return `${delay} ms`;
}
