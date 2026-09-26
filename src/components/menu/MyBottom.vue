<script setup lang="ts">
import {useMenuStore} from "@/store/menuStore";
import {useRouter} from "vue-router";
import Language from "@/components/menu/Language.vue";
import Off from "@/components/menu/Off.vue";
import Skin from "@/components/menu/Skin.vue";

const menuStore = useMenuStore()
const router = useRouter()

// 上次打开页面
onBeforeMount(() => {
  if (menuStore.path) {
    router.push(menuStore.path)
  }
})
</script>

<template>
  <div class="bottom no-drag">
    <Language/>
    <Skin/>
    <Off/>
  </div>
</template>

<style scoped>
.bottom {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 4px;
}

.bottom :deep(.side-round) {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--side-bg);
  backdrop-filter: var(--side-blur);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-2);
  cursor: pointer;
  padding: 0;
  transition: background .15s, color .15s;
}

.bottom :deep(.side-round:hover),
.bottom :deep(.side-round[aria-expanded="true"]) {
  background: var(--hover-bg);
  color: var(--text);
}
</style>
