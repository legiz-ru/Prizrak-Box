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
import {restartBackendAndSync, waitBackendReady} from "@/util/backendConn";
import {notifyServiceStatusChanged, TUN_FORCE_OFF_EVENT} from "@/util/serviceEvents";

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

  // Проверяем, есть ли права администратора или сервис в админ-режиме.
  // isPxPrivileged() swallows errors on purpose: this used to call
  // api.getAdmin() directly, so a single unreachable request (px restarting
  // after a service install, a stale secret, …) rejected here and aborted the
  // whole handler — the click then did nothing at all, with no dialog and no
  // message. An unreachable px simply means "not privileged".
  const hasAdmin = await isPxPrivileged();
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
    // Step 1 — the install itself. Only a failure *here* means "install failed";
    // errors from the restart/TUN steps below must not be reported as one (they
    // used to be, which is why a successful install was immediately followed by
    // "Не удалось установить сервис" while the service was in fact installed).
    let installed = false;
    try {
      // @ts-ignore
      installed = await window.pxService.install();
    } catch (e) {
      installed = false;
    }
    if (!installed) {
      pError(t('service.install-failed'));
      return;
    }
    pSuccess(t('service.install-success'));

    // Step 2 — relaunch px through the freshly installed elevated service and
    // pick up its new host/port/secret.
    const restarted = await restartBackendAfterInstall();
    notifyServiceStatusChanged();
    if (!restarted) {
      return;
    }

    // Step 3 — bring TUN up on the now-privileged backend. Verify the privilege
    // rather than assuming it: if px did not actually come back through the
    // service, enabling TUN would write the config and silently pass no traffic.
    try {
      if (!(await isPxPrivileged())) {
        pWarning(t('service.restart-required'));
        return;
      }
      const select = await selected();
      if (select) {
        await enableTun();
      }
      // No profile selected: selected() already told the user. TUN stays off,
      // and the toggle now works because px is privileged.
    } catch (e) {
      pWarning(t('service.restart-required'));
    }
  });
}

async function restartBackendAfterInstall(): Promise<boolean> {
  // NOTE: this deliberately does NOT call api.exit() first. The shell's
  // restartBackend() already stops the running px (and asks the service to stop
  // the one it owns) before spawning a fresh one; killing px from here as well
  // only widened the window in which both sides raced over a half-dead process.
  try {
    // restartBackendAndSync() applies the connection info the shell reports for
    // the new px process. Without it the app kept using the port/secret it was
    // launched with, so every request failed until the app was restarted.
    if (!(await restartBackendAndSync())) {
      pWarning(t('service.restart-required'));
      return false;
    }
    // The shell only reports connection info once px has called back, so px is
    // already running here. Its control API can need a moment longer — and can
    // stay unhappy for reasons that have nothing to do with the restart (no
    // profile loaded yet, for instance) — so this wait is best effort and must
    // not fail the install: doing so skipped step 3 and told the user to
    // restart the app even though the service was installed and working.
    if (!(await waitBackendReady(api, 20000))) {
      console.warn('px restarted but its control API did not answer in time');
    }
    return true;
  } catch (e) {
    pWarning(t('service.restart-required'));
    return false;
  }
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

// Removing the privileged service (settings → "Удалить сервис") takes TUN down
// with it: px is relaunched unprivileged, so the toggle must stop showing "on".
function forceTunOff() {
  menuStore.setTun(false);
  tunOn.value = false;
  Events.Emit({name: "tun", data: false});
}

onMounted(() => {
  window.addEventListener(TUN_FORCE_OFF_EVENT, forceTunOff);
});

onUnmounted(() => {
  window.removeEventListener(TUN_FORCE_OFF_EVENT, forceTunOff);
});


onMounted(async () => {
  if (menuStore.proxy) {
    // This call is the only thing that re-asserts the system proxy on a normal
    // launch, and it has no retry: if it fails, whatever the previous session
    // (or another proxy client) last wrote is what the OS keeps using. Bootstrap
    // normally awaits px's connection info before mounting, but it deliberately
    // mounts anyway when px was slow or failed, so gate on the backend actually
    // answering — same as the TUN branch below.
    await api.waitRunning();
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
          const restarted = await restartBackendAndSync();
          if (restarted) {
            await waitBackendReady(api);
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