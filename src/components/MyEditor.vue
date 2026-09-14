<script setup lang="ts">
import {VAceEditor} from "vue3-ace-editor";
import "ace-builds/src-noconflict/ace";
import "ace-builds/src-noconflict/ext-searchbox"; // 查找替换
import "ace-builds/src-noconflict/mode-yaml"; // YAML 支持
import "ace-builds/src-noconflict/ext-beautify";
import "ace-builds/src-noconflict/ext-language_tools"; // YAML 支持
import "ace-builds/src-noconflict/theme-monokai"; // 主题支持
import {useI18n} from "vue-i18n";

// 入参
const props = defineProps({
  load: Function,
  save: Function,
});

// 编辑器使用
const editorOptions = {
  showPrintMargin: false,
};
// 编辑器显示内容
const yamlContent = ref("");

// i18n
const {t} = useI18n();


onMounted(() => {
  // 加载编辑器内容
  props.load(yamlContent);
});

const disabled = ref(true)
let isFirst = true
// 监听内容变化
const onContentChange = () => {
  if (isFirst) {
    isFirst = false;
    return
  }

  disabled.value = false;
};

</script>

<template>
  <div class="group">
    <el-space class="op">
      <button
          class="px-btn"
          :disabled="disabled"
          @click="save(yamlContent)">
        {{ t("save") }}
      </button>
    </el-space>

    <VAceEditor
        v-model:value="yamlContent"
        lang="yaml"
        theme="monokai"
        :options="editorOptions"
        style="width: 100%; height: calc(100vh - 300px)"
        class="editor"
        @change="onContentChange"
    />
  </div>
</template>

<style scoped>
.group {
  margin-top: 5px;
}

.op {
  margin-top: 8px;
}

.st {
  color: var(--text-color);
}

.editor {
  /* Тот же зазор тулбар→контент, что у вкладок правил (var(--px-space-5)). */
  margin-top: var(--px-space-5);
}

/* Тот же край, что у редактора шаблона в разделе "Настройки ядра → Unified
   rule grouping": 1px var(--sub-card-border), а не собственная рамка 2px
   var(--text-color). */
:deep(.ace_editor) {
  border: 1px solid var(--sub-card-border);
  border-radius: var(--px-r-lg);
  font: 15px "Twemoji", "Monaco", "Menlo", "Ubuntu Mono", "Consolas",
  "Source Code Pro", "source-code-pro", monospace;
}

:deep(.ace_gutter) {
  border-top-left-radius: var(--px-r-lg);
  border-bottom-left-radius: var(--px-r-lg);
}

:deep(.ace_search.right) {
  width: 420px;
  margin-left: 10px;
  margin-right: -4px;
  padding-left: 8px;
  margin-top: 0;
  border: none;
  float: right;
  color: var(--text-color);
}

:deep(.ace_search_form, .ace_replace_form) {
  margin: 0;
}

:deep(.ace_search_form.ace_nomatch) {
  width: 374px;
}

:deep(.ace_button, .ace_searchbtn_close) {
  color: #cccccc;
}

/* Было color: black — эта панель поиска всегда лежит на тёмном фоне ace
   (monokai, независимо от темы приложения), и чёрная иконка при наведении
   пропадала на нём совсем. */
:deep(.ace_button:hover) {
  color: #ffffff;
}
</style>
