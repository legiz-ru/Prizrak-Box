import {defineStore} from 'pinia';
import {defaultPersist} from "@/types/persist";

export const useProxiesStore = defineStore('proxies', {
    state: () => ({
        isHide: false,
        isSort: false,
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
