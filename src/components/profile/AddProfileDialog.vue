<script setup lang="ts">
// "Add profile" dialog shared by the Home toolbar, the welcome screen and the
// Profiles page. It only collects the input; each caller keeps its own add
// flow (switching to the new profile, refreshing, order sync…).
import {useI18n} from "vue-i18n";

const props = withDefaults(defineProps<{
  modelValue: boolean;
  loading?: boolean;
  /** Pre-filled content (clipboard import). */
  initialContent?: string;
}>(), {loading: false, initialContent: ''});

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'submit', payload: { content: string; ageSecretKey: string }): void;
}>();

const {t} = useI18n();

const content = ref('');
const useAgeKey = ref(false);
const ageSecretKey = ref('');

watch(() => props.modelValue, (open) => {
  if (!open) return;
  content.value = props.initialContent ?? '';
  useAgeKey.value = false;
  ageSecretKey.value = '';
}, {immediate: true});

const canSubmit = computed(() => !!content.value.trim() && !props.loading);

function submit() {
  if (!canSubmit.value) return;
  emit('submit', {
    content: content.value,
    ageSecretKey: useAgeKey.value ? ageSecretKey.value.trim() : '',
  });
}
</script>

<template>
  <UiModal :model-value="modelValue"
           :title="t('profiles.add')"
           divided
           @update:model-value="emit('update:modelValue', $event)">
    <textarea v-model="content"
              class="px-textarea add-textarea"
              rows="5"
              autocapitalize="off"
              autocomplete="off"
              spellcheck="false"
              :aria-label="t('profiles.placeholder')"
              :placeholder="t('profiles.placeholder')"
              @keydown.ctrl.enter.prevent="submit"
              @keydown.meta.enter.prevent="submit"></textarea>
    <label v-if="useAgeKey" class="age-field">
      <icon-tabler-key width="15" height="15"/>
      <input v-model="ageSecretKey"
             autocapitalize="off"
             autocomplete="off"
             spellcheck="false"
             :aria-label="t('age.profile.keyPlaceholder')"
             :placeholder="t('age.profile.keyPlaceholder')">
    </label>
    <template #footer-left>
      <button type="button"
              class="age-toggle"
              :class="{ 'is-on': useAgeKey }"
              :aria-pressed="useAgeKey ? 'true' : 'false'"
              :aria-label="useAgeKey ? t('age.profile.toggleOn') : t('age.profile.toggleOff')"
              v-tip="useAgeKey ? t('age.profile.toggleOn') : t('age.profile.toggleOff')"
              @click="useAgeKey = !useAgeKey">
        <icon-tabler-key width="17" height="17"/>
      </button>
    </template>
    <template #footer>
      <button type="button" class="px-btn" @click="emit('update:modelValue', false)">{{ t('cancel') }}</button>
      <button type="button" class="px-btn px-btn--primary" :disabled="!canSubmit" @click="submit">
        <UiSpinner v-if="loading"/>
        {{ t('add') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.add-textarea {
  min-height: 120px;
  font-family: inherit;
  resize: vertical;
  padding: 12px 14px;
}

.age-field {
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 12px;
  background: var(--input-bg);
  color: var(--text-3);
}

.age-field:focus-within {
  border-color: var(--accent);
}

.age-field input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  color: var(--text);
  font-size: 13px;
  outline: none;
}

.age-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-3);
  cursor: pointer;
  padding: 0;
}

.age-toggle:hover {
  background: var(--hover-bg);
}

.age-toggle.is-on {
  color: var(--accent);
}
</style>
