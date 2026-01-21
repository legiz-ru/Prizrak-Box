<script setup lang="ts">
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {pSuccess, pError, pWarning} from "@/util/pLoad";

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
  try {
    // @ts-ignore
    const success = await window.pxService.install();
    if (success) {
      pSuccess(t('service.install-success'));
      await restartBackendAfterInstall();
      notifyServiceStatusChanged();
      await fetchServiceStatus();
    } else {
      pError(t('service.install-failed'));
    }
  } catch (e) {
    pError(t('service.install-failed'));
  }
  loading.value = false;
}

// Удаление сервиса
async function uninstallService() {
  loading.value = true;
  try {
    // @ts-ignore
    const success = await window.pxService.uninstall();
    if (success) {
      pSuccess(t('service.uninstall-success'));
      // После удаления сервиса перезапускаем backend в обычном режиме
      await restartBackendAfterUninstall();
      notifyServiceStatusChanged();
      await fetchServiceStatus();
    } else {
      pError(t('service.uninstall-failed'));
    }
  } catch (e) {
    pError(t('service.uninstall-failed'));
  }
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
    // @ts-ignore
    const restarted = await window.pxService.restartBackend();
    if (!restarted) {
      pWarning(t('service.restart-required'));
    }
  } catch (e) {
    pWarning(t('service.restart-required'));
  }
}

async function restartBackendAfterUninstall() {
  try {
    // Останавливаем текущий backend (который работал через сервис)
    await api.exit();
  } catch (e) {
    // ignore exit errors - процесс может быть уже остановлен
  }

  // Ждём, пока Electron автоматически перезапустит backend в обычном режиме
  await new Promise((resolve) => setTimeout(resolve, 1500));

  try {
    // Проверяем, что backend успешно запустился
    await api.waitRunning();
  } catch (e) {
    // Backend не запустился автоматически, требуется ручной перезапуск приложения
    pWarning(t('service.restart-required'));
  }
}

function notifyServiceStatusChanged() {
  window.dispatchEvent(new CustomEvent('service-status-updated'));
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
  window.addEventListener('service-status-updated', fetchServiceStatus);
});

onUnmounted(() => {
  window.removeEventListener('service-status-updated', fetchServiceStatus);
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
      <el-button
          type="primary"
          :loading="loading"
          @click="installService"
      >
        {{ t('service.install-btn') }}
      </el-button>
      <el-button
          v-if="serviceStatus.installed || serviceStatus.running"
          type="danger"
          :loading="loading"
          @click="uninstallService"
      >
        {{ t('service.uninstall-btn') }}
      </el-button>
      <el-button
          :loading="loading"
          @click="fetchServiceStatus"
      >
        {{ t('service.check-status') }}
      </el-button>
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
}

.service-setting__description {
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.7;
  margin: 8px 0;
}

.service-setting__actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}
</style>
