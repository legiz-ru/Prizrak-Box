<script setup lang="ts">
// Hotkey field: click (or Enter) and press the combination — it is recorded
// right in the field, no separate dialog. Esc cancels, leaving the field stops.
import {useI18n} from "vue-i18n";

const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ (e: 'update:modelValue', val: string): void }>();
const {t} = useI18n();

const isRecording = ref(false);

// macOS Sequoia (15.0–15.1, and reportedly recurring after screen unlock on
// later releases too — Apple bug FB15168205) silently drops Carbon global
// hotkeys whose modifiers are Option/Option+Shift alone: RegisterEventHotKey
// is called and appears to succeed, but the OS never delivers the keypress.
// Cmd or Ctrl in the combo sidesteps the restriction entirely, so warn (not
// block, since the combo does work on unaffected macOS versions) whenever
// the recorded combo would fall into the broken set.
const isMac = /Mac OS X|Macintosh/i.test(navigator.userAgent || '');

// True when `combo` (e.g. "Alt+Shift+M") uses only Alt/Shift as modifiers —
// the combination macOS silently swallows (see isMac comment above).
function isOptionOnlyCombo(combo: string): boolean {
  if (!combo) return false;
  const mods = combo.split('+').slice(0, -1).map(m => m.toLowerCase());
  return mods.includes('alt') && !mods.includes('ctrl') && !mods.includes('cmd');
}

const showMacOptionWarning = computed(() => isMac && isOptionOnlyCombo(props.modelValue));

function startRecording() {
  isRecording.value = true;
}

function stopRecording() {
  isRecording.value = false;
}

function handleKey(e: KeyboardEvent) {
  if (!isRecording.value) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      startRecording();
    }
    return;
  }

  e.preventDefault();
  e.stopPropagation();

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

  // A bare key is not a global hotkey; keep listening for a modifier combo.
  if (parts.length > 1) {
    emit('update:modelValue', parts.join('+'));
    isRecording.value = false;
  }
}
</script>

<template>
  <div class="hotkey">
    <div class="hotkey-field"
         :class="{ 'is-recording': isRecording }"
         role="button"
         tabindex="0"
         :aria-label="t('setting.shortcut.edit') + ': ' + modelValue"
         :aria-pressed="isRecording ? 'true' : 'false'"
         v-tip="t('setting.shortcut.edit')"
         @click="startRecording"
         @keydown="handleKey"
         @blur="stopRecording">
      <icon-tabler-keyboard width="15" height="15" class="hotkey-field__icon"/>
      <span class="mono">{{ isRecording ? t('setting.shortcut.recording') : modelValue }}</span>
    </div>
    <span class="px-kbd-hint">{{ t('hotkey.record-hint') }}</span>
    <div v-if="showMacOptionWarning" class="px-alert px-alert--warning">
      <icon-tabler-alert-triangle width="15" height="15"/>
      <span>{{ t('setting.shortcut.macOptionWarning') }}</span>
    </div>
  </div>
</template>

<style scoped>
.hotkey {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.hotkey-field {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 190px;
  align-self: flex-start;
  padding: 7px 14px;
  border-radius: 999px;
  font-size: 13px;
  cursor: pointer;
  border: 1.5px solid var(--border);
  background: var(--panel-soft);
  color: var(--text);
  user-select: none;
}

.hotkey-field:hover {
  border-color: color-mix(in srgb, var(--accent) 60%, var(--border));
}

.hotkey-field.is-recording {
  border-color: var(--accent);
  color: var(--accent);
}

.hotkey-field__icon {
  flex-shrink: 0;
  color: var(--text-3);
}
</style>
