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

const shortcutDialogVisible = ref(false);

const ageKeypairDialogVisible = ref(false);

// Секрет ядра скрыт по умолчанию и открывается по кнопке: раздел настроек
// регулярно попадает в скриншоты, а ключ даёт полный доступ к API ядра.
const secretVisible = ref(false);

// Версия ядра для подсказки у заголовка раздела. Отдельного запроса не жалко:
// /version — самый дешёвый маршрут ядра, и он же показывает, что ядро живо.
const coreVersion = ref('');
onMounted(async () => {
  try {
    coreVersion.value = await api.getVersion();
  } catch {
    coreVersion.value = '';
  }
});

</script>

<template>
  <!-- Настройки ядра -->
  <div v-if="props.section !== 'app'" class="setting-stack">
    <SettingSection title="Mihomo">
      <template #title-after>
        <PxInfo
            :content="coreVersion ? `${t('setting.mihomo.coreVersion')}: ${coreVersion}` : t('setting.mihomo.coreVersionUnknown')"
            :label="t('setting.mihomo.coreVersion')"
        />
      </template>

      <!-- Api и Secret наверху: это то, что отсюда чаще всего копируют. -->
      <SettingRow label="Api">
        <button class="px-value" :title="$t('copy.title')" @click="copy(webStore.baseUrl, t)">
          <span class="px-value__text">{{ webStore.baseUrl }}</span>
          <el-icon class="px-value__icon"><icon-tabler-copy/></el-icon>
        </button>
        <el-dropdown trigger="click" @command="handleDashboardCommand">
          <button class="px-btn">
            <el-icon><icon-tabler-layout-dashboard/></el-icon>
            {{ t('setting.dashboard.open') }}
            <el-icon><icon-tabler-chevron-down/></el-icon>
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
      </SettingRow>

      <!-- Секрет по умолчанию скрыт: раздел настроек часто попадает в
           скриншоты и на стримы, а ключ даёт полный доступ к ядру. -->
      <SettingRow label="Secret">
        <PxInfo :content="$t('setting.hints.secret')"/>
        <button class="px-value" :title="$t('copy.title')" @click="copy(webStore.secret, t)">
          <span class="px-value__text" :class="{ 'px-value__text--masked': !secretVisible }">
            {{ secretVisible ? webStore.secret : '••••••••••••' }}
          </span>
          <el-icon class="px-value__icon"><icon-tabler-copy/></el-icon>
        </button>
        <button
            class="px-iconbtn px-iconbtn--plain"
            :aria-label="$t('setting.hints.secret')"
            :aria-pressed="secretVisible"
            @click="secretVisible = !secretVisible"
        >
          <el-icon>
            <icon-tabler-eye-off v-if="secretVisible"/>
            <icon-tabler-eye v-else/>
          </el-icon>
        </button>
      </SettingRow>

      <SettingRow :label="$t('setting.mihomo.port')" :hint="$t('setting.hints.port')">
        <MyPort/>
      </SettingRow>

      <SettingRow :label="$t('setting.mihomo.bindAddress')">
        <MyBind/>
      </SettingRow>

      <SettingRow label="Tun Stack">
        <MyTun/>
      </SettingRow>

      <SettingRow :label="$t('setting.mihomo.dns')" :hint="$t('setting.hints.dns')">
        <button
            class="px-iconbtn px-iconbtn--plain"
            :aria-label="$t('setting.mihomo.dns')"
            @click.stop="changeMenu('Setting/Dns', router)"
        >
          <el-icon><icon-tabler-pencil/></el-icon>
        </button>
        <PxToggle v-model="settingStore.dns" :label="$t('setting.mihomo.dns')"/>
      </SettingRow>

      <SettingRow label="IPv6">
        <PxToggle v-model="settingStore.ipv6" label="IPv6"/>
      </SettingRow>

      <SettingRow
          :label="$t('setting.mihomo.independentDelayTest')"
          :hint="$t('setting.hints.independentDelayTest')"
      >
        <PxToggle
            v-model="settingStore.independentDelayTest"
            :label="$t('setting.mihomo.independentDelayTest')"
        />
      </SettingRow>

      <!-- Кнопка добавления стоит в колонке контролов, на месте тумблера:
           широкая кнопка под списком ломала правую границу раздела. -->
      <SettingRow v-if="settingStore.independentDelayTest" :label="$t('setting.mihomo.groupTestUrls')">
        <button
            class="px-iconbtn"
            :aria-label="$t('setting.mihomo.addGroupUrl')"
            :title="$t('setting.mihomo.addGroupUrl')"
            @click="addGroupTestUrl"
        >
          <el-icon><icon-tabler-plus/></el-icon>
        </button>
        <template #hint>
          <div v-if="settingStore.groupTestUrls.length" class="group-urls">
            <div
                v-for="(item, index) in settingStore.groupTestUrls"
                :key="index"
                class="group-urls__row"
            >
              <input
                  v-model="item.name"
                  class="px-value-input"
                  :placeholder="$t('setting.mihomo.groupName')"
                  :aria-label="$t('setting.mihomo.groupName')"
              />
              <input
                  v-model="item.url"
                  class="px-value-input"
                  :placeholder="$t('setting.mihomo.testUrlPlaceholder')"
                  :aria-label="$t('setting.mihomo.testUrlPlaceholder')"
              />
              <button
                  class="px-iconbtn px-iconbtn--plain"
                  :aria-label="$t('setting.dashboard.remove')"
                  @click="removeGroupTestUrl(index)"
              >
                <el-icon><icon-tabler-x/></el-icon>
              </button>
            </div>
          </div>
        </template>
      </SettingRow>

      <SettingRow :label="$t('setting.mihomo.dnsQuery.queryTitle')">
        <template #hint>
          <div v-if="dnsQueryError" class="dns-query__error">{{ dnsQueryError }}</div>
          <div v-if="dnsQueryResults.length > 0" class="dns-query__results">
            <div v-for="(rec, i) in dnsQueryResults" :key="i" class="dns-query__record">
              <span
                  class="dns-type-badge"
                  :class="'dns-type--' + (DNS_TYPE_MAP[rec.type] ?? 'other').toLowerCase()"
              >{{ DNS_TYPE_MAP[rec.type] ?? rec.type }}</span>
              <span class="dns-query__data px-num">{{ rec.data }}</span>
              <span class="dns-query__ttl px-num">TTL {{ rec.TTL }}</span>
            </div>
          </div>
        </template>
        <input
            v-model="dnsQueryName"
            class="px-value-input dns-query__input"
            placeholder="example.com"
            autocapitalize="off"
            autocomplete="off"
            autocorrect="off"
            spellcheck="false"
            :aria-label="$t('setting.mihomo.dnsQuery.queryTitle')"
            @keyup.enter="runDnsQuery"
        />
        <el-dropdown trigger="click" @command="(cmd: string) => dnsQueryType = cmd">
          <button class="px-btn dns-query__type">
            {{ dnsQueryType }}
            <el-icon><icon-tabler-chevron-down/></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                  v-for="type in ['A','AAAA','CNAME','MX','TXT','NS']"
                  :key="type"
                  :command="type"
              >{{ type }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <button class="px-btn" :disabled="dnsQueryLoading" @click="runDnsQuery">
          <el-icon>
            <icon-tabler-loader-2 v-if="dnsQueryLoading" class="px-spin"/>
            <icon-tabler-world-search v-else/>
          </el-icon>
          {{ $t('setting.mihomo.dnsQuery.query') }}
        </button>
      </SettingRow>

      <!-- Генерация ключей — редкое действие, поэтому в самом низу. -->
      <SettingRow :label="$t('age.settings.label')">
        <button class="px-btn" @click="ageKeypairDialogVisible = true">
          <el-icon><icon-tabler-key/></el-icon>
          {{ $t('age.settings.generateBtn') }}
        </button>
      </SettingRow>
    </SettingSection>
  </div>

  <!-- Настройки приложения -->
  <div v-if="props.section !== 'core'" class="setting-stack">
    <SettingSection
        title="Prizrak-Box"
        :note="manualUpdateStatus.text"
        :note-type="manualUpdateStatus.type"
    >
      <!-- Обновление относится ко всему разделу, а не к отдельной настройке,
           поэтому живёт в заголовке иконками, а не строкой внизу списка. -->
      <template #actions>
        <el-tooltip :content="t('updates.actions.open')" placement="top" :show-after="150">
          <button
              class="px-iconbtn px-iconbtn--plain px-iconbtn--sm"
              :aria-label="t('updates.actions.open')"
              @click="openReleasesPage"
          >
            <el-icon><icon-tabler-external-link/></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip :content="t('updates.actions.check')" placement="top" :show-after="150">
          <button
              class="px-iconbtn px-iconbtn--plain px-iconbtn--sm"
              :aria-label="t('updates.actions.check')"
              :disabled="updateChecking"
              @click="checkForUpdatesManually"
          >
            <el-icon>
              <icon-tabler-loader-2 v-if="updateChecking" class="px-spin"/>
              <icon-tabler-refresh v-else/>
            </el-icon>
          </button>
        </el-tooltip>
      </template>

      <SettingRow label="HWID">
        <PxInfo :lines="hwidTooltipContent" label="HWID"/>
        <PxToggle v-model="settingStore.hwid" label="HWID"/>
      </SettingRow>

      <SettingRow :label="$t('service.mode')">
        <PxInfo :content="$t('service.mode-description')"/>
        <MyService/>
      </SettingRow>

      <SettingRow :label="$t('setting.px.startup')">
        <PxToggle v-model="settingStore.startup" :label="$t('setting.px.startup')"/>
      </SettingRow>

      <SettingRow :label="$t('setting.px.auth')">
        <PxInfo :content="$t('setting.hints.auth')"/>
        <PxToggle v-model="settingStore.auth" :label="$t('setting.px.auth')"/>
      </SettingRow>

      <SettingRow :label="$t('setting.px.systemProxyMode')">
        <PxToggle v-model="settingStore.systemProxyMode" :label="$t('setting.px.systemProxyMode')"/>
      </SettingRow>

      <SettingRow :label="$t('setting.shortcut.title')">
        <button
            class="px-iconbtn px-iconbtn--plain"
            :aria-label="$t('setting.shortcut.edit')"
            @click.stop="shortcutDialogVisible = true"
        >
          <el-icon><icon-tabler-pencil/></el-icon>
        </button>
        <PxToggle v-model="settingStore.sc_switch" :label="$t('setting.shortcut.title')"/>
      </SettingRow>

      <SettingRow :label="$t('setting.subscriptionAlerts.title')">
        <PxInfo :content="$t('setting.subscriptionAlerts.tooltip')"/>
        <PxToggle
            v-model="settingStore.notifySubscriptionAlerts"
            :label="$t('setting.subscriptionAlerts.title')"
        />
      </SettingRow>

      <SettingRow :label="$t('setting.px.dir')">
        <button class="px-btn" @click="pxConfigDir">
          <el-icon><icon-tabler-folder-open/></el-icon>
          {{ $t('setting.px.open') }}
        </button>
        <button class="px-btn px-btn--quiet" @click="changeConfigDir">
          <el-icon><icon-tabler-edit/></el-icon>
          {{ $t('setting.px.change') }}
        </button>
        <button class="px-btn px-btn--quiet" @click="openImportDialog">
          <el-icon><icon-tabler-file-import/></el-icon>
          {{ $t('setting.px.import') }}
        </button>
        <input
            ref="importInputRef"
            type="file"
            accept=".yaml,.yml"
            hidden
            @change="handleImportFile"
        />
      </SettingRow>

      <SettingRow :label="$t('setting.px.startMinimized')">
        <PxToggle v-model="settingStore.startMinimized" :label="$t('setting.px.startMinimized')"/>
      </SettingRow>
    </SettingSection>
  </div>

  <MyAgeKeypair v-model="ageKeypairDialogVisible"/>

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
              <icon-tabler-pencil class="dashboard-dialog__action-icon"/>
            </el-button>
            <el-button
                type="danger"
                plain
                circle
                :title="t('setting.dashboard.remove')"
                :aria-label="t('setting.dashboard.remove')"
                @click="removeCustomDashboardEntry(index)"
            >
              <icon-tabler-trash class="dashboard-dialog__action-icon"/>
            </el-button>
          </div>
        </li>
      </ul>
    </div>
  </el-dialog>
</template>

<style scoped>
/* Раскладка разделов. Внешние поля контента задаёт правая панель, здесь
   только расстояние между группами — одно значение на все экраны. */
.setting-stack {
  display: flex;
  flex-direction: column;
  gap: var(--px-section-gap);
}

/* --- Строка «URL тестов для групп» ---
   Список живёт под меткой, а кнопка добавления — в колонке контролов, на
   месте тумблера: так правая граница раздела остаётся единой. */
.group-urls {
  display: flex;
  flex-direction: column;
  gap: var(--px-space-2);
  width: 100%;
  margin-top: var(--px-space-2);
}

.group-urls__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 2fr) auto;
  gap: var(--px-space-2);
  align-items: center;
}

