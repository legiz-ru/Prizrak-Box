<script setup lang="ts">

const props = defineProps({
  section: { type: String, default: 'all' }
})

import MyPort from "@/components/setting/MyPort.vue";
import MyBind from "@/components/setting/MyBind.vue";
import MyTun from "@/components/setting/MyTun.vue";
import MyService from "@/components/setting/MyService.vue";
import MyHotkeyInput from "@/components/setting/MyHotkeyInput.vue";
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
          <li>
            <strong>
              {{ $t('setting.mihomo.dns') }} :
            </strong>
            <el-icon
                @click="changeMenu('Setting/Dns',router)"
                class="btn">
              <EditPen/>
            </el-icon>
            <el-switch
                v-model="settingStore.dns"
                class="set-switch"
                style="margin-left: 28px"
            />
          </li>
          <li>
            <strong>IPV6 :</strong>
            <el-switch
                v-model="settingStore.ipv6"
                class="set-switch"
            />
          </li>
          <li>
            <strong>{{ $t('setting.mihomo.independentDelayTest') }} :</strong>
            <el-switch
                v-model="settingStore.independentDelayTest"
                class="set-switch"
            />
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
          <li style="height: 30px">
            <strong>Secret:</strong>
            {{ webStore.secret }}
            <button class="pill-btn" @click="copy(webStore.secret,t)">{{ $t('copy.title') }}</button>
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
          <li>
            <el-tooltip placement="top" effect="dark" class="hwid-tooltip__trigger">
              <template #content>
                <div class="hwid-tooltip">
                  <div v-for="line in hwidTooltipContent" :key="line">{{ line }}</div>
                </div>
              </template>
              <strong class="hwid-label">HWID :</strong>
            </el-tooltip>
            <el-switch
                v-model="settingStore.hwid"
                class="set-switch"
            />
          </li>
          <li>
            <strong>{{ $t('setting.px.startup') }} :</strong>
            <el-switch
                v-model="settingStore.startup"
                class="set-switch"
            />
          </li>
          <li>
            <strong>{{ $t('setting.px.startMinimized') }} :</strong>
            <el-switch
                v-model="settingStore.startMinimized"
                class="set-switch"
            />
          </li>
          <li>
            <strong>{{ $t('setting.px.systemProxyMode') }} :</strong>
            <el-switch
                v-model="settingStore.systemProxyMode"
                class="set-switch"
            />
          </li>
          <li>
            <strong>{{ $t('setting.px.auth') }} :</strong>
            <el-switch
                v-model="settingStore.auth"
                class="set-switch"
            />
          </li>
          <li>
            <MyService />
          </li>
          <li style="height: 30px">
            <strong>{{ $t('setting.shortcut.title') }} :</strong>
            <el-icon
                @click="shortcutDialogVisible = true"
                class="btn">
              <EditPen/>
            </el-icon>
            <el-switch
                v-model="settingStore.sc_switch"
                class="set-switch"
                style="margin-left: 28px"
            />
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
          <el-switch v-model="settingStore.sc_switch" class="set-switch"/>
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

.set-switch {
  margin-left: 10px;
  --el-switch-border-color: var(--text-color);
  --el-switch-on-color: var(--left-item-selected-bg);
  --el-switch-off-color: transparent;
}

:deep(.el-switch__core) {
  width: 46px;
  height: 26px;
  border-radius: 12px;
  border: 2px solid var(--text-color);
}

:deep(.el-switch__core .el-switch__action) {
  margin-left: 2px;
  background-color: var(--text-color);
}

:deep(.el-switch.is-checked .el-switch__core .el-switch__action) {
  left: calc(100% - 21px);
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

.btn {
  font-size: 18px;
  position: absolute;
  margin-top: 6px;
}

.btn:hover {
  cursor: pointer;
  color: var(--hr-color);
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

/* Переключатель внутри диалога: всегда видимая рамка */
.shortcut-controls :deep(.el-switch__core) {
  border: 2px solid var(--el-border-color) !important;
}

.shortcut-controls :deep(.el-switch__core .el-switch__action) {
  background-color: var(--el-text-color-regular) !important;
}

.shortcut-controls :deep(.el-switch.is-checked .el-switch__core) {
  border-color: var(--el-color-primary) !important;
  background-color: var(--el-color-primary) !important;
}

.shortcut-controls :deep(.el-switch.is-checked .el-switch__core .el-switch__action) {
  background-color: #fff !important;
}
</style>