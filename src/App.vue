<template>
  <div class="cBody"
       :style="{ backgroundImage: currentBackground }"
       key="prizrak-box-body"
  >
    <div class="left">
      <div :class="isWindows?'top-title win':'top-title'">
        <div class="top-icon"></div>
        <span class="top-title-text">Prizrak-Box</span>
      </div>
      <div v-if="showUpdateBanner" class="update-banner">
        <div class="update-banner__content">
          <span class="update-banner__message">{{ updateBannerMessage }}</span>
          <icon-mdi-close-circle
              class="update-banner__dismiss"
              @click="dismissUpdateNotification"
          />
        </div>
        <el-button
            class="update-banner__open"
            type="primary"
            size="small"
            @click="openLatestRelease"
        >
          {{ t('updates.actions.open') }}
        </el-button>
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
    <DeepLinkImportOverlay/>
  </div>
</template>


<script setup lang="ts">
import {useMenuStore} from "@/store/menuStore";
import {preloadBackgroundImage} from "@/util/theme";
import DeepLinkImportOverlay from "@/components/DeepLinkImportOverlay.vue";
import {useUpdateStore} from "@/store/updateStore";
import {storeToRefs} from "pinia";
import {Browser} from "@/runtime";
import {useI18n} from "vue-i18n";

const menuStore = useMenuStore();
const updateStore = useUpdateStore();
const {t} = useI18n();

const {hasVisibleUpdate, latestUrl} = storeToRefs(updateStore);

const showUpdateBanner = computed(() => hasVisibleUpdate.value);
const updateBannerMessage = computed(() => t('updates.banner.message'));

const openExternalLink = (url: string) => {
  if (!url) {
    return;
  }

  try {
    Browser.OpenURL(url);
  } catch (error) {
    window.open(url, '_blank');
  }
};

const openLatestRelease = () => {
  const url = latestUrl.value || 'https://github.com/legiz-ru/Prizrak-Box/releases/latest';
  openExternalLink(url);
};

const dismissUpdateNotification = () => {
  updateStore.dismissCurrentUpdate();
};

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
  padding-right: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  -webkit-app-region: drag;
  user-select: none;
}

.win {
  padding-top: 32px;
}

.top-icon {
  width: 80px;
  height: 80px;
  background-image: url("@/assets/images/appicon.png");
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.top-title-text {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-color);
  text-align: center;
  line-height: 1.2;
  width: 100%;
}

.update-banner {
  margin: 12px 0 0 22px;
  padding: 14px 16px 16px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
  color: var(--text-color);
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 193px;
  align-self: flex-start;
  box-sizing: border-box;
}

.update-banner__content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.update-banner__message {
  flex: 1;
  font-size: 0.9rem;
  line-height: 1.4;
  opacity: 0.85;
}

.update-banner__open {
  align-self: stretch;
  width: 100%;
}

.update-banner__dismiss {
  color: inherit;
  opacity: 0.7;
  cursor: pointer;
  font-size: 1.1rem;
  flex-shrink: 0;
  transition: opacity 0.2s ease;
}

.update-banner__dismiss:hover {
  opacity: 1;
}
</style>