<template>
  <div
      class="hotkey-input"
      :class="{ recording: isRecording }"
      tabindex="0"
      @click="startRecording"
      @keydown.prevent="handleKey"
      @blur="stopRecording"
  >
    <span v-if="!isRecording">{{ displayValue }}</span>
    <span v-else class="recording-hint">{{ $t('setting.shortcut.recording') }}</span>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ (e: 'update:modelValue', val: string): void }>();

const isRecording = ref(false);
const displayValue = computed(() => props.modelValue || 'Not set');

function startRecording() {
  isRecording.value = true;
}

function stopRecording() {
  isRecording.value = false;
}

function handleKey(e: KeyboardEvent) {
  if (!isRecording.value) return;

  if (e.key === 'Escape') {
    isRecording.value = false;
    return;
  }

  const modifiers = ['Control', 'Alt', 'Shift', 'Meta'];
  if (modifiers.includes(e.key)) return;

  const parts: string[] = [];
  if (e.ctrlKey) parts.push('Ctrl');
  if (e.altKey) parts.push('Alt');
  if (e.shiftKey) parts.push('Shift');
  if (e.metaKey) parts.push('Cmd');

  const key = e.key.length === 1 ? e.key.toUpperCase() : e.key;
  parts.push(key);

  if (parts.length > 1) {
    emit('update:modelValue', parts.join('+'));
    isRecording.value = false;
  }
}
</script>

<style scoped>
.hotkey-input {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 140px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--text-color);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  outline: none;
  background: transparent;
  color: var(--text-color);
  user-select: none;
  transition: border-color 0.2s;
}

.hotkey-input:focus,
.hotkey-input.recording {
  border-color: var(--left-item-selected-bg);
  color: var(--left-item-selected-bg);
}

.recording-hint {
  opacity: 0.7;
  font-style: italic;
}
</style>
