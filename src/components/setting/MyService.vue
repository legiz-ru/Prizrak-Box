<script setup lang="ts">
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {pSuccess, pError, pWarning} from "@/util/pLoad";
import {restartBackendAndSync, waitBackendReady} from "@/util/backendConn";
import {
  notifyServiceStatusChanged,
  notifyTunForcedOff,
  SERVICE_STATUS_UPDATED_EVENT,
} from "@/util/serviceEvents";

const {t} = useI18n();
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// Состояние сервиса
const serviceStatus = ref<{
  installed: boolean;
  running: boolean;
  isAdmin: boolean;
  version?: string;
}>({
  installed: false,
  running: false,
  isAdmin: false
});

const loading = ref(false);

// Получение статуса сервиса
async function fetchServiceStatus() {
  loading.value = true;
  try {
    // @ts-ignore
    const status = await window.pxService.getStatus();
    serviceStatus.value = status;
  } catch (e) {
    serviceStatus.value = {installed: false, running: false, isAdmin: false};
  }
  loading.value = false;
}

// Установка сервиса
async function installService() {
  loading.value = true;
  // Only a failure of install() itself means "install failed" — problems while
  // relaunching px afterwards are reported as "restart required" instead.
  let success = false;
  try {
    // @ts-ignore
    success = await window.pxService.install();
  } catch (e) {
    success = false;
  }
  if (success) {
    pSuccess(t('service.install-success'));
    await restartBackendAfterInstall();
  } else {
    pError(t('service.install-failed'));
  }
  notifyServiceStatusChanged();
  await fetchServiceStatus();
  loading.value = false;
}

// Удаление сервиса
async function uninstallService() {
  loading.value = true;

  // Stop px gracefully *before* removing the service: it is the service that
  // owns the running (elevated) px, and once the service is gone there is no
  // way left to ask it to shut down cleanly — it would keep the control port
  // and the TUN device.
  try {
    await api.exit();
  } catch (e) {
    // ignore exit errors - процесс может быть уже остановлен
  }

  let success = false;
  try {
    // @ts-ignore
    success = await window.pxService.uninstall();
  } catch (e) {
    success = false;
  }

  if (success) {
    pSuccess(t('service.uninstall-success'));
    // TUN can no longer work: px is now an ordinary, unprivileged process.
    notifyTunForcedOff();
    // После удаления сервиса перезапускаем backend в обычном режиме
    await restartBackendAfterUninstall();
    // Even when px came back cleanly, the app is no longer running against the
    // elevated service it was started with, so ask for a restart. This warning
    // never appeared before (see restartBackendAfterUninstall).
    pWarning(t('service.restart-required'));
  } else {
    // Uninstall refused/cancelled — px was stopped above, so bring it back
    // (through the still-installed service) instead of leaving the app dead.
    pError(t('service.uninstall-failed'));
    await restartBackendAfterUninstall();
  }

  notifyServiceStatusChanged();
  await fetchServiceStatus();
  loading.value = false;
}

async function restartBackendAfterInstall() {
  try {
    await api.exit();
  } catch (e) {
    // ignore exit errors
  }

  await new Promise((resolve) => setTimeout(resolve, 800));

  try {
    // restartBackendAndSync() applies the new host/port/secret reported by the
    // shell; without it the app keeps talking to the px it was launched with.
    const restarted = await restartBackendAndSync();
    if (!restarted || !(await waitBackendReady(api))) {
      pWarning(t('service.restart-required'));
    }
  } catch (e) {
    pWarning(t('service.restart-required'));
  }
}

// Relaunches px after the service was removed. The shell falls back to a plain
// (unprivileged) spawn once the service is unreachable; its fresh host/port/
// secret are applied so the running app keeps working.
//
// NOTE: this used to wait via api.waitRunning(), which swallows every error by
// design (see src/api/mihomo/index.ts). Its catch block could therefore never
// run and the "restart the app" warning was never shown — the bug this fixes.
async function restartBackendAfterUninstall(): Promise<boolean> {
  try {
    if (!(await restartBackendAndSync())) {
      return false;
    }
    return await waitBackendReady(api);
  } catch (e) {
    return false;
  }
}

// Статус в читаемом виде
const statusText = computed(() => {
  if (!serviceStatus.value.installed) {
    return t('service.status-not-installed');
  }
  if (serviceStatus.value.running && serviceStatus.value.isAdmin) {
    return t('service.status-running');
  }
  if (serviceStatus.value.running && !serviceStatus.value.isAdmin) {
    return t('service.status-no-admin');
  }
  return t('service.status-stopped');
});

const statusType = computed(() => {
  if (!serviceStatus.value.installed) {
    return 'info';
  }
  if (serviceStatus.value.running && serviceStatus.value.isAdmin) {
    return 'success';
  }
  if (serviceStatus.value.running && !serviceStatus.value.isAdmin) {
    return 'danger';
  }
  return 'warning';
});

// Проверяем статус при монтировании
onMounted(() => {
  fetchServiceStatus();
  window.addEventListener(SERVICE_STATUS_UPDATED_EVENT, fetchServiceStatus);
});

onUnmounted(() => {
  window.removeEventListener(SERVICE_STATUS_UPDATED_EVENT, fetchServiceStatus);
});
</script>

<template>
  <div class="service-setting">
    <div class="service-setting__header">
      <strong>{{ t('service.mode') }}:</strong>
      <el-tag :type="statusType" size="small" class="service-setting__status">
        {{ statusText }}
      </el-tag>
    </div>
    <p class="service-setting__description">{{ t('service.mode-description') }}</p>
    <div class="service-setting__actions">
      <button class="pill-btn" :disabled="loading" @click="installService">
        <icon-mdi-loading v-if="loading" class="pill-spin"/>
        {{ t('service.install-btn') }}
      </button>
      <button
          v-if="serviceStatus.installed || serviceStatus.running"
          class="pill-btn pill-btn--danger"
          :disabled="loading"
          @click="uninstallService"
      >
        <icon-mdi-loading v-if="loading" class="pill-spin"/>
        {{ t('service.uninstall-btn') }}
      </button>
      <button class="pill-btn" :disabled="loading" @click="fetchServiceStatus">
        <icon-mdi-loading v-if="loading" class="pill-spin"/>
        {{ t('service.check-status') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.service-setting {
  margin: 8px 0;
}

.service-setting__header {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 18px;
}

.service-setting__status {
  margin-left: 8px;
  --el-tag-border-radius: 999px;
  border-radius: 999px;
}

.service-setting__description {
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.7;
  margin: 8px 0;
}

.service-setting__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 10px;
}

.pill-btn {
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  padding: 6px 18px;
  font-size: 14px;
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

.pill-btn--danger:hover {
  background-color: #f56c6c;
}

.pill-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
