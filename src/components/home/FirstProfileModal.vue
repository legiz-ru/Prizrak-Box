<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useOnboardingStore } from '@/store/onboardingStore';

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
  <n-modal
    v-if="hasContent"
    v-model:show="localVisible"
    preset="card"
    :title="t('onboarding.first-profile-info.title')"
    :bordered="false"
    :closable="false"
    :mask-closable="false"
    :close-on-esc="false"
    style="width: 520px"
  >
    <div class="modal-content">
      <p>{{ message }}</p>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <n-button
          type="primary"
          size="large"
          @click="closeModal"
          class="ok-btn"
        >
          {{ t('onboarding.first-profile-info.ok') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.modal-content {
  padding: 10px 0;
  font-size: 16px;
  line-height: 1.6;
  color: var(--text-color);
  text-align: center;
}

.dialog-footer {
  display: flex;
  justify-content: center;
}

.ok-btn {
  min-width: 120px;
}
</style>
