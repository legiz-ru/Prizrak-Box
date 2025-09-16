<script setup lang="ts">

import MyPort from "@/components/setting/MyPort.vue";
import MyBind from "@/components/setting/MyBind.vue";
import MyTun from "@/components/setting/MyTun.vue";
import {ArrowDown, EditPen} from "@element-plus/icons-vue";
import {useWebStore} from "@/store/webStore";
import {useHomeStore} from "@/store/homeStore";
import {copy} from "@/util/pLoad";
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

interface DashboardOption {
  key: string;
  name: string;
  url: string;
  isCustom?: boolean;
}

const defaultDashboards: DashboardOption[] = [
  {
    key: 'metacubexd',
    name: 'MetaCubeXD',
    url: 'https://metacubex.github.io/metacubexd/#/setup?http=true&hostname=%host&port=%port&secret=%secret',
  },
  {
    key: 'yacd',
    name: 'Yacd',
    url: 'https://yacd.metacubex.one/?hostname=%host&port=%port&secret=%secret',
  },
  {
    key: 'zashboard',
    name: 'Zashboard',
    url: 'https://board.zash.run.place/#/setup?http=true&hostname=%host&port=%port&secret=%secret',
  },
];

const customDashboardOptions = computed<DashboardOption[]>(() =>
  customDashboards.value.map((entry, index) => ({
    key: `custom-${index}`,
    name: entry.name,
    url: entry.url,
    isCustom: true,
  })),
);

const dashboardOptions = computed(() => [...defaultDashboards, ...customDashboardOptions.value]);

const dashboardDialogVisible = ref(false);
const newDashboard = reactive({name: '', url: ''});
const dashboardFormError = ref('');

const formatDashboardUrl = (template: string) => template
    .replace(/%host/g, webStore.host)
    .replace(/%port/g, webStore.port)
    .replace(/%secret/g, webStore.secret);

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
  const formattedUrl = formatDashboardUrl(dashboard.url);
  openExternalLink(formattedUrl);
};

const handleDashboardCommand = (command: DashboardOption | 'manage') => {
  if (typeof command === 'string') {
    dashboardDialogVisible.value = true;
    return;
  }

  openDashboard(command);
};

const addCustomDashboardEntry = () => {
  const name = newDashboard.name.trim();
  const url = newDashboard.url.trim();

  if (!name || !url) {
    dashboardFormError.value = t('setting.dashboard.error');
    return;
  }

  dashboardFormError.value = '';
  webStore.addCustomDashboard({name, url});
  newDashboard.name = '';
  newDashboard.url = '';
};

