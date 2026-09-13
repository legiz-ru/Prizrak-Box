import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import { useMenuStore } from '@/store/menuStore';

import Home from '@/views/Home.vue';
import Setting from '@/views/Setting.vue';
import Proxies from '@/views/Proxies.vue';
import Profiles from '@/views/Profiles.vue';
import Dns from '@/views/setting/Dns.vue';
import Shortcut from '@/views/setting/Shortcut.vue';

const RULE_TABS = ['Now', 'Group', 'Providers', 'Ignore'];

/**
 * Переводит старый отдельный экран на соответствующую вкладку настроек и
 * чинит сохранённый путь, чтобы перенаправление сработало один раз, а не при
 * каждом запуске.
 */
function legacyTabRedirect(tab: 'connection' | 'log' | 'rule'): string {
    const menuStore = useMenuStore();
    menuStore.setSettingTab(tab);
    menuStore.setPath('/Setting');
    return '/Setting';
}

const routes: Array<RouteRecordRaw> = [
    {
        path: '/',
        name: 'Start',
        component: Home,
    },
    {
        path: '/Home',
        name: 'Home',
        component: Home,
    },
    {
        path: '/Setting',
        name: 'Setting',
        component: Setting,
    },
    {
        path: '/Setting/Dns',
        name: 'Dns',
        component: Dns,
    },
    {
        path: '/Setting/Shortcut',
        name: 'Shortcut',
        component: Shortcut,
    },
    {
        path: '/Proxies',
        name: 'Proxies',
        component: Proxies,
    },
    {
        path: '/Profiles',
        name: 'Profiles',
        component: Profiles,
    },
    // Правила тоже были отдельным экраном со своей навигацией по подразделам;
    // сам подраздел сохраняется в сторе, поэтому переносим и его.
    {
        path: '/Rule/:sub?',
        redirect: (to) => {
            const sub = String(to.params.sub ?? '');
            if (RULE_TABS.includes(sub)) {
                useMenuStore().setRuleMenu(sub);
            }
            return legacyTabRedirect('rule');
        },
    },
    // Наследие: до перехода на вкладки настроек соединения и журнал были
    // отдельными экранами, а путь последнего открытого раздела сохраняется в
    // сторе. У пользователя, закрывшего приложение на одном из них, там до сих
    // пор лежит /Connection или /Log — без этих перенаправлений он получил бы
    // при запуске пустую правую панель.
    {
        path: '/Connection',
        redirect: () => legacyTabRedirect('connection'),
    },
    {
        path: '/Log',
        redirect: () => legacyTabRedirect('log'),
    },
    // Любой другой сохранённый путь из прошлых версий ведёт на главную, а не в
    // пустоту.
    {
        path: '/:pathMatch(.*)*',
        redirect: '/Home',
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

export default router;