<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { Browser } from '@/runtime';
import { useHwidStatusStore } from '@/store/hwidStatusStore';
import IconAlertOctagon from '~icons/tabler/alert-octagon';

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
  <UiNotice :model-value="visible"
            tone="error"
            icon-size="lg"
            :width="420"
            :icon="IconAlertOctagon"
            :title="t('hwid.max-devices.title')"
            @update:model-value="(v: boolean) => { if (!v) close() }">
    <span class="hwid-text">{{ t('hwid.max-devices.message') }}</span>
    <template #actions>
      <button type="button" class="px-btn" @click="close">{{ t('cancel') }}</button>
      <button v-if="supportUrl" type="button" class="px-btn px-btn--primary" @click="openSupport">
        <icon-tabler-headset width="15" height="15"/>
        {{ t('hwid.max-devices.support') }}
      </button>
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
