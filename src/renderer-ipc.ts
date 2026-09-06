// Renderer-side IPC helpers.
// Call initRendererIPC() once during app bootstrap (before mount).
// Exposes window.pxDrawEmoji so the Electron main process can request
// emoji rendering via webContents.executeJavaScript without touching the CDN.

// Vite bundles this asset into the app — no network required at runtime.
import twemojiFontUrl from './assets/fonts/TwemojiCountryFlags.woff2?url';

// TwemojiCountryFlags is loaded for HTML text rendering (CSS ligature substitution
// works correctly there). It is intentionally excluded from the canvas font stack
// because canvas does NOT support color emoji from custom-loaded FontFace fonts —
// only system-installed fonts get color rendering in Chromium canvas.
let flagFontPromise: Promise<FontFace> | null = null;

function loadFlagFont(): Promise<FontFace> {
    if (!flagFontPromise) {
        flagFontPromise = (async () => {
            const face = new FontFace('PxTwemojiFlags', `url(${twemojiFontUrl})`, {
                style: 'normal',
                weight: '400',
                unicodeRange: 'U+1F1E6-1F1FF, U+1F3F4, U+E0062-E0063, U+E0065, U+E0067, U+E006C, U+E006E, U+E0073-E0074, U+E0077, U+E007F',
            });
            await face.load();
            document.fonts.add(face);
            return face;
        })();
    }
    return flagFontPromise;
}

// Converts a single emoji to its Twemoji SVG codepoint string.
// e.g. 🇨🇭 → "1f1e8-1f1ed", 🔥 → "1f525"
// Strips variation selector U+FE0F which is not part of Twemoji filenames.
function emojiToTwemojiCodepoint(emoji: string): string {
    return [...emoji]
        .map(char => char.codePointAt(0)!.toString(16))
        .filter(hex => hex !== 'fe0f')
        .join('-');
}

// Splits a string of concatenated emoji into individual emoji.
// e.g. "🇨🇭🇩🇪" → ["🇨🇭", "🇩🇪"]
function splitEmoji(combined: string): string[] {
    const matches = combined.match(/[\u{1F1E6}-\u{1F1FF}]{2}|[\u{1F300}-\u{1FAFF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/gu);
    return matches ?? [];
}

// SVG cache: codepoint → single-emoji data URL
const svgCache = new Map<string, string>();

async function renderSingleEmojiViaSvg(emoji: string): Promise<string | null> {
    const codepoint = emojiToTwemojiCodepoint(emoji);
    if (svgCache.has(codepoint)) {
        return svgCache.get(codepoint)!;
    }

    // jdecked/twemoji is the actively maintained Twemoji fork (post-Twitter/X).
    const svgUrl = `https://cdn.jsdelivr.net/gh/jdecked/twemoji@15.1.0/assets/svg/${codepoint}.svg`;

    try {
        const img = new Image();
        img.crossOrigin = 'anonymous';
        await new Promise<void>((resolve, reject) => {
            img.onload = () => resolve();
            img.onerror = () => reject(new Error(`SVG not found: ${codepoint}`));
            img.src = svgUrl;
        });

        const canvas = document.createElement('canvas');
        canvas.width = 64;
        canvas.height = 64;
        const ctx = canvas.getContext('2d');
        if (!ctx) return null;
        ctx.drawImage(img, 0, 0, 64, 64);
        const dataUrl = canvas.toDataURL('image/png');
        svgCache.set(codepoint, dataUrl);
        return dataUrl;
    } catch {
        return null;
    }
}

// Renders one or more emoji into a single composite PNG data URL.
// Multiple emoji are drawn side by side: "🇨🇭🇩🇪" → 128×64 canvas.
async function renderEmojiComposite(combined: string): Promise<string | null> {
    const SIZE = 64;
    const emojis = splitEmoji(combined);
    if (emojis.length === 0) return null;

    const canvas = document.createElement('canvas');
    canvas.width = SIZE * emojis.length;
    canvas.height = SIZE;
    const ctx = canvas.getContext('2d');
    if (!ctx) return null;

    for (let i = 0; i < emojis.length; i++) {
        const dataUrl = await renderSingleEmojiViaSvg(emojis[i]);
        if (dataUrl) {
            const img = new Image();
            await new Promise<void>(res => { img.onload = () => res(); img.src = dataUrl; });
            ctx.drawImage(img, i * SIZE, 0, SIZE, SIZE);
        } else {
            // Fallback: draw via system emoji font for this slot
            ctx.font = `${SIZE * 0.75}px 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(emojis[i], i * SIZE + SIZE / 2, SIZE / 2 + SIZE * 0.06);
        }
    }

    return canvas.toDataURL('image/png');
}

export function initRendererIPC(): void {
    // Pre-load the flag font for HTML text rendering.
    void loadFlagFont();

    // Accepts a string of one or more concatenated emoji.
    // Returns a PNG data URL where all emoji are composited side by side.
    (window as any).pxDrawEmoji = async (emoji: string): Promise<string | null> => {
        try {
            return await renderEmojiComposite(emoji);
        } catch {
            return null;
        }
    };
}
