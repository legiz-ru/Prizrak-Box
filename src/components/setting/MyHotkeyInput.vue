<template>
  <!-- Кнопка-триггер: показывает текущее значение -->
  <button class="hotkey-trigger" @click="openDialog">
    {{ modelValue || 'Not set' }}
  </button>

  <!-- Диалог 2: запись комбинации -->
  <el-dialog
      v-model="dialogVisible"
      :title="$t('setting.shortcut.edit')"
      width="360"
      :close-on-press-escape="false"
      append-to-body
      @opened="onDialogOpened"
      @closed="onDialogClosed"
  >
    <div
        ref="recordingAreaRef"
        class="hotkey-area"
        :class="{ recording: isRecording }"
        tabindex="0"
        @click="startRecording"
        @keydown.prevent.stop="handleKey"
        @blur="stopRecording"
    >
      <span v-if="!isRecording">{{ pendingValue || $t('setting.shortcut.recording') }}</span>
      <span v-else class="recording-hint">{{ $t('setting.shortcut.recording') }}</span>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="cancel">{{ $t('cancel') }}</el-button>
        <el-button type="primary" :disabled="!pendingValue" @click="confirm">
          {{ $t('confirm') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ (e: 'update:modelValue', val: string): void }>();

const dialogVisible = ref(false);
const isRecording = ref(false);
const pendingValue = ref('');
const recordingAreaRef = ref<HTMLElement | null>(null);

function openDialog() {
  pendingValue.value = props.modelValue;
  dialogVisible.value = true;
}

function onDialogOpened() {
  nextTick(() => {
    recordingAreaRef.value?.focus();
    isRecording.value = true;
  });
}

function onDialogClosed() {
  isRecording.value = false;
  pendingValue.value = '';
}

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
    pendingValue.value = parts.join('+');
    isRecording.value = false;
  }
}

function confirm() {
  if (pendingValue.value) {
    emit('update:modelValue', pendingValue.value);
  }
  dialogVisible.value = false;
}

function cancel() {
  dialogVisible.value = false;
}
</script>

<style scoped>
.hotkey-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 140px;
  height: 32px;
  padding: 0 16px;
  border: 1.5px solid var(--el-border-color);
  border-radius: 999px;
  cursor: pointer;
  font-size: 14px;
  background: transparent;
  color: var(--el-text-color-primary);
  user-select: none;
  transition: border-color 0.2s, color 0.2s;
}

.hotkey-trigger:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.hotkey-area {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 56px;
  border: 2px dashed var(--el-border-color);
  border-radius: 20px;
  cursor: pointer;
  font-size: 20px;
  font-weight: 600;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  user-select: none;
  transition: border-color 0.2s, color 0.2s;
}

.hotkey-area:focus,
.hotkey-area.recording {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
  border-style: solid;
}

.recording-hint {
  opacity: 0.7;
  font-size: 15px;
  font-weight: 400;
  font-style: italic;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
