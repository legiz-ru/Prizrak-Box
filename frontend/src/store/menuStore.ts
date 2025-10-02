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
        useWhite: true
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
        }
    },
    persist: defaultPersist,
});
