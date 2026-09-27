// Resolves the theme (mode, accent, panel opacity/blur, background dimming)
// and writes it onto <html>: `data-theme` plus the dynamic custom properties
// that override src/styles/tokens.css.
//
// Mode "auto": with a background image, light/dark follows the image
// (util/theme.ts analyzeImage → white); without one, the OS preference.
// Light mode over an image keeps panels at least 74 % opaque and whitens the
// image by at least 30 % so text stays readable.
import {computed, ref, watchEffect} from "vue";
import {useMenuStore} from "@/store/menuStore";
import {accentFor, onColor, type ImageTheme} from "@/util/theme";
import {Events} from "@/runtime";

/** Result of analysing the current background image (null: none / unreadable). */
export const imageTheme = ref<ImageTheme | null>(null);

const systemDark = ref(true);
let mediaBound = false;

function bindSystemMode() {
    if (mediaBound) return;
    mediaBound = true;
    try {
        const mq = window.matchMedia('(prefers-color-scheme: light)');
        systemDark.value = !mq.matches;
        mq.addEventListener('change', (e) => {
            systemDark.value = !e.matches;
        });
    } catch {
        systemDark.value = true;
    }
}

/**
 * The look actually in effect: the active profile's pxd-theme (unless the user
 * switched it off) on top of the user's own settings. Fields the profile does
 * not set — or a profile without the header — fall back to the user's values,
 * which are never overwritten.
 */
export function useEffectiveTheme() {
    const menuStore = useMenuStore();
    const profile = computed(() => (menuStore.useProfileTheme ? menuStore.profileTheme : null) ?? null);
    return {
        /** The applied profile theme, or null. */
        profile,
        themePref: computed(() => profile.value?.mode || menuStore.themePref),
        useImage: computed(() => (profile.value?.image ? true : menuStore.useBgImage)),
        background: computed(() => {
            const image = profile.value?.image;
            return image ? `url('${image.replace(/'/g, '%27')}')` : menuStore.background;
        }),
        uiTrans: computed(() => profile.value?.transparency ?? menuStore.uiTrans),
        uiBlur: computed(() => profile.value?.blur ?? menuStore.uiBlur),
        bgDim: computed(() => profile.value?.dim ?? menuStore.bgDim),
        /** A fixed accent from the profile ('#rrggbb'); null = image or user accent. */
        fixedAccent: computed(() => {
            const value = profile.value?.accent;
            return value && value !== 'auto' ? value : null;
        }),
    };
}

export function useAppTheme() {
    bindSystemMode();
    const menuStore = useMenuStore();
    const effective = useEffectiveTheme();

    const useImage = effective.useImage;

    const mode = computed<'dark' | 'light'>(() => {
        const pref = effective.themePref.value;
        if (pref === 'light' || pref === 'dark') return pref;
        if (useImage.value && imageTheme.value) return imageTheme.value.white ? 'dark' : 'light';
        return systemDark.value ? 'dark' : 'light';
    });

    const accentFromImage = computed(() => !effective.fixedAccent.value && useImage.value && !!imageTheme.value);

    const vars = computed<Record<string, string | null>>(() => {
        const dark = mode.value === 'dark';
        const out: Record<string, string | null> = {};
        if (useImage.value) {
            const base = dark ? '20,21,27' : '255,255,255';
            const trans = effective.uiTrans.value, blur = effective.uiBlur.value, dim = effective.bgDim.value;
            const alpha = dark ? 1 - trans / 100 : Math.max(0.74, 1 - trans / 100);
            out['--panel-bg'] = `rgba(${base},${alpha.toFixed(2)})`;
            out['--overlay'] = dark
                ? `rgba(6,7,11,${(dim / 100).toFixed(2)})`
                : `rgba(255,255,255,${(Math.max(30, dim) / 100).toFixed(2)})`;
            out['--ui-blur'] = blur + 'px';
            out['--side-bg'] = out['--panel-bg'];
            out['--side-blur'] = `blur(${blur}px)`;
            out['--title-bg'] = out['--panel-bg'];
            out['--title-shadow'] = dark ? '0 1px 3px rgba(0,0,0,.55)' : '0 1px 3px rgba(255,255,255,.7)';
            out['--logo-shadow'] = 'drop-shadow(0 2px 6px rgba(0,0,0,.35))';
        } else {
            for (const key of ['--panel-bg', '--overlay', '--ui-blur', '--side-bg', '--side-blur', '--title-bg', '--title-shadow', '--logo-shadow']) {
                out[key] = null;
            }
        }
        if (accentFromImage.value && imageTheme.value) {
            out['--accent'] = accentFor(imageTheme.value, dark);
            out['--on-accent'] = dark ? '#fff' : '#000';
        } else {
            const accent = effective.fixedAccent.value ?? menuStore.accent;
            out['--accent'] = accent;
            out['--on-accent'] = onColor(accent);
        }
        return out;
    });

    watchEffect(() => {
        const root = document.documentElement;
        root.setAttribute('data-theme', mode.value);
        root.classList.toggle('dark', mode.value === 'dark');
        for (const [key, value] of Object.entries(vars.value)) {
            if (value === null) root.style.removeProperty(key);
            else root.style.setProperty(key, value);
        }
    });

    // Keep the shells' first-frame colour in sync (Wails paints the native
    // window background; index.html's boot placeholder reads px:darkBg).
    watchEffect(() => {
        const dark = mode.value === 'dark';
        if (menuStore.useWhite !== dark) menuStore.setUseWhite(dark);
        Events.Emit({name: 'darkBg', data: dark});
        try {
            localStorage.setItem('px:darkBg', dark ? '1' : '0');
        } catch {
            /* ignore */
        }
    });

    return {mode, useImage, accentFromImage, effective};
}
