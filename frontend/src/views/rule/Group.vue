<script setup lang="ts">
import {VAceEditor} from "vue3-ace-editor";
import "ace-builds/src-noconflict/ace";
import "ace-builds/src-noconflict/ext-searchbox"; // 查找替换
import "ace-builds/src-noconflict/mode-yaml"; // YAML 支持
import "ace-builds/src-noconflict/ext-beautify";
import "ace-builds/src-noconflict/ext-language_tools"; // YAML 支持
import "ace-builds/src-noconflict/theme-monokai"; // 主题支持
import createApi from "@/api";
import {useI18n} from "vue-i18n";
import {pError, pLoad, pSuccess} from "@/util/pLoad";
import {useMenuStore} from "@/store/menuStore";
import {useProxiesStore} from "@/store/proxiesStore";
import {getTemplateTitle} from "@/util/format";

// 编辑器使用
const editorOptions = {
  showPrintMargin: false,
};
// 编辑器显示内容
const yamlContent = ref("");

// 当前页面使用store
const menuStore = useMenuStore();
const proxiesStore = useProxiesStore();

// i18n
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);


// Template列表
let tList = reactive([]);
// Template
let now = reactive({
  id: "",
  title: "m1",
  selected: false
})

const innerTemplate = ['m1', 'm2', 'm3']
// 是否可删除
const canDelete = ref(false)

function isDefault(data: any) {
  return innerTemplate.indexOf(data) !== -1
}

// 添加逻辑
const addVisible = ref(false)
const isNowAdd = ref(false)
const addForm = reactive({
  content: '',
})

const initPage = async () => {
  // 初始化
  tList = await api.getTemplateList();
  Object.assign(now, tList[0]);

  // 处理选中项
  for (const item of tList) {
    canDelete.value = !isDefault(item.title);
    if (item.selected) {
      Object.assign(now, item);
      break;
    }
  }

  // 处理编辑器内容
  yamlContent.value = await api.getTemplateById(now.id);
}

onMounted(initPage);

// Template 下拉列表逻辑
const isDropdownOpen = ref(false);
const selectOption = async (item: any) => {
  Object.assign(now, item);
  // 处理编辑器内容
  yamlContent.value = await api.getTemplateById(item.id);
  isDropdownOpen.value = false;

  canDelete.value = !isDefault(item.title);
};

// 添加逻辑
const addTemplate = async () => {
  if (!addForm.content) {
    pError(t('profiles.edit.title-tip'))
    return
  }
  Object.assign(now, {
    id: "",
    title: addForm.content,
    selected: false
  });
  yamlContent.value = ""
  addVisible.value = false
  canDelete.value = false;
}

// 删除逻辑
const deleteTemplate = async () => {
  if (!now.id) {
    return
  }
  try {
    await api.deleteTemplateById(now.id);
    await initPage()
    pSuccess(t('rule.group.delete.success'))
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }
}

// 保存逻辑
const saveTemplate = async () => {
  const trim = yamlContent.value.trim();
  if (!trim) {
    pError(t('rule.group.add.tip'))
    return
  }

  await pLoad(t('rule.group.save-ing'), async () => {
    try {
      // 测试
      await api.testTemplate({
        data: trim,
      });
      // 如果ID存在进行更新
      if (now.id) {
        await api.updateTemplate({
          data: trim,
          template: now,
        });
        // 如果是启用中的 进行切换
        if (now.selected) {
          await api.switchTemplate(now);
          proxiesStore.active = ""
          api.getRuleNum().then((res) => {
            menuStore.setRuleNum(res);
          });
        }
        pSuccess(t('rule.success'))
      } else {
        // 如果ID不存在进行添加
        await api.createTemplate({
          data: trim,
          title: now.title,
        });

        tList = await api.getTemplateList();
        for (const item of tList) {
          if (now.title == item.title) {
            canDelete.value = true;
            Object.assign(now, item);
            break;
          }
        }

        pSuccess(t('rule.group.add.success'))
      }
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    }
  })
}

// 切换逻辑
const switchTemplate = async () => {
  if (!now.id) {
    return
  }
  await pLoad(t('rule.group.switch.ing'), async () => {
    try {
      await api.switchTemplate(now);
      tList = await api.getTemplateList();

      await api.waitRunning()
      pSuccess(t('rule.group.switch.success'))

      proxiesStore.active = ""
      api.getRuleNum().then((res) => {
        menuStore.setRuleNum(res);
      });
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    }
  })
}


</script>

