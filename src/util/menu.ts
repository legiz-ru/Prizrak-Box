import {useMenuStore} from "@/store/menuStore";

export function changeMenu(value: string, router: any): void {
    let path = ''
    if (!value.startsWith("/")) {
        path = "/" + value
    }
    const menuStore = useMenuStore();
    menuStore.setPath(path);
    router.push(path);
}


export function detectLanguage() {
    const lang = navigator.language.toLowerCase();
    if (lang.startsWith('zh')) return 'zh';
    if (lang.startsWith('en')) return 'en';
    if (lang.startsWith('ru')) return 'ru';
    return 'en'; // 默认 fallback 英文
}