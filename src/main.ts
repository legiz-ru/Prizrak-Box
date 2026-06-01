import {createApp, watch, toRaw} from "vue";
import App from "./App.vue";
import router from "@/router";
import {createPinia} from "pinia";
import piniaPluginPersistence from "pinia-plugin-persistedstate";
import {createI18n} from "vue-i18n";
import messages from "@intlify/unplugin-vue-i18n/messages";
import ElementPlus from "element-plus";
import VueApexCharts from "vue3-apexcharts";
import "element-plus/dist/index.css";
import 'element-plus/theme-chalk/dark/css-vars.css'
import "./styles/global.css";
import "./styles/basic.css";
import {useMenuStore} from "@/store/menuStore";
import {useWebStore} from "@/store/webStore";
import {AxiosRequest} from "@/util/axiosRequest";
import {useHomeStore} from "@/store/homeStore";
import {useSettingStore} from "@/store/settingStore";
import {memoryCache} from "@/types/persist"
import {initBgCache} from "@/util/bgCache"
import {detectLanguage} from "@/util/menu";
import createApi from "@/api";
import {Profile} from "@/types/profile";
import {pError, pSuccess, pWarning} from "@/util/pLoad";
import {isHttpOrHttps} from "@/util/format";
import {useDeepLinkImportStore} from "@/store/deepLinkStore";
import {useUpdateStore} from "@/store/updateStore";
import {Browser, Events} from "@/runtime";
import {createDashboardLinks} from "@/util/dashboard";
import {initRendererIPC} from "./renderer-ipc";
import {installWailsShim} from "./wails-shim";

const app = createApp(App);
const lang = detectLanguage();
const DEEP_LINK_IMPORTED_EVENT = 'deeplink-profile-imported';
const DEEP_LINK_HOST = 'install-config';
const KNOWN_DEEP_LINK_EXTRA_KEYS = new Set(['name']);
let deepLinkHandlerRegistered = false;
let updateCheckerRegistered = false;
const UPDATE_CHECK_INTERVAL = 6 * 60 * 60 * 1000;

function isCanceledError(error: any) {
    if (!error || typeof error !== 'object') {
        return false;
    }

    const code = (error as any).code;
    const name = (error as any).name;
    const message = typeof (error as any).message === 'string' ? (error as any).message.toLowerCase() : '';

    return code === 'ERR_CANCELED'
        || name === 'CanceledError'
        || (error as any).__CANCEL__ === true
        || message === 'canceled'
        || message === 'cancelled'
        || message === 'aborted';
}

