import {defineStore} from 'pinia';
import {defaultPersist} from "@/types/persist";

export const useMenuStore = defineStore('menu', {
    state: () => ({
        menu: 'Home',
        path: '/Home',
        rule: 'rule',
        ruleNum: 0,
        proxy: false,
        tun: false,
        language: '',
        ruleMenu: 'Now',
        background: 'url("/images/default.jpg")',
        useWhite: true,
        settingTab: 'app',
        providersView: 'cards' as 'cards' | 'table',
        // Theme dialog (sidebar → palette button)
        themePref: 'auto' as 'auto' | 'light' | 'dark',
        useBgImage: true,
        // Percentages / px, defaults from the old styles/basic.css look
        uiTrans: 78,
        uiBlur: 1,
        bgDim: 18,
        // Accent used when no background image drives it
        accent: '#5b67e8',
        bgTheme: 'default',
    }),
    actions: {
        setMenu(menu: string) {
            this.menu = menu;
        },
        setPath(path: string) {
            this.path = path;
        },
        setRule(rule: string) {
            this.rule = rule;
        },
        setProxy(proxy: boolean) {
            this.proxy = proxy;
        },
        setTun(tun: boolean) {
            this.tun = tun;
        },
        setLanguage(language: string) {
            this.language = language;
        },
        setRuleMenu(ruleMenu: string) {
            this.ruleMenu = ruleMenu;
        },
        setRuleNum(ruleNum: number) {
            this.ruleNum = ruleNum;
        },
        setBackground(background: string) {
            this.background = background;
        },
        setUseWhite(useWhite: boolean) {
            this.useWhite = useWhite;
        },
        setSettingTab(settingTab: string) {
            this.settingTab = settingTab;
        },
        setProvidersView(view: 'cards' | 'table') {
            this.providersView = view;
        },
        resetThemeTweaks() {
            this.themePref = 'auto';
            this.useBgImage = true;
            this.uiTrans = 78;
            this.uiBlur = 1;
            this.bgDim = 18;
            this.bgTheme = 'default';
            this.background = 'url("/images/default.jpg")';
        },
    },
    persist: defaultPersist,
});
