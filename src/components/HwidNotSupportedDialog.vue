<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { useHwidStatusStore } from '@/store/hwidStatusStore';

const { t } = useI18n();
const router = useRouter();
const hwidStatusStore = useHwidStatusStore();

const visible = computed(() => hwidStatusStore.errorType === 'not-supported');

function close() {
  hwidStatusStore.clear();
}

function goToSettings() {
  hwidStatusStore.clear();
  router.push('/Setting');
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('hwid.not-supported.title')"
    width="420"
    draggable
    :close-on-click-modal="true"
    @close="close"
  >
    <div class="hwid-dialog__body">
      <el-icon class="hwid-dialog__icon hwid-dialog__icon--warning">
        <icon-mdi-alert-circle-outline />
      </el-icon>
      <p class="hwid-dialog__text">{{ t('hwid.not-supported.message') }}</p>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="close">{{ t('cancel') }}</el-button>
        <el-button type="primary" @click="goToSettings">
          {{ t('hwid.not-supported.go-settings') }}
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

.hwid-dialog__icon--warning {
  color: var(--el-color-warning);
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
