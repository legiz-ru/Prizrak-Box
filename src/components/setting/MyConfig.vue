<script setup lang="ts">

const props = defineProps({
  section: { type: String, default: 'all' }
})

import MyPort from "@/components/setting/MyPort.vue";
import MyBind from "@/components/setting/MyBind.vue";
import MyTun from "@/components/setting/MyTun.vue";
import MyService from "@/components/setting/MyService.vue";
import MyHotkeyInput from "@/components/setting/MyHotkeyInput.vue";
import MyAgeKeypair from "@/components/setting/MyAgeKeypair.vue";
import {useWebStore} from "@/store/webStore";
import {useHomeStore} from "@/store/homeStore";
import {copy, pError, pLoad, pSuccess, pWarning} from "@/util/pLoad";
import {Profile} from "@/types/profile";
import {useI18n} from "vue-i18n";
import {useSettingStore} from "@/store/settingStore";
import createApi from "@/api";
import {changeMenu} from "@/util/menu";
import {useRouter} from "vue-router";
import {pUpdateMihomo} from "@/util/mihomo";
import {useMenuStore} from "@/store/menuStore";
import {Browser, Events} from "@/runtime";
import {useUpdateStore} from "@/store/updateStore";
import {storeToRefs} from "pinia";
import type {DashboardOption} from "@/util/dashboard";
import {formatDashboardUrl as buildDashboardUrl, resolveDashboardOptions} from "@/util/dashboard";
import {updateSystemProxy} from "@/util/systemProxy";
import {confirm} from "@/components/ui";
import UiDropdown from "@/components/ui/UiDropdown.vue";

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 使用 store
const webStore = useWebStore()
const homeStore = useHomeStore()
const menuStore = useMenuStore()
const settingStore = useSettingStore()
const {t} = useI18n()
const updateStore = useUpdateStore()

const {checking: updateChecking, lastCheckStatus, lastError, latestDisplayName, updateAvailable} = storeToRefs(updateStore)
const {customDashboards} = storeToRefs(webStore)

const manualUpdateStatus = computed(() => {
  if (updateChecking.value) {
    return {type: 'info', text: t('updates.status.checking')};
  }

  if (updateAvailable.value) {
    const label = latestDisplayName.value || t('updates.banner.version-unknown');
    return {type: 'warning', text: t('updates.status.available', {version: label})};
  }

  if (lastCheckStatus.value === 'up-to-date') {
    return {type: 'success', text: t('updates.status.up-to-date')};
  }

  if (lastCheckStatus.value === 'error') {
    const errorKey = lastError.value ? 'updates.status.error-details' : 'updates.status.error';
    return {type: 'danger', text: t(errorKey, {message: lastError.value ?? ''})};
  }

  return {type: '', text: ''};
});

const dashboardOptions = computed<DashboardOption[]>(() =>
  resolveDashboardOptions(customDashboards.value),
);

const dashboardDialogVisible = ref(false);
const newDashboard = reactive({name: '', url: ''});
const dashboardFormError = ref('');
const editingDashboardIndex = ref<number | null>(null);
const isEditingDashboard = computed(() => editingDashboardIndex.value !== null);

const resetDashboardForm = () => {
  dashboardFormError.value = '';
  editingDashboardIndex.value = null;
  newDashboard.name = '';
  newDashboard.url = '';
};

const hwidTooltipContent = computed(() => {
  const headers = settingStore.hwidHeaders;
  const lines: string[] = [];

  if (headers.hwid) {
    lines.push(`HWID=${headers.hwid}`);
  }
  if (headers.os) {
    lines.push(`OS=${headers.os}`);
  }
  if (headers.osVersion) {
    lines.push(`OS Version=${headers.osVersion}`);
  }
  if (headers.model) {
    lines.push(`Model=${headers.model}`);
  }

  if (lines.length === 0) {
    return ['HWID=—', 'OS=—', 'OS Version=—', 'Model=—'];
  }
  return lines;
});

const importInputRef = ref<HTMLInputElement | null>(null);

const openImportDialog = () => {
  importInputRef.value?.click();
};

