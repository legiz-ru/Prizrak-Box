<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { Browser } from '@/runtime';
import { useHwidStatusStore } from '@/store/hwidStatusStore';

const { t } = useI18n();
const hwidStatusStore = useHwidStatusStore();

const visible = computed(() => hwidStatusStore.errorType === 'max-devices-reached');
const supportUrl = computed(() => hwidStatusStore.supportUrl);

function close() {
  hwidStatusStore.clear();
}

function openSupport() {
  if (supportUrl.value) {
    try {
      Browser.OpenURL(supportUrl.value);
    } catch {
      window.open(supportUrl.value, '_blank');
    }
  }
  hwidStatusStore.clear();
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('hwid.max-devices.title')"
    width="420"
    draggable
    :close-on-click-modal="true"
    @close="close"
  >
    <div class="hwid-dialog__body">
      <el-icon class="hwid-dialog__icon hwid-dialog__icon--danger">
        <icon-mdi-alert-octagon-outline />
      </el-icon>
      <p class="hwid-dialog__text">{{ t('hwid.max-devices.message') }}</p>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="close">{{ t('cancel') }}</el-button>
        <el-button
          v-if="supportUrl"
          type="primary"
          @click="openSupport"
        >
          {{ t('hwid.max-devices.support') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.hwid-dialog__body {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 8px 0;
  text-align: center;
}

.hwid-dialog__icon {
  font-size: 48px;
}

.hwid-dialog__icon--danger {
  color: var(--el-color-danger);
}

.hwid-dialog__text {
  font-size: 14px;
  line-height: 1.6;
  margin: 0;
  opacity: 0.8;
}

.dialog-footer {
  display: flex;
  justify-content: center;
  gap: 12px;
}
</style>
