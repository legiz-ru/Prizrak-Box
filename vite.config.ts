import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import Icons from 'unplugin-icons/vite';
import IconsResolver from "unplugin-icons/resolver";
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import {ElementPlusResolver} from 'unplugin-vue-components/resolvers'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'
import path from 'path'

const pathSrc = path.resolve(__dirname, 'src')

// https://vitejs.dev/config/
export default defineConfig({
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
                ElementPlusResolver(),
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
                    enabledCollections: ["ep", "mdi"],
                }),
                ElementPlusResolver()
            ],
            dts: path.resolve(pathSrc, 'components.d.ts'),
        }),
        Icons({
            autoInstall: true,
            compiler: "vue3",
        }),
        VueI18nPlugin({
            include: [path.resolve(pathSrc, './locales/**')],
        }),
    ],
    clearScreen: false
})