const handleImportFile = async (event: Event) => {
  const target = event.target as HTMLInputElement | null;
  const files = target?.files ? Array.from(target.files) : [];
  if (files.length === 0) {
    return;
  }

  if (files.length > 1) {
    pWarning(t("drag.size"));
    if (target) {
      target.value = '';
    }
    return;
  }

  const file = files[0];
  const reader = new FileReader();
  reader.onload = async (loadEvent) => {
    await pLoad(t("drag.add"), async () => {
      const profile = new Profile();
      profile.content = loadEvent.target?.result ?? '';
      profile.title = file.name;
      try {
        const pList = await api.addProfileFromInput(profile);
        if (pList && pList.length > 0) {
          webStore.dProfile = pList;
          pSuccess(t("drag.success"));
          api.getProfileList().then((list) => {
            Events.Emit({
              name: "profiles",
              data: list,
            });
          });
        }
      } catch (e) {
        if (e && typeof e === 'object' && 'message' in e && typeof e.message === 'string') {
          pError(e.message);
        } else {
          pError(String(e));
        }
      }
    });
  };
  reader.onerror = (error) => {
    console.error(`Error reading ${file.name}:`, error);
    pError(t("drag.error"));
  };
  reader.readAsText(file);

  if (target) {
    target.value = '';
  }
};


const openExternalLink = (url: string) => {
  if (!url) {
    return;
  }

  try {
    Browser.OpenURL(url)
  } catch (e) {
    window.open(url, '_blank')
  }
};

const openDashboard = (dashboard: DashboardOption) => {
  const formattedUrl = buildDashboardUrl(dashboard.url, {
    host: webStore.host,
    port: webStore.port,
    secret: webStore.secret,
  });
  openExternalLink(formattedUrl);
};

const handleDashboardCommand = (command: DashboardOption | 'manage') => {
  if (typeof command === 'string') {
    dashboardDialogVisible.value = true;
    return;
  }

  openDashboard(command);
};

const submitCustomDashboardEntry = () => {
  const name = newDashboard.name.trim();
  const url = newDashboard.url.trim();

  if (!name || !url) {
    dashboardFormError.value = t('setting.dashboard.error');
    return;
  }

  dashboardFormError.value = '';

  if (editingDashboardIndex.value === null) {
    webStore.addCustomDashboard({name, url});
  } else {
    webStore.updateCustomDashboard(editingDashboardIndex.value, {name, url});
  }

  resetDashboardForm();
};

// Accepted product change: removing a custom dashboard asks first.
const askRemoveCustomDashboardEntry = async (index: number) => {
  const dashboard = customDashboards.value[index];
  if (!dashboard) return;
  const ok = await confirm({
    title: t('confirm.delete-dashboard.title'),
    text: t('confirm.delete-dashboard.text', {name: dashboard.name}),
    okLabel: t('confirm.delete-dashboard.ok'),
  });
  if (ok) removeCustomDashboardEntry(index);
};

const removeCustomDashboardEntry = (index: number) => {
  webStore.removeCustomDashboard(index);

  if (editingDashboardIndex.value === null) {
    return;
  }

  if (editingDashboardIndex.value === index) {
    resetDashboardForm();
    return;
  }

  if (index < editingDashboardIndex.value) {
    editingDashboardIndex.value -= 1;
  }
};

const startEditingCustomDashboardEntry = (index: number) => {
  const dashboard = customDashboards.value[index];

  if (!dashboard) {
    return;
  }

  editingDashboardIndex.value = index;
  newDashboard.name = dashboard.name;
  newDashboard.url = dashboard.url;
  dashboardFormError.value = '';
};

const cancelEditingCustomDashboardEntry = () => {
  resetDashboardForm();
};

// 使用路由
const router = useRouter()

// DNS Query Tool
const dnsQueryName = ref('')
const dnsQueryType = ref('A')
const dnsQueryLoading = ref(false)
const dnsQueryResults = ref<Array<{ name: string; type: number; TTL: number; data: string }>>([])
const dnsQueryError = ref('')

