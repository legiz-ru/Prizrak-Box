<script setup lang="ts">
import {useI18n} from "vue-i18n";
import {useWebStore} from "@/store/webStore";
import {confirm, toast} from "@/components/ui";
import IconTrash from "~icons/tabler/trash";
import IconFileText from "~icons/tabler/file-text";
import IconSearchOff from "~icons/tabler/search-off";

interface LogEntry {
  time: string;
  type: string;
  payload: string;
}

// Keep the DOM light: never render more than this many lines.
const MAX_LINES = 1000;

const {t} = useI18n();
const webStore = useWebStore();

const search = ref('');
const paused = ref(false);
const autoscroll = ref(true);
const snapshot = ref<LogEntry[]>([]);
const listEl = ref<HTMLElement | null>(null);

watch(paused, (value) => {
  snapshot.value = value ? webStore.logs.slice() : [];
});

const source = computed<LogEntry[]>(() => paused.value ? snapshot.value : webStore.logs);

const filtered = computed(() => {
  const query = search.value.trim().toLowerCase();
  const list = query
      ? source.value.filter((item) =>
          String(item.payload).toLowerCase().includes(query) || String(item.type).toLowerCase().includes(query))
      : source.value;
  return list.length > MAX_LINES ? list.slice(list.length - MAX_LINES) : list;
});

function tone(type: string) {
  switch (String(type).toUpperCase()) {
    case 'ERROR':
      return 'error';
    case 'WARNING':
    case 'WARN':
      return 'warning';
    case 'DEBUG':
      return 'debug';
    default:
      return 'info';
  }
}

function segments(text: string) {
  const query = search.value.trim();
  const value = String(text ?? '');
  if (!query) return [{text: value, hit: false}];
  const lower = value.toLowerCase();
  const needle = query.toLowerCase();
  const out: { text: string, hit: boolean }[] = [];
  let from = 0;
  let at = lower.indexOf(needle);
  while (at !== -1) {
    if (at > from) out.push({text: value.slice(from, at), hit: false});
    out.push({text: value.slice(at, at + needle.length), hit: true});
    from = at + needle.length;
    at = lower.indexOf(needle, from);
  }
  if (from < value.length) out.push({text: value.slice(from), hit: false});
  return out;
}

function scrollToBottom() {
  const el = listEl.value;
  if (el) el.scrollTop = el.scrollHeight;
}

watch(() => [filtered.value.length, filtered.value[filtered.value.length - 1]], () => {
  if (autoscroll.value) nextTick(scrollToBottom);
});

watch(autoscroll, (value) => {
  if (value) nextTick(scrollToBottom);
});

onMounted(() => nextTick(scrollToBottom));

function asText(list: LogEntry[]) {
  return list.map((item) => `${item.time} [${item.type}] ${item.payload}`).join('\n');
}

async function copyAll() {
  const list = filtered.value;
  if (!list.length) return;
  try {
    await navigator.clipboard.writeText(asText(list));
    toast('success', t('logs.copied', {n: list.length}));
  } catch {
    toast('error', t('copy.fail'));
  }
}

