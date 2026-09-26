// Presentation helpers shared by the Home card and the Profiles page.
import {prettyBytes} from "@/util/format";

type Translate = (key: string, values?: Record<string, unknown>) => string;

export function hasValue(value: any) {
    return value !== undefined && value !== null && value !== '';
}

export function formatTrafficValue(value: any) {
    if (!hasValue(value)) {
        return '';
    }
    const num = Number(value);
    if (Number.isFinite(num)) {
        return prettyBytes(num);
    }
    return String(value);
}

/** Accepts dd.mm.yyyy, yyyy-mm-dd, any Date.parse()-able string, unix s/ms or a Date. */
export function parseDateValue(value: any): Date | null {
    if (!hasValue(value)) return null;
    if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value;
    if (typeof value === 'number') {
        const date = new Date(value > 1e12 ? value : value * 1000);
        return Number.isNaN(date.getTime()) ? null : date;
    }
    if (typeof value === 'string') {
        const trimmed = value.trim();
        let m = trimmed.match(/^(\d{2})[-/.](\d{2})[-/.](\d{4})$/);
        if (m) return new Date(Number(m[3]), Number(m[2]) - 1, Number(m[1]));
        m = trimmed.match(/^(\d{4})[-/.](\d{2})[-/.](\d{2})$/);
        if (m) return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
        const parsed = Date.parse(trimmed);
        return Number.isNaN(parsed) ? null : new Date(parsed);
    }
    return null;
}

const pad = (n: number) => String(n).padStart(2, '0');

export function formatDateValue(value: any) {
    if (!hasValue(value)) {
        return '';
    }
    const date = parseDateValue(value);
    if (!date) return String(value);
    return `${pad(date.getDate())}.${pad(date.getMonth() + 1)}.${date.getFullYear()}`;
}

/** Exact local date-time, for the tooltip next to a relative time. */
export function formatExact(value: any) {
    const date = parseDateValue(value);
    if (!date) return hasValue(value) ? String(value) : '';
    const time = date.getHours() || date.getMinutes() ? ` ${pad(date.getHours())}:${pad(date.getMinutes())}` : '';
    return `${pad(date.getDate())}.${pad(date.getMonth() + 1)}.${date.getFullYear()}${time}`;
}

/** "6 дн. назад", "сегодня", … for a past date (future dates fall back to the date). */
export function relativeDate(t: Translate, value: any) {
    const date = parseDateValue(value);
    if (!date) return hasValue(value) ? String(value) : '';
    const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
    const days = Math.round((startOf(new Date()) - startOf(date)) / 864e5);
    if (days < 0) return formatDateValue(date);
    if (days === 0) return t('time.today');
    if (days === 1) return t('time.yesterday');
    if (days < 7) return t('time.days-ago', {n: days});
    if (days < 30) return t('time.weeks-ago', {n: Math.floor(days / 7)});
    return t('time.months-ago', {n: Math.floor(days / 30)});
}

/** "только что", "12 сек назад", "3 мин назад", "2 ч назад" for an age in seconds. */
export function relativeSeconds(t: Translate, seconds: number) {
    if (!Number.isFinite(seconds) || seconds < 10) return t('time.just-now');
    if (seconds < 60) return t('time.sec-ago', {n: Math.floor(seconds)});
    if (seconds < 3600) return t('time.min-ago', {n: Math.floor(seconds / 60)});
    if (seconds < 86400) return t('time.hour-ago', {n: Math.floor(seconds / 3600)});
    return t('time.days-ago', {n: Math.floor(seconds / 86400)});
}

const flagEmojiRegex = /([\u{1F1E6}-\u{1F1FF}]{2}|\u{1F3F3}|\u{1F3F4}|\u{1F6A9})/u;

function containsFlagEmoji(value: any) {
    return typeof value === 'string' && flagEmojiRegex.test(value);
}

/** title vs. headerTitle, preferring the one that carries a flag emoji. */
export function profileDisplayTitle(profile: any) {
    const title = typeof profile?.title === 'string' ? profile.title.trim() : '';
    const headerTitle = typeof profile?.headerTitle === 'string' ? profile.headerTitle.trim() : '';

    if (title) {
        if (!headerTitle) {
            return title;
        }
        if (containsFlagEmoji(title) || !containsFlagEmoji(headerTitle)) {
            return title;
        }
    }

    return headerTitle || title || '';
}