<template>
  <div class="group">
    <el-space class="op">
      <div class="dropdown">
        <button class="dropdown-btn" @click="isDropdownOpen = !isDropdownOpen">
          {{ getTemplateTitle(t, now.title) }}
        </button>
        <ul v-if="isDropdownOpen" class="dropdown-list">
          <li
              :key="item.id + index"
              @click="selectOption(item)"
              class="dropdown-item"
              v-for="(item, index) in tList"
          >
            {{ getTemplateTitle(t, item.title) }}
          </li>
        </ul>
      </div>
      <el-divider direction="vertical" border-style="dashed"/>
      <el-button @click="saveTemplate">
        {{ t("save") }}
      </el-button>
      <el-button @click="addVisible=true;addForm.content=''">
        {{ t("add") }}
      </el-button>
      <el-button @click="deleteTemplate" v-if="canDelete">
        {{ t("delete") }}
      </el-button>
      <el-divider direction="vertical" border-style="dashed"/>
      <el-text :class="now.selected ? 'sf' : 'st'">{{ t("off") }}</el-text>
      <el-switch
          @click="switchTemplate"
          v-model="now.selected"
          :disabled="!now.id"
          class="set-switch"/>
      <el-text :class="now.selected ? 'st' : 'sf'">{{ t("on") }}</el-text>
    </el-space>

    <VAceEditor
        v-model:value="yamlContent"
        lang="yaml"
        theme="monokai"
        :options="editorOptions"
        style="width: 100%; height: calc(100vh - 325px)"
        class="editor"
    />
  </div>


  <el-dialog v-model="addVisible"
             :title="t('add')"
             width="520"
             draggable
             center
  >
    <el-form :model="addForm" label-position="top">
      <el-form-item :label="t('rule.group.add.title')">
        <el-input
            :rows="3"
            type="text"
            autocapitalize="off"
            autocomplete="off"
            spellcheck="false"
            :placeholder="t('rule.group.add.placeholder')"
            v-model="addForm.content"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="addVisible = false">
          {{ t('cancel') }}
        </el-button>
        <el-button
            :loading="isNowAdd"
            type="primary"
            @click="addTemplate">
          {{ t('confirm') }}
        </el-button>
      </div>
    </template>
  </el-dialog>

</template>

<style scoped>
.group {
  width: 95%;
  margin-left: 10px;
  margin-top: 5px;
}

.op {
  margin-top: 2px;
}

.dropdown {
  position: relative;
  display: inline-block;
}

.dropdown-btn {
  background: transparent;
  color: var(--text-color);
  border: 2px solid var(--text-color);
  padding: 5px 10px;
  cursor: pointer;
  font-size: 15px;
  outline: none;
  border-radius: 8px;
  min-width: 150px;
}

.dropdown-btn:hover {
  opacity: 0.8;
}

.dropdown-list {
  position: absolute;
  background: var(--skin-bg-color);
  border: 2px solid var(--text-color);
  margin-top: 4px;
  padding: 0;
  list-style: none;
  min-width: 146px;
  z-index: 20;
  border-radius: 8px;
  font-size: 15px;
  text-align: center;
}

.dropdown-item {
  color: var(--text-color);
  padding: 8px;
  cursor: pointer;
}

.dropdown-item:hover {
  background: var(--skin-hover-color);
}

.set-switch {
  margin-left: 10px;
  --el-switch-border-color: var(--text-color);
  --el-switch-on-color: var(--left-item-selected-bg);
  --el-switch-off-color: transparent;
}

:deep(.el-switch__core) {
  width: 46px;
  height: 26px;
  border-radius: 12px;
  border: 2px solid var(--text-color);
}

:deep(.el-switch__core .el-switch__action) {
  margin-left: 2px;
}

:deep(.el-switch.is-checked .el-switch__core .el-switch__action) {
  left: calc(100% - 21px);
}

.op :deep(.el-button) {
  padding: 2px 10px;
  --el-button-bg-color: transparent;
  --el-button-text-color: var(--text-color);
  --el-button-hover-text-color: var(--left-item-selected-bg);
  --el-button-hover-bg-color: var(--text-color);
}

.st {
  color: var(--top-hr-color);
}

.sf {
  color: var(--text-color);
}

.editor {
  margin-top: 25px;
}

:deep(.ace_editor) {
  border: 2px solid var(--text-color);
  border-radius: 8px;
  font: 15px "Twemoji", "Monaco", "Menlo", "Ubuntu Mono", "Consolas",
  "Source Code Pro", "source-code-pro", monospace;
}

:deep(.ace_gutter) {
  border-top-left-radius: 8px;
  border-bottom-left-radius: 8px;
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

:deep(.ace_button:hover) {
  color: black;
}
</style>
