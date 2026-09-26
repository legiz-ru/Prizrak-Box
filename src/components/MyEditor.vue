<script setup lang="ts">
import {VAceEditor} from "vue3-ace-editor";
import "ace-builds/src-noconflict/ace";
import "ace-builds/src-noconflict/ext-searchbox"; // 查找替换
import "ace-builds/src-noconflict/mode-yaml"; // YAML 支持
import "ace-builds/src-noconflict/ext-beautify";
import "ace-builds/src-noconflict/ext-language_tools"; // YAML 支持
import "ace-builds/src-noconflict/theme-tomorrow_night";
import "ace-builds/src-noconflict/theme-tomorrow";
import {useI18n} from "vue-i18n";
import {useMenuStore} from "@/store/menuStore";

// 入参: load(holder) fills holder.value, save(content) persists it (returns false on failure).
const props = defineProps<{
  load: (content: { value: string }) => void | Promise<void>;
  save: (content: string) => unknown;
  hint?: string;
}>();

const {t} = useI18n();
const menuStore = useMenuStore();

// Editor colours follow the app's light/dark mode.
const theme = computed(() => menuStore.useWhite ? 'tomorrow_night' : 'tomorrow');

const editorOptions = {
  showPrintMargin: false,
  fontSize: 13,
  tabSize: 2,
  useSoftTabs: true,
};
// 编辑器显示内容
const yamlContent = ref("");
const loaded = ref("");

const dirty = computed(() => yamlContent.value !== loaded.value);
const saving = ref(false);
const saved = ref(false);
let savedTimer: number | undefined;

async function reload() {
  const holder = {value: ''};
  await props.load(holder);
  yamlContent.value = holder.value;
  loaded.value = holder.value;
}

async function doSave() {
  if (saving.value) return;
  saving.value = true;
  try {
    const result = await props.save(yamlContent.value);
    if (result !== false) {
      loaded.value = yamlContent.value;
      saved.value = true;
      window.clearTimeout(savedTimer);
      savedTimer = window.setTimeout(() => (saved.value = false), 1800);
    }
  } finally {
    saving.value = false;
  }
}

onMounted(reload);
onBeforeUnmount(() => window.clearTimeout(savedTimer));
</script>

<template>
  <div class="editor-wrap">
    <div v-if="hint" class="editor-hint">
      <icon-tabler-info-circle width="16" height="16"/>
      {{ hint }}
    </div>
    <VAceEditor
        v-model:value="yamlContent"
        lang="yaml"
        :theme="theme"
        :options="editorOptions"
        class="editor"
    />
    <div class="editor-actions">
      <button type="button" class="px-btn px-btn--pill" :disabled="!dirty || saving" @click="reload">{{ t('theme.reset') }}</button>
      <button type="button"
              class="px-btn px-btn--pill"
              :class="saved ? 'editor-saved' : 'px-btn--primary'"
              :disabled="(!dirty && !saved) || saving"
              @click="doSave">
        <UiSpinner v-if="saving" :size="12"/>
        <icon-tabler-check v-else-if="saved" width="14" height="14"/>
        {{ saved ? t('ui.saved') : t('save') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.editor-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.editor-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.5;
}

.editor-hint svg {
  flex-shrink: 0;
  color: var(--info);
}

.editor {
  flex: 1;
  min-height: 360px;
  width: 100%;
  border-radius: 12px;
  border: 1px solid var(--border);
  overflow: hidden;
}

.editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.editor-saved {
  background: var(--success);
  color: #fff;
  border-color: transparent;
}

:deep(.ace_editor) {
  font-family: 'SF Mono', Consolas, Menlo, monospace;
  line-height: 1.7;
}
</style>
