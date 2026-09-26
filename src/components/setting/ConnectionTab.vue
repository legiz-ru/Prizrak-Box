<script setup lang="ts">
import {computed, getCurrentInstance, onBeforeUnmount, onMounted, ref, watch} from "vue";
import ConnectionRow from "@/components/setting/ConnectionRow.vue";
import MihomoCoreIcon from "@/components/setting/MihomoCoreIcon.vue";
import UiDropdown from "@/components/ui/UiDropdown.vue";
import {confirm, toast} from "@/components/ui";
import type {UiSelectOption, UiPillOption} from "@/components/ui";
import {relativeSeconds} from "@/util/profileView";
import IconList from "~icons/tabler/list";
import IconChartSankey from "~icons/tabler/chart-sankey";
import IconAppWindow from "~icons/tabler/app-window";
import IconPlugX from "~icons/tabler/plug-connected-x";
import IconWifiOff from "~icons/tabler/wifi-off";
import ConnectionTopology from "@/components/topology/ConnectionTopology.vue";
import {WS} from "@/util/ws";
import {useWebStore} from "@/store/webStore";
import {useConnectionStore} from "@/store/connectionStore";
import {prettyBytes} from "@/util/format";
import {useI18n} from "vue-i18n";
import createApi from "@/api";

const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const {t} = useI18n()
// Relative age ("3 мин назад") with the exact start time in the tooltip.
const nowTick = ref(Date.now())
function fAge(start: any): string {
  const ms = new Date(start).getTime()
  if (!Number.isFinite(ms)) return ''
  return relativeSeconds(t, Math.max(0, (nowTick.value - ms) / 1000))
}
function fExact(start: any): string {
  const date = new Date(start)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
}

const search = ref('')
const logDialogVisible = ref(false)
const logItem = ref<any>(null)

const dialogTick = ref(0)
let durationTimer: ReturnType<typeof setInterval> | null = null

watch(logDialogVisible, (visible) => {
  if (visible) {
    dialogTick.value = 0
    durationTimer = setInterval(() => { dialogTick.value++ }, 1000)
  } else {
    if (durationTimer) { clearInterval(durationTimer); durationTimer = null }
  }
})

interface LogRow {
  label: string
  value: string
  type: 'text' | 'ip' | 'host' | 'process' | 'path'
  explorerPath?: string
}
interface LogSection {
  title: string
  rows: LogRow[]
}

function formatDuration(start: string): string {
  const _ = dialogTick.value
  const elapsed = Math.floor((Date.now() - new Date(start).getTime()) / 1000)
  if (elapsed < 0) return '—'
  const h = Math.floor(elapsed / 3600)
  const m = Math.floor((elapsed % 3600) / 60)
  const s = elapsed % 60
  return [h, m, s].map(v => String(v).padStart(2, '0')).join(':')
}

function formatChains(chains: string[]): string {
  if (!chains || chains.length === 0) return '—'
  return [...chains].reverse().join(' → ')
}

function formatConnType(metadata: any): string {
  const net = (metadata?.network || '').toUpperCase()
  const type = metadata?.type || ''
  if (net && type && net !== type) return `${net}(${type})`
  return net || type || '—'
}

const dialogTitle = computed(() => {
  if (!logItem.value) return t('connections.dialog-title')
  const m = logItem.value.metadata
  if (!m) return t('connections.dialog-title')
  const host = m.host || m.destinationIP || ''
  const port = m.destinationPort || ''
  return host && port ? `${host}:${port}` : host || t('connections.dialog-title')
})

