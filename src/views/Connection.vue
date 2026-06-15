<script setup lang="ts">
import {computed, getCurrentInstance, onBeforeUnmount, onMounted, ref, watch} from "vue";
import MySimpleInput from "@/components/MySimpleInput.vue";
import ConnectionTopology from "@/components/topology/ConnectionTopology.vue";
import {WS} from "@/util/ws";
import {useWebStore} from "@/store/webStore";
import {useConnectionStore} from "@/store/connectionStore";
import {prettyBytes} from "@/util/format";
import {onBeforeRouteLeave} from "vue-router";
import {formatDistance, Locale} from 'date-fns';
import {enUS, ru, zhCN} from 'date-fns/locale'
import {useI18n} from "vue-i18n";
import {ElMessage} from "element-plus";
import createApi from "@/api";

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 获取 i18n
const {t} = useI18n()
const localeMap: Record<string, Locale> = {
  '简体中文': zhCN,
  'English': enUS,
  'Русский': ru,
};

function fDate(start: any): string {
  const startTime = new Date(start);
  return formatDistance(new Date(), startTime, {locale: localeMap[t('language')]})
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

// Полная строка маршрута (правило + цепочка) — используется и для отображения,
// и для тултипа (title), когда строка не помещается и усекается.
function routingText(item: any): string {
  const rule = [item.rule, item.rulePayload].filter(Boolean).join(' / ')
  const chains = item.chains?.length ? formatChains(item.chains) : ''
  return [rule, chains].filter(Boolean).join(' · ')
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
  if (!navigator.clipboard) { ElMessage.error(t('copy.fail')); return }
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('copy.success'))
  } catch {
    ElMessage.error(t('copy.fail'))
  }
}

function openInBrowser(url: string) {
  ;(window as any).pxOpen?.(url)
}

function showInExplorer(path: string) {
  ;(window as any).pxShowInFolder?.(path)
}

function handleInputChange(value: any) {
  search.value = value
}

function fHost(metadata: any): string {
  return (metadata.host || metadata.destinationIP) + ':' + metadata.destinationPort
}

function filterData(cacheData: any): any {

  if (!cacheData || cacheData.length === 0) {
    return
  }

  const cache = cacheData.filter((data: any) => {
    const searchLower = search.value.toLowerCase();
    return (
        (!search.value || fHost(data.metadata).toLowerCase().includes(searchLower)) || // 主机过滤
        data.rule.toLowerCase().includes(searchLower) || // 规则过滤
        (data.metadata.process && data.metadata.process.toLowerCase().includes(searchLower)) // 程序过滤
    );
  });

  cache.sort((obj1: any, obj2: any) => obj2.start.localeCompare(obj1.start));

  return cache;
}

const paginatedData = ref<any[]>([]);

function onConn(ev: MessageEvent) {
  const parsedData = JSON.parse(ev.data);
  const next: any[] = parsedData['connections'] ?? [];
  connectionStore.recordClosed(paginatedData.value, next);
  paginatedData.value = next;
}

const displayData = computed(() =>
  connectionStore.showClosed ? connectionStore.closedConnections : paginatedData.value
);

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
  if (!data) {
    return
  }
  if (!navigator.clipboard) {
    ElMessage.error(t('copy.fail'))
    return
  }
  try {
    await navigator.clipboard.writeText(data)
    ElMessage.success(t('copy.success'))
  } catch (error) {
    ElMessage.error(t('copy.fail'))
  }
}

const webStore = useWebStore()
const connectionStore = useConnectionStore()

// View mode options for segmented control
const viewModeOptions = computed(() => [
  {
    label: t('connections.list'),
    value: 'list',
  },
  {
    label: t('connections.topology-view'),
    value: 'topology',
  },
  {
    label: t('connections.process-view'),
    value: 'process',
  },
]);

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

// 路由切换前关闭 WebSocket
onBeforeRouteLeave(() => {
  if (wsConn) {
    wsConn.close();
    wsConn = null
  }
});

