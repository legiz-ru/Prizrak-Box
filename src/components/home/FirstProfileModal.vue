<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useOnboardingStore } from '@/store/onboardingStore';
import IconInfoCircle from '~icons/tabler/info-circle';

const { t } = useI18n();
const onboardingStore = useOnboardingStore();

interface Props {
  visible: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>();

const localVisible = ref(props.visible);

// Синхронизация с родительским компонентом
watch(() => props.visible, (newVal) => {
  localVisible.value = newVal;
});

watch(localVisible, (newVal) => {
  emit('update:visible', newVal);
});

// Проверяем, что переводы загружены
const message = computed(() => t('onboarding.first-profile-info.message'));
const hasContent = computed(() => {
  const msg = message.value;
  return msg && msg !== 'onboarding.first-profile-info.message' && msg.length > 0;
});

// Закрыть модальное окно
function closeModal() {
  onboardingStore.markFirstProfileInfoShown();
  localVisible.value = false;
}
</script>

<template>
  <!-- Esc and overlay clicks are ignored: the info must be acknowledged. -->
  <UiNotice v-if="hasContent"
            v-model="localVisible"
            tone="info"
            :width="440"
            :icon="IconInfoCircle"
            :close-on-esc="false"
            :close-on-overlay="false"
            :title="t('onboarding.first-profile-info.title')">
    <span class="first-profile-text">{{ message }}</span>
    <template #actions>
      <button type="button" class="px-btn px-btn--primary first-profile-ok" @click="closeModal">
        {{ t('onboarding.first-profile-info.ok') }}
      </button>
    </template>
  </UiNotice>
</template>

<style scoped>
.first-profile-text {
  font-size: 14px;
  line-height: 1.6;
}

.first-profile-ok {
  flex: 0 0 auto !important;
  min-width: 120px;
  margin: 0 auto;
  padding: 10px 24px;
  font-size: 14px;
}
</style>
