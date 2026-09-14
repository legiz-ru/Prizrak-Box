<script lang="ts" setup>
import {useRouter} from "vue-router";
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {useProxiesStore} from "@/store/proxiesStore";
import {useSettingStore} from "@/store/settingStore";
import {changeProxyAndCloseConnections} from "@/util/proxy";
import {pError, pWarning} from "@/util/pLoad";
import {Events} from "@/runtime";
import type {ProxyGroupInfo} from "@/api/proxies";
import {useWebStore} from "@/store/webStore";

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);
const {t} = useI18n();

// 当前组件使用store
const proxiesStore = useProxiesStore();
const settingStore = useSettingStore();
const webStore = useWebStore();

// Group and Proxy dropdowns
const groupList = ref<ProxyGroupInfo[]>([]);
const proxyList = ref<any[]>([]);
// Pre-initialize from persisted store so the panel shows the correct value
// immediately on mount, before any API call completes.
const selectedGroup = ref(proxiesStore.active || '');
const selectedProxy = ref(proxiesStore.now || '');
const isGroupDropdownOpen = ref(false);
const isProxyDropdownOpen = ref(false);

const router = useRouter()

const isWindows = ref(false)

// Load groups
async function loadGroups() {
  if (!webStore.fProfile || !webStore.fProfile['id']) {
    groupList.value = [];
    selectedGroup.value = '';
    proxiesStore.setActive('');
    return;
  }
  try {
    const groups = await api.getGroups();
    groupList.value = groups;

    // Set initial selected group from store
    if (proxiesStore.active && groups.some(g => g.name === proxiesStore.active)) {
      selectedGroup.value = proxiesStore.active;
    } else if (groups.length > 0) {
      selectedGroup.value = groups[0].name;
      proxiesStore.setActive(groups[0].name);
    }
  } catch (error) {
    console.error('Failed to load groups:', error);
  }
}

// Load proxies for selected group
async function loadProxies() {
  if (!webStore.fProfile || !webStore.fProfile['id']) {
    proxyList.value = [];
    selectedProxy.value = '';
    proxiesStore.setNow('');
    return;
  }
  if (!selectedGroup.value) {
    proxyList.value = [];
    return;
  }

  try {
    // isHide: false - показывать все прокси (даже без пинга)
    // isSort: false - сохранять оригинальный порядок из API
    const proxies = await api.getProxies(selectedGroup.value, false, false);
    proxyList.value = proxies;

    // Set current selected proxy
    const currentProxy = proxies.find((p: any) => p.now);
    if (currentProxy) {
      selectedProxy.value = currentProxy.name;
      proxiesStore.setNow(currentProxy.name);
    }
  } catch (error) {
    console.error('Failed to load proxies:', error);
  }
}

// Get server description for info tooltip
function getServerDescription(proxy: any): string | undefined {
  return proxy?.displayType !== proxy?.type ? proxy?.displayType : undefined;
}

// Get latency color class
function getLatencyColor(toClass: string): string {
  if (toClass === 'toLow') return 'latency-low';
  if (toClass === 'toMiddle') return 'latency-medium';
  if (toClass === 'toHigh') return 'latency-high';
  return 'latency-hidden';
}

// Handle group selection
async function selectGroup(group: ProxyGroupInfo) {
  selectedGroup.value = group.name;
  proxiesStore.setActive(group.name);
  isGroupDropdownOpen.value = false;
  await loadProxies();
}

// Handle proxy selection
async function selectProxy(proxy: any) {
  if (proxy.now) {
    isProxyDropdownOpen.value = false;
    return;
  }

  const group = groupList.value.find(g => g.name === selectedGroup.value);
  if (group?.type !== 'Selector') {
    pWarning(t('proxies.auto-group-no-manual-select'));
    isProxyDropdownOpen.value = false;
    return;
  }

  try {
    await changeProxyAndCloseConnections(
        api,
        selectedGroup.value,
        proxy.name,
    );
    selectedProxy.value = proxy.name;
    proxiesStore.setNow(proxy.name);
    isProxyDropdownOpen.value = false;

    // Reload proxies to update 'now' status
    await loadProxies();
  } catch (error) {
    if (error && typeof error === 'object' && 'message' in error) {
      const message = (error as {message?: unknown}).message;
      if (typeof message === 'string') {
        pError(message);
      } else {
        console.error(error);
      }
    } else {
      console.error(error);
    }
  }
}