const dialogSections = computed<LogSection[]>(() => {
  const _ = dialogTick.value
  if (!logItem.value) return []
  const item = logItem.value
  const m = item.metadata || {}
  const isInner = m.type === 'Inner'

  const row = (label: string, value: any, type: LogRow['type'] = 'text', explorerPath?: string): LogRow => ({
    label, value: String(value ?? ''), type, explorerPath
  })

  const sections: (LogSection | null)[] = [
    {
      title: t('connections.section-traffic'),
      rows: [
        row(t('connections.upload-speed'), item.uploadSpeed ? prettyBytes(item.uploadSpeed) + '/s' : '—'),
        row(t('connections.download-speed'), item.downloadSpeed ? prettyBytes(item.downloadSpeed) + '/s' : '—'),
        row(t('connections.upload'), prettyBytes(item.upload || 0)),
        row(t('connections.download'), prettyBytes(item.download || 0)),
        row(t('connections.duration'), formatDuration(item.start)),
      ]
    },
    {
      title: t('connections.section-routing'),
      rows: [
        item.rule ? row(t('connections.rule'), [item.rule, item.rulePayload].filter(Boolean).join(' / ')) : null,
        row(t('connections.proxy-chain'), formatChains(item.chains)),
        row(t('connections.conn-type'), formatConnType(m)),
      ].filter(Boolean) as LogRow[]
    },
    {
      title: t('connections.section-network'),
      rows: [
        m.host ? row(t('connections.host'), m.host, 'host') : null,
        m.sniffHost && m.sniffHost !== m.host ? row(t('connections.sniff-host'), m.sniffHost, 'host') : null,
        m.destinationIP ? row(t('connections.dest-ip'), m.destinationIP, 'ip') : null,
        m.sourceIP ? row(t('connections.source-ip'), m.sourceIP, 'ip') : null,
        m.sourcePort ? row(t('connections.source-port'), m.sourcePort) : null,
        m.destinationPort ? row(t('connections.dest-port'), m.destinationPort) : null,
        m.remoteDestination ? row(t('connections.remote-dest'), m.remoteDestination, 'ip') : null,
      ].filter(Boolean) as LogRow[]
    },
    !isInner && (m.process || m.processPath) ? {
      title: t('connections.section-process'),
      rows: [
        m.process ? row(t('connections.process-label'), m.process, 'process', m.processPath || undefined) : null,
        m.processPath && m.processPath !== m.process ? row(t('connections.process-path'), m.processPath, 'path', m.processPath) : null,
      ].filter(Boolean) as LogRow[]
    } : null,
    (m.inboundIP || m.inboundPort || m.inboundName || m.inboundUser) ? {
      title: t('connections.section-inbound'),
      rows: [
        m.inboundIP ? row(t('connections.inbound-ip'), m.inboundIP, 'ip') : null,
        m.inboundPort ? row(t('connections.inbound-port'), m.inboundPort) : null,
        m.inboundName ? row(t('connections.inbound-name'), m.inboundName) : null,
        m.inboundUser ? row(t('connections.inbound-user'), m.inboundUser) : null,
      ].filter(Boolean) as LogRow[]
    } : null,
    (m.dnsMode || m.specialProxy || m.specialRules || m.dscp) ? {
      title: t('connections.section-other'),
      rows: [
        m.dnsMode ? row(t('connections.dns-mode'), m.dnsMode) : null,
        m.specialProxy ? row(t('connections.special-proxy'), m.specialProxy) : null,
        m.specialRules ? row(t('connections.special-rules'), m.specialRules) : null,
        m.dscp ? row(t('connections.dscp'), String(m.dscp)) : null,
      ].filter(Boolean) as LogRow[]
    } : null,
  ]

  return sections.filter(Boolean) as LogSection[]
})

async function copyText(text: string) {
  if (!navigator.clipboard) { toast('error', t('copy.fail')); return }
  try {
    await navigator.clipboard.writeText(text)
    toast('success', t('copy.success'))
  } catch {
    toast('error', t('copy.fail'))
  }
}

function openInBrowser(url: string) {
  ;(window as any).pxOpen?.(url)
}

function showInExplorer(path: string) {
  ;(window as any).pxShowInFolder?.(path)
}


function fHost(metadata: any): string {
  return (metadata.host || metadata.destinationIP) + ':' + metadata.destinationPort
}

function isLive(item: any): boolean {
  return !connectionStore.showClosed && ((item?.uploadSpeed || 0) > 0 || (item?.downloadSpeed || 0) > 0)
}

// Per-connection live speed (bytes/s), diffed from the previous websocket frame
// (Mihomo's /connections payload carries only cumulative upload/download).
let prevConnStats = new Map<string, { u: number; d: number; t: number }>()

