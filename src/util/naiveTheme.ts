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
        },
        InternalSelection: {
            borderRadius: "999px",
        },
        Dialog: {
            borderRadius: "20px",
        },
        Card: {
            borderRadius: "20px",
        },
        Popover: {
            borderRadius: "14px",
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
