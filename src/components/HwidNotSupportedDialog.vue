<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { useHwidStatusStore } from '@/store/hwidStatusStore';
import IconAlertCircle from '~icons/tabler/alert-circle';

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
  <UiNotice :model-value="visible"
            tone="warning"
            icon-size="lg"
            :width="420"
            :icon="IconAlertCircle"
            :title="t('hwid.not-supported.title')"
            @update:model-value="(v: boolean) => { if (!v) close() }">
    <span class="hwid-text">{{ t('hwid.not-supported.message') }}</span>
    <template #actions>
      <button type="button" class="px-btn" @click="close">{{ t('cancel') }}</button>
      <button type="button" class="px-btn px-btn--primary" @click="goToSettings">{{ t('hwid.not-supported.go-settings') }}</button>
    </template>
  </UiNotice>
</template>

<style scoped>
.hwid-text {
  font-size: 14px;
  line-height: 1.6;
  text-wrap: pretty;
}
</style>
