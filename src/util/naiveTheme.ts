// Naive UI theme bridge.
//
// Prizrak-Box derives its accent colour from the user's background image
// (see theme.ts's changeTheme()) and writes it as plain CSS variables
// consumed by hand-authored component CSS. Naive UI components don't read
// CSS variables for their palette — they take a reactive JS object
// (GlobalThemeOverrides) via <n-config-provider>. This module is the single
// place that turns the same computed accent into that object, so migrated
// n-* components repaint with the wallpaper exactly like the legacy ones do.
import {computed, ref} from "vue";
import {createDiscreteApi, darkTheme, type DiscreteApi, type GlobalTheme, type GlobalThemeOverrides} from "naive-ui";

interface NaiveAccentColors {
    primary: string;
    text: string;
}

const DEFAULT_ACCENT: NaiveAccentColors = {
    primary: "#3FD3E0",
    text: "#ffffff",
};

// Mirrors menuStore.useWhite: true when the current background is dark
// enough to need white text, which is also when Naive UI's own dark
// component theme (dark surfaces, light text) is the right base to layer
// the accent on top of.
const isDarkBg = ref(true);
const accent = ref<NaiveAccentColors>(DEFAULT_ACCENT);

export function setNaiveDarkBg(useWhite: boolean): void {
    isDarkBg.value = useWhite;
}

export function setNaiveAccentColors(colors: NaiveAccentColors): void {
    accent.value = colors;
}

export const naiveTheme = computed<GlobalTheme | null>(() => (isDarkBg.value ? darkTheme : null));

// Pill-shaped inputs/buttons and rounded 20px surfaces match the
// hand-rolled overrides this app already applies to Element Plus
// (see styles/global.css's .el-overlay-dialog rules) — kept here so the
// same shape language carries over as screens move to Naive UI.
//
// Surface/text/border colours below are wired to the app's own
// wallpaper-adaptive CSS variables (theme.ts's changeTheme(), applied via
// document.documentElement.style.setProperty) rather than left on Naive's
// static dark/light preset values. Passing a `var(...)` string here is fine:
// Naive bakes it into the CSS it injects, so the browser re-resolves it live
// whenever the underlying custom property changes — no re-render needed.
// Without this, dialogs/dropdowns/inputs only had two fixed looks (Naive's
// built-in dark or light palette) that didn't track the wallpaper the way
// every hand-styled surface in the app already does, so a card/modal could
// end up visibly mismatched against its background after a wallpaper change.
export const naiveThemeOverrides = computed<GlobalThemeOverrides>(() => {
    const {primary, text} = accent.value;
    return {
        common: {
            primaryColor: primary,
            primaryColorHover: primary,
            primaryColorPressed: primary,
            primaryColorSuppl: primary,
            textColorBase: text,
        },
        Button: {
            borderRadiusTiny: "999px",
            borderRadiusSmall: "999px",
            borderRadiusMedium: "999px",
            borderRadiusLarge: "999px",
        },
        Input: {
            borderRadius: "999px",
            color: "var(--search-input-bg)",
            colorHover: "var(--search-input-bg)",
            colorFocus: "var(--search-input-bg)",
            textColor: "var(--text-color)",
            placeholderColor: "var(--placeholder-color)",
            border: "1px solid var(--sub-card-border)",
            borderHover: "1px solid var(--text-color)",
            borderFocus: "1px solid var(--text-color)",
            caretColor: "var(--left-item-selected-bg)",
        },
        InternalSelection: {
            borderRadius: "999px",
            color: "var(--search-input-bg)",
            colorActive: "var(--search-input-bg)",
            textColor: "var(--text-color)",
            placeholderColor: "var(--placeholder-color)",
            border: "1px solid var(--sub-card-border)",
            borderHover: "1px solid var(--text-color)",
            borderActive: "1px solid var(--text-color)",
            borderFocus: "1px solid var(--text-color)",
        },
        Dialog: {
            borderRadius: "20px",
        },
        Card: {
            borderRadius: "20px",
            // Plain in-page cards keep the translucent glass look: they sit on
            // top of the app's own wallpaper-darkening backdrop, which is what
            // makes a 10%-alpha fill readable in the first place.
            color: "var(--sub-card-bg)",
            // colorModal/colorPopover back floating chrome (dialogs, popovers)
            // that renders with no such backdrop behind it — Naive's own modal
            // mask is a fixed, non-adaptive rgba(0,0,0,.4), nowhere near enough
            // to compensate on a bright photo wallpaper. --dropdown-list-bg is
            // solid (no alpha) and already proven readable for exactly this
            // kind of floating surface.
            colorModal: "var(--dropdown-list-bg)",
            colorPopover: "var(--dropdown-list-bg)",
            colorEmbedded: "var(--sub-card-bg)",
            textColor: "var(--text-color)",
            titleTextColor: "var(--text-color)",
            borderColor: "var(--sub-card-border)",
            actionColor: "var(--sub-card-bg)",
        },
        Modal: {
            color: "var(--dropdown-list-bg)",
            textColor: "var(--text-color)",
        },
        Popover: {
            borderRadius: "14px",
            color: "var(--dropdown-list-bg)",
            textColor: "var(--text-color)",
            dividerColor: "var(--sub-card-border)",
        },
        Dropdown: {
            color: "var(--dropdown-list-bg)",
            dividerColor: "var(--sub-card-border)",
            optionTextColor: "var(--text-color)",
            optionTextColorHover: "var(--text-color)",
            optionTextColorActive: "var(--text-color)",
            optionColorHover: "var(--left-nav-btn-hover-bg)",
            optionColorActive: "var(--left-item-selected-bg)",
        },
        Tooltip: {
            color: "var(--skin-bg-color)",
            textColor: "var(--text-color)",
        },
        Form: {
            labelTextColor: "var(--text-color)",
        },
    };
});

// Lazily created: createDiscreteApi() mounts its own app instance, and
// nothing here should run before naive-ui is actually needed (e.g. before
// the first toast). Reused afterwards as a singleton.
let discreteApi: DiscreteApi<"message"> | null = null;

function getDiscreteApi() {
    if (!discreteApi) {
        discreteApi = createDiscreteApi(["message"], {
            configProviderProps: computed(() => ({
                theme: naiveTheme.value,
                themeOverrides: naiveThemeOverrides.value,
            })),
        });
    }
    return discreteApi;
}

export function naiveMessage() {
    return getDiscreteApi().message;
}
