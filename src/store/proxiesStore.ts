import {defineStore} from 'pinia';
import {defaultPersist} from "@/types/persist";

export type ProxyViewMode = 'horizontal' | 'dropdown' | 'full';

export const useProxiesStore = defineStore('proxies', {
    state: () => ({
        isHide: false,
        isSort: false,
        viewMode: 'full' as ProxyViewMode,
        active: '',
        now: "",
        groupExpansion: {} as Record<string, boolean>,
    }),
    actions: {
        setHide(isHide: boolean) {
            this.isHide = isHide;
        },
        setSort(isSort: boolean) {
            this.isSort = isSort;
        },
        setViewMode(viewMode: ProxyViewMode) {
            this.viewMode = viewMode;
        },
        setActive(active: string) {
            this.active = active;
        },
        setNow(now: string) {
            this.now = now;
        },
        setGroupExpansionState(group: string, expanded: boolean) {
            this.groupExpansion = {
                ...this.groupExpansion,
                [group]: expanded,
            };
        },
        replaceGroupExpansions(expansions: Record<string, boolean>) {
            this.groupExpansion = expansions;
        },
    },
    persist: defaultPersist,
});