function computeSpeeds(conns: any[]): void {
  const now = Date.now()
  const nextStats = new Map<string, { u: number; d: number; t: number }>()
  for (const c of conns) {
    const up = c.upload || 0
    const down = c.download || 0
    const prev = prevConnStats.get(c.id)
    const dt = prev ? (now - prev.t) / 1000 : 0
    if (prev && dt > 0) {
      c.uploadSpeed = Math.max(0, Math.round((up - prev.u) / dt))
      c.downloadSpeed = Math.max(0, Math.round((down - prev.d) / dt))
    } else {
      c.uploadSpeed = 0
      c.downloadSpeed = 0
    }
    nextStats.set(c.id, { u: up, d: down, t: now })
  }
  prevConnStats = nextStats
}

// Sorting (zashboard-style): field + direction. "time" = newest first when
// descending.
type SortKey = 'time' | 'download' | 'upload' | 'dl-speed' | 'ul-speed' | 'host'
const sortKey = ref<SortKey>('time')
const sortDesc = ref(true)
const SORT_VALUE: Record<SortKey, (c: any) => number | string> = {
  time: (c) => new Date(c.start).getTime() || 0,
  download: (c) => c.download || 0,
  upload: (c) => c.upload || 0,
  'dl-speed': (c) => c.downloadSpeed || 0,
  'ul-speed': (c) => c.uploadSpeed || 0,
  host: (c) => fHost(c.metadata).toLowerCase(),
}
const sortOptions = computed<UiSelectOption<SortKey>[]>(() =>
  (Object.keys(SORT_VALUE) as SortKey[]).map(key => ({value: key, label: t('connections.sort.' + key)})))

function filterData(cacheData: any): any[] {
  if (!cacheData || cacheData.length === 0) {
    return []
  }
  const searchLower = search.value.toLowerCase();
  const cache = cacheData.filter((data: any) => {
    return (
        (!search.value || fHost(data.metadata).toLowerCase().includes(searchLower)) ||
        (data.rule || '').toLowerCase().includes(searchLower) ||
        (data.metadata.process && data.metadata.process.toLowerCase().includes(searchLower))
    );
  });
  const pick = SORT_VALUE[sortKey.value]
  const dir = sortDesc.value ? -1 : 1
  cache.sort((a: any, b: any) => {
    const x = pick(a), y = pick(b)
    return (x > y ? 1 : x < y ? -1 : 0) * dir
  });
  return cache;
}

const paginatedData = ref<any[]>([]);

// Pause freezes what the list shows; the socket keeps running underneath so
// closed connections are still recorded.
const paused = ref(false)
const frozen = ref<any[] | null>(null)
watch(paused, (value) => {
  frozen.value = value ? [...displaySource.value] : null
})

function onConn(ev: MessageEvent) {
  const parsedData = JSON.parse(ev.data);
  const next: any[] = parsedData['connections'] ?? [];
  computeSpeeds(next);
  connectionStore.recordClosed(paginatedData.value, next);
  paginatedData.value = next;
  if (!paused.value) nowTick.value = Date.now();
}

const displaySource = computed(() =>
  connectionStore.showClosed ? connectionStore.closedConnections : paginatedData.value
);
const displayData = computed(() => frozen.value ?? displaySource.value);
const sortedList = computed(() => filterData(displayData.value));

// Virtual list: fixed 58px rows, only the visible window (+ margin) is rendered.
const ROW = 58
const listEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewH = ref(600)
let listObserver: ResizeObserver | null = null
watch(listEl, (el) => {
  listObserver?.disconnect()
  if (!el) return
  viewH.value = el.clientHeight || 600
  listObserver = new ResizeObserver(() => { viewH.value = el.clientHeight || 600 })
  listObserver.observe(el)
})
let scrollRaf = 0
function onListScroll(e: Event) {
  const top = (e.currentTarget as HTMLElement).scrollTop
  cancelAnimationFrame(scrollRaf)
  scrollRaf = requestAnimationFrame(() => { scrollTop.value = top })
}
const vStart = computed(() => Math.max(0, Math.floor(scrollTop.value / ROW) - 6))
const vEnd = computed(() => Math.min(sortedList.value.length, Math.ceil((scrollTop.value + viewH.value) / ROW) + 6))
const visibleRows = computed(() => sortedList.value.slice(vStart.value, vEnd.value))
watch([sortKey, sortDesc, search, () => connectionStore.showClosed], () => {
  if (listEl.value) listEl.value.scrollTop = 0
  scrollTop.value = 0
})

