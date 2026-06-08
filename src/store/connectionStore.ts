import { defineStore } from 'pinia';
import { defaultPersist } from '@/types/persist';

export type ConnectionViewMode = 'list' | 'topology' | 'process';

const MAX_CLOSED = 500;

export const useConnectionStore = defineStore('connections', {
  state: () => ({
    viewMode: 'topology' as ConnectionViewMode,
    showClosed: false,
    closedConnections: [] as any[],
    _prevIds: new Set<string>() as Set<string>,
  }),
  actions: {
    setViewMode(viewMode: ConnectionViewMode) {
      this.viewMode = viewMode;
    },
    toggleViewMode() {
      this.viewMode = this.viewMode === 'list' ? 'topology' : 'list';
    },
    setShowClosed(val: boolean) {
      this.showClosed = val;
    },
    updateConnections(active: any[]) {
      const activeIds = new Set<string>(active.map((c: any) => c.id));
      const closed: any[] = [];
      for (const id of this._prevIds) {
        if (!activeIds.has(id)) {
          // find the connection object from previous snapshot via active list won't work,
          // so we store prev snapshot
        }
      }
      this._prevIds = activeIds;
    },
    recordClosed(prevSnapshot: any[], nextSnapshot: any[]) {
      const nextIds = new Set<string>(nextSnapshot.map((c: any) => c.id));
      for (const conn of prevSnapshot) {
        if (!nextIds.has(conn.id)) {
          this.closedConnections.unshift({ ...conn, closedAt: Date.now() });
        }
      }
      if (this.closedConnections.length > MAX_CLOSED) {
        this.closedConnections.length = MAX_CLOSED;
      }
    },
    clearClosed() {
      this.closedConnections = [];
    },
  },
  persist: {
    pick: ['viewMode'],
  },
});