// Тихий тест задержек без индикатора загрузки
async function runDelayTestSilent() {
  const group = selectedGroup.value || proxiesStore.active;
  if (!group) return;
  try {
    await api.getDelay(group, settingStore.testUrl, 3000);
    await loadProxies();
  } catch (_) {
    // silently ignore
  }
}

// Toggle dropdowns
async function toggleGroupDropdown() {
  const nextOpen = !isGroupDropdownOpen.value;
  if (nextOpen) {
    await loadGroups();
    isProxyDropdownOpen.value = false;
    runDelayTestSilent(); // fire-and-forget: auto-test when group dropdown opens
  }
  isGroupDropdownOpen.value = nextOpen;
}

async function toggleProxyDropdown() {
  const nextOpen = !isProxyDropdownOpen.value;
  if (nextOpen) {
    await loadProxies();
    isGroupDropdownOpen.value = false;
    runDelayTestSilent(); // fire-and-forget: auto-test when proxy dropdown opens
  }
  isProxyDropdownOpen.value = nextOpen;
}

// Close dropdowns when clicking outside
function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement;
  if (!target.closest('.header-content')) {
    isGroupDropdownOpen.value = false;
    isProxyDropdownOpen.value = false;
  }
}

const handleProfileChanged = async () => {
  proxiesStore.setActive('');
  proxiesStore.setNow('');
  isGroupDropdownOpen.value = false;
  isProxyDropdownOpen.value = false;
  await loadGroups();
  await loadProxies();
};

onMounted(async () => {
  // @ts-ignore
  if (window["pxShowBar"]) {
    isWindows.value = true;
  }

  // Load groups and proxies
  await loadGroups();
  await loadProxies();

  // Add click outside handler
  document.addEventListener('click', handleClickOutside);
  window.addEventListener('profile-changed', handleProfileChanged as EventListener);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
  window.removeEventListener('profile-changed', handleProfileChanged as EventListener);
});

// Listen to proxy change events from other sources (menu, tray, etc.)
Events.On("proxyChanged", async (data: any) => {
  if (data?.group) {
    selectedGroup.value = data.group;
    proxiesStore.setActive(data.group);
  }
  if (data?.proxy) {
    selectedProxy.value = data.proxy;
    proxiesStore.setNow(data.proxy);
  }
  await loadProxies();
});

// Listen to profile change events
Events.On("profileChanged", async () => {
  await handleProfileChanged();
});

// Watch for store changes from other components (e.g. Proxies.vue).
// Only update local refs — no API calls, no clearing proxyList.
// proxyList is refreshed on next dropdown open or on a backend proxyChanged event.
watch(() => proxiesStore.active, (newActive) => {
  if (newActive && newActive !== selectedGroup.value) {
    selectedGroup.value = newActive;
    // Do NOT clear proxyList here — that would flash "Не выбрано".
  }
});

watch(() => proxiesStore.now, (newNow) => {
  // Ignore empty resets — they happen transiently during profile/group changes.
  if (!newNow || newNow === selectedProxy.value) return;
  selectedProxy.value = newNow;
  // Optimistically mark the active proxy in the cached list without an API call.
  if (proxyList.value.length > 0) {
    proxyList.value = proxyList.value.map(p => ({ ...p, now: p.name === newNow }));
  }
});

</script>

