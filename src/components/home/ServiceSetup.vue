<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useOnboardingStore } from '@/store/onboardingStore';
import { Service } from '@/runtime';
import { pSuccess, pWarning } from '@/util/pLoad';

const { t } = useI18n();
const onboardingStore = useOnboardingStore();

const isInstalling = ref(false);

const emit = defineEmits<{
  (e: 'complete'): void
}>();

// Установка сервиса
async function installService() {
  isInstalling.value = true;

  try {
    const result = await Service.Install();

    if (result) {
      pSuccess(t('onboarding.service-setup.install-success'));
      onboardingStore.markServiceInstalled();
    } else {
      // Показываем предупреждение, но не ошибку
      pWarning(t('onboarding.service-setup.install-warning'));
      onboardingStore.markServiceSkipped();
    }
  } catch (error) {
    // При любой ошибке показываем предупреждение и помечаем как пропущенное
    pWarning(t('onboarding.service-setup.install-warning'));
    onboardingStore.markServiceSkipped();
  } finally {
    isInstalling.value = false;
    emit('complete');
  }
}

// Пропустить установку
function skipInstallation() {
  onboardingStore.markServiceSkipped();
  emit('complete');
}
</script>

<template>
  <div class="service-setup-container">
    <div class="service-setup-content">
      <h1 class="service-setup-title">{{ t('onboarding.service-setup.title') }}</h1>
      <p class="service-setup-description">{{ t('onboarding.service-setup.description') }}</p>

      <div class="service-setup-actions">
        <el-button
          type="primary"
          size="large"
          :loading="isInstalling"
          @click="installService"
          class="install-btn"
        >
          {{ isInstalling ? t('onboarding.service-setup.installing') : t('onboarding.service-setup.install-btn') }}
        </el-button>

        <el-button
          size="large"
          :disabled="isInstalling"
          @click="skipInstallation"
          class="skip-btn"
        >
          {{ t('onboarding.service-setup.skip-btn') }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.service-setup-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  width: 100%;
}

.service-setup-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-width: 600px;
  padding: 40px 20px;
}

.service-setup-title {
  font-size: 32px;
  font-weight: 600;
  margin-bottom: 20px;
  color: var(--text-color);
}

.service-setup-description {
  font-size: 18px;
  line-height: 1.6;
  margin-bottom: 40px;
  color: var(--text-color);
  opacity: 0.8;
}

.service-setup-actions {
  display: flex;
  gap: 20px;
  justify-content: center;
}

.install-btn {
  min-width: 180px;
  --el-button-bg-color: var(--left-item-selected-bg);
  --el-button-hover-bg-color: var(--left-item-selected-bg);
  --el-button-active-bg-color: var(--left-item-selected-bg);
  --el-button-border-color: transparent;
  --el-button-hover-border-color: transparent;
  --el-button-active-border-color: transparent;
  --el-button-text-color: var(--text-color);
  --el-button-hover-text-color: var(--text-color);
  --el-button-active-text-color: var(--text-color);
  box-shadow: var(--left-nav-shadow);
}

.skip-btn {
  min-width: 120px;
  background: var(--left-nav-btn-active-bg);
  border: 1px solid var(--left-nav-btn-active-bg);
  color: var(--text-color);
  box-shadow: var(--left-nav-shadow);
}

.skip-btn:hover {
  background: var(--left-nav-btn-hover-bg);
  border-color: var(--left-nav-btn-hover-bg);
}
</style>
