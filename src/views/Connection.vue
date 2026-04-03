<script setup lang="ts">
import {computed, getCurrentInstance, onBeforeUnmount, onMounted, ref, watch} from "vue";
import MySimpleInput from "@/components/MySimpleInput.vue";
import ConnectionTopology from "@/components/topology/ConnectionTopology.vue";
import {WS} from "@/util/ws";
import {useWebStore} from "@/store/webStore";
import {useConnectionStore} from "@/store/connectionStore";
import {prettyBytes, rJoin} from "@/util/format";
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
const logContent = ref('')

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

// 分页数据状态
const paginatedData = ref([]);

function onConn(ev: MessageEvent) {
  const parsedData = JSON.parse(ev.data);
  paginatedData.value = parsedData['connections']
}

function openLogDialog(item: any) {
  logContent.value = JSON.stringify(item, null, 2)
  logDialogVisible.value = true
}

function closeLogDialog() {
  logDialogVisible.value = false
}

async function copyLog(item?: any) {
  const data = item ? JSON.stringify(item, null, 2) : logContent.value
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
}

const processGroups = computed<ProcessGroup[]>(() => {
  const groups = new Map<string, ProcessGroup>()
  for (const conn of (paginatedData.value || [])) {
    const name: string = (conn as any).metadata?.process || t('connections.unknown-process')
    const path: string = (conn as any).metadata?.processPath || name
    if (!groups.has(path)) {
      groups.set(path, { processName: name, processPath: path, count: 0, download: 0, upload: 0, iconUrl: null })
      if (path !== name) loadIcon(path)
    }
    const g = groups.get(path)!
    g.count++
    g.download += (conn as any).download ?? 0
    g.upload += (conn as any).upload ?? 0
    g.iconUrl = iconCache.value[path] ?? null
  }
  return [...groups.values()].sort((a, b) => b.count - a.count)
})

const selectedProcessConnections = computed(() => {
  if (!selectedProcess.value) return []
  return filterData(paginatedData.value)?.filter((c: any) => {
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
})


function closeAll() {
  const data = filterData(paginatedData.value)
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
          <div class="search" v-if="connectionStore.viewMode === 'list' || connectionStore.viewMode === 'process'">
            <MySimpleInput
                :onInputChange="handleInputChange"
                :placeholder="$t('connections.search')"
                class="search"
            ></MySimpleInput>
          </div>
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
      </div>

      <div class="content" v-if="connectionStore.viewMode === 'list'">
        <div class="info-list">
          <el-row
              class="info"
              v-for="(item, i) in filterData(paginatedData)"
              :key="i"
          >
            <el-col :span="24">
              <div class="info-header">
                <div class="info-actions">
                  <span
                      class="icon-btn"
                      role="button"
                      tabindex="0"
                      :title="$t('connections.view-log')"
                      @click="openLogDialog(item)"
                      @keydown.enter.prevent="openLogDialog(item)"
                      @keydown.space.prevent="openLogDialog(item)"
                  >
                    <icon-mdi-information-outline/>
                  </span>
                  <span
                      class="icon-btn"
                      role="button"
                      tabindex="0"
                      :title="$t('connections.copy-log')"
                      @click="copyLog(item)"
                      @keydown.enter.prevent="copyLog(item)"
                      @keydown.space.prevent="copyLog(item)"
                  >
                    <icon-mdi-content-copy/>
                  </span>
                </div>
                <div class="info-tags">
                  <el-tag type="success" size="small">{{ item.metadata.type }}</el-tag>
                  &emsp;
                  <el-tag type="danger" size="small">
                    {{ fDate(item.start) }}
                  </el-tag>
                  <template v-if="item.metadata.process">
                    &emsp;
                    <el-tag type="primary" size="small">{{ item.metadata.process }}</el-tag>
                  </template>
                </div>
              </div>
              <div class="od">
                <span class="ot">{{ $t('connections.host') }} : </span>
                {{ fHost(item.metadata) }}
              </div>
              <div class="od">
                <span class="ot">{{ $t('connections.download') }} : </span>
                {{ prettyBytes(item.download) }}
                &emsp;
                <span class="ot">{{ $t('connections.upload') }} : </span>
                {{ prettyBytes(item.upload) }}
              </div>
              <div class="od" v-if="item.rule">
                <span class="ot">{{ $t('connections.rule') }} : </span>
                {{ item.rule }}
                {{ item.rulePayload ? ' / ' + item.rulePayload : '' }}
              </div>
              <div class="od">
                <span class="ot">{{ $t('connections.chains') }} : </span>
                {{ rJoin(item.chains, '&nbsp;/&nbsp;') }}
              </div>
            </el-col>
          </el-row>
        </div>
      </div>

      <div class="content topology-content" v-else-if="connectionStore.viewMode === 'topology'">
        <ConnectionTopology :connections="paginatedData" />
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
                  <img
                      v-if="group.iconUrl"
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
                <div class="info-header">
                  <div class="info-actions">
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
                  <div class="info-tags">
                    <el-tag type="success" size="small">{{ item.metadata.type }}</el-tag>
                    &emsp;
                    <el-tag type="danger" size="small">{{ fDate(item.start) }}</el-tag>
                  </div>
                </div>
                <div class="od"><span class="ot">{{ $t('connections.host') }}: </span>{{ fHost(item.metadata) }}</div>
                <div class="od">
                  <span class="ot">{{ $t('connections.download') }}: </span>{{ prettyBytes(item.download) }}
                  &emsp;
                  <span class="ot">{{ $t('connections.upload') }}: </span>{{ prettyBytes(item.upload) }}
                </div>
                <div class="od" v-if="item.rule">
                  <span class="ot">{{ $t('connections.rule') }}: </span>{{ item.rule }}{{ item.rulePayload ? ' / ' + item.rulePayload : '' }}
                </div>
                <div class="od">
                  <span class="ot">{{ $t('connections.chains') }}: </span>{{ rJoin(item.chains, '&nbsp;/&nbsp;') }}
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
          :title="$t('connections.dialog-title')"
          :show-close="false"
          modal-class="log-dialog__overlay"
          class="log-dialog"
      >
        <template #header>
          <div class="log-dialog__header">
            <span class="log-dialog__title">{{ $t('connections.dialog-title') }}</span>
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
        <pre class="log-dialog__content">{{ logContent }}</pre>
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
}

.title {
  font-size: 32px;
  font-weight: bold;
  margin-left: 10px;
}

.search {
  width: 400px;
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
}

.info-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.info {
  border-bottom: 1px solid var(--sub-card-border);
  padding: 5px 10px 5px 16px;
  font-size: 15px;
  line-height: 1.6;
  background-color: var(--left-bg-color);
}

.info-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.info-actions {
  display: flex;
  gap: 8px;
}

.info-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  --el-tag-border-radius: 999px;
}

.info-tags :deep(.el-tag) {
  border-radius: 999px;
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

.log-dialog__content {
  flex: 1;
  overflow: auto;
  background-color: rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  padding: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  width: 100%;
  box-sizing: border-box;
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

.od {
  user-select: text;
}

.ot {
  font-weight: bold;
  font-size: 15px;
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
}

.process-stats {
  font-size: 14px;
  opacity: 0.75;
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