function routeOf(item: any): string {
  return [[item.rule, item.rulePayload].filter(Boolean).join(' / '), item.chains?.length ? formatChains(item.chains) : ''].filter(Boolean).join(' · ')
}

const viewOptions = computed<UiPillOption<'list' | 'topology' | 'process'>[]>(() => [
  {value: 'list', label: t('connections.list'), icon: IconList},
  {value: 'topology', label: t('connections.topology-view'), icon: IconChartSankey},
  {value: 'process', label: t('connections.process-view'), icon: IconAppWindow},
])

function openLogDialog(item: any) {
  logItem.value = item
  logDialogVisible.value = true
}

function closeLogDialog() {
  logDialogVisible.value = false
}

async function copyLog(item?: any) {
  const target = item ?? logItem.value
  const data = target ? JSON.stringify(target, null, 2) : ''
  if (!data) return
  if (!navigator.clipboard) {
    toast('error', t('copy.fail'))
    return
  }
  try {
    await navigator.clipboard.writeText(data)
    toast('success', t('copy.success'))
  } catch (error) {
    toast('error', t('copy.fail'))
  }
}

const webStore = useWebStore()
const connectionStore = useConnectionStore()

// ── Process view ──────────────────────────────────────────────────────────────
const selectedProcess = ref<string | null>(null)
const iconCache = ref<Record<string, string | null>>({})
const iconLoadingSet = new Set<string>()

const MIHOMO_CORE_KEY = '__prizrak_mihomo_core__'
const MIHOMO_CORE_NAME = 'Prizrak (mihomo core)'

async function loadIcon(processPath: string) {
  if (!processPath || processPath in iconCache.value || iconLoadingSet.has(processPath)) return
  iconLoadingSet.add(processPath)
  try {
    const dataUrl = await (window as any).electron?.invoke('get-file-icon', processPath)
    iconCache.value = { ...iconCache.value, [processPath]: dataUrl ?? null }
  } catch {
    iconCache.value = { ...iconCache.value, [processPath]: null }
  } finally {
    iconLoadingSet.delete(processPath)
  }
}

interface ProcessGroup {
  processName: string
  processPath: string
  count: number
  download: number
  upload: number
  iconUrl: string | null
  isMihomoCore?: boolean
}

const processGroups = computed<ProcessGroup[]>(() => {
  const groups = new Map<string, ProcessGroup>()
  for (const conn of (displayData.value || [])) {
    const isInner = (conn as any).metadata?.type === 'Inner'
    const name: string = isInner ? MIHOMO_CORE_NAME : ((conn as any).metadata?.process || t('connections.unknown-process'))
    const path: string = isInner ? MIHOMO_CORE_KEY : ((conn as any).metadata?.processPath || name)
    if (!groups.has(path)) {
      groups.set(path, { processName: name, processPath: path, count: 0, download: 0, upload: 0, iconUrl: null, isMihomoCore: isInner })
      if (!isInner && path !== name) loadIcon(path)
    }
    const g = groups.get(path)!
    g.count++
    g.download += (conn as any).download ?? 0
    g.upload += (conn as any).upload ?? 0
    if (!isInner) g.iconUrl = iconCache.value[path] ?? null
  }
  return [...groups.values()].sort((a, b) => b.count - a.count)
})

const filteredProcessGroups = computed(() => processGroups.value.filter(g => !search.value || g.processName.toLowerCase().includes(search.value.toLowerCase())))