/* --- Строка DNS-запроса ---
   Высота инпута, выпадающего списка и кнопки задана общим --px-control-h,
   здесь остаётся только ширина поля. */
.dns-query__input {
  width: 190px;
}

.dns-query__type {
  min-width: 74px;
  justify-content: space-between;
}

.dns-query__error {
  margin-top: var(--px-space-2);
  color: var(--px-danger);
  font-size: var(--px-fs-small);
}

.dns-query__results {
  display: flex;
  flex-direction: column;
  gap: var(--px-space-1);
  margin-top: var(--px-space-2);
}

.dns-query__record {
  display: flex;
  align-items: center;
  gap: var(--px-space-2);
}

.dns-query__data {
  flex: 1;
  min-width: 0;
  font-size: var(--px-fs-small);
  overflow-wrap: anywhere;
}

.dns-query__ttl {
  font-size: var(--px-fs-caption);
  opacity: .55;
  white-space: nowrap;
}

.dns-type-badge {
  display: inline-block;
  min-width: 48px;
  padding: 2px 8px;
  border-radius: var(--px-r-pill);
  font-size: var(--px-fs-caption);
  font-weight: 600;
  text-align: center;
  background-color: rgba(128, 128, 128, .2);
  color: var(--text-color);
}

.dns-type--a { background-color: rgba(56, 189, 248, .18); color: var(--px-info); }
.dns-type--aaaa { background-color: rgba(74, 222, 128, .18); color: var(--px-ok); }
.dns-type--cname { background-color: rgba(250, 204, 21, .18); color: var(--px-warn); }
.dns-type--mx { background-color: rgba(248, 113, 113, .18); color: var(--px-danger); }
.dns-type--txt { background-color: rgba(148, 163, 184, .18); color: var(--px-text-muted); }
.dns-type--ns { background-color: rgba(160, 90, 220, .18); color: #a05adc; }

/* Крутящийся индикатор в кнопках («Проверить обновления», DNS-запрос). */
.px-spin {
  animation: px-spin 1s linear infinite;
}

@keyframes px-spin {
  to { transform: rotate(360deg); }
}

/* Маскированный секрет: точки набираются моноширинным шрифтом, поэтому ширина
   строки не скачет при показе и скрытии. */
.px-value__text--masked {
  letter-spacing: .18em;
}

/* --- Диалог горячих клавиш --- */
.shortcut-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.shortcut-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--px-space-3);
  padding: var(--px-space-2) 0;
  font-size: var(--px-fs-lead);
}

