<script setup lang="ts">
import {useMenuStore} from "@/store/menuStore";
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {Events} from "@/runtime";
import {pError, pLoad, pSuccess, pWarning} from "@/util/pLoad";
import {useSettingStore} from "@/store/settingStore";
import {pUpdateMihomo} from "@/util/mihomo";
import {useHomeStore} from "@/store/homeStore";
import {updateSystemProxy} from "@/util/systemProxy";

// 使用store
const menuStore = useMenuStore();
const settingStore = useSettingStore();
const homeStore = useHomeStore();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 国际化
const {t} = useI18n();

// 页面使用参数
const tunOn = ref(false)

// Keep the native shell's persisted "TUN desired" flag in sync. The shell reads
// it at boot to decide whether to wait for the privileged service before
// spawning px (so TUN survives an autostart instead of silently failing).
watch(() => menuStore.tun, (v) => {
  Events.Emit({name: "tunDesired", data: !!v});
}, {immediate: true});

// Whether the *running* px process is actually privileged. api.getAdmin() asks
// px itself (admin token on Windows / uid 0 on unix), which is the real signal
// for "can TUN come up": it's true when the app runs as admin OR px was started
// by the elevated service, and false for a plain non-elevated spawn. We rely on
// this instead of the persisted config flag, which used to make the UI show TUN
// "on" while it did nothing.
async function isPxPrivileged(): Promise<boolean> {
  try {
    const admin = await api.getAdmin();
    return !!admin?.data;
  } catch (e) {
    return false;
  }
}


async function selected() {
  const list = await api.getProfileList()
  if (!list || list.length == 0) {
    pWarning(t("no-profile-warning"));
    return false
  }

  for (let profile of list) {
    if (profile['selected']) {
      return true
    }
  }

  pWarning(t("select-profile-warning"));
  return false
}


// 代理开关
async function doSwitch() {
  let ok = false

  // 检测通过执行后续操作
  if (!menuStore.proxy) {
    try {
      // 添加配置后执行
      const select = await selected()
      if (!select) {
        return
      }
      // 检测端口是否被占用
      await api.checkAddressPort({
        "bindAddress": settingStore.bindAddress,
        "port": settingStore.port,
      })
      // 开启代理
      await api.updateConfigs({
        "allow-lan": true,
        "mixed-port": settingStore.port,
        "bind-address": settingStore.bindAddress,
      })
      await applySystemProxyMode(settingStore.systemProxyMode, false);
      ok = true
      pSuccess(t("proxy-switch-on"));
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    }
  } else {
    // Отключаем mixed-port в Mihomo, чтобы он перестал слушать порт
    await api.updateConfigs({
      "mixed-port": 0,
    })
    // Всегда отключаем системный прокси при выключении переключателя прокси
    await applySystemProxyMode(false, false);
    ok = true
    pWarning(t("proxy-switch-off"));
  }

  // 同步配置
  if (ok) {
    menuStore.setProxy(!menuStore.proxy);
    pUpdateMihomo(menuStore, settingStore, api)
  }

  // 发送事件通知
  Events.Emit({name: "proxy", data: menuStore.proxy});
}

const proxySwitch = async () => {
  await pLoad(t('switch.ing'), doSwitch)
}

Events.On("switchProxy", async () => {
  await proxySwitch()
});


// Диалог предложения установки сервиса
const showServiceDialog = ref(false);
// Диалог выбора при наличии админских прав
const showAdminChoiceDialog = ref(false);

// 虚拟网卡开关
async function tunSwitch() {
  if (tunOn.value) {
    await enableTun();
    return;
  }

  // Проверяем, есть ли права администратора или сервис в админ-режиме
  const admin = await api.getAdmin();
  const hasAdmin = !!admin.data;
  let allowTun = hasAdmin;

  if (!allowTun) {
    try {
      // @ts-ignore
      const status = await window.pxService.getStatus();
      allowTun = status?.running && status?.isAdmin;
    } catch (e) {
      allowTun = false;
    }
  }

  if (!allowTun) {
    // Нет прав администратора - предлагаем установить сервис
    showServiceDialog.value = true;
    Events.Emit({name: "tun", data: false});
    return;
  }

  if (hasAdmin) {
    try {
      // @ts-ignore
      const status = await window.pxService.getStatus();
      const serviceElevated = status?.running && status?.isAdmin;
      if (!serviceElevated) {
        showAdminChoiceDialog.value = true;
        return;
      }
    } catch (e) {
      showAdminChoiceDialog.value = true;
      return;
    }
  }

  // 添加配置后执行
  const select = await selected()
  if (!select) {
    Events.Emit({name: "tun", data: false});
    return
  }

  // Включаем TUN
  await enableTun();
}

