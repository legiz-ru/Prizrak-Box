import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import Icons from 'unplugin-icons/vite';
import IconsResolver from "unplugin-icons/resolver";
import {FileSystemIconLoader} from 'unplugin-icons/loaders';
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'
import path from 'path'
import fs from 'fs'

const pathSrc = path.resolve(__dirname, 'src')

// Prizrak-Core version, read from src-go/go.mod at build time (the
// `replace ... => github.com/legiz-ru/Prizrak-Core vX.Y.Z` line), the same way
// the Android app does. Empty when the line is missing — the UI then hides it.
function readCoreVersion(): string {
    try {
        const goMod = fs.readFileSync(path.resolve(__dirname, 'src-go/go.mod'), 'utf8')
        return /legiz-ru\/Prizrak-Core\s+(v[\w.\-]+)/.exec(goMod)?.[1] ?? ''
    } catch {
        return ''
    }
}

// https://vitejs.dev/config/
export default defineConfig({
    define: {
        __CORE_VERSION__: JSON.stringify(readCoreVersion()),
    },
    resolve: {
        alias: {
            '@': pathSrc,
            // Wails v3 generated Go bindings (used only by the Wails shell via
            // src/wails-shim.ts; dynamically imported, so the Electron build
            // never executes them).
            '@wbind': path.resolve(__dirname, 'src-wails/frontend/bindings'),
        },
    },
    build: {
        rollupOptions: {
            // Resolve the Wails JS runtime at load time from the Go binary's
            // asset server (/wails/runtime.js) instead of bundling the
            // @wailsio/runtime npm package. The published npm package lags the
            // Go module (e.g. 3.0.0-alpha.97 lacks the appregion module that
            // native non-client regions need), while the served runtime is
            // built from the exact source of the pinned wails Go module — the
            // two sides can never drift apart. The npm dependency remains for
            // types/dev. All imports of '@wailsio/runtime' are dynamic (see
            // wails-shim.ts / generated bindings), so the Electron build never
            // requests this URL.
            external: ['@wailsio/runtime'],
            output: {
                paths: {'@wailsio/runtime': '/wails/runtime.js'},
            },
        },
    },
    plugins: [vue(),
        AutoImport({
            imports: ["vue"],
            resolvers: [
                IconsResolver({
                    prefix: "Icon",
                }),
            ],
            dts: path.resolve(pathSrc, "auto-imports.d.ts"),
        }),
        Components({
            resolvers: [
                IconsResolver({
                    prefix: 'icon',
                    // Tabler for UI icons; "proto" is the local collection of
                    // protocol brand marks below. Its files are not in an
                    // Iconify package, so the resolver needs to be told which
                    // names belong to it.
                    enabledCollections: ["tabler", "proto"],
                    customCollections: ["proto"],
                }),
            ],
            dts: path.resolve(pathSrc, 'components.d.ts'),
        }),
        Icons({
            autoInstall: false,
            compiler: "vue3",
            customCollections: {
                // Protocol brand marks, normalised to a 24x24 box and to
                // currentColor so one asset serves both themes. See
                // src/assets/icons/proto/ATTRIBUTION.md for sources, licences
                // and the normalisation pipeline. Used as <icon-proto-xray/>.
                proto: FileSystemIconLoader(
                    path.resolve(pathSrc, 'assets/icons/proto'),
                ),
            },
        }),
        VueI18nPlugin({
            include: [path.resolve(pathSrc, './locales/**')],
        }),
    ],
    clearScreen: false
})
