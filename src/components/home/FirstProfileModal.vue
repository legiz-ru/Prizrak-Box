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
  <el-dialog
    v-if="hasContent"
    v-model="localVisible"
    :title="t('onboarding.first-profile-info.title')"
    width="520"
    center
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
  >
    <div class="modal-content">
      <p>{{ message }}</p>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button
          type="primary"
          size="large"
          @click="closeModal"
          class="ok-btn"
        >
          {{ t('onboarding.first-profile-info.ok') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.modal-content {
  padding: 10px 0;
  font-size: 16px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  text-align: center;
}

.ok-btn {
  min-width: 120px;
}
</style>
