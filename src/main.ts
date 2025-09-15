import {createApp} from "vue";
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
import {memoryCache} from "@/types/persist"
import {detectLanguage} from "@/util/menu";

const app = createApp(App);
const lang = detectLanguage();

async function bootstrap() {
    // 加载缓存数据
    // @ts-ignore
    if (window["pxStore"]) {
        const keys = ['menu', 'home', 'proxies', 'setting', 'web'];
        for (const key of keys) {
            // @ts-ignore
            const val = await window["pxStore"].get(key);
            if (val) {
                memoryCache[key] = val;
            }
        }
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

    // 注册 Axios 实例到全局
    app.config.globalProperties.$http = new AxiosRequest(
        webStore.baseUrl,
        webStore.secret
    );

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

    // 设置起始时间 和 操作系统类型
    const homeStore = useHomeStore();

    // 获取系统类型
    homeStore.setOS(window.pxOs());

    // 设置软件开始时间
    homeStore.setStartTime(Date.now());
    
    // 设置深度链接处理器  
    setupDeepLinkHandler();
}

// 处理深度链接导入配置
function setupDeepLinkHandler() {
    // @ts-ignore
    if (window.pxDeepLink) {
        // @ts-ignore
        window.pxDeepLink.onImportProfile((data: { url: string; name?: string }) => {
            // 发送全局事件，让 Profiles 页面处理
            window.dispatchEvent(new CustomEvent('deeplink-import-profile', {
                detail: data
            }));
        });
    }
}

// 🚀 启动应用
bootstrap().then(() => app.mount("#app"));