const DNS_TYPE_MAP: Record<number, string> = { 1: 'A', 5: 'CNAME', 28: 'AAAA', 15: 'MX', 16: 'TXT', 2: 'NS' }

async function runDnsQuery() {
  const name = dnsQueryName.value.trim()
  if (!name) return
  dnsQueryLoading.value = true
  dnsQueryResults.value = []
  dnsQueryError.value = ''
  try {
    const data: any = await proxy.$http.get(`/dns/query?name=${encodeURIComponent(name)}&type=${dnsQueryType.value}`)
    const answers = data?.Answer ?? []
    if (answers.length === 0) {
      dnsQueryError.value = t('setting.mihomo.dnsQuery.noResults')
    } else {
      dnsQueryResults.value = answers
    }
  } catch (e: any) {
    dnsQueryError.value = e?.message || String(e)
  } finally {
    dnsQueryLoading.value = false
  }
}

// 数据监听
// dns
watch(() => settingStore.dns, (newValue) => {
  // 更新配置
  api.switchDNS({
    enable: newValue,
  });
});

// ipv6
watch(() => settingStore.ipv6, (newValue) => {
  // 更新配置
  api.updateConfigs({
    ipv6: newValue,
  }).then(() => {
    // 同步 mihomo 配置
    pUpdateMihomo(menuStore, settingStore, api)
  });
});

// 开机自启
// NOTE: the shell is notified from MyEvent.vue instead — that component is
// always mounted, so it can also re-apply the registration on every launch.
watch(() => settingStore.startup, () => {
  // 同步 mihomo 配置
  pUpdateMihomo(menuStore, settingStore, api)
});

// Порт - обновляем системный прокси если он включен и прокси активен
watch(() => settingStore.port, async (newValue, oldValue) => {
  if (menuStore.proxy && settingStore.systemProxyMode && newValue !== oldValue) {
    try {
      await updateSystemProxy(api, settingStore, true);
    } catch (e) {
      console.error('Failed to update system proxy port:', e);
    }
  }
});

// Адрес привязки - обновляем системный прокси если он включен и прокси активен
watch(() => settingStore.bindAddress, async (newValue, oldValue) => {
  if (menuStore.proxy && settingStore.systemProxyMode && newValue !== oldValue) {
    try {
      await updateSystemProxy(api, settingStore, true);
    } catch (e) {
      console.error('Failed to update system proxy bind address:', e);
    }
  }
});

// 打开配置目录
function pxConfigDir() {
  // @ts-ignore
  api.configDir().then(res => window["pxConfigDir"](res))
}

// 修改配置目录
async function changeConfigDir() {
  try {
    // @ts-ignore
    const preConfigDir = await window["pxPreConfigDir"]();

    if (!preConfigDir.endsWith("Prizrak-Box-V3")) {
      pWarning(t('setting.px.change-warn'))
    }

    // @ts-ignore
    const newDir = await window.electron.invoke('select-directory');
    if (!newDir) {
      return;
    }

    // @ts-ignore
    await window["pxChangeConfigDir"](newDir);
    pSuccess(t('setting.px.change-success'));
  } catch (e) {
    if (e && typeof e === 'object' && 'message' in e && typeof e.message === 'string') {
      pError(e.message);
    } else {
      pError(String(e));
    }
  }
}

const releasesPageUrl = 'https://github.com/legiz-ru/Prizrak-Box/releases/latest'

// 打开更新页面
function openReleasesPage() {
  openExternalLink(releasesPageUrl)
}

// 手动检查更新
async function checkForUpdatesManually() {
  await updateStore.checkForUpdates()
}

function addGroupTestUrl() {
  settingStore.groupTestUrls = [...settingStore.groupTestUrls, { name: '', url: '' }];
}

function removeGroupTestUrl(index: number) {
  settingStore.groupTestUrls = settingStore.groupTestUrls.filter((_, i) => i !== index);
}

watch(dashboardDialogVisible, (visible) => {
  if (!visible) {
    resetDashboardForm();
  }
});

const secretVisible = ref(false);
const dnsTypeOptions = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS'].map(value => ({value, label: value}));

const ageKeypairDialogVisible = ref(false);