async function bootstrap() {
    // Install the Wails compatibility shim before anything reads window.px*.
    // No-op under Electron (preload already provides the bridge).
    installWailsShim();

    initRendererIPC();

    // 加载缓存数据
    // @ts-ignore
    if (window["pxStore"]) {
        const keys = ['menu', 'home', 'proxies', 'setting', 'web', 'onboarding'];
        for (const key of keys) {
            // @ts-ignore
            const val = await window["pxStore"].get(key);
            if (val) {
                memoryCache[key] = val;
            }
        }
    }

    // Загрузить кэш фона до монтирования Vue, чтобы applyBackground мог использовать его синхронно
    // @ts-ignore
    if (window["pxBgCache"]) {
        try {
            // @ts-ignore
            const cached = await window["pxBgCache"].read();
            if (cached?.forBg && cached?.dataUrl) {
                initBgCache(cached.forBg, cached.dataUrl);
            }
        } catch {}
    }

    // 国际化设置
    const i18n = createI18n({
        locale: lang,
        messages,
        globalInjection: true,
    });

    // 全局状态管理
    const pinia = createPinia();
    pinia.use(piniaPluginPersistence);


    // 加载所需组件
    app.use(pinia);
    app.use(ElementPlus);
    app.use(VueApexCharts);
    app.use(i18n);
    app.use(router);

    const translate = (key: string, values?: Record<string, unknown>) => {
        try {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-return
            return i18n.global.t(key, values);
        } catch {
            return key;
        }
    };

    (app.config.globalProperties as any).$translate = translate;
    (window as any).pxTranslate = translate;

    // 获取api地址、端口、密钥
    const url = window.location.search;
    const params = new URLSearchParams(url);
    const webStore = useWebStore();
    const host = params.get("host");
    const port = params.get("port");
    const secret = params.get("secret");
    if (host) {
        webStore.setHost(host);
    }
    if (port) {
        webStore.setPort(port);
    }
    if (secret) {
        webStore.setSecret(secret);
    }

    const emitDashboardLinks = () => {
        const dashboards = createDashboardLinks(webStore.customDashboards, {
            host: webStore.host,
            port: webStore.port,
            secret: webStore.secret,
        });

        if ((window as any)?.pxTray?.emit) {
            Events.Emit({name: "dashboards", data: dashboards});
        }
    };

    watch(() => [webStore.host, webStore.port, webStore.secret], () => {
        emitDashboardLinks();
    }, {immediate: true});

    watch(() => webStore.customDashboards, () => {
        emitDashboardLinks();
    }, {deep: true});

    // 注册 Axios 实例到全局
    app.config.globalProperties.$http = new AxiosRequest(
        webStore.baseUrl,
        webStore.secret
    );

    const api = createApi(app.config.globalProperties);

    const homeStore = useHomeStore();
    const settingStore = useSettingStore();

    const updateHttpClientConfig = async () => {
        try {
            const version = await api.getVersion();
            const details = await api.updateHTTPClientConfig({
                enableHWID: settingStore.hwid,
                version,
                deviceOS: "",
                deviceOSVer: "",
                deviceModel: "",
            });

            settingStore.setHwidHeaders({
                hwid: details?.hwid ?? '',
                os: details?.os ?? '',
                osVersion: details?.osVersion ?? '',
                model: details?.model ?? '',
            });
        } catch (error) {
            console.error("Failed to update HTTP client config", error);
        }
    };

    // Start the backend HWID config request immediately so the backend receives
    // enableHWID=true before it can fire a startup auto-refresh of subscriptions.
    // We still await the promise at the end of bootstrap before mounting the app.
    const httpConfigPromise = updateHttpClientConfig();

    watch(() => settingStore.hwid, () => {
        void updateHttpClientConfig();
    });

    setupDeepLinkHandler();
    setupUpdateChecker();

    // 激活menu
    const menuStore = useMenuStore();
    router.afterEach((to) => {
        const split = to.path.split("/");
        menuStore.setMenu(split[1]);
        if (split.length > 2 && split[1] === "Rule") {
            menuStore.setRuleMenu(split[2]);
        }
    });
    if (!menuStore.language) {
        menuStore.setLanguage(lang);
    }

    // Sync i18n locale to stored preference immediately — before app.mount()
    // so that any tray interaction before Language.vue mounts shows the correct language
    i18n.global.locale.value = menuStore.language;

    // Pre-send tray translations with the correct language before the slow HTTP call
    // (updateHttpClientConfig below). Without this the tray shows Chinese default labels
    // until Language.vue mounts, which can take a few seconds on slow Mihomo startup.
    const _trayMenuId = ['tray.show','tray.rule','tray.global','tray.direct',
        'tray.profiles','tray.proxyGroups','tray.dashboard','tray.proxy','tray.tun','tray.quit'];
    if ((window as any)?.pxTray?.emit) {
        const _translate: Record<string, string> = {};
        _trayMenuId.forEach(k => { _translate[k] = i18n.global.t(k); });
        (window as any).pxTray.emit('translate', _translate);
        (window as any).pxTray.emit('tunAuthTip', i18n.global.t('tun-auth-tip'));
    }

    // 设置起始时间 和 操作系统类型
    // 获取系统类型
    homeStore.setOS(window.pxOs());

    // 设置软件开始时间
    homeStore.setStartTime(Date.now());

    await httpConfigPromise;

}

type DeepLinkPayload = string | { rawUrl?: string; url?: string; name?: string };

