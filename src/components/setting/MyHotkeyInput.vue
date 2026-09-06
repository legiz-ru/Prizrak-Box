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
    <p v-if="showMacOptionWarning" class="mac-option-warning">
      {{ $t('setting.shortcut.macOptionWarning') }}
    </p>

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

// macOS Sequoia (15.0–15.1, and reportedly recurring after screen unlock on
// later releases too — Apple bug FB15168205) silently drops Carbon global
// hotkeys whose modifiers are Option/Option+Shift alone: RegisterEventHotKey
// is called and appears to succeed, but the OS never delivers the keypress.
// Cmd or Ctrl in the combo sidesteps the restriction entirely, so warn (not
// block, since the combo does work on unaffected macOS versions) whenever
// the recorded combo would fall into the broken set.
const isMac = /Mac OS X|Macintosh/i.test(navigator.userAgent || '');
const showMacOptionWarning = ref(false);

function openDialog() {
  pendingValue.value = props.modelValue;
  showMacOptionWarning.value = isMac && isOptionOnlyCombo(props.modelValue);
  dialogVisible.value = true;
}

// True when `combo` (e.g. "Alt+Shift+M") uses only Alt/Shift as modifiers —
// the combination macOS silently swallows (see isMac comment above).
function isOptionOnlyCombo(combo: string): boolean {
  if (!combo) return false;
  const mods = combo.split('+').slice(0, -1).map(m => m.toLowerCase());
  return mods.includes('alt') && !mods.includes('ctrl') && !mods.includes('cmd');
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
  showMacOptionWarning.value = false;
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
    showMacOptionWarning.value = isMac && isOptionOnlyCombo(pendingValue.value);
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

.mac-option-warning {
  margin: 10px 0 0;
  font-size: 12.5px;
  line-height: 1.4;
  color: var(--el-color-warning);
}
</style>
