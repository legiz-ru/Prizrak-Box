import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';

import Home from '@/views/Home.vue';
import Setting from '@/views/Setting.vue';
import Proxies from '@/views/Proxies.vue';
import Profiles from '@/views/Profiles.vue';
import Dns from '@/views/setting/Dns.vue';
import Shortcut from '@/views/setting/Shortcut.vue';
import {useMenuStore} from '@/store/menuStore';

function settingsTab(tab: string) {
    useMenuStore().setSettingTab(tab);
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
    // Legacy screens now live inside Settings; keep old paths (persisted menu
    // state, tray/deeplink callers) working by redirecting to the right tab.
    {path: '/Rule/:sub(.*)*', redirect: () => settingsTab('rule')},
    {path: '/Connection', redirect: () => settingsTab('connection')},
    {path: '/Log', redirect: () => settingsTab('log')},
    {path: '/Crawl', redirect: '/Home'},
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

export default router;