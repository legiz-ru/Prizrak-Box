<template>
  <Teleport to="body">
    <transition name="deeplink-import-fade">
      <div v-if="isImporting" class="deeplink-import-overlay" role="dialog" aria-live="assertive">
        <div class="deeplink-import-overlay__content">
          <div class="deeplink-import-overlay__spinner" aria-hidden="true"></div>
          <p class="deeplink-import-overlay__message">{{ message }}</p>
          <p class="deeplink-import-overlay__hint">{{ cancelHint }}</p>
          <el-button
            class="deeplink-import-overlay__cancel"
            type="danger"
            plain
            @click="deepLinkImportStore.cancelImport"
          >
            {{ cancelText }}
          </el-button>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import {computed, onBeforeUnmount, onMounted} from 'vue';
import {storeToRefs} from 'pinia';
import {useI18n} from 'vue-i18n';
import {useDeepLinkImportStore} from '@/store/deepLinkStore';

const deepLinkImportStore = useDeepLinkImportStore();
const {isImporting, message, cancelLabel} = storeToRefs(deepLinkImportStore);
const {t} = useI18n();

const cancelText = computed(() => cancelLabel.value || t('profiles.deeplink.cancel-import'));
const cancelHint = computed(() => t('profiles.deeplink.cancel-hint'));

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && isImporting.value) {
    event.preventDefault();
    deepLinkImportStore.cancelImport();
  }
};

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown);
});
</script>

<style scoped>
.deeplink-import-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background-color: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.deeplink-import-overlay__content {
  width: min(420px, 100%);
  border-radius: 20px;
  background: rgba(16, 16, 16, 0.85);
  box-shadow: 0 18px 38px rgba(0, 0, 0, 0.35);
  padding: 36px 32px 28px;
  text-align: center;
  color: #fff;
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.deeplink-import-overlay__spinner {
  width: 64px;
  height: 64px;
  margin: 0 auto 24px auto;
  border-radius: 50%;
  border: 4px solid rgba(255, 255, 255, 0.25);
  border-top-color: var(--el-color-primary, #409eff);
  animation: deeplink-import-spin 1s linear infinite;
}

.deeplink-import-overlay__message {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0 0 12px 0;
}

.deeplink-import-overlay__hint {
  font-size: 0.9rem;
  opacity: 0.75;
  margin: 0 0 20px 0;
}

.deeplink-import-overlay__cancel {
  width: 100%;
  font-weight: 600;
}

.deeplink-import-fade-enter-active,
.deeplink-import-fade-leave-active {
  transition: opacity 0.2s ease;
}

.deeplink-import-fade-enter-from,
.deeplink-import-fade-leave-to {
  opacity: 0;
}

@keyframes deeplink-import-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