<template>
  <div :class="isWindows?'search-container win':'search-container'">
    <div class="header-content no-drag">
      <div class="proxy-selector">
        <!-- Group Dropdown -->
        <div class="dropdown-wrapper">
          <button
              type="button"
              class="dropdown-button"
              :class="{ 'is-open': isGroupDropdownOpen }"
              @click="toggleGroupDropdown"
          >
            <span class="dropdown-text">
              <span class="dropdown-label-text">{{ t('proxySelector.group') }}</span>
              <span class="dropdown-value">{{ selectedGroup || '—' }}</span>
            </span>
            <n-icon class="dropdown-icon">
              <icon-tabler-chevron-down v-if="!isGroupDropdownOpen" />
              <icon-tabler-chevron-up v-else />
            </n-icon>
          </button>
          <div v-if="isGroupDropdownOpen" class="dropdown-list">
            <div
                v-for="group in groupList"
                :key="group.name"
                class="dropdown-item"
                :class="{ 'dropdown-item-selected': group.name === selectedGroup }"
                @click="selectGroup(group)"
            >
              <span class="dropdown-item-text">{{ group.name }}</span>
              <icon-tabler-check v-if="group.name === selectedGroup" class="dropdown-item-check"/>
            </div>
          </div>
        </div>

        <!-- Proxy Dropdown -->
        <div class="dropdown-wrapper">
          <button
              type="button"
              class="dropdown-button"
              :class="{ 'is-open': isProxyDropdownOpen }"
              @click="toggleProxyDropdown"
          >
            <span class="dropdown-text">
              <span class="dropdown-label-text">{{ t('proxySelector.proxy') }}</span>
              <span class="dropdown-value">
                {{ (proxyList.find(p => p.now)?.displayName ?? proxyList.find(p => p.now)?.name ?? selectedProxy) || 'Не выбрано' }}
              </span>
            </span>
            <n-icon class="dropdown-icon">
              <icon-tabler-chevron-down v-if="!isProxyDropdownOpen" />
              <icon-tabler-chevron-up v-else />
            </n-icon>
          </button>
          <div v-if="isProxyDropdownOpen" class="dropdown-list">
            <div
                v-for="proxyItem in proxyList"
                :key="proxyItem.name"
                class="dropdown-item proxy-item"
                :class="{ 'dropdown-item-selected': proxyItem.now }"
                @click="selectProxy(proxyItem)"
            >
              <span :class="['latency-dot', getLatencyColor(proxyItem.toClass)]"></span>
              <span class="proxy-item-name">{{ proxyItem.displayName ?? proxyItem.name }}</span>
              <n-tooltip
                  v-if="getServerDescription(proxyItem)"
                  trigger="hover"
                  placement="top"
              >
                <template #trigger>
                  <n-icon class="proxy-info-icon">
                    <icon-tabler-info-circle />
                  </n-icon>
                </template>
                {{ getServerDescription(proxyItem) }}
              </n-tooltip>
              <icon-tabler-check v-if="proxyItem.now" class="dropdown-item-check"/>
            </div>
          </div>
        </div>
      </div>

      <MyTitleBar :class="isWindows?'minus-win':'minus'"></MyTitleBar>
    </div>
  </div>
</template>

<style scoped>
.search-container {
  padding-top: 25px;
  position: relative;
  -webkit-app-region: drag; /* Electron */
  --wails-draggable: drag;  /* Wails (frameless on Windows/Linux) */
}

.win {
  padding-top: 15px;
}

.no-drag {
  -webkit-app-region: no-drag;
  --wails-draggable: no-drag;
}

/* Header Content - одна линия */
.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

/* Proxy Selector Container */
.proxy-selector {
  display: flex;
  align-items: center;
  gap: var(--px-space-3);
  margin-left: var(--px-space-2);
}

/* Dropdown Wrapper */
.dropdown-wrapper {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 200px;
}

/* Dropdown Button — сплошная плашка-контрол в языке остального приложения
   (тот же --left-nav-btn-bg/--sub-card-border, что у кнопок и полей), а не
   подпись, врезанная в разрыв рамки. */
