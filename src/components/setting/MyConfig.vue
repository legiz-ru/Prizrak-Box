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
import {ArrowDown, EditPen} from "@element-plus/icons-vue";
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


  return lines;
});

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
watch(() => settingStore.startup, (newValue) => {
  // 更新配置
  Events.Emit({name: "boot", data: newValue});
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

const shortcutDialogVisible = ref(false);

const ageKeypairDialogVisible = ref(false);

</script>

<template>
  <el-row v-if="props.section !== 'app'" :gutter="20" class="spark"
          style="margin-left: 0;
          margin-top: 2px;
          margin-right: 0;">
    <el-col :span="24">
      <div class="box box1">
        <div class="title">
          Mihomo
        </div>
        <hr/>
        <ul class="info-list">
          <li>
            <MyPort></MyPort>
          </li>
          <li>
            <MyBind></MyBind>
          </li>
          <li>
            <MyTun></MyTun>
          </li>
          <li class="toggle-row">
            <strong>
              {{ $t('setting.mihomo.dns') }} :
            </strong>
            <div :class="['px-toggle', { 'is-on': settingStore.dns }]" @click="settingStore.dns = !settingStore.dns">
              <div class="px-toggle__thumb"></div>
            </div>
            <button class="pencil-btn" @click.stop="changeMenu('Setting/Dns',router)">
              <el-icon><EditPen/></el-icon>
            </button>
          </li>
          <li class="toggle-row">
            <strong>IPV6 :</strong>
            <div :class="['px-toggle', { 'is-on': settingStore.ipv6 }]" @click="settingStore.ipv6 = !settingStore.ipv6">
              <div class="px-toggle__thumb"></div>
            </div>
          </li>
          <li class="toggle-row">
            <strong>{{ $t('setting.mihomo.independentDelayTest') }} :</strong>
            <div :class="['px-toggle', { 'is-on': settingStore.independentDelayTest }]" @click="settingStore.independentDelayTest = !settingStore.independentDelayTest">
              <div class="px-toggle__thumb"></div>
            </div>
          </li>
          <li v-if="settingStore.independentDelayTest" class="group-test-urls-section">
            <div class="group-test-urls-header">
              <strong>{{ $t('setting.mihomo.groupTestUrls') }} :</strong>
              <button class="pill-btn" @click="addGroupTestUrl">+ {{ $t('setting.mihomo.addGroupUrl') }}</button>
            </div>
            <div class="group-test-urls-list">
              <div
                v-for="(item, index) in settingStore.groupTestUrls"
                :key="index"
                class="group-test-url-row"
              >
                <el-input
                  v-model="item.name"
                  :placeholder="$t('setting.mihomo.groupName')"
                  size="small"
                  class="group-test-url-input"
                />
                <el-input
                  v-model="item.url"
                  :placeholder="$t('setting.mihomo.testUrlPlaceholder')"
                  size="small"
                  class="group-test-url-input"
                />
                <button class="pill-btn pill-btn--danger" @click="removeGroupTestUrl(index)">✕</button>
              </div>
            </div>
          </li>
          <li class="age-row">
            <strong>{{ $t('age.settings.label') }} :</strong>
            <button class="pill-btn" @click="ageKeypairDialogVisible = true">
              {{ $t('age.settings.generateBtn') }}
            </button>
          </li>
          <MyAgeKeypair v-model="ageKeypairDialogVisible" />
          <li class="api-row">
            <div class="api-row__info">
              <strong>Api :</strong>
              <span class="api-row__value">{{ webStore.baseUrl }}</span>
            </div>
            <div class="api-row__actions">
              <button class="pill-btn" @click="copy(webStore.baseUrl,t)">{{ $t('copy.title') }}</button>
              <el-dropdown trigger="click" @command="handleDashboardCommand" class="api-row__dropdown">
                <button class="pill-btn pill-btn--arrow">
                  {{ t('setting.dashboard.open') }}
                  <el-icon class="api-row__icon"><ArrowDown/></el-icon>
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                        v-for="dashboard in dashboardOptions"
                        :key="dashboard.key"
                        :command="dashboard"
                    >
                      {{ dashboard.name }}
                    </el-dropdown-item>
                    <el-dropdown-item divided command="manage">
                      {{ t('setting.dashboard.manage') }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </li>
          <li class="secret-row">
            <strong>Secret:</strong>
            <span class="secret-row__value">{{ webStore.secret }}</span>
            <button class="pill-btn" @click="copy(webStore.secret,t)">{{ $t('copy.title') }}</button>
          </li>
          <li class="api-row dns-query-row">
            <strong>{{ $t('setting.mihomo.dnsQuery.queryTitle') }} :</strong>
            <input
              v-model="dnsQueryName"
              class="dns-query-input"
              placeholder="example.com"
              autocapitalize="off"
              autocomplete="off"
              autocorrect="off"
              spellcheck="false"
              @keyup.enter="runDnsQuery"
            />
            <el-dropdown trigger="click" @command="(cmd: string) => dnsQueryType = cmd">
              <button class="pill-btn pill-btn--arrow">
                {{ dnsQueryType }}
                <el-icon class="api-row__icon"><ArrowDown/></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-for="type in ['A','AAAA','CNAME','MX','TXT','NS']" :key="type" :command="type">{{ type }}</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <button class="pill-btn" :disabled="dnsQueryLoading" @click="runDnsQuery">
              {{ dnsQueryLoading ? '...' : $t('setting.mihomo.dnsQuery.query') }}
            </button>
          </li>
          <li v-if="dnsQueryError || dnsQueryResults.length > 0" class="dns-query-results-row">
            <div v-if="dnsQueryError" class="dns-query-error">{{ dnsQueryError }}</div>
            <div v-if="dnsQueryResults.length > 0" class="dns-query-results">
              <div v-for="(rec, i) in dnsQueryResults" :key="i" class="dns-query-record">
                <span class="dns-type-badge" :class="'dns-type--' + (DNS_TYPE_MAP[rec.type] ?? 'other').toLowerCase()">
                  {{ DNS_TYPE_MAP[rec.type] ?? rec.type }}
                </span>
                <span class="dns-record-data">{{ rec.data }}</span>
                <span class="dns-record-ttl">TTL {{ rec.TTL }}</span>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </el-col>
  </el-row>

  <el-row v-if="props.section !== 'core'" :gutter="20" class="spark"
          :style="{
            marginLeft: '0',
            marginTop: props.section === 'app' ? '2px' : '30px',
            marginRight: '0'
          }">
    <el-col :span="24">
      <div class="box box2">
        <div class="title title--status">
          <span class="title__label">Prizrak-Box</span>
          <span
              v-if="manualUpdateStatus.text"
              :class="['title__status', manualUpdateStatus.type && `title__status--${manualUpdateStatus.type}`]"
          >
            {{ manualUpdateStatus.text }}
          </span>
        </div>
        <hr/>
        <ul class="info-list">
          <li class="toggle-row">
            <el-tooltip placement="top" effect="dark" class="hwid-tooltip__trigger">
              <template #content>
                <div class="hwid-tooltip">
                  <div v-for="line in hwidTooltipContent" :key="line">{{ line }}</div>
                </div>
              </template>
              <strong class="hwid-label">HWID :</strong>
            </el-tooltip>
            <div :class="['px-toggle', { 'is-on': settingStore.hwid }]" @click="settingStore.hwid = !settingStore.hwid">
              <div class="px-toggle__thumb"></div>
            </div>
          </li>
          <li class="toggle-row">
            <strong>{{ $t('setting.px.startup') }} :</strong>
            <div :class="['px-toggle', { 'is-on': settingStore.startup }]" @click="settingStore.startup = !settingStore.startup">
              <div class="px-toggle__thumb"></div>
            </div>
          </li>
          <li class="toggle-row">
            <strong>{{ $t('setting.px.startMinimized') }} :</strong>
            <div :class="['px-toggle', { 'is-on': settingStore.startMinimized }]" @click="settingStore.startMinimized = !settingStore.startMinimized">
              <div class="px-toggle__thumb"></div>
            </div>
          </li>
          <li class="toggle-row">
            <strong>{{ $t('setting.px.systemProxyMode') }} :</strong>
            <div :class="['px-toggle', { 'is-on': settingStore.systemProxyMode }]" @click="settingStore.systemProxyMode = !settingStore.systemProxyMode">
              <div class="px-toggle__thumb"></div>
            </div>
          </li>
          <li class="toggle-row">
            <strong>{{ $t('setting.px.auth') }} :</strong>
            <div :class="['px-toggle', { 'is-on': settingStore.auth }]" @click="settingStore.auth = !settingStore.auth">
              <div class="px-toggle__thumb"></div>
            </div>
          </li>
          <li>
            <MyService />
          </li>
          <li class="toggle-row">
            <strong>{{ $t('setting.shortcut.title') }} :</strong>
            <div :class="['px-toggle', { 'is-on': settingStore.sc_switch }]" @click="settingStore.sc_switch = !settingStore.sc_switch">
              <div class="px-toggle__thumb"></div>
            </div>
            <button class="pencil-btn" @click.stop="shortcutDialogVisible = true">
              <el-icon><EditPen/></el-icon>
            </button>
          </li>
          <li class="btn-row">
            <strong>{{ $t('setting.px.dir') }} :</strong>
            <button class="pill-btn" @click="pxConfigDir">{{ $t('setting.px.open') }}</button>
            <button class="pill-btn" @click="changeConfigDir">{{ $t('setting.px.change') }}</button>
            <button class="pill-btn" @click="openImportDialog">{{ $t('setting.px.import') }}</button>
            <input
                ref="importInputRef"
                type="file"
                accept=".yaml,.yml"
                hidden
                @change="handleImportFile"
            />
          </li>
          <li class="update-row">
            <strong>{{ $t('setting.px.update') }} :</strong>
            <button class="pill-btn" @click="openReleasesPage">{{ t('updates.actions.open') }}</button>
            <button class="pill-btn" @click="checkForUpdatesManually" :disabled="updateChecking">
              <icon-mdi-loading v-if="updateChecking" class="pill-spin"/>
              {{ t('updates.actions.check') }}
            </button>
          </li>
        </ul>
      </div>
    </el-col>
  </el-row>

  <!-- Диалог 1: Горячие клавиши -->
  <el-dialog
      v-model="shortcutDialogVisible"
      :title="t('setting.shortcut.title')"
      width="420"
  >
    <ul class="shortcut-list">
      <li class="shortcut-item">
        <span class="shortcut-label">{{ t('setting.shortcut.showHide') }}</span>
        <div class="shortcut-controls">
          <div :class="['px-toggle', { 'is-on': settingStore.sc_switch }]" @click="settingStore.sc_switch = !settingStore.sc_switch">
            <div class="px-toggle__thumb"></div>
          </div>
          <MyHotkeyInput v-model="settingStore.sc_switch_key"/>
        </div>
      </li>
    </ul>
    <template #footer>
      <el-button @click="shortcutDialogVisible = false">{{ t('close') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog
      v-model="dashboardDialogVisible"
      :title="t('setting.dashboard.custom-title')"
      width="520px"
  >
    <div class="dashboard-dialog">
      <div class="dashboard-dialog__form">
        <el-form label-position="top" class="dashboard-dialog__form-fields">
          <el-form-item :label="t('setting.dashboard.name')">
            <el-input v-model="newDashboard.name" placeholder="Zashboard"/>
          </el-form-item>
          <el-form-item :label="t('setting.dashboard.url')">
            <el-input
                v-model="newDashboard.url"
                placeholder="https://legiz-ru.github.io/zashboard/#/setup?disableUpgradeCore=1&http=true&hostname=%host&port=%port&secret=%secret"
            />
          </el-form-item>
        </el-form>
        <p class="dashboard-dialog__hint">{{ t('setting.dashboard.hint') }}</p>
        <div class="dashboard-dialog__actions">
          <el-button type="primary" plain @click="submitCustomDashboardEntry">
            <component
                :is="isEditingDashboard ? 'icon-mdi-content-save' : 'icon-mdi-plus'"
                class="dashboard-dialog__action-icon dashboard-dialog__action-icon--with-label"
            />
            {{ isEditingDashboard ? t('setting.dashboard.save') : t('setting.dashboard.add') }}
          </el-button>
          <el-button v-if="isEditingDashboard" link @click="cancelEditingCustomDashboardEntry">
            {{ t('setting.dashboard.cancel') }}
          </el-button>
        </div>
        <p v-if="dashboardFormError" class="dashboard-dialog__error">{{ dashboardFormError }}</p>
      </div>
      <el-divider/>
      <div v-if="customDashboards.length === 0" class="dashboard-dialog__empty">
        {{ t('setting.dashboard.empty') }}
      </div>
      <ul v-else class="dashboard-dialog__list">
        <li v-for="(item, index) in customDashboards" :key="item.name + index" class="dashboard-dialog__item">
          <div class="dashboard-dialog__item-info">
            <span class="dashboard-dialog__item-name">{{ item.name }}</span>
            <span class="dashboard-dialog__item-url">{{ item.url }}</span>
          </div>
          <div class="dashboard-dialog__item-actions">
            <el-button
                type="primary"
                plain
                circle
                :title="t('setting.dashboard.edit')"
                :aria-label="t('setting.dashboard.edit')"
                @click="startEditingCustomDashboardEntry(index)"
            >
              <icon-mdi-pencil class="dashboard-dialog__action-icon"/>
            </el-button>
            <el-button
                type="danger"
                plain
                circle
                :title="t('setting.dashboard.remove')"
                :aria-label="t('setting.dashboard.remove')"
                @click="removeCustomDashboardEntry(index)"
            >
              <icon-mdi-trash-can-outline class="dashboard-dialog__action-icon"/>
            </el-button>
          </div>
        </li>
      </ul>
    </div>
  </el-dialog>
</template>

<style scoped>
.spark {
  max-width: 95%;
}

.box {
  padding: 10px;
  border-radius: 20px;
  text-align: left;
}

.box hr {
  border: none;
  height: 1px;
  background-color: var(--hr-color);
  margin: 10px 0;
}

.info-list {
  list-style: none;
  padding: 0;
}

.info-list li {
  font-size: 18px;
  margin: 8px 0;
}

.api-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  min-height: 30px;
}

.api-row__info {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.api-row__value {
  word-break: break-all;
}

.api-row__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.api-row__dropdown {
  display: inline-flex;
}

.api-row__icon {
  margin-left: 4px;
  font-size: 0.85rem;
}

.secret-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  min-height: 30px;
}

.age-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 30px;
}

.secret-row__value {
  word-break: break-all;
}

.info-list .dns-query-row {
  margin-top: 8px;
}

.update-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-height: 30px;
}

.update-row strong {
  margin-right: 6px;
}

.update-row__button {
  margin-left: 0;
}

.title--status {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title__status {
  font-size: 0.85rem;
}

.title__status--info {
  color: #909399;
}

.title__status--success {
  color: #67c23a;
}

.title__status--warning {
  color: #e6a23c;
}

.title__status--danger {
  color: #f56c6c;
}

.dashboard-dialog {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dashboard-dialog__form-fields {
  display: grid;
  gap: 12px;
}

.dashboard-dialog__actions {
  margin-top: 4px;
}

.dashboard-dialog__action-icon {
  display: inline-flex;
  vertical-align: middle;
}

.dashboard-dialog__action-icon--with-label {
  margin-right: 6px;
}

.dashboard-dialog__item-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.dashboard-dialog__hint {
  font-size: 0.85rem;
  opacity: 0.75;
  margin: 0;
}

.dashboard-dialog__error {
  color: #f56c6c;
  font-size: 0.85rem;
  margin: 6px 0 0;
}

.dashboard-dialog__empty {
  text-align: center;
  opacity: 0.7;
  font-size: 0.9rem;
}

.dashboard-dialog__list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dashboard-dialog__item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.dashboard-dialog__item-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 75%;
}

.dashboard-dialog__item-name {
  font-weight: 600;
}

.dashboard-dialog__item-url {
  font-size: 0.85rem;
  word-break: break-all;
  opacity: 0.75;
}

.box1 {
}

.box2 {
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.px-toggle {
  position: relative;
  display: inline-block;
  width: 58px;
  height: 36px;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  cursor: pointer;
  flex-shrink: 0;
  transition: background-color 0.25s ease, box-shadow 0.25s ease;
}

.px-toggle:hover {
  box-shadow: var(--left-nav-hover-shadow);
}

.px-toggle.is-on {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.px-toggle__thumb {
  position: absolute;
  top: 4px;
  left: 4px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background-color: var(--text-color);
  transition: left 0.25s ease, background-color 0.25s ease;
}

.px-toggle.is-on .px-toggle__thumb {
  left: 26px;
  background-color: #fff;
}

.pencil-btn {
  height: 36px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  box-shadow: var(--left-nav-shadow);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  flex-shrink: 0;
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.pencil-btn:hover {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.pill-btn {
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  padding: 9px 16px;
  font-size: 14px;
  height: 36px !important;
  cursor: pointer;
  box-shadow: var(--left-nav-shadow);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.pill-btn:hover {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.pill-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.pill-btn--arrow {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.pill-btn--danger {
  background-color: rgba(255, 80, 80, 0.15);
  color: var(--el-color-danger, #f56c6c);
  padding: 4px 10px;
}

.pill-btn--danger:hover {
  background-color: rgba(255, 80, 80, 0.3);
}

.group-test-urls-section {
  flex-direction: column !important;
  align-items: flex-start !important;
  gap: 8px;
}

.group-test-urls-header {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.group-test-urls-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
}

.group-test-url-row {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
}

.group-test-url-input {
  flex: 1;
}

.pill-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.btn-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-height: 30px;
}

.dashboard-dialog :deep(.el-button--primary) {
  --el-button-bg-color: var(--left-item-selected-bg);
  --el-button-border-color: var(--left-item-selected-bg);
  --el-button-text-color: #fff;
  --el-button-hover-bg-color: var(--left-item-selected-bg);
  --el-button-hover-text-color: #fff;
}

.dashboard-dialog :deep(.el-button.is-link) {
  --el-button-bg-color: transparent;
}


.shortcut-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.shortcut-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 16px;
  padding: 6px 0;
}

.shortcut-label {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.shortcut-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* The shared .px-toggle colours derive from the background image (--text-color
   / --left-nav-btn-bg) and can blend into the dialog's own surface, leaving the
   switch invisible on some themes. Inside the dialog use self-contained colours
   with a clear off/on contrast and a white thumb so it reads on any theme. */
.shortcut-controls .px-toggle {
  background-color: rgba(120, 120, 120, 0.45);
  box-shadow: inset 0 0 0 1.5px rgba(150, 150, 150, 0.55);
}
.shortcut-controls .px-toggle.is-on {
  background-color: var(--el-color-primary, #409eff);
  box-shadow: inset 0 0 0 1.5px rgba(0, 0, 0, 0.15);
}
.shortcut-controls .px-toggle__thumb {
  background-color: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.45);
}
.shortcut-controls .px-toggle.is-on .px-toggle__thumb {
  background-color: #fff;
}

/* DNS Query Tool */
.dns-query-results-row {
  flex-direction: column !important;
  align-items: flex-start !important;
  gap: 6px !important;
}

.dns-query-input {
  width: 160px;
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  padding: 0 16px;
  font-size: 14px;
  height: 36px;
  box-shadow: var(--left-nav-shadow);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.dns-query-input:focus {
  outline: none;
  box-shadow: var(--left-nav-hover-shadow);
}

.dns-query-error {
  font-size: 13px;
  color: var(--el-color-danger, #f56c6c);
  word-break: break-all;
}

.dns-query-results {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
}

.dns-query-record {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
}

.dns-type-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  min-width: 48px;
  text-align: center;
  background-color: rgba(128, 128, 128, 0.2);
  color: var(--text-color);
}

.dns-type--a      { background-color: rgba(64, 158, 255, 0.2); color: #409eff; }
.dns-type--aaaa   { background-color: rgba(103, 194, 58, 0.2); color: #67c23a; }
.dns-type--cname  { background-color: rgba(230, 162, 60, 0.2); color: #e6a23c; }
.dns-type--mx     { background-color: rgba(245, 108, 108, 0.2); color: #f56c6c; }
.dns-type--txt    { background-color: rgba(144, 147, 153, 0.2); color: #909399; }
.dns-type--ns     { background-color: rgba(160, 90, 220, 0.2); color: #a05adc; }

.dns-record-data {
  color: var(--text-color);
  word-break: break-all;
  flex: 1;
}

.dns-record-ttl {
  font-size: 12px;
  opacity: 0.55;
  white-space: nowrap;
}

</style>