// Prizrak-Core version, baked in at build time from src-go/go.mod.
const coreVersion = __CORE_VERSION__;

</script>

<template>
  <section v-if="props.section !== 'app'" class="px-card cfg-card">
    <div class="cfg-head">
      <span class="px-card-title">Mihomo</span>
      <span v-if="coreVersion"
            class="px-tag tabular"
            v-tip="$t('setting.mihomo.core-version-tip')">{{ coreVersion }}</span>
    </div>
    <div class="px-divider cfg-divider"></div>
    <div class="px-rows">
      <MyPort/>
      <MyBind/>
      <MyTun/>

      <div class="px-row px-row--wrap">
        <span class="px-row__label">Api</span>
        <span class="px-row__value ellipsis">{{ webStore.baseUrl }}</span>
        <div class="px-row__end">
          <button type="button" class="px-btn px-btn--soft px-btn--sm" @click="copy(webStore.baseUrl,t)">{{ $t('copy.title') }}</button>
          <UiDropdown align="right" role="menu" :min-width="220">
            <template #trigger="{ open, toggle, attrs }">
              <button type="button" v-bind="attrs" class="px-btn px-btn--soft px-btn--sm" @click="toggle">
                <icon-tabler-layout-dashboard width="14" height="14"/>
                {{ t('setting.dashboard.open') }}
                <icon-tabler-chevron-down width="12" height="12" class="cfg-chev" :class="{ 'is-open': open }"/>
              </button>
            </template>
            <template #default="{ close }">
              <button v-for="dashboard in dashboardOptions"
                      :key="dashboard.key"
                      type="button"
                      role="menuitem"
                      data-dd-item
                      class="px-dd-item"
                      @click="close(); openDashboard(dashboard)">
                <span class="ellipsis" style="flex:1">{{ dashboard.name }}</span>
                <icon-tabler-external-link width="13" height="13" style="color:var(--text-3);flex-shrink:0"/>
              </button>
              <div class="px-dd-sep"></div>
              <button type="button" role="menuitem" data-dd-item class="px-dd-item" @click="close(); dashboardDialogVisible = true">
                <icon-tabler-settings width="14" height="14" style="color:var(--text-2);flex-shrink:0"/>
                {{ t('setting.dashboard.manage') }}
              </button>
            </template>
          </UiDropdown>
        </div>
      </div>

      <div class="px-row px-row--wrap">
        <span class="px-row__label">
          Secret
          <span class="px-info" tabindex="0" :aria-label="t('setting.tips.secret')" v-tip="t('setting.tips.secret')">
            <icon-tabler-info-circle width="13" height="13"/>
          </span>
        </span>
        <span class="px-row__value mono ellipsis">{{ secretVisible ? webStore.secret : '••••••••••••' }}</span>
        <UiIconButton :size="26" :label="secretVisible ? t('ui.hide') : t('ui.show')" @click="secretVisible = !secretVisible">
          <icon-tabler-eye-off v-if="secretVisible" width="14" height="14"/>
          <icon-tabler-eye v-else width="14" height="14"/>
        </UiIconButton>
        <div class="px-row__end">
          <button type="button" class="px-btn px-btn--soft px-btn--sm" @click="copy(webStore.secret,t)">{{ $t('copy.title') }}</button>
        </div>
      </div>

      <div class="px-row">
        <span class="px-row__label">{{ $t('setting.mihomo.dns') }}</span>
        <UiSwitch v-model="settingStore.dns" :aria-label="$t('setting.mihomo.dns')"/>
        <UiIconButton :size="26" :label="$t('setting.section.dns')" @click="changeMenu('Setting/Dns',router)">
          <icon-tabler-edit width="14" height="14"/>
        </UiIconButton>
      </div>

      <div class="px-row">
        <span class="px-row__label">IPv6</span>
        <UiSwitch v-model="settingStore.ipv6" aria-label="IPv6"/>
      </div>

      <div class="px-row">
        <span class="px-row__label">
          {{ $t('setting.mihomo.independentDelayTest') }}
          <span class="px-info" tabindex="0" :aria-label="t('setting.tips.independent-test')" v-tip="t('setting.tips.independent-test')">
            <icon-tabler-info-circle width="13" height="13"/>
          </span>
        </span>
        <UiSwitch v-model="settingStore.independentDelayTest" :aria-label="$t('setting.mihomo.independentDelayTest')"/>
      </div>

      <div v-if="settingStore.independentDelayTest" class="px-row cfg-urls-row">
        <span class="px-row__label cfg-urls-label">{{ $t('setting.mihomo.groupTestUrls') }}</span>
        <div class="cfg-urls">
          <div v-for="(item, index) in settingStore.groupTestUrls" :key="index" class="cfg-url">
            <input v-model="item.name" class="px-input px-input--sm" :aria-label="$t('setting.mihomo.groupName')" :placeholder="$t('setting.mihomo.groupName')">
            <input v-model="item.url" class="px-input px-input--sm" :aria-label="$t('setting.mihomo.testUrlPlaceholder')" :placeholder="$t('setting.mihomo.testUrlPlaceholder')">
            <UiIconButton :size="26" :label="$t('delete')" @click="removeGroupTestUrl(index)">
              <icon-tabler-x width="14" height="14"/>
            </UiIconButton>
          </div>
          <UiIconButton :size="26" :label="$t('setting.mihomo.addGroupUrl')" @click="addGroupTestUrl">
            <icon-tabler-plus width="14" height="14"/>
          </UiIconButton>
        </div>
      </div>

      <div class="px-row px-row--wrap">
        <span class="px-row__label">{{ $t('setting.mihomo.dnsQuery.queryTitle') }}</span>
        <input v-model="dnsQueryName"
               class="px-input px-input--sm cfg-dns-input"
               placeholder="example.com"
               aria-label="example.com"
               autocapitalize="off"
               autocomplete="off"
               autocorrect="off"
               spellcheck="false"
               @keyup.enter="runDnsQuery">
        <UiSelect v-model="dnsQueryType" :options="dnsTypeOptions" class="cfg-dns-type" :min-width="90" align="left" :aria-label="$t('setting.mihomo.dnsQuery.queryTitle')"/>
        <button type="button" class="px-btn px-btn--soft px-btn--sm" :disabled="dnsQueryLoading" @click="runDnsQuery">
          <UiSpinner v-if="dnsQueryLoading" :size="12"/>
          <icon-tabler-search v-else width="13" height="13"/>
          {{ $t('setting.mihomo.dnsQuery.query') }}
        </button>
      </div>
      <div v-if="dnsQueryError || dnsQueryResults.length > 0" class="cfg-dns-results">
        <div v-if="dnsQueryError" class="px-alert px-alert--error">
          <icon-tabler-alert-circle width="15" height="15"/>
          <span>{{ dnsQueryError }}</span>
        </div>
        <div v-for="(rec, i) in dnsQueryResults" :key="i" class="cfg-dns-record">
          <span class="px-tag px-tag--accent">{{ DNS_TYPE_MAP[rec.type] ?? rec.type }}</span>
          <span class="mono ellipsis" style="flex:1">{{ rec.data }}</span>
          <span class="tabular cfg-ttl">TTL {{ rec.TTL }}</span>
        </div>
      </div>

      <div class="px-row">
        <span class="px-row__label">{{ $t('age.settings.label') }}</span>
        <button type="button" class="px-btn px-btn--soft px-btn--sm" @click="ageKeypairDialogVisible = true">
          {{ $t('age.settings.generateBtn') }}
        </button>
      </div>
    </div>
    <MyAgeKeypair v-model="ageKeypairDialogVisible"/>
  </section>

  <section v-if="props.section !== 'core'" class="px-card cfg-card">
    <div class="cfg-head">
      <span class="px-card-title">Prizrak-Box</span>
      <span v-if="manualUpdateStatus.text"
            class="px-tag"
            :class="{
              'px-tag--success': manualUpdateStatus.type === 'success',
              'px-tag--warning': manualUpdateStatus.type === 'warning',
              'px-tag--error': manualUpdateStatus.type === 'danger',
            }"
            role="status">
        <UiSpinner v-if="manualUpdateStatus.type === 'info'" :size="10"/>
        {{ manualUpdateStatus.text }}
      </span>
    </div>
    <div class="px-divider cfg-divider"></div>
    <div class="px-rows">
      <MyService/>
      <div class="px-divider"></div>

      <div class="px-row">
        <span class="px-row__label">
          HWID
          <span class="px-info" tabindex="0" :aria-label="t('setting.tips.hwid')" v-tip="[t('setting.tips.hwid'), '', ...hwidTooltipContent].join('\n')">
            <icon-tabler-info-circle width="13" height="13"/>
          </span>
        </span>
        <UiSwitch v-model="settingStore.hwid" aria-label="HWID"/>
      </div>
      <div class="px-row">
        <span class="px-row__label">{{ $t('setting.px.startup') }}</span>
        <UiSwitch v-model="settingStore.startup" :aria-label="$t('setting.px.startup')"/>
      </div>
      <div class="px-row">
        <span class="px-row__label">{{ $t('setting.px.startMinimized') }}</span>
        <UiSwitch v-model="settingStore.startMinimized" :aria-label="$t('setting.px.startMinimized')"/>
      </div>
      <div class="px-row">
        <span class="px-row__label">{{ $t('setting.px.systemProxyMode') }}</span>
        <UiSwitch v-model="settingStore.systemProxyMode" :aria-label="$t('setting.px.systemProxyMode')"/>
      </div>
      <div class="px-row">
        <span class="px-row__label">
          {{ $t('setting.px.auth') }}
          <span class="px-info" tabindex="0" :aria-label="t('setting.tips.auth')" v-tip="t('setting.tips.auth')">
            <icon-tabler-info-circle width="13" height="13"/>
          </span>
        </span>
        <UiSwitch v-model="settingStore.auth" :aria-label="$t('setting.px.auth')"/>
      </div>

      <div class="px-divider"></div>

      <div class="px-row">
        <span class="px-row__label">{{ $t('setting.shortcut.title') }}</span>
        <UiSwitch v-model="settingStore.sc_switch" :aria-label="$t('setting.shortcut.title')"/>
        <UiIconButton :size="26" :label="$t('setting.shortcut.edit')" @click="changeMenu('Setting/Shortcut',router)">
          <icon-tabler-edit width="14" height="14"/>
        </UiIconButton>
      </div>
      <div class="px-row">
        <span class="px-row__label">
          {{ $t('setting.subscriptionAlerts.title') }}
          <span class="px-info" tabindex="0" :aria-label="$t('setting.subscriptionAlerts.tooltip')" v-tip="$t('setting.subscriptionAlerts.tooltip')">
            <icon-tabler-info-circle width="13" height="13"/>
          </span>
        </span>
        <UiSwitch v-model="settingStore.notifySubscriptionAlerts" :aria-label="$t('setting.subscriptionAlerts.title')"/>
      </div>
      <div class="px-row px-row--wrap">
        <span class="px-row__label">{{ $t('setting.px.dir') }}</span>
        <button type="button" class="px-btn px-btn--soft px-btn--sm" @click="pxConfigDir">{{ $t('setting.px.open') }}</button>
        <button type="button" class="px-btn px-btn--soft px-btn--sm" @click="changeConfigDir">{{ $t('setting.px.change') }}</button>
        <button type="button" class="px-btn px-btn--soft px-btn--sm" @click="openImportDialog">{{ $t('setting.px.import') }}</button>
        <input ref="importInputRef" type="file" accept=".yaml,.yml" hidden @change="handleImportFile"/>
      </div>
      <div class="px-row px-row--wrap">
        <span class="px-row__label">{{ $t('setting.px.update') }}</span>
        <button type="button" class="px-btn px-btn--soft px-btn--sm" @click="openReleasesPage">{{ t('updates.actions.open') }}</button>
        <button type="button" class="px-btn px-btn--soft px-btn--sm" :disabled="updateChecking" @click="checkForUpdatesManually">
          <UiSpinner v-if="updateChecking" :size="12"/>
          {{ t('updates.actions.check') }}
        </button>
      </div>
    </div>
  </section>

  <!-- Custom dashboards -->
  <UiModal v-model="dashboardDialogVisible" :title="t('setting.dashboard.custom-title')" :width="520">
    <label class="px-field">
      <span class="px-field__label">{{ t('setting.dashboard.name') }}</span>
      <input v-model="newDashboard.name" class="px-input" placeholder="Zashboard">
    </label>
    <label class="px-field">
      <span class="px-field__label">{{ t('setting.dashboard.url') }}</span>
      <input v-model="newDashboard.url" class="px-input" placeholder="https://example.com/?host=%host&port=%port&secret=%secret">
    </label>
    <span class="px-field__hint">{{ t('setting.dashboard.hint') }}</span>
    <span v-if="dashboardFormError" class="px-field__error" role="alert">{{ dashboardFormError }}</span>
    <div class="cfg-dash-actions">
      <button v-if="isEditingDashboard" type="button" class="px-btn px-btn--pill" @click="cancelEditingCustomDashboardEntry">
        {{ t('setting.dashboard.cancel') }}
      </button>
      <button type="button" class="px-btn px-btn--primary px-btn--pill" @click="submitCustomDashboardEntry">
        <icon-tabler-device-floppy v-if="isEditingDashboard" width="15" height="15"/>
        <icon-tabler-plus v-else width="15" height="15"/>
        {{ isEditingDashboard ? t('setting.dashboard.save') : t('setting.dashboard.add') }}
      </button>
    </div>
    <div class="px-divider"></div>
    <div v-if="customDashboards.length === 0" class="cfg-dash-empty">{{ t('setting.dashboard.empty') }}</div>
    <div v-else class="cfg-dash-list">
      <div v-for="(item, index) in customDashboards"
           :key="item.name + index"
           class="cfg-dash-item"
           :class="{ 'is-editing': editingDashboardIndex === index }">
        <div class="cfg-dash-info">
          <span class="ellipsis cfg-dash-name">{{ item.name }}</span>
          <span class="ellipsis cfg-dash-url" v-tip="item.url">{{ item.url }}</span>
        </div>
        <UiIconButton :size="30" :label="t('setting.dashboard.edit')" @click="startEditingCustomDashboardEntry(index)">
          <icon-tabler-edit width="15" height="15"/>
        </UiIconButton>
        <UiIconButton :size="30" danger :label="t('setting.dashboard.remove')" @click="askRemoveCustomDashboardEntry(index)">
          <icon-tabler-trash width="15" height="15"/>
        </UiIconButton>
      </div>
    </div>
  </UiModal>
