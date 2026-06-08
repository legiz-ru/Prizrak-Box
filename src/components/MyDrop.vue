<template>
  <div class="mask" @click="webStore.dnd = false" v-show="webStore.dnd">
    <div class="mask-card" @click.stop>
      <h3>{{ t("drag.hear") }}</h3>
      <el-button class="import-button" @click="openImportDialog">
        {{ t("drag.open") }}
      </el-button>
      <input
          ref="importInputRef"
          type="file"
          accept=".yaml,.yml,.age"
          hidden
          @change="handleImportFile"
      />
    </div>
  </div>

  <el-dialog
      v-model="ageKeyDialogVisible"
      :title="t('age.file.title')"
      width="420"
      draggable
      append-to-body
      :close-on-click-modal="false"
  >
    <div class="age-file-body">
      <p class="age-file-hint">{{ t('age.file.hint') }}</p>
      <el-input
          v-model="ageKeyInput"
          :placeholder="t('age.profile.keyPlaceholder')"
          autocapitalize="off"
          autocomplete="off"
          spellcheck="false"
          clearable
      >
        <template #prefix>
          <el-icon><icon-mdi-key-variant/></el-icon>
        </template>
      </el-input>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="cancelAgeFileImport">{{ t('cancel') }}</el-button>
        <el-button type="primary" :disabled="!ageKeyInput.trim()" @click="confirmAgeFileImport">{{ t('confirm') }}</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import {useWebStore} from "@/store/webStore.js";
import {pError, pLoad, pSuccess, pWarning} from "@/util/pLoad";
import {useI18n} from "vue-i18n";
import {Profile} from "@/types/profile.js";
import createApi from "@/api/index.js";
import {Events} from "@/runtime";
import {changeMenu} from "@/util/menu";
import {useRouter} from "vue-router";

const {t} = useI18n();
const webStore = useWebStore();
const router = useRouter();
// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const importInputRef = ref<HTMLInputElement | null>(null);
const ageKeyDialogVisible = ref(false);
const ageKeyInput = ref('');
let pendingAgeImport: { content: string; title: string } | null = null;

const openImportDialog = () => {
  importInputRef.value?.click();
};

async function doImportProfile(content: string, title: string, ageSecretKey?: string) {
  await pLoad(t("drag.add"), async () => {
    const p = new Profile();
    p.content = content;
    p.title = title;
    if (ageSecretKey) {
      p.ageSecretKey = ageSecretKey;
    }
    try {
      const pList = await api.addProfileFromInput(p);
      if (pList && pList.length > 0) {
        webStore.dProfile = pList;
        pSuccess(t("drag.success"));
        webStore.dnd = false;
        changeMenu("Profiles", router);

        api.getProfileList().then((list) => {
          Events.Emit({
            name: "profiles",
            data: list,
          });
        });
      }
    } catch (e) {
      if (e && typeof e === 'object' && 'message' in e && typeof e.message === 'string') {
        pError(e.message);
      } else {
        pError(t("drag.error"));
      }
    }
  });
}

async function confirmAgeFileImport() {
  if (!pendingAgeImport) return;
  const { content, title } = pendingAgeImport;
  const key = ageKeyInput.value.trim();
  ageKeyDialogVisible.value = false;
  ageKeyInput.value = '';
  pendingAgeImport = null;
  await doImportProfile(content, title, key);
}

function cancelAgeFileImport() {
  ageKeyDialogVisible.value = false;
  ageKeyInput.value = '';
  pendingAgeImport = null;
}

const handleImportFile = async (event: Event) => {
  const target = event.target as HTMLInputElement | null;
  const files = target?.files ? Array.from(target.files) : [];
  if (files.length === 0) {
    return;
  }

  if (files.length > 1) {
    pWarning(t("drag.size"));
    if (target) {
      target.value = '';
    }
    return;
  }

  const file = files[0];
  const isAgefile = file.name.toLowerCase().endsWith('.age');

  const reader = new FileReader();
  reader.onload = async (loadEvent) => {
    const content = (loadEvent.target?.result ?? '') as string;
    if (isAgefile) {
      pendingAgeImport = { content, title: file.name };
      ageKeyInput.value = '';
      ageKeyDialogVisible.value = true;
    } else {
      await doImportProfile(content, file.name);
    }
  };

  reader.onerror = (error) => {
    console.error(`Error reading ${file.name}:`, error);
    pError(t("drag.error"));
  };

  reader.readAsText(file);

  if (target) {
    target.value = '';
  }
};

onMounted(() => manageDragEvents("add"));
onUnmounted(() => manageDragEvents("remove"));

function manageDragEvents(action: any) {
  const method = action === "add" ? "addEventListener" : "removeEventListener";
  document.body[method]("dragenter", handleDragEnter);
  document.body[method]("dragover", preventDefault);
  document.body[method]("drop", handleDrop);
}

function handleDragEnter(e: any) {
  if (e.dataTransfer && e.dataTransfer.types.includes("Files")) {
    webStore.dnd = true;
  }
}

function preventDefault(e: any) {
  e.preventDefault();
}

function handleDrop(e: any) {
  e.preventDefault();
  webStore.dnd = false;

  const files = Array.from(e.dataTransfer.files);
  if (files.length > 1) {
    pWarning(t("drag.size"));
    return;
  }

  files.forEach((file: any) => {
    const isAgefile = (file.name as string).toLowerCase().endsWith('.age');
    const reader = new FileReader();

    reader.onload = async (event) => {
      const content = event.target.result as string;
      if (isAgefile) {
        pendingAgeImport = { content, title: file.name };
        ageKeyInput.value = '';
        ageKeyDialogVisible.value = true;
      } else {
        await doImportProfile(content, file.name);
      }
    };

    reader.onerror = (error) => {
      console.error(`Error reading ${file.name}:`, error);
    };

    reader.readAsText(file);
  });
}
</script>

<style scoped>
.mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  background: var(--skin-bg-color);
  display: flex;
  justify-content: center;
  align-items: center;
  color: var(--text-color);
  font-size: 1.5rem;
}

.mask-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

h3 {
  margin: 0;
  width: 450px;
  height: 100px;
  border: 2px dashed var(--text-color);
  text-align: center;
  padding-top: 70px;
  border-radius: 20px;
}

.import-button {
  min-width: 200px;
  --el-border-radius-base: 999px;
  border-radius: 999px;
}

.age-file-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.age-file-hint {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
