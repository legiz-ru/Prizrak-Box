<script setup lang="ts">
// One 58px row of the connections list (also used in the process detail view).
import {prettyBytes} from "@/util/format";

defineProps<{
  host: string;
  connType: string;
  process?: string;
  route: string;
  upload: number;
  download: number;
  uploadSpeed: number;
  downloadSpeed: number;
  live: boolean;
  age: string;
  ageExact: string;
}>();

defineEmits<{ (e: 'log'): void; (e: 'copy'): void }>();
</script>

<template>
  <div class="conn-row" :class="{ 'is-live': live }">
    <div class="conn-row__actions">
      <UiIconButton :size="26" :label="$t('connections.view-log')" @click="$emit('log')">
        <icon-tabler-info-circle width="16" height="16"/>
      </UiIconButton>
      <UiIconButton :size="26" :label="$t('connections.copy-log')" @click="$emit('copy')">
        <icon-tabler-copy width="15" height="15"/>
      </UiIconButton>
    </div>
    <div class="conn-row__main">
      <div class="conn-row__host ellipsis" v-tip="host.length > 40 ? host : ''">{{ host }}</div>
      <div class="conn-row__sub">
        <span class="conn-row__type">{{ connType }}</span>
        <template v-if="process">
          <span class="conn-row__dot">·</span>
          <span class="conn-row__proc ellipsis" v-tip="process">{{ process }}</span>
        </template>
        <template v-if="route">
          <span class="conn-row__dot">·</span>
          <span class="conn-row__route ellipsis" v-tip="route">{{ route }}</span>
        </template>
      </div>
    </div>
    <div class="conn-row__meta">
      <div class="conn-row__traffic tabular">
        <span class="conn-row__up" v-tip="$t('connections.upload')">
          <icon-tabler-arrow-up width="13" height="13" stroke-width="2.4"/>{{ prettyBytes(upload) }}
        </span>
        <span class="conn-row__down" v-tip="$t('connections.download')">
          <icon-tabler-arrow-down width="13" height="13" stroke-width="2.4"/>{{ prettyBytes(download) }}
        </span>
      </div>
      <div v-if="live" class="conn-row__speed tabular">
        <span class="is-up">↑ {{ prettyBytes(uploadSpeed) }}/s</span>
        <span class="is-down">↓ {{ prettyBytes(downloadSpeed) }}/s</span>
      </div>
      <div v-else class="conn-row__age" v-tip="ageExact">{{ age }}</div>
    </div>
  </div>
</template>

<style scoped>
.conn-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 14px 0 10px;
  height: 58px;
  border-bottom: 1px solid var(--border);
  min-width: 0;
}

.conn-row:hover {
  background: var(--hover-bg);
}

.conn-row.is-live {
  box-shadow: inset 3px 0 0 var(--accent);
}

.conn-row__actions {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex-shrink: 0;
}

.conn-row__main {
  flex: 1;
  min-width: 0;
}

.conn-row__host {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 2px;
}

.conn-row__sub {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--text-2);
  white-space: nowrap;
  min-width: 0;
}

.conn-row__type {
  flex-shrink: 0;
  letter-spacing: .02em;
}

.conn-row__dot {
  opacity: .5;
  flex-shrink: 0;
}

.conn-row__proc {
  flex-shrink: 0;
  max-width: 32%;
}

.conn-row__meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  flex-shrink: 0;
}

.conn-row__traffic {
  display: flex;
  gap: 10px;
  font-size: 13px;
  white-space: nowrap;
}

.conn-row__up, .conn-row__down {
  display: flex;
  align-items: center;
  gap: 3px;
}

.conn-row__up svg, .conn-row__speed .is-up { color: var(--upload-color); }
.conn-row__down svg, .conn-row__speed .is-down { color: var(--download-color); }

.conn-row__speed {
  display: flex;
  gap: 8px;
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.conn-row__age {
  font-size: 11px;
  color: var(--text-3);
  white-space: nowrap;
}
</style>
