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

// 主题切换
const changeTheme = (useWhite:boolean) => {
  if (useWhite) {
    document.documentElement.classList.remove('dark')
    document.documentElement.setAttribute('data-theme', '');
  } else {
    document.documentElement.classList.add('dark')
    document.documentElement.setAttribute('data-theme', 'dark');
  }
}

//
onMounted(()=>{
  changeTheme(menuStore.useWhite)
})

// 监控黑白切换
watch(() => menuStore.useWhite, changeTheme);
</script>

<template>
  <div class="bottom-text">

    <Off></Off>
    <Language></Language>
    <Skin></Skin>

  </div>
</template>

<style scoped>
.bottom-text {
  position: absolute;
  bottom: 18px;
  left: 22px;
  color: var(--text-color);
  font-size: 20px;
}
</style>