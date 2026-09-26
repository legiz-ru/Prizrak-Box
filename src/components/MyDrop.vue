<template>
  <Teleport to="body">
    <Transition name="px-drop">
      <div v-if="webStore.dnd" class="drop-overlay" @click="webStore.dnd = false">
        <div class="drop-dialog" role="dialog" aria-modal="true" :aria-label="t('drag.hear')" tabindex="-1" @click.stop>
          <div class="drop-zone">
            <icon-tabler-cloud-upload width="32" height="32" class="drop-zone__icon"/>
            <span class="drop-zone__title">{{ t("drag.hear") }}</span>
            <span class="drop-zone__hint">{{ t("drag.formats") }}</span>
          </div>
          <button type="button" class="px-btn px-btn--primary px-btn--pill drop-open" @click="openImportDialog">
            <icon-tabler-folder-open width="15" height="15"/>
            {{ t("drag.open") }}
          </button>
          <input
              ref="importInputRef"
              type="file"
              accept=".yaml,.yml,.age"
              hidden
              @change="handleImportFile"
          />
        </div>
      </div>
    </Transition>
  </Teleport>

  <UiModal v-model="ageKeyDialogVisible"
           :title="t('age.file.title')"
           :icon="IconKey"
           :width="420"
           :z-index="72"
           :close-on-overlay="false"
           @close="cancelAgeFileImport">
    <span class="age-file-hint">{{ t('age.file.hint') }}</span>
    <label class="age-file-field">
      <icon-tabler-key width="15" height="15"/>
      <input v-model="ageKeyInput"
             :placeholder="t('age.profile.keyPlaceholder')"
             :aria-label="t('age.profile.keyPlaceholder')"
             autocapitalize="off"
             autocomplete="off"
             spellcheck="false"
             @keydown.enter.prevent="confirmAgeFileImport">
    </label>
    <template #footer>
      <button type="button" class="px-btn" @click="cancelAgeFileImport">{{ t('cancel') }}</button>
      <button type="button" class="px-btn px-btn--primary" :disabled="!ageKeyInput.trim()" @click="confirmAgeFileImport">{{ t('confirm.label') }}</button>
    </template>
  </UiModal>
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
import {registerModal} from "@/components/ui/services";
import IconKey from "~icons/tabler/key";

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

// The drop mask behaves like a dialog: Esc or a click outside the box closes it.
let unregisterMask: (() => void) | null = null;
watch(() => webStore.dnd, (open) => {
  unregisterMask?.();
  unregisterMask = open ? registerModal(() => { webStore.dnd = false; }, () => true) : null;
}, {immediate: true});
onBeforeUnmount(() => unregisterMask?.());

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
.drop-overlay {
  position: fixed;
  inset: 0;
  z-index: 75;
  background: rgba(0, 0, 0, .5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  -webkit-app-region: no-drag;
  --wails-draggable: no-drag;
}

.drop-dialog {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.drop-zone {
  width: 400px;
  max-width: calc(100vw - 40px);
  padding: 34px 20px;
  border: 2px dashed var(--border);
  border-radius: 12px;
  background: var(--dialog-bg);
  box-shadow: 0 24px 60px rgba(0, 0, 0, .4);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  text-align: center;
  color: var(--text);
}

.drop-zone__icon {
  color: var(--text-3);
}

.drop-zone__title {
  font-size: 14px;
  font-weight: 600;
}

.drop-zone__hint {
  font-size: 12px;
  color: var(--text-3);
}

.drop-open {
  padding: 9px 24px;
}

.age-file-hint {
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.5;
}

.age-file-field {
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 12px;
  background: var(--input-bg);
  color: var(--text-3);
}

.age-file-field:focus-within {
  border-color: var(--accent);
}

.age-file-field input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  color: var(--text);
  font-size: 13px;
  outline: none;
}

.px-drop-enter-active, .px-drop-leave-active {
  transition: opacity .15s ease;
}

.px-drop-enter-from, .px-drop-leave-to {
  opacity: 0;
}
</style>