const selectedProcessConnections = computed(() => {
  if (!selectedProcess.value) return []
  return filterData(displayData.value)?.filter((c: any) => {
    if (c.metadata?.type === 'Inner') return selectedProcess.value === MIHOMO_CORE_KEY
    const path = c.metadata?.processPath || c.metadata?.process || t('connections.unknown-process')
    return path === selectedProcess.value
  }) ?? []
})

watch(() => connectionStore.viewMode, () => {
  selectedProcess.value = null
})

let wsConn: WS | null = null
onMounted(() => {
  const urlTraffic = webStore.wsUrl + "/connections?token=" + webStore.secret;
  wsConn = new WS(urlTraffic, null, onConn);
})

onBeforeUnmount(() => {
  if (wsConn) {
    wsConn.close();
    wsConn = null
  }
  if (durationTimer) { clearInterval(durationTimer); durationTimer = null }
  listObserver?.disconnect()
})

// Accepted product change: closing all connections asks first.
async function askCloseAll() {
  const ok = await confirm({
    title: t('confirm.close-all.title'),
    text: t('confirm.close-all.text'),
    okLabel: t('confirm.close-all.ok'),
    icon: IconPlugX,
  })
  if (ok) closeAll()
}

function closeAll() {
  const data = filterData(paginatedData.value)  // always use active connections for closing
  if (data && data.length > 0) {
    if (search.value) {
      for (let connection of data) {
        api.closeConnection(connection.id)
      }
    } else {
      api.closeAllConnection()
    }
  }
}
</script>

