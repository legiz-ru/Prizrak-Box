<script lang="ts" setup>
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {useProxiesStore} from "@/store/proxiesStore";
import {useSettingStore} from "@/store/settingStore";
import {changeProxyAndCloseConnections} from "@/util/proxy";
import {pError, pWarning} from "@/util/pLoad";
import {Events} from "@/runtime";
import type {ProxyGroupInfo} from "@/api/proxies";
import {useWebStore} from "@/store/webStore";
import {proxyTypeIcon, proxyTypeTooltip} from "@/util/proxyType";
import {withFlag} from "@/util/flag";
import {delayColor, delayLabel} from "@/util/delay";
import UiDropdown from "@/components/ui/UiDropdown.vue";

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
const groupDd = ref<InstanceType<typeof UiDropdown> | null>(null);
const proxyDd = ref<InstanceType<typeof UiDropdown> | null>(null);

// Load groups
async function loadGroups() {
  if (!webStore.fProfile || !(webStore.fProfile as any)['id']) {
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
  if (!webStore.fProfile || !(webStore.fProfile as any)['id']) {
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

// Type badge. The search list never showed the adapter type at all; the icon
// adds it without spending horizontal space in an already narrow dropdown.
// A node named by a known group is a group — groupList holds every group.
const isGroupProxy = (p: any) => groupList.value.some((g) => g.name === p?.name);
const typeIcon = (p: any) => proxyTypeIcon(p?.type, isGroupProxy(p));
const typeTooltip = (p: any) => proxyTypeTooltip(p?.type, isGroupProxy(p), t);

// Handle group selection
async function selectGroup(group: ProxyGroupInfo) {
  selectedGroup.value = group.name;
  proxiesStore.setActive(group.name);
  groupDd.value?.close();
  await loadProxies();
}

// Handle proxy selection
async function selectProxy(proxy: any) {
  if (proxy.now) {
    proxyDd.value?.close();
    return;
  }

  const group = groupList.value.find(g => g.name === selectedGroup.value);
  if (group?.type !== 'Selector') {
    pWarning(t('proxies.auto-group-no-manual-select'));
    proxyDd.value?.close();
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
    proxyDd.value?.close();

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

// Opening a list refreshes it and fires a silent delay test (as in dev_21).
async function onGroupOpen() {
  await loadGroups();
  runDelayTestSilent();
}

async function onProxyOpen() {
  await loadProxies();
  runDelayTestSilent();
}

const currentGroup = computed(() => groupList.value.find(g => g.name === selectedGroup.value));
const currentProxy = computed(() => proxyList.value.find(p => p.now));
const currentProxyName = computed(() =>
    withFlag((currentProxy.value?.displayName ?? currentProxy.value?.name ?? selectedProxy.value) || '') || t('ui.not-selected'));

const handleProfileChanged = async () => {
  proxiesStore.setActive('');
  proxiesStore.setNow('');
  groupDd.value?.close();
  proxyDd.value?.close();
  await loadGroups();
  await loadProxies();
};

onMounted(async () => {
  // Load groups and proxies
  await loadGroups();
  await loadProxies();

  window.addEventListener('profile-changed', handleProfileChanged as EventListener);
});

onUnmounted(() => {
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
  <div class="search-pill no-drag">
    <UiDropdown ref="groupDd" :min-width="230" :max-width="320" @open="onGroupOpen">
      <template #trigger="{ open, toggle, attrs }">
        <button type="button"
                v-bind="attrs"
                class="search-seg"
                :class="{ 'is-open': open }"
                :aria-label="t('proxySelector.group') + ': ' + selectedGroup"
                @click="toggle">
          <img v-if="currentGroup?.icon" class="search-seg__img" :src="currentGroup.icon" alt="">
          <span v-else class="search-seg__type" v-tip="typeTooltip({type: currentGroup?.type, name: selectedGroup})">
            <component :is="proxyTypeIcon(currentGroup?.type, true)" width="15" height="15"/>
          </span>
          <span class="search-seg__label">{{ t('proxySelector.group') }}</span>
          <span class="search-seg__value ellipsis" v-tip="withFlag(selectedGroup)">{{ withFlag(selectedGroup) || '—' }}</span>
          <icon-tabler-chevron-down class="search-seg__chev" :class="{ 'is-open': open }" width="14" height="14"/>
        </button>
      </template>
      <button v-for="group in groupList"
              :key="group.name"
              type="button"
              role="option"
              data-dd-item
              class="px-dd-item"
              :class="{ 'is-selected': group.name === selectedGroup }"
              :aria-selected="group.name === selectedGroup ? 'true' : 'false'"
              @click="selectGroup(group)">
        <img v-if="group.icon" class="search-seg__img" :src="group.icon" alt="">
        <span v-else class="search-seg__type" v-tip="typeTooltip(group)">
          <component :is="proxyTypeIcon(group.type, true)" width="15" height="15"/>
        </span>
        <span class="ellipsis" style="flex:1" v-tip="withFlag(group.name)">{{ withFlag(group.name) }}</span>
        <icon-tabler-check v-if="group.name === selectedGroup" width="14" height="14" class="search-check"/>
      </button>
    </UiDropdown>

    <span class="search-sep"></span>

    <UiDropdown ref="proxyDd" :min-width="260" :max-width="340" @open="onProxyOpen">
      <template #trigger="{ open, toggle, attrs }">
        <button type="button"
                v-bind="attrs"
                class="search-seg"
                :class="{ 'is-open': open }"
                :aria-label="t('proxySelector.proxy') + ': ' + currentProxyName"
                @click="toggle">
          <icon-tabler-world class="search-seg__type" width="15" height="15"/>
          <span class="search-seg__label">{{ t('proxySelector.proxy') }}</span>
          <span class="search-seg__value ellipsis" v-tip="currentProxyName">{{ currentProxyName }}</span>
          <span class="search-dot" :style="{ background: delayColor(currentProxy?.delay) }"></span>
          <icon-tabler-chevron-down class="search-seg__chev" :class="{ 'is-open': open }" width="14" height="14"/>
        </button>
      </template>
      <button v-for="proxyItem in proxyList"
              :key="proxyItem.name"
              type="button"
              role="option"
              data-dd-item
              class="px-dd-item"
              :class="{ 'is-selected': proxyItem.now }"
              :aria-selected="proxyItem.now ? 'true' : 'false'"
              @click="selectProxy(proxyItem)">
        <span class="search-dot" :style="{ background: delayColor(proxyItem.delay) }"></span>
        <span class="ellipsis" style="flex:1" v-tip="withFlag(proxyItem.displayName ?? proxyItem.name)">{{ withFlag(proxyItem.displayName ?? proxyItem.name) }}</span>
        <span class="search-delay tabular" :style="{ color: delayColor(proxyItem.delay) }">{{ delayLabel(proxyItem.delay) }}</span>
      </button>
    </UiDropdown>
  </div>
</template>

<style scoped>
.search-pill {
  display: flex;
  align-items: center;
  height: 38px;
  padding: 3px;
  gap: 2px;
  border-radius: 999px;
  background: var(--side-bg);
  backdrop-filter: var(--side-blur);
  border: 1px solid var(--border);
  min-width: 0;
  max-width: 560px;
}

.search-pill > :deep(.px-dd) {
  flex: 0 1 auto;
}

.search-seg {
  display: flex;
  align-items: center;
  gap: 7px;
  height: 30px;
  padding: 0 10px 0 12px;
  border-radius: 999px;
  border: none;
  cursor: pointer;
  min-width: 0;
  max-width: 250px;
  color: var(--text);
  background: transparent;
}

.search-seg:hover, .search-seg.is-open {
  background: var(--hover-bg);
}

.search-seg__img {
  width: 18px;
  height: 18px;
  object-fit: contain;
  flex-shrink: 0;
}

.search-seg__type {
  display: flex;
  color: var(--text-2);
  flex-shrink: 0;
}

.search-seg__label {
  font-size: 12px;
  color: var(--text-2);
  flex-shrink: 0;
}

.search-seg__value {
  font-size: 13px;
  font-weight: 600;
}

.search-seg__chev {
  flex-shrink: 0;
  opacity: .7;
  transition: transform .15s;
}

.search-seg__chev.is-open {
  transform: rotate(180deg);
}

.search-sep {
  width: 1px;
  height: 18px;
  background: var(--border);
  flex-shrink: 0;
}

.search-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.search-delay {
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.search-check {
  flex-shrink: 0;
  color: var(--accent);
}
</style>
