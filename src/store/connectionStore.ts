import { defineStore } from 'pinia';
import { defaultPersist } from '@/types/persist';

export type ConnectionViewMode = 'list' | 'topology';

export const useConnectionStore = defineStore('connections', {
  state: () => ({
    viewMode: 'list' as ConnectionViewMode,
  }),
  actions: {
    setViewMode(viewMode: ConnectionViewMode) {
      this.viewMode = viewMode;
    },
    toggleViewMode() {
      this.viewMode = this.viewMode === 'list' ? 'topology' : 'list';
    },
  },
  persist: defaultPersist,
});