<template>
  <div class="conn">
    <div class="conn-bar">
      <button type="button" class="px-btn px-btn--chip" @click="askCloseAll">{{ $t('connections.close') }}</button>
      <template v-if="connectionStore.viewMode === 'list'">
        <UiSelect v-model="sortKey"
                  variant="pill"
                  align="left"
                  :min-width="220"
                  :options="sortOptions"
                  class="conn-sort"
                  :aria-label="t('connections.sort.label')"
                  :tip="t('connections.sort.label')">
          <template #prefix><icon-tabler-arrows-sort width="14" height="14"/></template>
        </UiSelect>
        <UiIconButton round soft :size="36"
                      :label="sortDesc ? t('connections.sort.desc') : t('connections.sort.asc')"
                      @click="sortDesc = !sortDesc">
          <icon-tabler-arrow-down width="15" height="15" class="conn-dir" :class="{ 'is-asc': !sortDesc }"/>
        </UiIconButton>
        <button type="button" class="px-btn px-btn--chip" :aria-pressed="paused ? 'true' : 'false'" @click="paused = !paused">
          <icon-tabler-player-play-filled v-if="paused" width="14" height="14"/>
          <icon-tabler-player-pause-filled v-else width="14" height="14"/>
          {{ paused ? t('resume') : t('pause') }}
        </button>
        <span class="conn-count tabular" role="status">{{ t('connections.count', {n: sortedList.length}) }}</span>
      </template>
      <UiPillTabs v-model="connectionStore.viewMode"
                  class="conn-views"
                  size="md"
                  :options="viewOptions"
                  :aria-label="t('connections.toggleView')"/>
    </div>

    <div v-if="connectionStore.viewMode !== 'topology'" class="px-search conn-search">
      <icon-tabler-search width="15" height="15"/>
      <input v-model="search" class="px-input px-input--pill" type="search" :placeholder="$t('connections.search')" :aria-label="$t('connections.search')">
    </div>

    <!-- List (virtualised) -->
    <div v-if="connectionStore.viewMode === 'list'" ref="listEl" class="conn-panel conn-list" @scroll.passive="onListScroll">
      <UiEmpty v-if="sortedList.length === 0" :icon="IconWifiOff" :title="t('empty.connections.title')" :text="t('empty.connections.text')"/>
      <template v-else>
        <div :style="{ height: vStart * ROW + 'px' }"></div>
        <ConnectionRow v-for="item in visibleRows"
                       :key="item.id"
                       :host="fHost(item.metadata)"
                       :conn-type="formatConnType(item.metadata)"
                       :process="item.metadata.type === 'Inner' ? '' : item.metadata.process"
                       :route="routeOf(item)"
                       :upload="item.upload"
                       :download="item.download"
                       :upload-speed="item.uploadSpeed"
                       :download-speed="item.downloadSpeed"
                       :live="isLive(item)"
                       :age="fAge(item.start)"
                       :age-exact="fExact(item.start)"
                       @log="openLogDialog(item)"
                       @copy="copyLog(item)"/>
        <div :style="{ height: Math.max(0, (sortedList.length - vEnd) * ROW) + 'px' }"></div>
      </template>
    </div>

    <!-- Topology -->
    <div v-else-if="connectionStore.viewMode === 'topology'" class="conn-panel conn-topology">
      <ConnectionTopology :connections="paginatedData"/>
    </div>

    <!-- Processes -->
    <div v-else class="conn-panel conn-process">
      <template v-if="selectedProcess === null">
        <button v-for="group in filteredProcessGroups"
                :key="group.processPath"
                type="button"
                class="proc-row"
                @click="selectedProcess = group.processPath">
          <span class="proc-icon">
            <MihomoCoreIcon v-if="group.isMihomoCore"/>
            <img v-else-if="group.iconUrl" :src="group.iconUrl" alt="">
            <icon-tabler-app-window v-else width="30" height="30" stroke-width="1.6"/>
          </span>
          <span class="proc-body">
            <span class="proc-name-row">
              <span class="proc-name ellipsis" v-tip="group.processName">{{ group.processName }}</span>
              <span class="px-tag px-tag--accent">{{ $t('connections.connections-count', {count: group.count}) }}</span>
            </span>
            <span class="proc-stats tabular">
              {{ $t('connections.download') }}: <b>{{ prettyBytes(group.download) }}</b>
              &nbsp; {{ $t('connections.upload') }}: <b>{{ prettyBytes(group.upload) }}</b>
            </span>
          </span>
          <icon-tabler-chevron-right width="20" height="20" class="proc-chev"/>
        </button>
        <UiEmpty v-if="filteredProcessGroups.length === 0" :icon="IconWifiOff" :title="t('empty.connections.title')" :text="t('empty.connections.text')"/>
      </template>
      <template v-else>
        <button type="button" class="proc-back" @click="selectedProcess = null">
          <icon-tabler-arrow-left width="18" height="18"/>
          {{ $t('connections.back') }}
          <span class="proc-back__name ellipsis">— {{ processGroups.find(g => g.processPath === selectedProcess)?.processName }}</span>
        </button>
        <ConnectionRow v-for="item in selectedProcessConnections"
                       :key="item.id"
                       :host="fHost(item.metadata)"
                       :conn-type="formatConnType(item.metadata)"
                       :route="routeOf(item)"
                       :upload="item.upload"
                       :download="item.download"
                       :upload-speed="item.uploadSpeed"
                       :download-speed="item.downloadSpeed"
                       :live="isLive(item)"
                       :age="fAge(item.start)"
                       :age-exact="fExact(item.start)"
                       @log="openLogDialog(item)"
                       @copy="copyLog(item)"/>
        <UiEmpty v-if="selectedProcessConnections.length === 0" :icon="IconWifiOff" :title="t('empty.connections.title')" :text="t('empty.connections.text')"/>
      </template>
    </div>
  </div>

  <!-- Connection log -->
  <UiModal v-model="logDialogVisible" :width="600" :aria-label="dialogTitle" body-class="log-body">
    <template #header>
      <span class="log-title ellipsis" v-tip="dialogTitle">{{ dialogTitle }}</span>
      <UiIconButton round :label="$t('connections.copy-log')" @click="copyLog()">
        <icon-tabler-copy width="17" height="17"/>
      </UiIconButton>
    </template>
    <div v-if="logItem" class="log-parsed">
      <section v-for="section in dialogSections" :key="section.title" class="log-section">
        <div class="log-section__title">{{ section.title }}</div>
        <div v-for="row in section.rows" :key="row.label" class="log-row">
          <span class="log-label">{{ row.label }}</span>
          <span v-if="row.type === 'text'" class="log-value">{{ row.value }}</span>
          <UiDropdown v-else role="menu" :min-width="190" class="log-dd">
            <template #trigger="{ toggle, attrs }">
              <button type="button" v-bind="attrs" class="log-value log-clickable" @click="toggle">{{ row.value }}</button>
            </template>
            <template #default="{ close }">
              <button type="button" role="menuitem" data-dd-item class="px-dd-item" @click="close(); copyText(row.value)">
                <icon-tabler-copy width="14" height="14"/>{{ $t('connections.copy-value') }}
              </button>
              <button v-if="row.type === 'ip'" type="button" role="menuitem" data-dd-item class="px-dd-item" @click="close(); openInBrowser('https://ipinfo.io/' + row.value)">
                <icon-tabler-external-link width="14" height="14"/>ipinfo.io
              </button>
              <button v-if="row.type === 'host'" type="button" role="menuitem" data-dd-item class="px-dd-item" @click="close(); openInBrowser('https://' + row.value)">
                <icon-tabler-external-link width="14" height="14"/>{{ $t('connections.open-in-browser') }}
              </button>
              <button v-if="(row.type === 'process' || row.type === 'path') && row.explorerPath" type="button" role="menuitem" data-dd-item class="px-dd-item" @click="close(); showInExplorer(row.explorerPath!)">
                <icon-tabler-folder-open width="14" height="14"/>{{ $t('connections.show-in-explorer') }}
              </button>
            </template>
          </UiDropdown>
        </div>
      </section>
    </div>
  </UiModal>