function exportLog() {
  const list = filtered.value;
  if (!list.length) return;
  const blob = new Blob([asText(list) + '\n'], {type: 'text/plain;charset=utf-8'});
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  link.href = url;
  link.download = `prizrak-box-${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}.log`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

async function clearAll() {
  if (!source.value.length) return;
  const ok = await confirm({title: t('logs.clear'), icon: IconTrash, okLabel: t('logs.clear')});
  if (!ok) return;
  webStore.clearLogs();
  snapshot.value = [];
  toast('info', t('logs.cleared'));
}
</script>

<template>
  <div class="logs">
    <div class="logs-bar">
      <div class="px-search logs-search">
        <icon-tabler-search width="15" height="15"/>
        <input v-model="search"
               class="px-input px-input--pill"
               type="search"
               :placeholder="t('logs.search')"
               :aria-label="t('logs.search')">
      </div>
      <button type="button" class="px-btn px-btn--chip" :aria-pressed="paused ? 'true' : 'false'" @click="paused = !paused">
        <icon-tabler-player-play-filled v-if="paused" width="14" height="14"/>
        <icon-tabler-player-pause-filled v-else width="14" height="14"/>
        {{ paused ? t('resume') : t('pause') }}
      </button>
      <button type="button"
              class="px-btn px-btn--chip logs-auto"
              role="switch"
              :aria-checked="autoscroll ? 'true' : 'false'"
              :class="{ 'is-on': autoscroll }"
              @click="autoscroll = !autoscroll">
        <icon-tabler-arrow-bar-to-down width="14" height="14"/>
        {{ t('logs.autoscroll') }}
      </button>
      <span class="logs-spacer"/>
      <UiIconButton round soft :size="36" :label="t('logs.copy-all')" :disabled="!filtered.length" @click="copyAll">
        <icon-tabler-copy width="16" height="16"/>
      </UiIconButton>
      <UiIconButton round soft :size="36" :label="t('logs.export')" :disabled="!filtered.length" @click="exportLog">
        <icon-tabler-download width="16" height="16"/>
      </UiIconButton>
      <UiIconButton round soft danger :size="36" :label="t('logs.clear')" :disabled="!source.length" @click="clearAll">
        <icon-tabler-trash width="16" height="16"/>
      </UiIconButton>
    </div>

    <div v-if="!source.length" class="logs-panel logs-panel--empty">
      <UiEmpty :icon="IconFileText" :title="t('empty.logs.title')" :text="t('empty.logs.text')"/>
    </div>
    <div v-else-if="!filtered.length" class="logs-panel logs-panel--empty">
      <UiEmpty :icon="IconSearchOff" :title="t('empty.search.title')" :text="t('empty.search.text')"/>
    </div>
    <div v-else ref="listEl" class="logs-panel logs-list mono" role="log" aria-live="off">
      <div v-for="(item, i) in filtered" :key="i" class="log-line">
        <span class="log-time tabular">{{ item.time }}</span>
        <span class="log-type" :class="'is-' + tone(item.type)">{{ item.type }}</span>
        <span class="log-payload"><template v-for="(part, j) in segments(item.payload)" :key="j"><mark v-if="part.hit" class="log-hit">{{ part.text }}</mark><template v-else>{{ part.text }}</template></template></span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.logs {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.logs-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.logs-search {
  flex: 1 1 240px;
  min-width: 200px;
}

.logs-search > svg {
  left: 14px;
}

.logs-search > .px-input {
  padding-left: 36px;
}

.logs-auto.is-on {
  color: var(--accent);
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
}

.logs-spacer {
  flex: 1 0 0;
}

.logs-panel {
  flex: 1;
  min-height: 0;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--input-bg);
}

.logs-panel--empty {
  display: flex;
  align-items: center;
  justify-content: center;
}

.logs-list {
  overflow-y: auto;
  overflow-x: hidden;
  padding: 6px 0;
  user-select: text;
}

.log-line {
  display: grid;
  grid-template-columns: auto auto 1fr;
  gap: 10px;
  align-items: baseline;
  padding: 4px 14px;
  font-size: 12px;
  line-height: 1.5;
}

.log-line:hover {
  background: var(--hover-bg);
}

.log-time {
  color: var(--text-3);
  white-space: nowrap;
}

.log-type {
  font-size: 11px;
  font-weight: 700;
  min-width: 56px;
}

.log-type.is-info {
  color: var(--info);
}

.log-type.is-warning {
  color: var(--warning);
}

.log-type.is-error {
  color: var(--error);
}

.log-type.is-debug {
  color: var(--text-3);
}

.log-payload {
  color: var(--text);
  word-break: break-word;
  min-width: 0;
}

.log-hit {
  background: color-mix(in srgb, var(--warning) 35%, transparent);
  color: inherit;
  border-radius: 3px;
}
</style>
