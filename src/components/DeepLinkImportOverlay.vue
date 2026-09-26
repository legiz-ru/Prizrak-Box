<template>
  <Teleport to="body">
    <transition name="deeplink-import-fade">
      <div v-if="isImporting" class="deeplink-overlay">
        <div class="deeplink-box" role="dialog" aria-modal="true" aria-live="assertive" :aria-label="message" tabindex="-1">
          <div class="deeplink-spinner" aria-hidden="true"></div>
          <span class="deeplink-message">{{ message }}</span>
          <span class="deeplink-hint">{{ cancelHint }}</span>
          <button type="button" class="px-btn px-btn--danger-soft px-btn--block deeplink-cancel" @click="deepLinkImportStore.cancelImport">
            {{ cancelText }}
          </button>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import {computed, onBeforeUnmount, watch} from 'vue';
import {storeToRefs} from 'pinia';
import {useI18n} from 'vue-i18n';
import {useDeepLinkImportStore} from '@/store/deepLinkStore';
import {registerModal} from '@/components/ui/services';

const deepLinkImportStore = useDeepLinkImportStore();
const {isImporting, message, cancelLabel} = storeToRefs(deepLinkImportStore);
const {t} = useI18n();

const cancelText = computed(() => cancelLabel.value || t('profiles.deeplink.cancel-import'));
const cancelHint = computed(() => t('profiles.deeplink.cancel-hint'));

// Esc cancels the import (it sits on top of every other dialog).
let unregister: (() => void) | null = null;
watch(isImporting, (active) => {
  unregister?.();
  unregister = active ? registerModal(() => deepLinkImportStore.cancelImport(), () => true) : null;
}, {immediate: true});
onBeforeUnmount(() => unregister?.());
</script>

<style scoped>
.deeplink-overlay {
  position: fixed;
  inset: 0;
  z-index: 95;
  background: rgba(0, 0, 0, .55);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  -webkit-app-region: no-drag;
  --wails-draggable: no-drag;
}

.deeplink-box {
  width: 100%;
  max-width: 400px;
  background: var(--dialog-bg);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, .4);
  padding: 34px 28px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  color: var(--text);
}

.deeplink-spinner {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  border: 4px solid var(--border);
  border-top-color: var(--accent);
  animation: px-spin 1s linear infinite;
  margin-bottom: 22px;
}

.deeplink-message {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 8px;
}

.deeplink-hint {
  font-size: 13px;
  color: var(--text-2);
  margin-bottom: 20px;
}

.deeplink-cancel {
  border: 1px solid color-mix(in srgb, var(--error) 50%, transparent);
  padding: 10px 0;
  font-size: 14px;
}

.deeplink-import-fade-enter-active,
.deeplink-import-fade-leave-active {
  transition: opacity .2s ease;
}

.deeplink-import-fade-enter-from,
.deeplink-import-fade-leave-to {
  opacity: 0;
}
</style>