// Включение TUN режима
async function enableTun() {
  menuStore.setTun(!tunOn.value);
  if (menuStore.tun) {
    api.updateConfigs({
      tun: {
        enable: true,
        stack: settingStore.stack,
      },
    }).then(() => {
      tunOn.value = true;
      pSuccess(t("tun-switch-on"));

      // 同步 mihomo 配置
      pUpdateMihomo(menuStore, settingStore, api)
      notifyServiceStatusChanged();

      // 发送事件通知
      Events.Emit({name: "tun", data: menuStore.tun});
    });
  } else {
    api.updateConfigs({
      tun: {
        enable: false,
      },
    }).then(() => {
      tunOn.value = false;
      pWarning(t("tun-switch-off"));

      // 同步 mihomo 配置
      pUpdateMihomo(menuStore, settingStore, api)
      notifyServiceStatusChanged();

      // 发送事件通知
      Events.Emit({name: "tun", data: menuStore.tun});
    });
  }
}

// Установка сервиса
async function installServiceHandler() {
  showServiceDialog.value = false;
  showAdminChoiceDialog.value = false;
  pLoad(t('service.installing'), async () => {
    try {
      // @ts-ignore
      const installed = await window.pxService.install();
      if (installed) {
        pSuccess(t('service.install-success'));
        const restarted = await restartBackendAfterInstall();
        await notifyServiceStatusChanged();
        if (restarted) {
          await api.waitRunning();
          const select = await selected();
          if (select) {
            await enableTun();
          }
        }
      } else {
        pError(t('service.install-failed'));
      }
    } catch (e) {
      pError(t('service.install-failed'));
    }
  });
}

async function restartBackendAfterInstall(): Promise<boolean> {
  try {
    await api.exit();
  } catch (e) {
    // ignore exit errors
  }

  await new Promise((resolve) => setTimeout(resolve, 800));

  try {
    // @ts-ignore
    const restarted = await window.pxService.restartBackend();
    if (!restarted) {
      pWarning(t('service.restart-required'));
    }
    return restarted;
  } catch (e) {
    pWarning(t('service.restart-required'));
    return false;
  }
}

async function notifyServiceStatusChanged() {
  window.dispatchEvent(new CustomEvent('service-status-updated'));
}

async function runTunWithoutService() {
  showAdminChoiceDialog.value = false;
  const select = await selected();
  if (!select) {
    Events.Emit({name: "tun", data: false});
    return;
  }
  await enableTun();
}

// Закрыть диалог
function closeServiceDialog() {
  showServiceDialog.value = false;
}

function closeAdminChoiceDialog() {
  showAdminChoiceDialog.value = false;
}

Events.On("switchTun", async () => {
  await tunSwitch()
});


onMounted(async () => {
  if (menuStore.proxy) {
    await applySystemProxyMode(settingStore.systemProxyMode, false);
  }

  // Restore TUN state if it was previously enabled. TUN can only work when the
  // running px is privileged; we check that directly (isPxPrivileged) rather
  // than trusting the persisted config flag, which used to leave the UI showing
  // TUN "on" after an autostart while no traffic actually passed.
  if (menuStore.tun) {
    await api.waitRunning();

    let privileged = await isPxPrivileged();

    // Not privileged but the service is installed → px likely lost the boot race
    // (spawned before the service was reachable). Recover by relaunching px
    // through the elevated service, then re-check.
    if (!privileged) {
      let installed = false;
      try {
        // @ts-ignore
        installed = !!(await window.pxService.getStatus())?.installed;
      } catch (e) {
        installed = false;
      }
      if (installed) {
        try {
          // @ts-ignore
          const restarted = await window.pxService.restartBackend();
          if (restarted) {
            await api.waitRunning();
            privileged = await isPxPrivileged();
          }
        } catch (e) {
          // fall through to the not-privileged branch below
        }
      }
    }

    if (privileged) {
      // Silently enable TUN without showing dialogs.
      const select = await selected();
      if (select) {
        await enableTun();
      } else {
        // No profile selected - silently disable TUN
        menuStore.setTun(false);
        tunOn.value = false;
      }
    } else {
      // Can't bring TUN up (no elevation / service unavailable). Reflect the real
      // state instead of pretending it's on, and tell the user.
      menuStore.setTun(false);
      tunOn.value = false;
      pWarning(t("tun-unavailable"));
    }
  }
})

// Отслеживание изменения настройки "Режим системного прокси"
watch(() => settingStore.systemProxyMode, async (newValue, oldValue) => {
  // Применяем изменения только если прокси уже включен
  if (menuStore.proxy) {
    await applySystemProxyMode(newValue, true);
  }
})

async function applySystemProxyMode(enable: boolean, notify: boolean) {
  try {
    await updateSystemProxy(api, settingStore, enable);
    if (!notify) {
      return;
    }

    if (enable) {
      pSuccess(t("proxy-switch-on"));
    } else {
      pWarning(t("proxy-switch-off"));
    }
  } catch (e) {
    if (notify && e['message']) {
      pError(e['message'])
    }
  }
}


</script>