onBeforeUnmount(() => {
  if (wsConn) {
    wsConn.close();
    wsConn = null
  }
  if (durationTimer) { clearInterval(durationTimer); durationTimer = null }
})


function closeAll() {
  const data = filterData(paginatedData.value)  // always active
  if (data.length > 0) {
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
  <MyLayout>
    <template #top>
      <el-space class="space">
        <div class="title">
          {{ $t('connections.title') }}
        </div>
      </el-space>
    </template>
    <template #bottom>
      <div class="conn">
        <el-space class="op">
          <el-button @click="closeAll">
            {{ $t('connections.close') }}
          </el-button>
          <div class="view-mode-switch">
            <el-segmented v-model="connectionStore.viewMode" :options="viewModeOptions">
              <template #default="scope">
                <div>
                  {{ (scope as any).item["label"] }}
                </div>
              </template>
            </el-segmented>
          </div>
        </el-space>
        <div class="search" v-if="connectionStore.viewMode === 'list' || connectionStore.viewMode === 'process'">
          <MySimpleInput
              :onInputChange="handleInputChange"
              :placeholder="$t('connections.search')"
          ></MySimpleInput>
        </div>
      </div>

      <div class="content" v-if="connectionStore.viewMode === 'list'">
        <div class="info-list">
          <el-row
              class="info"
              v-for="(item, i) in filterData(displayData)"
              :key="i"
          >
            <el-col :span="24">
              <div class="info-card">
                <div class="info-card__actions">
                  <span class="icon-btn" role="button" tabindex="0"
                        :title="$t('connections.view-log')"
                        @click="openLogDialog(item)"
                        @keydown.enter.prevent="openLogDialog(item)"
                        @keydown.space.prevent="openLogDialog(item)">
                    <icon-mdi-information-outline/>
                  </span>
                  <span class="icon-btn" role="button" tabindex="0"
                        :title="$t('connections.copy-log')"
                        @click="copyLog(item)"
                        @keydown.enter.prevent="copyLog(item)"
                        @keydown.space.prevent="copyLog(item)">
                    <icon-mdi-content-copy/>
                  </span>
                </div>
                <div class="info-card__body">
                  <div class="info-card__row info-card__row--top">
                    <div class="info-card__host">{{ fHost(item.metadata) }}</div>
                    <div class="info-card__traffic">
                      <span class="info-traffic-item" :title="$t('connections.upload')">
                        <icon-mdi-arrow-up class="traffic-icon traffic-icon--up"/>
                        {{ prettyBytes(item.upload) }}
                      </span>
                      <span class="info-traffic-item" :title="$t('connections.download')">
                        <icon-mdi-arrow-down class="traffic-icon traffic-icon--down"/>
                        {{ prettyBytes(item.download) }}
                      </span>
                    </div>
                  </div>
                  <div class="info-card__row info-card__row--bottom">
                    <div class="info-card__tags">
                      <el-tag type="success" size="small">{{ item.metadata.type }}</el-tag>
                      <el-tag v-if="item.metadata.process" type="primary" size="small">{{ item.metadata.process }}</el-tag>
                      <el-tag type="danger" size="small">{{ fDate(item.start) }}</el-tag>
                    </div>
                    <div class="info-card__routing" :title="routingText(item)">{{ routingText(item) }}</div>
                  </div>
                </div>
              </div>
            </el-col>
          </el-row>
        </div>
      </div>

      <div class="content topology-content" v-else-if="connectionStore.viewMode === 'topology'">
        <ConnectionTopology :connections="paginatedData" /><!-- topology always shows active -->
      </div>

      <div class="content" v-else-if="connectionStore.viewMode === 'process'">
        <!-- Process list -->
        <div v-if="selectedProcess === null" class="info-list">
          <el-row
              class="info process-row"
              v-for="group in processGroups.filter(g => !search || g.processName.toLowerCase().includes(search.toLowerCase()))"
              :key="group.processPath"
              @click="selectedProcess = group.processPath"
          >
            <el-col :span="24">
              <div class="process-item-inner">
                <div class="process-icon-wrap">
                  <svg v-if="group.isMihomoCore" class="process-app-icon" viewBox="0 0 50 50" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect width="50" height="50" rx="12" fill="#FAFAFA"/>
                    <path d="M16.519 18.4845C20.3639 15.5401 19.5043 8.43845 25.5007 8.02291C33.2105 7.48864 31.627 16.4316 35.9102 20.9755C37.7728 22.9515 40.6866 20.2735 42.9969 24.2991C43.5516 25.2656 42.3157 25.9003 41.9326 26.9469C41.6858 27.6212 42.0469 28.5879 41.3356 28.6861C40.8008 28.7599 40.7312 27.8598 40.1934 27.9073C39.4708 27.9711 39.5785 28.8762 39.3627 29.5687C39.1434 30.2722 39.6046 30.8978 39.1031 31.4377C38.6755 31.8981 38.1616 31.7148 37.5715 31.9309C36.7954 32.2153 36.4479 32.8363 35.6246 32.7616C33.6258 33.2289 34.6642 30.1897 33.6258 30.9186C32.8696 31.2003 32.9645 32.0023 32.6913 32.7616C32.3217 33.7889 32.5443 34.4779 32.1981 35.5133C31.9248 36.3305 31.9241 36.9588 31.2636 37.5121C30.4392 38.2025 29.5223 37.469 28.5379 37.9015C27.6449 38.2938 27.4682 39.0648 26.565 39.433C25.6123 39.8215 24.9507 39.4839 23.9432 39.6926C22.7231 39.9453 23.0087 42.001 19.4523 42.001C15.5585 42.001 16.6042 39.3379 14.624 38.7841C12.9879 38.3265 10.9987 39.2893 9.5101 38.4706C7.79682 37.2246 8.69139 35.8487 8.316 35.3555C8.15793 35.1479 7.95257 34.9661 6.8623 34.8104C7.45936 34.1355 8.70037 34.0238 9.38031 34.447C10.205 34.9602 9.56982 36.0004 10.2889 36.6535C11.4288 37.6888 12.9175 37.0968 14.0529 36.0564C14.9801 35.2067 15.4964 34.3272 14.8836 33.2289C14.3945 32.3524 12.7639 32.7683 12.0541 32.0588C16.6747 31.5655 17.765 27.7256 17.765 27.7256C17.765 27.7256 16.0244 30.2636 14.8836 29.5687C14.1053 29.0946 14.7147 28.0146 14.0529 27.3882C12.9151 26.3111 11.0686 28.9205 10.0552 27.7256C9.18772 26.7027 10.8597 25.5128 10.2889 24.2991C9.77 23.1959 7.84874 23.4924 7.84874 22.3522C8.316 19.0534 13.1644 21.0535 16.519 18.4845Z" fill="#0A0A0A"/>
                    <path d="M16.5192 18.4845C20.364 15.5401 19.5044 8.43845 25.5008 8.02291C33.2106 7.48864 31.6272 16.4316 35.9104 20.9756C37.773 22.9516 40.6868 20.2735 42.9971 24.2991C42.3654 23.8398 40.7438 23.3366 39.3109 24.998C38.854 23.6274 37.8572 24.8076 37.5457 25.3095C36.7884 26.5295 36.5229 28.4038 37.2082 29.3591C36.5333 30.7349 35.016 30.4493 34.6643 30.1378C33.7557 29.3331 33.7038 25.4393 33.4702 24.1154C33.4442 26.2613 32.9147 30.7245 31.0041 31.4098C28.6159 32.2664 27.4478 31.8251 27.0843 33.0971C26.7936 34.1147 27.5516 34.7498 27.9669 34.9402C27.6468 35.6497 26.5807 37.0117 24.8778 36.7833C22.7492 36.4977 21.6589 35.044 21.2436 34.7585C20.8283 34.4729 19.8937 34.3172 19.0371 34.6027C18.3179 34.8425 17.0805 35.667 17.194 36.7833C17.2785 37.6139 18.5179 37.6918 17.5055 38.1331C16.0456 38.7695 15.1173 37.588 15.2212 37.0169C14.728 37.1467 13.1704 36.965 13.7675 35.3555C14.0605 34.5658 14.6241 33.9538 14.3126 33.2788C13.892 32.3675 12.7641 32.7683 12.0542 32.0588C16.6749 31.5656 17.7651 27.7256 17.7651 27.7256C17.7651 27.7256 16.8654 29.0487 15.9615 29.5007C16.9739 28.4624 17.0642 24.6865 15.1173 24.7903C14.7539 24.9461 14.6241 25.2057 14.5462 25.8287C12.7032 25.6729 13.4076 23.6481 12.5993 22.7136C11.7687 21.7531 8.57572 20.4812 7.84888 22.3522C8.31613 19.0534 13.1645 21.0535 16.5192 18.4845Z" fill="#FAFAFA" stroke="#0A0A0A" stroke-linejoin="round"/>
                    <ellipse cx="23.451" cy="15.017" rx="1.78635" ry="3.18529" transform="rotate(7.04122 23.451 15.017)" fill="#0A0A0A"/>
                    <ellipse cx="1.78635" cy="3.18529" rx="1.78635" ry="3.18529" transform="matrix(-0.992458 0.122583 0.122583 0.992458 29.7302 11.6367)" fill="#0A0A0A"/>
                    <ellipse cx="25.9047" cy="20.9553" rx="1.31286" ry="2.3244" fill="#0A0A0A"/>
                  </svg>
                  <img
                      v-else-if="group.iconUrl"
                      :src="group.iconUrl"
                      class="process-app-icon"
                      :alt="group.processName"
                  />
                  <icon-mdi-application-outline v-else class="process-app-icon-placeholder" />
                </div>
                <div class="process-item-body">
                  <div class="process-name-row">
                    <span class="process-name">{{ group.processName }}</span>
                    <el-tag type="primary" size="small">
                      {{ $t('connections.connections-count', { count: group.count }) }}
                    </el-tag>
                  </div>
                  <div class="process-stats">
                    <span class="ot">{{ $t('connections.download') }}: </span>{{ prettyBytes(group.download) }}
                    &emsp;
                    <span class="ot">{{ $t('connections.upload') }}: </span>{{ prettyBytes(group.upload) }}
                  </div>
                </div>
                <icon-mdi-chevron-right class="process-chevron" />
              </div>
            </el-col>
          </el-row>
          <div v-if="processGroups.length === 0" class="process-empty">
            {{ $t('connections.noData') }}
          </div>
        </div>

        <!-- Selected process connections -->
        <div v-else class="process-connections-wrap">
          <div class="process-back-bar" @click="selectedProcess = null">
            <icon-mdi-arrow-left class="process-back-icon" />
            <span>{{ $t('connections.back') }}</span>
            <span class="process-back-name">— {{ processGroups.find(g => g.processPath === selectedProcess)?.processName }}</span>
          </div>
          <div class="info-list">
            <el-row
                class="info"
                v-for="(item, i) in selectedProcessConnections"
                :key="i"
            >
              <el-col :span="24">
                <div class="info-card">
                  <div class="info-card__actions">
                    <span class="icon-btn" role="button" tabindex="0"
                          :title="$t('connections.view-log')"
                          @click.stop="openLogDialog(item)"
                          @keydown.enter.prevent="openLogDialog(item)"
                          @keydown.space.prevent="openLogDialog(item)">
                      <icon-mdi-information-outline/>
                    </span>
                    <span class="icon-btn" role="button" tabindex="0"
                          :title="$t('connections.copy-log')"
                          @click.stop="copyLog(item)"
                          @keydown.enter.prevent="copyLog(item)"
                          @keydown.space.prevent="copyLog(item)">
                      <icon-mdi-content-copy/>
                    </span>
                  </div>
                  <div class="info-card__body">
                    <div class="info-card__row info-card__row--top">
                      <div class="info-card__host">{{ fHost(item.metadata) }}</div>
                      <div class="info-card__traffic">
                        <span class="info-traffic-item" :title="$t('connections.upload')">
                          <icon-mdi-arrow-up class="traffic-icon traffic-icon--up"/>
                          {{ prettyBytes(item.upload) }}
                        </span>
                        <span class="info-traffic-item" :title="$t('connections.download')">
                          <icon-mdi-arrow-down class="traffic-icon traffic-icon--down"/>
                          {{ prettyBytes(item.download) }}
                        </span>
                      </div>
                    </div>
                    <div class="info-card__row info-card__row--bottom">
                      <div class="info-card__tags">
                        <el-tag type="success" size="small">{{ item.metadata.type }}</el-tag>
                        <el-tag type="danger" size="small">{{ fDate(item.start) }}</el-tag>
                      </div>
                      <div class="info-card__routing" :title="routingText(item)">{{ routingText(item) }}</div>
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>
            <div v-if="selectedProcessConnections.length === 0" class="process-empty">
              {{ $t('connections.noData') }}
            </div>
          </div>
        </div>
      </div>

      <el-dialog
          v-model="logDialogVisible"
          :show-close="false"
          modal-class="log-dialog__overlay"
          class="log-dialog"
      >
        <template #header>
          <div class="log-dialog__header">
            <span class="log-dialog__title">{{ dialogTitle }}</span>
            <div class="log-dialog__actions">
              <el-button
                  class="log-dialog__action log-dialog__copy"
                  circle
                  :title="$t('connections.copy-log')"
                  :aria-label="$t('connections.copy-log')"
                  @click="copyLog()"
              >
                <icon-mdi-content-copy/>
              </el-button>
              <el-button
                  class="log-dialog__action log-dialog__close"
                  circle
                  :title="$t('connections.dialog-close')"
                  :aria-label="$t('connections.dialog-close')"
                  @click="closeLogDialog()"
              >
                <icon-mdi-close/>
              </el-button>
            </div>
          </div>
        </template>
        <div v-if="logItem" class="log-parsed">
          <template v-for="section in dialogSections" :key="section.title">
            <div class="log-section">
              <div class="log-section__title">{{ section.title }}</div>
              <div
                  v-for="row in section.rows"
                  :key="row.label"
                  class="log-row"
              >
                <span class="log-label">{{ row.label }}</span>
                <span v-if="row.type === 'text'" class="log-value">{{ row.value }}</span>
                <el-popover
                    v-else
                    trigger="click"
                    :width="190"
                    placement="bottom-start"
                    popper-class="log-popover"
                >
                  <template #reference>
                    <span class="log-value log-clickable">{{ row.value }}</span>
                  </template>
                  <div class="log-popover__inner">
                    <button class="log-popover__btn" @click="copyText(row.value)">
                      <icon-mdi-content-copy class="log-popover__icon"/>
                      {{ $t('connections.copy-value') }}
                    </button>
                    <button
                        v-if="row.type === 'ip'"
                        class="log-popover__btn"
                        @click="openInBrowser('https://ipinfo.io/' + row.value)"
                    >
                      <icon-mdi-open-in-new class="log-popover__icon"/>
                      ipinfo.io
                    </button>
                    <button
                        v-if="row.type === 'host'"
                        class="log-popover__btn"
                        @click="openInBrowser('https://' + row.value)"
                    >
                      <icon-mdi-open-in-new class="log-popover__icon"/>
                      {{ $t('connections.open-in-browser') }}
                    </button>
                    <button
                        v-if="(row.type === 'process' || row.type === 'path') && row.explorerPath"
                        class="log-popover__btn"
                        @click="showInExplorer(row.explorerPath!)"
                    >
                      <icon-mdi-folder-open-outline class="log-popover__icon"/>
                      {{ $t('connections.show-in-explorer') }}
                    </button>
                  </div>
                </el-popover>
              </div>
            </div>
          </template>
        </div>
      </el-dialog>

    </template>
  </MyLayout>
</template>

<style scoped>
.space {
  margin-top: 20px;
}

:deep(.bottom) {
  padding-bottom: 0;
  overflow-y: hidden;
  display: flex;
  flex-direction: column;
}

.conn {
  width: 100%;
  margin-left: 0;
  margin-top: 2px;
  flex-shrink: 0;
}

.title {
  font-size: 32px;
  font-weight: bold;
  margin-left: 10px;
}

.search {
  width: 100%;
  margin-top: 12px;
}

.search :deep(.custom-input) {
  border-radius: 999px;
  padding-left: 16px;
}

.search :deep(.clear-button) {
  right: 14px;
}

:deep(.el-button) {
  padding: 2px 10px;
  --el-button-bg-color: transparent;
  --el-button-text-color: var(--text-color);
  --el-button-hover-text-color: var(--left-item-selected-bg);
  --el-button-hover-bg-color: var(--text-color)
}

.content {
  border: 2px solid var(--text-color);
  margin-top: 20px;
  border-radius: 20px;
  overflow: hidden;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  box-sizing: border-box;
  /* Полупрозрачная подложка с блюром: текст читается поверх любых обоев,
     но сохраняется «стеклянный» эффект (фон просвечивает размытым). */
  background-color: rgba(0, 0, 0, 0.28);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.info-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.info {
  border-bottom: 1px solid var(--sub-card-border);
  padding: 0;
  background-color: transparent;
}

.info-card {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr);
  column-gap: 10px;
  align-items: center;
  padding: 8px 12px 8px 8px;
  font-size: 14px;
  line-height: 1.45;
}

.info-card__actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.info-card__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.info-card__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.info-card__host {
  flex: 1 1 auto;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.55);
}

.info-card__tags {
  flex: 0 1 auto;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
}

.info-card__tags :deep(.el-tag) {
  border-radius: 999px;
  flex-shrink: 0;
  max-width: 160px;
}

.info-card__traffic {
  flex: 0 0 auto;
  display: flex;
  gap: 10px;
  font-size: 13px;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

.info-traffic-item {
  display: flex;
  align-items: center;
  gap: 3px;
}

.traffic-icon {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
}

.traffic-icon--up {
  color: #f59e0b;
}

.traffic-icon--down {
  color: #10b981;
}

.info-card__routing {
  flex: 1 1 auto;
  min-width: 0;
  font-size: 12px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: right;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-color);
  transition: color 0.2s ease, background-color 0.2s ease;
}

.icon-btn svg {
  width: 18px;
  height: 18px;
  display: block;
}

.icon-btn:hover,
.icon-btn:focus {
  color: var(--left-item-selected-bg);
  background-color: rgba(255, 255, 255, 0.08);
  outline: none;
}

.log-dialog__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.log-dialog__title {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.log-dialog__actions {
  display: flex;
  gap: 8px;
}

.log-dialog__action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  background-color: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.log-dialog__action svg {
  width: 18px;
  height: 18px;
}

.log-dialog__action:hover,
.log-dialog__action:focus-visible {
  background-color: rgba(0, 0, 0, 0.08);
  color: var(--left-item-selected-bg);
}

.log-dialog__action:focus-visible {
  outline: 2px solid var(--left-item-selected-bg);
  outline-offset: 2px;
}

.log-parsed {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0 8px;
}

.log-parsed::-webkit-scrollbar {
  width: 5px;
}

.log-parsed::-webkit-scrollbar-track {
  background: transparent;
}

.log-parsed::-webkit-scrollbar-thumb {
  background: var(--scrollbar-bg);
  border-radius: 2px;
}

.log-parsed::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-hover-bg);
}

