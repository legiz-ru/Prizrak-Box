// Background image analysis and accent colour.
//
// Accent (accepted product change vs. dev_21's ColorThief dominant colour): take
// the most frequent *saturated* colour of the image — 5-5-5 bins over a 64×64
// downscale, skipping pixels with max-min < 22 or max < 28, weighting each bin
// by count * (0.35 + (max-min)/255). If there is none, or it covers < 1 % of
// the pixels, fall back to the most frequent colour overall. The hue is then
// shifted and saturation/lightness tuned exactly like dev_21's
// adjustSelectedColor, and the lightness is walked until the contrast with the
// current text colour reaches 4.5 (see accentFor).
//
// Light/dark for the "Auto" mode keeps dev_21's rule (shouldUseWhiteText): the
// average perceived luminance of the image below 0.55 means white text.

export interface ImageTheme {
    /** Accent in HSL, before the per-mode contrast pass. */
    h: number;
    s: number;
    l: number;
    /** The image is dark: white text reads better on it. */
    white: boolean;
}

type RGB = [number, number, number];

const IMAGE_LOAD_TIMEOUT = 15000;
const DEFAULT_BACKGROUND_IMAGE = "url('/images/default.jpg')";
const WHITE_TEXT_LUMINANCE_THRESHOLD = 0.55;
const MIN_CONTRAST = 4.5;

export function hslToRgb(h: number, s: number, l: number): RGB {
    const k = (n: number) => (n + h / 30) % 12;
    const a = s * Math.min(l, 1 - l);
    const f = (n: number) => l - a * Math.max(-1, Math.min(k(n) - 3, 9 - k(n), 1));
    return [f(0) * 255, f(8) * 255, f(4) * 255];
}

export function rgbToHsl(r: number, g: number, b: number): RGB {
    r /= 255;
    g /= 255;
    b /= 255;
    const max = Math.max(r, g, b), min = Math.min(r, g, b);
    const l = (max + min) / 2, d = max - min;
    if (!d) return [0, 0, l];
    const s = d / (1 - Math.abs(2 * l - 1));
    const hh = max === r ? ((g - b) / d) % 6 : max === g ? (b - r) / d + 2 : (r - g) / d + 4;
    return [(hh * 60 + 360) % 360, s, l];
}

export function contrast(a: RGB | number[], b: RGB | number[]): number {
    const lum = (c: number[]) => {
        const v = c.map((x) => {
            x /= 255;
            return x <= 0.03928 ? x / 12.92 : Math.pow((x + 0.055) / 1.055, 2.4);
        });
        return 0.2126 * v[0] + 0.7152 * v[1] + 0.0722 * v[2];
    };
    const l1 = lum(a), l2 = lum(b);
    return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
}

export function parseColor(color: string): RGB | null {
    const hex = color.trim().match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i);
    if (hex) {
        const v = hex[1].length === 3 ? hex[1].split('').map(c => c + c).join('') : hex[1];
        return [0, 2, 4].map(i => parseInt(v.slice(i, i + 2), 16)) as RGB;
    }
    const rgb = color.match(/rgba?\(\s*([\d.]+)[\s,]+([\d.]+)[\s,]+([\d.]+)/i);
    return rgb ? [Number(rgb[1]), Number(rgb[2]), Number(rgb[3])] : null;
}

/**
 * Samples the image and returns the accent and light/dark verdict, or null when
 * the pixels cannot be read (cross-origin image without CORS: tainted canvas).
 */
export function analyzeImage(img: HTMLImageElement): ImageTheme | null {
    const N = 64;
    const canvas = document.createElement('canvas');
    canvas.width = canvas.height = N;
    const ctx = canvas.getContext('2d', {willReadFrequently: true});
    if (!ctx) return null;
    ctx.drawImage(img, 0, 0, N, N);
    let data: Uint8ClampedArray;
    try {
        data = ctx.getImageData(0, 0, N, N).data;
    } catch {
        return null;
    }

    const bins = new Map<number, number[]>();
    let lumSum = 0, n = 0;
    for (let i = 0; i < data.length; i += 4) {
        if (data[i + 3] < 125) continue;
        const r = data[i], g = data[i + 1], b = data[i + 2];
        lumSum += (0.299 * r + 0.587 * g + 0.114 * b) / 255;
        n++;
        if (r > 250 && g > 250 && b > 250) continue;
        const key = (r >> 3) << 10 | (g >> 3) << 5 | (b >> 3);
        const bin = bins.get(key) ?? [0, 0, 0, 0];
        bin[0] += r;
        bin[1] += g;
        bin[2] += b;
        bin[3]++;
        bins.set(key, bin);
    }
    if (!n) return null;

    let best: number[] | null = null, bestScore = 0;
    let dominant: number[] | null = null;
    for (const bin of bins.values()) {
        if (!dominant || bin[3] > dominant[3]) dominant = bin;
        const r = bin[0] / bin[3], g = bin[1] / bin[3], b = bin[2] / bin[3];
        const max = Math.max(r, g, b), min = Math.min(r, g, b);
        if (max - min < 22 || max < 28) continue;
        const score = bin[3] * (0.35 + (max - min) / 255);
        if (score > bestScore) {
            bestScore = score;
            best = bin;
        }
    }
    if (!best || best[3] < n * 0.01) best = dominant;
    if (!best) return null;

    const base: RGB = [best[0] / best[3], best[1] / best[3], best[2] / best[3]];
    const white = lumSum / n < WHITE_TEXT_LUMINANCE_THRESHOLD;
    const [h0, s0, l0] = rgbToHsl(...base);
    // dev_21 calculateHueShift / adjustSelectedColor
    let h = h0 >= 200 && h0 <= 250 ? h0 - 10 : h0 > 250 && h0 < 320 ? h0 - 20 : (h0 + 25) % 360;
    const s = Math.min(1, Math.min(s0 + 0.22 + (l0 < 0.5 ? 0.05 : -0.05), 0.88) + 0.28);
    let l = Math.min(0.9, Math.min(Math.max(l0, 0.48), 0.68) + 0.03);
    if (h0 > 40 && h0 < 65 && l0 > 0.7) {
        l -= 0.09;
        h = (h0 + 20) % 360;
    }
    const text: RGB = white ? [255, 255, 255] : [0, 0, 0];
    for (let i = 0; i < 5 && contrast(hslToRgb(h, s, l), text) < MIN_CONTRAST; i++) {
        l += white ? -0.035 : 0.035;
    }
    if (contrast(hslToRgb(h, s, l), base) < 2.5) l = Math.min(0.9, l + 0.08);
    return {h: Math.round(h), s, l, white};
}

