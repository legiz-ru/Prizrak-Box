import RuleProviders from '@/views/rule/Providers.vue';
import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import { useMenuStore } from '@/store/menuStore';

import Home from '@/views/Home.vue';
import Setting from '@/views/Setting.vue';
import Proxies from '@/views/Proxies.vue';
import Profiles from '@/views/Profiles.vue';
import Rule from '@/views/Rule.vue';
import Now from '@/views/rule/Now.vue';
import Group from '@/views/rule/Group.vue';
import Ignore from '@/views/rule/Ignore.vue';
import Crawl from '@/views/Crawl.vue';
import Dns from '@/views/setting/Dns.vue';
import Shortcut from '@/views/setting/Shortcut.vue';

/**
 * Переводит старый отдельный экран на соответствующую вкладку настроек и
 * чинит сохранённый путь, чтобы перенаправление сработало один раз, а не при
 * каждом запуске.
 */
function legacyTabRedirect(tab: 'connection' | 'log'): string {
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
    {
        path: '/Rule',
        name: 'Rule',
        component: Rule,
        children: [
            {
                path: 'Now',
                name: 'Now',
                component: Now,
            },
            {
                path: 'Group',
                name: 'Group',
                component: Group,
            },
            {
                path: 'Ignore',
                name: 'Ignore',
                component: Ignore,
            },
            {
                path: 'Providers',
                name: 'RuleProviders',
                component: RuleProviders,
            },
        ],
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
    {
        path: '/Crawl',
        name: 'Crawl',
        component: Crawl,
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