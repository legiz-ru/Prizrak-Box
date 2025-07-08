<template>
  <div class="cBody"
       :style="{ backgroundImage: currentBackground }"
       key="pandora-box-body"
  >
    <div class="left">
      <div :class="isWindows?'top-title win':'top-title'">
        <div class="top-icon"></div>
        <span class="top-title-text">Pandora-Box</span>
      </div>
      <MyEvent/>
      <MyNav/>
      <MyRule/>
      <MyProxy/>
      <MySecNav/>
      <MyBottom/>
    </div>

    <div class="right">
      <router-view/>
      <MyDrop/>
    </div>
  </div>
</template>


<script setup lang="ts">
import {useMenuStore} from "@/store/menuStore";
import {preloadBackgroundImage} from "@/util/theme";

const menuStore = useMenuStore();

// 当前背景
const currentBackground = ref("linear-gradient(to bottom, #434343, #000000)");

// 切换背景
const changeBg = (bg: string, useWhite: boolean) => {
  currentBackground.value = bg;
  menuStore.setUseWhite(useWhite);
}

const isWindows = ref(false)
onMounted(() => {
  preloadBackgroundImage(menuStore.background, changeBg);
  // @ts-ignore
  if (window["pxShowBar"]) {
    isWindows.value = true;
  }
});

// 监控背景切换
watch(() => menuStore.background, (nextBackground) => {
  preloadBackgroundImage(nextBackground, changeBg);
});

</script>

<style scoped>
.cBody {
  margin: 0;
  display: flex;
  height: 100vh;
  color: var(--text-color);
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  background-attachment: fixed;
  background-color: var(--blend-color);
  background-blend-mode: overlay;
  transition: background-image 0.6s ease-in-out, background-color 0.4s ease-in-out;
  position: relative;
  overflow: hidden;
}

.cBody::before {
  content: "";
  position: absolute;
  inset: 0;
  background-color: var(--body-blur-color);
  backdrop-filter: var(--body-blur);
  z-index: 0;
  pointer-events: none;
}

.left {
  padding-right: 18px;
  z-index: 1;
  display: flex;
  flex-direction: column;
}

.right {
  z-index: 1;
  overflow: hidden;
  position: relative;
  width: 100%;
  flex-grow: 1;
  margin: 15px 15px 15px 0;
  border-radius: 15px;
  background-color: var(--right-bg-color);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15),
  0 2px 8px rgba(0, 0, 0, 0.08);
  display: flex;
  flex-direction: column;
  border: var(--right-boder);
}

.top-title {
  padding-top: 40px;
  padding-left: 24px;
  display: flex;
  align-items: center;
  gap: 12px;
  -webkit-app-region: drag;
  user-select: none;
}

.win {
  padding-top: 32px;
}

.top-icon {
  width: 28px;
  height: 28px;
  background-image: url("@/assets/images/appicon.png");
  background-size: cover;
  background-position: center;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.top-title-text {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-color);
  flex-grow: 1;
  line-height: 1.2;
}
</style>