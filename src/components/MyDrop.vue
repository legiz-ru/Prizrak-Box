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
          accept=".yaml,.yml"
          hidden
          @change="handleImportFile"
      />
    </div>
  </div>
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

const openImportDialog = () => {
  importInputRef.value?.click();
};

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
  const reader = new FileReader();
  reader.onload = async (loadEvent) => {
    await pLoad(t("drag.add"), async () => {
      const p = new Profile();
      p.content = loadEvent.target?.result ?? '';
      p.title = file.name;
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
    const reader = new FileReader();

    reader.onload = async (event) => {
      console.log(`Content of ${file.name}:`);
      // console.log(event.target.result);

      await pLoad(t("drag.add"), async () => {
        const p = new Profile();
        p.content = event.target.result;
        p.title = file.name;
        try {
          const pList = await api.addProfileFromInput(p);
          if (pList && pList.length > 0) {
            webStore.dProfile = pList;
            pSuccess(t("drag.success"));

            // 发送订阅配置数据
            api.getProfileList().then((list) => {
              Events.Emit({
                name: "profiles",
                data: list,
              });
            });
          }
        } catch (e) {
          if (e["message"]) {
            pError(e["message"]);
          }
        }
      });
    };

    reader.onerror = (error) => {
      console.error(`Error reading ${file.name}:`, error);
    };

    // 使用 readAsText 方法读取文件内容
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
  border-radius: 10px;
}

.import-button {
  min-width: 200px;
}
</style>