/**
 * The accent for the given mode: lightness walked until the text drawn on the
 * accent (white in dark mode, black in light mode) has contrast >= 4.5.
 */
export function accentFor(theme: ImageTheme, dark: boolean): string {
    const text: RGB = dark ? [255, 255, 255] : [0, 0, 0];
    let l = theme.l;
    let rgb = hslToRgb(theme.h, theme.s, l);
    for (let i = 0; i < 40 && contrast(rgb, text) < MIN_CONTRAST; i++) {
        l = Math.max(0.05, Math.min(0.95, l + (dark ? -0.02 : 0.02)));
        rgb = hslToRgb(theme.h, theme.s, l);
    }
    return `rgb(${rgb.map(Math.round).join(',')})`;
}

/** Black or white, whichever reads better on the colour. */
export function onColor(color: string): string {
    const rgb = parseColor(color) ?? [91, 103, 232];
    return contrast(rgb, [255, 255, 255]) >= contrast(rgb, [0, 0, 0]) ? '#fff' : '#000';
}

const extractImageUrl = (style: string): string | null => {
    const match = style.match(/^url\(["']?(.*?)["']?\)$/);
    return match?.[1] || null;
};

let isBgLoading = false;

/**
 * Loads a background (CSS `url(...)` value) and analyses it. Falls back to the
 * default image on error/timeout. A cross-origin image without CORS headers is
 * still shown, just without analysis (theme = null).
 */
export function preloadBackgroundImage(
    bg: string,
    cb: (bg: string, theme: ImageTheme | null, img?: HTMLImageElement) => void
): void {
    if (isBgLoading) {
        console.warn("Background is loading, ignore new request:", bg);
        return;
    }

    if (!bg.startsWith("url(")) {
        cb(bg, null);
        return;
    }

    const imgUrl = extractImageUrl(bg);
    if (!imgUrl) {
        return preloadBackgroundImage(DEFAULT_BACKGROUND_IMAGE, cb);
    }

    isBgLoading = true;
    let isResolved = false;
    let triedWithoutCors = false;

    const finish = (img: HTMLImageElement) => {
        if (isResolved) return;
        isResolved = true;
        isBgLoading = false;
        let theme: ImageTheme | null = null;
        try {
            theme = analyzeImage(img);
        } catch (e) {
            console.warn("Theme color analysis failed (tainted canvas?); applying background without recolor:", e);
        }
        cb(bg, theme, img);
    };

    const load = (withCors: boolean) => {
        const img = new Image();
        if (withCors) img.crossOrigin = "anonymous";

        img.onload = () => {
            if (isResolved) return;
            clearTimeout(timeoutId);
            finish(img);
        };

        img.onerror = () => {
            if (isResolved) return;
            if (withCors && !triedWithoutCors) {
                triedWithoutCors = true;
                load(false);
                return;
            }
            clearTimeout(timeoutId);
            isResolved = true;
            console.error(`Failed to load background image: ${imgUrl}`);
            isBgLoading = false;
            preloadBackgroundImage(DEFAULT_BACKGROUND_IMAGE, cb);
        };

        img.src = imgUrl;
    };

    const timeoutId = setTimeout(() => {
        if (!isResolved) {
            console.error(`Background image load timed out: ${imgUrl}`);
            isResolved = true;
            isBgLoading = false;
            preloadBackgroundImage(DEFAULT_BACKGROUND_IMAGE, cb);
        }
    }, IMAGE_LOAD_TIMEOUT);

    load(true);
}