function setupDeepLinkHandler() {
    if (deepLinkHandlerRegistered) {
        return;
    }

    if (!window.pxDeepLink || typeof window.pxDeepLink.onImportProfile !== 'function') {
        return;
    }

    const globalProperties: any = app.config.globalProperties;
    const api = createApi(globalProperties);
    const translate = (key: string) => {
        try {
            return typeof globalProperties.$t === 'function' ? globalProperties.$t(key) : key;
        } catch {
            return key;
        }
    };

    const ensureDeepLinkReady = () => {
        if (typeof window.pxDeepLink?.notifyReady !== 'function') {
            return;
        }

        try {
            window.pxDeepLink.notifyReady();
        } catch (error) {
            console.error('Failed to notify deeplink readiness', error);
        }
    };

    const deepLinkImportStore = useDeepLinkImportStore();

    const importProfileFromDeepLink = async (payload: DeepLinkPayload) => {
        const normalized = normalizeDeepLinkPayload(payload);
        const parsed = normalized.rawUrl ? parseDeepLinkUrl(normalized.rawUrl) : null;
        const subscriptionUrl = parsed?.url ?? normalized.directUrl;
        const profileName = normalized.name ?? parsed?.name;

        if (!subscriptionUrl) {
            pError(translate('profiles.deeplink.invalid-url'));
            return;
        }

        if (!isHttpOrHttps(subscriptionUrl)) {
            pError(translate('profiles.deeplink.invalid-url-format'));
            return;
        }

        const profile = new Profile();
        profile.content = subscriptionUrl;
        if (profileName) {
            profile.title = profileName;
        }

        const controller = new AbortController();
        let cancelledByUser = false;
        let overlayActive = false;

        try {
            deepLinkImportStore.startImport({
                message: translate('profiles.deeplink.importing'),
                cancelLabel: translate('profiles.deeplink.cancel-import'),
                onCancel: () => {
                    if (!controller.signal.aborted) {
                        cancelledByUser = true;
                        controller.abort();
                    }
                },
            });
            overlayActive = true;

            const result = await api.addProfileFromInput(profile, {signal: controller.signal});

            if (controller.signal.aborted || cancelledByUser) {
                pWarning(translate('profiles.deeplink.import-cancelled'));
                return;
            }

            if (Array.isArray(result) && result.length > 0) {
                const firstProfile = result[0];

                try {
                    await api.switchProfile({
                        id: firstProfile.id,
                        selected: true,
                        exclusive: true,
                    });

                    await api.waitRunning();

                    const fullList = await api.getProfileList();
                    const activeProfile = fullList?.find((item: any) => item?.id === firstProfile.id) ?? firstProfile;

                    Events.Emit({
                        name: "profileChanged",
                        data: {
                            profile: activeProfile,
                            exclusive: true,
                        }
                    });
                    window.dispatchEvent(new CustomEvent('profile-changed'));

                    Events.Emit({
                        name: "profiles",
                        data: toRaw(fullList)
                    });

                    window.dispatchEvent(new CustomEvent('vue-profiles-updated', {
                        detail: { profiles: toRaw(fullList) }
                    }));
                } catch (error) {
                    console.error('Failed to activate deeplink profile', error);
                }

                window.dispatchEvent(new CustomEvent(DEEP_LINK_IMPORTED_EVENT, {
                    detail: {profiles: result}
                }));
            }

            pSuccess(translate('profiles.deeplink.import-success'));
        } catch (error: any) {
            if (cancelledByUser || isCanceledError(error)) {
                pWarning(translate('profiles.deeplink.import-cancelled'));
            } else if (error && typeof error === 'object' && 'message' in error && error.message) {
                pError(error.message);
            } else {
                pError(translate('profiles.deeplink.import-failed'));
            }
        } finally {
            if (overlayActive) {
                deepLinkImportStore.finishImport();
            }
        }
    };

    window.pxDeepLink.onImportProfile(importProfileFromDeepLink);

    const handleWindowFocus = () => ensureDeepLinkReady();
    const handleVisibilityChange = () => {
        if (!document.hidden) {
            ensureDeepLinkReady();
        }
    };

    window.addEventListener('focus', handleWindowFocus);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    ensureDeepLinkReady();

    deepLinkHandlerRegistered = true;
}