</template>

<style scoped>
.conn {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.conn-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.conn-sort {
  width: auto;
}

.conn-sort :deep(.px-select) {
  padding: 0 14px;
  gap: 6px;
}

.conn-dir {
  transition: transform .15s;
}

.conn-dir.is-asc {
  transform: rotate(180deg);
}

.conn-count {
  font-size: 12px;
  color: var(--text-2);
}

.conn-views {
  height: 36px;
  align-items: center;
}

.conn-search {
  flex-shrink: 0;
}

.conn-search > svg {
  left: 14px;
}

.conn-search > .px-input {
  padding-left: 36px;
}

.conn-panel {
  flex: 1;
  min-height: 0;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--input-bg);
  overflow-y: auto;
  overflow-x: hidden;
}

.conn-topology {
  padding: 14px 16px 16px;
}

.proc-row {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 10px 14px 10px 16px;
  border: none;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  cursor: pointer;
  text-align: left;
}

.proc-row:hover {
  background: var(--hover-bg);
}

.proc-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--text-2);
}

.proc-icon img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.proc-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.proc-name-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.proc-name {
  font-size: 14px;
  font-weight: 600;
}

.proc-stats {
  font-size: 13px;
  color: var(--text-2);
}

.proc-stats b {
  color: var(--text);
  font-weight: 400;
}

.proc-chev {
  opacity: .4;
  flex-shrink: 0;
}

.proc-back {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 11px 16px;
  border: none;
  border-bottom: 1px solid var(--border);
  background: var(--panel-soft);
  color: var(--text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  position: sticky;
  top: 0;
  z-index: 1;
}

.proc-back:hover {
  background: var(--hover-bg);
}

.proc-back__name {
  font-weight: 400;
  color: var(--text-2);
}

.log-title {
  flex: 1;
  font-size: 14px;
  font-weight: 700;
}

:deep(.log-body) {
  padding: 0 0 12px;
  gap: 0;
}

.log-section {
  margin-bottom: 6px;
}

.log-section__title {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .06em;
  color: var(--accent);
  padding: 8px 20px 3px;
}

.log-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 3px 20px;
  font-size: 13px;
  line-height: 1.45;
}

.log-label {
  flex: 0 0 42%;
  color: var(--text-2);
  overflow-wrap: anywhere;
}

.log-value {
  flex: 1;
  overflow-wrap: anywhere;
  user-select: text;
}

.log-dd {
  flex: 1;
}

.log-clickable {
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  font-size: 13px;
  cursor: pointer;
  color: var(--accent);
  text-decoration: underline dotted;
  text-underline-offset: 3px;
}
</style>