.log-section {
  margin-bottom: 4px;
}

.log-section__title {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--left-item-selected-bg);
  padding: 6px 16px 2px;
}

.log-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 2px 16px;
  font-size: 13px;
  line-height: 1.4;
}

.log-label {
  flex: 0 0 42%;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  word-break: break-word;
}

.log-value {
  flex: 1;
  color: var(--el-text-color-primary);
  word-break: break-all;
}

.log-clickable {
  cursor: pointer;
  text-decoration: underline dotted;
  text-underline-offset: 3px;
  color: var(--left-item-selected-bg);
  transition: opacity 0.15s ease;
}

.log-clickable:hover {
  opacity: 0.75;
}

:deep(.log-dialog__overlay) {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  box-sizing: border-box;
  overflow: hidden;
}

:deep(.log-dialog__overlay .el-overlay-dialog) {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 !important;
}

:deep(.el-dialog.log-dialog) {
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  width: min(600px, calc(100vw - 32px));
  max-width: 100%;
  max-height: min(720px, calc(100vh - 32px));
  height: auto;
  margin: 0 !important;
  overflow: hidden;
}

:deep(.el-dialog.log-dialog .el-dialog__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}


.info-list::-webkit-scrollbar {
  width: 5px;
  padding-bottom: 20px;
}