.dropdown-button {
  width: 100%;
  min-width: 0;
  padding: var(--px-space-2) var(--px-space-3);
  border: 1px solid var(--sub-card-border);
  border-radius: var(--px-r-lg);
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--px-space-2);
  transition: background-color var(--px-dur) var(--px-ease),
  border-color var(--px-dur) var(--px-ease);
  font-family: 'Twemoji', 'Nunito', 'Microsoft YaHei', '微软雅黑', sans-serif;
  font-variant-emoji: emoji;
  box-sizing: border-box;
}

.dropdown-button:hover,
.dropdown-button.is-open {
  background-color: var(--left-nav-btn-hover-bg);
  border-color: var(--text-color);
}

/* Подпись роли ("Group"/"Proxy") над текущим значением. */
.dropdown-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.dropdown-label-text {
  font-size: var(--px-fs-caption);
  color: var(--text-color);
  opacity: .6;
  white-space: nowrap;
}

.dropdown-value {
  font-size: var(--px-fs-small);
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dropdown-icon {
  font-size: var(--px-fs-body);
  flex-shrink: 0;
  opacity: .7;
}

.dropdown-button.is-open .dropdown-icon {
  opacity: 1;
}

/* Dropdown List */
.dropdown-list {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  width: 100%;
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid var(--dropdown-border-color);
  border-radius: var(--px-r-lg);
  background-color: var(--dropdown-list-bg);
  box-shadow: var(--skin-box-shadow);
  z-index: 9999;
  font-family: 'Twemoji', 'Nunito', 'Microsoft YaHei', '微软雅黑', sans-serif;
  font-variant-emoji: emoji;
  box-sizing: border-box;
}

/* Тонкий скроллбар для индикации возможности прокрутки */
.dropdown-list::-webkit-scrollbar {
  width: 4px;
}

.dropdown-list::-webkit-scrollbar-track {
  background: transparent;
}

.dropdown-list::-webkit-scrollbar-thumb {
  background: var(--scrollbar-bg);
  border-radius: 2px;
}

.dropdown-list::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-hover-bg);
}

/* Firefox */
.dropdown-list {
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar-bg) transparent;
}

/* Dropdown Item */
.dropdown-item {
  padding: var(--px-space-2) var(--px-space-3);
  cursor: pointer;
  color: var(--text-color);
  font-size: var(--px-fs-small);
  transition: background-color var(--px-dur) var(--px-ease);
  display: flex;
  align-items: center;
  gap: var(--px-space-2);
  justify-content: space-between;
}

.dropdown-item:hover {
  background-color: var(--left-nav-btn-hover-bg);
}

.dropdown-item-selected {
  background-color: var(--left-item-selected-bg);
}

.dropdown-item-text {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Отмеченный элемент — та же галочка, что помечает выбранный узел в
   разделе Прокси: состояние не только цветом заливки строки. */
.dropdown-item-check {
  flex-shrink: 0;
  font-size: 15px;
  color: var(--text-color);
}

/* Proxy Item */
.proxy-item {
  gap: var(--px-space-2);
}

.proxy-item-name {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.proxy-info-icon {
  font-size: var(--px-fs-body);
  color: var(--text-color);
  opacity: 0.6;
  flex-shrink: 0;
}

.proxy-info-icon:hover {
  opacity: 1;
}

/* Latency Dot — тот же смысл, что у .toLow/.toMiddle/.toHigh в global.css,
   на общих статусных токенах, а не на своём отдельном наборе цветов. */
.latency-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.latency-low {
  background-color: var(--px-ok);
}

.latency-medium {
  background-color: var(--px-warn);
}

.latency-high {
  background-color: var(--px-danger);
}

.latency-hidden {
  background-color: var(--px-text-faint);
}

/* Title Bar */
.minus {
  margin-right: 25px;
  margin-left: auto;
  float: right;
  font-size: 18px;
  color: var(--text-color);
  cursor: pointer;
  -webkit-app-region: no-drag;
  --wails-draggable: no-drag;
}

.minus-win {
  margin-right: 12px;
  margin-left: auto;
  float: right;
  font-size: 20px;
  color: var(--text-color);
  cursor: pointer;
  -webkit-app-region: no-drag;
  --wails-draggable: no-drag;
}
</style>
