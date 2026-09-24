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

export function useAppTheme() {
    bindSystemMode();
    const menuStore = useMenuStore();

    const useImage = computed(() => menuStore.useBgImage);

    const mode = computed<'dark' | 'light'>(() => {
        if (menuStore.themePref === 'light' || menuStore.themePref === 'dark') return menuStore.themePref;
        if (useImage.value && imageTheme.value) return imageTheme.value.white ? 'dark' : 'light';
        return systemDark.value ? 'dark' : 'light';
    });

    const accentFromImage = computed(() => useImage.value && !!imageTheme.value);

    const vars = computed<Record<string, string | null>>(() => {
        const dark = mode.value === 'dark';
        const out: Record<string, string | null> = {};
        if (useImage.value) {
            const base = dark ? '20,21,27' : '255,255,255';
            const alpha = dark ? 1 - menuStore.uiTrans / 100 : Math.max(0.74, 1 - menuStore.uiTrans / 100);
            out['--panel-bg'] = `rgba(${base},${alpha.toFixed(2)})`;
            out['--overlay'] = dark
                ? `rgba(6,7,11,${(menuStore.bgDim / 100).toFixed(2)})`
                : `rgba(255,255,255,${(Math.max(30, menuStore.bgDim) / 100).toFixed(2)})`;
            out['--ui-blur'] = menuStore.uiBlur + 'px';
            out['--side-bg'] = out['--panel-bg'];
            out['--side-blur'] = `blur(${menuStore.uiBlur}px)`;
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
            out['--accent'] = menuStore.accent;
            out['--on-accent'] = onColor(menuStore.accent);
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

    return {mode, useImage, accentFromImage};
}