.info-list::-webkit-scrollbar-track {
  background: transparent;
}

.info-list::-webkit-scrollbar-thumb {
  background: var(--scrollbar-bg);
  border-radius: 2px;
  transition: background 0.3s ease, box-shadow 0.3s ease;
}

.info-list::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-hover-bg);
  box-shadow: var(--scrollbar-hover-shadow);
}

.topology-content {
  flex: 1;
  min-height: 0;
  height: auto;
  border: none;
  background: transparent;
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.view-mode-switch .el-segmented {
  min-width: 150px;
  border: 1px solid var(--sub-card-border);
  background: var(--left-proxy-bg);
  box-shadow: var(--left-nav-shadow);
  --el-segmented-item-selected-color: var(--text-color);
  --el-segmented-item-selected-bg-color: var(--left-item-selected-bg);
  --el-border-radius-base: 5px;
  color: var(--text-color);
  font-size: 14px;
}

.view-mode-switch .el-segmented:hover {
  box-shadow: var(--left-nav-hover-shadow);
}

.view-mode-switch :deep(.el-segmented__item) {
  padding: 0 12px;
}

/* ── Process view ─────────────────────────────────────────────────── */

.process-row {
  cursor: pointer;
  transition: background-color 0.15s ease;
  padding: 5px 10px 5px 16px;
}

.process-row:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.process-item-inner {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 2px 0;
}

.process-icon-wrap {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.process-app-icon {
  width: 36px;
  height: 36px;
  object-fit: contain;
  border-radius: 8px;
}

.process-app-icon-placeholder {
  width: 32px;
  height: 32px;
  color: var(--text-color);
  opacity: 0.45;
}

.process-item-body {
  flex: 1;
  min-width: 0;
}

.process-name-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}

.process-name {
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.55);
}

.process-stats {
  font-size: 14px;
  opacity: 0.85;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

.process-chevron {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  opacity: 0.4;
}

.process-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 80px;
  opacity: 0.5;
  font-size: 14px;
}

.process-connections-wrap {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.process-back-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--sub-card-border);
  font-size: 14px;
  font-weight: 600;
  background-color: var(--left-bg-color);
  transition: background-color 0.15s ease;
  flex-shrink: 0;
}

.process-back-bar:hover {
  background-color: rgba(255, 255, 255, 0.06);
}

.process-back-icon {
  width: 18px;
  height: 18px;
}

.process-back-name {
  opacity: 0.6;
  font-weight: 400;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.process-connections-wrap .info-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

</style>