function setupUpdateChecker() {
    if (updateCheckerRegistered) {
        return;
    }

    updateCheckerRegistered = true;

    const updateStore = useUpdateStore();
    const globalProperties: any = app.config.globalProperties;
    const translate = (key: string, values?: Record<string, any>) => {
        try {
            return typeof globalProperties.$t === 'function'
                ? globalProperties.$t(key, values)
                : key;
        } catch {
            return key;
        }
    };

    const openExternalLink = (url: string) => {
        if (!url) {
            return;
        }

        try {
            Browser.OpenURL(url);
        } catch (error) {
            window.open(url, '_blank');
        }
    };

    watch(() => updateStore.shouldNotify, (shouldNotify) => {
        if (!shouldNotify) {
            return;
        }

        const label = updateStore.latestDisplayName || translate('updates.banner.version-unknown');
        const title = translate('updates.notification.title');
        const message = translate('updates.notification.message', {version: label});

        const notify = async () => {
            const NotificationCtor = window.Notification;

            if (typeof NotificationCtor !== 'function') {
                updateStore.markNotified();
                return;
            }

            const showNotification = () => {
                try {
                    const notification = new NotificationCtor(title, {body: message});
                    notification.onclick = () => {
                        if (typeof window.focus === 'function') {
                            window.focus();
                        }

                        if (updateStore.latestUrl) {
                            openExternalLink(updateStore.latestUrl);
                        }
                    };
                } catch (error) {
                    console.error('Failed to display update notification', error);
                }
            };

            try {
                let permission = NotificationCtor.permission;

                if (permission === 'default' && typeof NotificationCtor.requestPermission === 'function') {
                    try {
                        permission = await NotificationCtor.requestPermission();
                    } catch (error) {
                        console.error('Failed to request notification permission', error);
                        permission = 'denied';
                    }
                }

                if (permission === 'granted') {
                    showNotification();
                }
            } finally {
                updateStore.markNotified();
            }
        };

        void notify();
    }, {immediate: false});

    const performCheck = async () => {
        await updateStore.checkForUpdates();
    };

    void performCheck();

    window.setInterval(() => {
        void performCheck();
    }, UPDATE_CHECK_INTERVAL);
}

function normalizeDeepLinkPayload(payload: DeepLinkPayload): { rawUrl?: string; directUrl?: string; name?: string } {
    if (typeof payload === 'string') {
        return {rawUrl: payload};
    }

    if (payload && typeof payload === 'object') {
        return {
            rawUrl: payload.rawUrl,
            directUrl: payload.url,
            name: payload.name,
        };
    }

    return {};
}

function parseDeepLinkUrl(link: string): { url: string; name?: string } | null {
    try {
        const parsed = new URL(link);
        if (parsed.protocol !== 'prizrak-box:') {
            return null;
        }

        const host = parsed.hostname || parsed.host;
        if (host && host.toLowerCase() !== DEEP_LINK_HOST) {
            return null;
        }

        const query = parsed.search.startsWith('?') ? parsed.search.slice(1) : '';
        if (!query) {
            return null;
        }

        const segments = query.split('&');
        let urlValue: string | null = null;
        const extras: Record<string, string> = {};

        for (const segment of segments) {
            if (!segment) {
                continue;
            }

            const [rawKey, ...rawRest] = segment.split('=');
            const key = rawKey;
            const value = rawRest.join('=');

            if (key === 'url' && urlValue === null) {
                urlValue = value;
                continue;
            }

            if (urlValue !== null && KNOWN_DEEP_LINK_EXTRA_KEYS.has(key)) {
                extras[key] = safeDecode(value);
                continue;
            }

            if (urlValue !== null) {
                urlValue += `&${segment}`;
            }
        }

        if (!urlValue) {
            return null;
        }

        const decodedUrl = safeDecode(urlValue);
        if (!decodedUrl) {
            return null;
        }

        return {
            url: decodedUrl,
            name: extras['name'],
        };
    } catch {
        return null;
    }
}

function safeDecode(value?: string) {
    if (value === undefined) {
        return undefined;
    }

    try {
        return decodeURIComponent(value);
    } catch {
        return value;
    }
}

// 🚀 启动应用
bootstrap().then(() => app.mount("#app"));