const removeCustomDashboardEntry = (index: number) => {
  webStore.removeCustomDashboard(index);
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

// HWID设置
watch(() => settingStore.hwid, async (newValue) => {
  await updateHTTPClientConfig();
});

// 更新HTTP客户端配置
async function updateHTTPClientConfig() {
  try {
    // 获取版本号
    const version = await api.getVersion();
    
    // 更新HTTP客户端配置
    await api.updateHTTPClientConfig({
      enableHWID: settingStore.hwid,
      version: version,
      deviceOS: homeStore.os,
      deviceOSVer: "", // 可以从系统信息获取，暂时留空
      deviceModel: "", // 可以从系统信息获取，暂时留空
    });
  } catch (e) {
    console.error("Failed to update HTTP client config:", e);
  }
}

// 打开配置目录
function pxConfigDir() {
  // @ts-ignore
  api.configDir().then(res => window["pxConfigDir"](res))
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

watch(dashboardDialogVisible, (visible) => {
  if (!visible) {
    dashboardFormError.value = '';
    newDashboard.name = '';
    newDashboard.url = '';
  }
});

// 初始化HTTP客户端配置
onMounted(async () => {
  await updateHTTPClientConfig();
});

</script>

<template>
  <el-row :gutter="20" class="spark"
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
          <li class="api-row">
            <strong>Api :</strong>
            <span class="api-row__value">{{ webStore.baseUrl }}</span>
            <el-button
                @click="copy(webStore.baseUrl,t)"
                class="api-row__button">
              {{ $t('copy.title') }}
            </el-button>
            <el-dropdown trigger="click" @command="handleDashboardCommand" class="api-row__dropdown">
              <el-button class="api-row__button" type="primary" plain>
                {{ t('setting.dashboard.open') }}
                <el-icon class="api-row__icon">
                  <ArrowDown/>
                </el-icon>
              </el-button>
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
          </li>
          <li style="height: 30px">
            <strong>Secret:</strong>
            {{ webStore.secret }}
            <el-button
                @click="copy(webStore.secret,t)">
              {{ $t('copy.title') }}
            </el-button>
          </li>
        </ul>
      </div>
    </el-col>
  </el-row>

  <el-row :gutter="20" class="spark"
          style="margin-left: 0;
          margin-top: 30px;
          margin-right: 0;">
    <el-col :span="24">
      <div class="box box2">
        <div class="title">
          Prizrak-Box
        </div>
        <hr/>
        <ul class="info-list">
          <li>
            <strong>HWID :</strong>
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
            <strong>{{ $t('setting.px.auth') }} :</strong>
            <el-switch
                v-model="settingStore.auth"
                class="set-switch"
            />
          </li>
          <li style="height: 30px">
            <strong>{{ $t('setting.px.dir') }} :</strong>
            <el-button @click="pxConfigDir" style="margin-left: 10px">
              {{ $t('setting.px.open') }}
            </el-button>
            <!--            <el-button>{{ $t('setting.px.export') }}</el-button>-->
            <!--            <el-button>{{ $t('setting.px.import') }}</el-button>-->
          </li>
          <li class="update-row">
            <strong>{{ $t('setting.px.update') }} :</strong>
            <el-button @click="openReleasesPage" class="update-row__button">{{ t('updates.actions.open') }}</el-button>
            <el-button
                @click="checkForUpdatesManually"
                :loading="updateChecking"
                class="update-row__button"
                type="primary"
                plain
            >
              {{ t('updates.actions.check') }}
            </el-button>
            <span v-if="manualUpdateStatus.text" :class="['update-status', manualUpdateStatus.type && `update-status--${manualUpdateStatus.type}`]">
              {{ manualUpdateStatus.text }}
            </span>
          </li>
        </ul>
      </div>
    </el-col>
  </el-row>


  <el-dialog
      v-model="dashboardDialogVisible"
      :title="t('setting.dashboard.custom-title')"
      width="520px"
  >
    <div class="dashboard-dialog">
      <div class="dashboard-dialog__form">
        <el-form label-position="top" class="dashboard-dialog__form-fields">
          <el-form-item :label="t('setting.dashboard.name')">
            <el-input v-model="newDashboard.name" placeholder="MetaCubeXD"/>
          </el-form-item>
          <el-form-item :label="t('setting.dashboard.url')">
            <el-input v-model="newDashboard.url" placeholder="https://example.com"/>
          </el-form-item>
        </el-form>
        <p class="dashboard-dialog__hint">{{ t('setting.dashboard.hint') }}</p>
        <div class="dashboard-dialog__actions">
          <el-button type="primary" @click="addCustomDashboardEntry">{{ t('setting.dashboard.add') }}</el-button>
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
          <el-button type="danger" link @click="removeCustomDashboardEntry(index)">
            {{ t('setting.dashboard.remove') }}
          </el-button>
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
  border-radius: 8px;
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
  gap: 8px;
  min-height: 30px;
}

.api-row strong {
  margin-right: 6px;
}

.api-row__value {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.95rem;
}

.api-row__button {
  margin-left: 10px;
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
  margin-left: 10px;
}

.update-status {
  font-size: 0.85rem;
}

.update-status--info {
  color: #909399;
}

.update-status--success {
  color: #67c23a;
}

.update-status--warning {
  color: #e6a23c;
}

.update-status--danger {
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
  box-shadow: var(--right-box-shadow);
}

.box2 {
  box-shadow: var(--right-box-shadow);
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

:deep(.el-button) {
  padding: 2px 10px;
  --el-button-bg-color: transparent;
  --el-button-text-color: var(--text-color);
  --el-button-hover-text-color: var(--left-item-selected-bg);
  --el-button-hover-bg-color: var(--text-color)
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


</style>