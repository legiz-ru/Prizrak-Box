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
//
// useWhite === "фон тёмный, текст белый". Element Plus включает тёмную тему по
// классу `dark` на <html> — значит он должен совпадать с useWhite, а не быть
// обратным ему: иначе диалоги, селекты и тултипы приезжают светлыми поверх
// тёмного приложения (и наоборот). Раньше здесь было наоборот.
//
// Атрибут data-theme трогать нельзя: в basic.css `[data-theme="dark"]`
// исторически означает ровно противоположное — «фон светлый». Имя legacy,
// значения под него уже написаны.
const changeTheme = (useWhite: boolean) => {
  document.documentElement.classList.toggle('dark', useWhite);
  document.documentElement.setAttribute('data-theme', useWhite ? '' : 'dark');
}

//
onMounted(()=>{
  changeTheme(menuStore.useWhite)
})

// 监控黑白切换
watch(() => menuStore.useWhite, changeTheme);
</script>

<template>
  <div class="bottom-text" :class="{ 'bottom-text--collapsed': menuStore.navCollapsed }">

    <Off></Off>
    <Language></Language>
    <Skin></Skin>

  </div>
</template>

<style scoped>
.bottom-text {
  margin-top: auto;
  margin-bottom: 18px;
  margin-left: 22px;
  width: 185px;
  display: flex;
  justify-content: center;
  gap: 16px;
  color: var(--text-color);
  font-size: 20px;
}

/* Collapsed sidebar: 185px of horizontal room shrinks to an icon column, so
   Off/Language/Skin no longer fit side by side — stack them instead. */
.bottom-text--collapsed {
  flex-direction: column;
  align-items: center;
  width: 56px;
  gap: 20px;
}
</style>