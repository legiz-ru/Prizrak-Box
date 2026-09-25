<script setup lang="ts">
import {VAceEditor} from "vue3-ace-editor";
import "ace-builds/src-noconflict/ace";
import "ace-builds/src-noconflict/ext-searchbox"; // 查找替换
import "ace-builds/src-noconflict/mode-yaml"; // YAML 支持
import "ace-builds/src-noconflict/ext-beautify";
import "ace-builds/src-noconflict/ext-language_tools"; // YAML 支持
import "ace-builds/src-noconflict/theme-tomorrow_night";
import "ace-builds/src-noconflict/theme-tomorrow";
import {confirm} from "@/components/ui";
import IconTrash from "~icons/tabler/trash";
import createApi from "@/api";
import {useI18n} from "vue-i18n";
import {pError, pLoad, pSuccess} from "@/util/pLoad";
import {useMenuStore} from "@/store/menuStore";
import {useProxiesStore} from "@/store/proxiesStore";
import {getTemplateTitle} from "@/util/format";

// 编辑器使用
const editorOptions = {
  showPrintMargin: false,
  fontSize: 13,
  tabSize: 2,
  useSoftTabs: true,
};
// 编辑器显示内容
const yamlContent = ref("");

// 当前页面使用store
const menuStore = useMenuStore();

const editorTheme = computed(() => menuStore.useWhite ? 'tomorrow_night' : 'tomorrow');

// A template being added has no id yet; it shows by its title until saved.
const templateOptions = computed(() => {
  const list = tList.value.map((item: any) => ({value: item.id as string, label: getTemplateTitle(t, item.title)}));
  if (!now.id) list.push({value: '', label: getTemplateTitle(t, now.title)});
  return list;
});

const onOff = computed({
  get: () => !!now.selected,
  set: (value: boolean) => {
    if (value === !!now.selected) return;
    now.selected = value;
    switchTemplate();
  },
});
const proxiesStore = useProxiesStore();

// i18n
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);


// Template列表 (a ref: the list is replaced after add/delete/switch)
const tList = ref<any[]>([]);
// Template
let now = reactive({
  id: "",
  title: "m1",
  selected: false
})

const innerTemplate = ['m1', 'm2', 'm3']
// 是否可删除
const canDelete = ref(false)
const isSwitchingTemplate = ref(false)

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
  tList.value = await api.getTemplateList();
  Object.assign(now, tList.value[0]);

  // 处理选中项
  for (const item of tList.value) {
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
const handleTemplateChange = async (id: string) => {
  const item = tList.value.find((i: any) => i.id === id);
  if (item) {
    Object.assign(now, item);
    // 处理编辑器内容
    yamlContent.value = await api.getTemplateById(item.id);
    canDelete.value = !isDefault(item.title);
  }
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

// 删除逻辑 — asks first (accepted product change)
const deleteTemplate = async () => {
  if (!now.id) {
    return
  }
  const ok = await confirm({
    title: t('confirm.delete-template.title'),
    text: t('confirm.delete-template.text'),
    okLabel: t('confirm.delete-template.ok'),
    icon: IconTrash,
  })
  if (!ok) {
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

        tList.value = await api.getTemplateList();
        for (const item of tList.value) {
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
  if (isSwitchingTemplate.value) {
    return
  }
  isSwitchingTemplate.value = true
  try {
    await pLoad(t('rule.group.switch.ing'), async () => {
      try {
        await api.switchTemplate(now);
        tList.value = await api.getTemplateList();

        // Sync now.selected from refreshed list so the toggle reflects actual state
        const updated = tList.value.find((i: any) => i.id === now.id);
        if (updated) Object.assign(now, updated);

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
  } finally {
    isSwitchingTemplate.value = false
  }
}


</script>

<template>
  <div class="group">
    <div class="group-bar">
      <UiSelect :model-value="now.id"
                :options="templateOptions"
                class="group-select"
                align="left"
                :min-width="180"
                :aria-label="t('rule.group.model')"
                @change="(id: string) => id && handleTemplateChange(id)"/>
      <span class="px-vdivider"></span>
      <button type="button" class="px-btn px-btn--primary" @click="saveTemplate">{{ t("save") }}</button>
      <button type="button" class="px-btn px-btn--input" @click="addVisible = true; addForm.content = ''">{{ t("add") }}</button>
      <button v-if="canDelete" type="button" class="px-btn px-btn--input" @click="deleteTemplate">{{ t("delete") }}</button>
      <UiPillTabs v-model="onOff"
                  class="group-toggle"
                  :options="[{value: false, label: t('off')}, {value: true, label: t('on')}]"
                  :aria-label="t('rule.group.title')"/>
    </div>

    <VAceEditor
        v-model:value="yamlContent"
        lang="yaml"
        :theme="editorTheme"
        :options="editorOptions"
        class="editor"
    />
  </div>

  <UiModal v-model="addVisible" :title="t('rule.group.add.new')" :width="420">
    <label class="px-field">
      <span class="px-field__label">{{ t('rule.group.add.title') }}</span>
      <input v-model="addForm.content"
             class="px-input"
             autocapitalize="off"
             autocomplete="off"
             spellcheck="false"
             :placeholder="t('rule.group.add.placeholder')"
             @keydown.enter.prevent="addTemplate">
    </label>
    <template #footer>
      <button type="button" class="px-btn" @click="addVisible = false">{{ t('cancel') }}</button>
      <button type="button" class="px-btn px-btn--primary" :disabled="isNowAdd" @click="addTemplate">{{ t('add') }}</button>
    </template>
  </UiModal>
</template>

<style scoped>
.group {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 0 28px 28px;
}

.group-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.group-select {
  width: 200px;
}

.group-select :deep(.px-select--field) {
  padding: 7px 10px 7px 12px;
  font-weight: 600;
}

.group-toggle {
  margin-left: auto;
}

.editor {
  flex: 1;
  min-height: 280px;
  width: 100%;
  border-radius: 12px;
  border: 1px solid var(--border);
}

:deep(.ace_editor) {
  font-family: 'SF Mono', Consolas, Menlo, monospace;
  line-height: 1.6;
}
</style>