.shortcut-label {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.shortcut-controls {
  display: flex;
  align-items: center;
  gap: var(--px-space-3);
}

/* Внутри диалога фон — не стекло приложения, поэтому выключенному состоянию
   нужна видимая граница, иначе переключатель сливается с поверхностью. */
.shortcut-controls :deep(.px-toggle:not(.is-on)) {
  box-shadow: inset 0 0 0 1.5px var(--el-border-color);
}

/* --- Диалог пользовательских панелей --- */
.dashboard-dialog {
  display: flex;
  flex-direction: column;
  gap: var(--px-space-3);
}

.dashboard-dialog__form-fields {
  display: grid;
  gap: var(--px-space-3);
}

.dashboard-dialog__actions {
  margin-top: var(--px-space-1);
}

.dashboard-dialog__action-icon {
  display: inline-flex;
  vertical-align: middle;
}

.dashboard-dialog__action-icon--with-label {
  margin-right: var(--px-space-2);
}

.dashboard-dialog__item-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--px-space-2);
}

.dashboard-dialog__hint {
  margin: 0;
  font-size: var(--px-fs-small);
  opacity: .75;
}

.dashboard-dialog__error {
  margin: var(--px-space-2) 0 0;
  color: var(--px-danger);
  font-size: var(--px-fs-small);
}

.dashboard-dialog__empty {
  text-align: center;
  opacity: .7;
  font-size: var(--px-fs-body);
}

.dashboard-dialog__list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--px-space-3);
}

.dashboard-dialog__item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--px-space-3);
}

.dashboard-dialog__item-info {
  display: flex;
  flex-direction: column;
  gap: var(--px-space-1);
  max-width: 75%;
}

.dashboard-dialog__item-name {
  font-weight: 600;
}

.dashboard-dialog__item-url {
  font-size: var(--px-fs-small);
  opacity: .75;
  overflow-wrap: anywhere;
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
</style>