</template>

<style scoped>
.cfg-card {
  padding: 18px 20px;
  flex-shrink: 0;
}

.cfg-divider {
  margin: 12px 0;
}

.cfg-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.cfg-chev {
  transition: transform .15s;
}

.cfg-chev.is-open {
  transform: rotate(180deg);
}

.cfg-urls-row {
  align-items: flex-start;
}

.cfg-urls-label {
  padding-top: 6px;
}

.cfg-urls {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.cfg-url {
  display: grid;
  grid-template-columns: 1fr 2fr auto;
  gap: 8px;
  align-items: center;
}

.cfg-dns-input {
  width: 170px;
}

.cfg-dns-type {
  width: 90px;
}

.cfg-dns-type :deep(.px-select--field) {
  padding: 6px 8px 6px 10px;
  font-size: 12px;
}

.cfg-dns-results {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-left: 180px;
}

.cfg-dns-record {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  min-width: 0;
}

.cfg-ttl {
  color: var(--text-3);
  flex-shrink: 0;
}

.cfg-dash-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.cfg-dash-empty {
  padding: 18px 0;
  text-align: center;
  font-size: 13px;
  color: var(--text-3);
}

.cfg-dash-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cfg-dash-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 10px 10px 14px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: var(--input-bg);
}

.cfg-dash-item.is-editing {
  border-color: var(--accent);
}

.cfg-dash-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cfg-dash-name {
  font-size: 13px;
  font-weight: 600;
}

.cfg-dash-url {
  font-size: 12px;
  color: var(--text-2);
}
</style>