<template>
  <div class="mode-switches">
    <button
        type="button"
        :class="['mode-button', { 'is-active': menuStore.proxy }]"
        @click="proxySwitch"
    >
      <span class="mode-left">
        <span class="mode-icon">
          <icon-mdi-access-point-network-off v-if="!menuStore.proxy"/>
          <icon-mdi-access-point-network v-else/>
        </span>
        <span class="mode-label">
          {{ $t("proxy-switch") }}
        </span>
      </span>
      <span
          class="mode-indicator"
          :class="{ 'is-visible': menuStore.proxy }"
          aria-hidden="true"
      >
        <span class="mode-indicator__pulse"></span>
      </span>
    </button>
    <button
        type="button"
        :class="['mode-button', { 'is-active': tunOn }]"
        @click="tunSwitch"
    >
      <span class="mode-left">
        <span class="mode-icon">
          <icon-mdi-help-network-outline v-if="!tunOn"/>
          <icon-mdi-security-network v-else/>
        </span>
        <span class="mode-label mode-label--tun">
          {{ $t("tun-switch") }}
        </span>
      </span>
      <span
          class="mode-indicator"
          :class="{ 'is-visible': tunOn }"
          aria-hidden="true"
      >
        <span class="mode-indicator__pulse"></span>
      </span>
    </button>
  </div>

  <!-- Диалог предложения установки сервиса -->
  <el-dialog
      v-model="showServiceDialog"
      :title="$t('service.dialog-title')"
      width="450px"
      :close-on-click-modal="true"
      :append-to-body="true"
      :modal="true"
      :z-index="9999"
  >
    <div class="service-dialog">
      <p class="service-dialog__message">{{ $t('service.dialog-message') }}</p>
      <p class="service-dialog__description">{{ $t('service.dialog-description') }}</p>
      <p class="service-dialog__description">{{ $t('service.dialog-restart-admin') }}</p>
    </div>
    <template #footer>
      <div class="service-dialog__footer">
        <el-button @click="closeServiceDialog">{{ $t('cancel') }}</el-button>
        <el-button type="primary" @click="installServiceHandler">{{ $t('service.install-btn') }}</el-button>
      </div>
    </template>
  </el-dialog>

  <el-dialog
      v-model="showAdminChoiceDialog"
      :title="$t('service.admin-title')"
      width="450px"
      :close-on-click-modal="true"
      :append-to-body="true"
      :modal="true"
      :z-index="9999"
  >
    <div class="service-dialog">
      <p class="service-dialog__message">{{ $t('service.admin-message') }}</p>
      <p class="service-dialog__description">{{ $t('service.admin-description') }}</p>
    </div>
    <template #footer>
      <div class="service-dialog__footer">
        <el-button @click="closeAdminChoiceDialog">{{ $t('cancel') }}</el-button>
        <el-button @click="runTunWithoutService">{{ $t('service.admin-run-btn') }}</el-button>
        <el-button type="primary" @click="installServiceHandler">{{ $t('service.admin-install-btn') }}</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.mode-switches {
  margin-left: 22px;
  margin-top: 23px;
  width: 185px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mode-button {
  width: 100%;
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  padding: 10px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--left-nav-text);
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.mode-button:hover {
  background-color: var(--left-nav-btn-hover-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.mode-button.is-active {
  background-color: var(--left-item-selected-bg);
}

.mode-left {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
}

.mode-icon {
  display: inline-flex;
  font-size: 18px;
}

.mode-label--tun {
  text-transform: uppercase;
}

.mode-indicator {
  position: relative;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0;
  visibility: hidden;
  transform: scale(0.5);
  transition: all 0.3s ease;
  flex-shrink: 0;
}

.mode-indicator.is-visible {
  opacity: 1;
  visibility: visible;
  transform: scale(1);
  box-shadow:
    0 0 0 2px color-mix(in srgb, currentColor 20%, transparent),
    0 0 6px 1px currentColor,
    0 0 12px 3px color-mix(in srgb, currentColor 50%, transparent);
}

.mode-indicator__pulse {
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  background: radial-gradient(
    circle,
    color-mix(in srgb, currentColor 60%, transparent) 0%,
    transparent 70%
  );
  animation: pulse-indicator 1.5s ease-in-out infinite;
}

@keyframes pulse-indicator {
  0% { transform: scale(0.8); opacity: 0.2; }
  50% { transform: scale(1.3); opacity: 0.8; }
  100% { transform: scale(0.8); opacity: 0.2; }
}
</style>

<style>
/* Стили для диалога сервиса (не scoped, т.к. el-dialog рендерится вне компонента) */
.service-dialog {
  padding: 10px 0;
}

.service-dialog__message {
  font-size: 16px;
  margin-bottom: 12px;
  font-weight: 500;
}

.service-dialog__description {
  font-size: 14px;
  opacity: 0.8;
  line-height: 1.6;
}

.service-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>