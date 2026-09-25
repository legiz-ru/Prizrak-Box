<template>
  <div class="app-root" key="prizrak-box-body">
    <div v-if="menuStore.useBgImage" class="app-bg" aria-hidden="true">
      <div class="app-bg__image" :style="{ backgroundImage: currentBackground }"></div>
      <div class="app-bg__overlay"></div>
    </div>

    <header class="app-top drag">
      <MySearch v-if="menuStore.menu !== 'Proxies'"/>
      <MyTitleBar/>
    </header>

    <aside class="app-side drag">
      <div class="app-brand no-drag">
        <div class="app-brand__logo" :style="topIconStyle" role="img" :aria-label="topTitle"></div>
        <div class="app-brand__title ellipsis" v-tip="topTitle.length > 18 ? topTitle : ''">{{ topTitle }}</div>
      </div>
      <MyEvent/>
      <MyNav/>
      <MyRule :active-profile="activeProfile"/>
      <MyProxy/>
      <div class="app-side__spacer"></div>
      <MyBottom/>
    </aside>

    <main class="app-main">
      <router-view/>
      <MyDrop/>
    </main>

    <DeepLinkImportOverlay/>
    <HwidNotSupportedDialog/>
    <HwidMaxDevicesDialog/>
    <SubscriptionAlertModal/>

    <UiNotice :model-value="showUpdateDialog"
              tone="accent"
              :icon="IconDownload"
              :title="t('updates.notification.title')"
              @update:model-value="(v: boolean) => { if (!v) dismissUpdateNotification() }">
      <span>{{ t('updates.notification.message', {version: updateStore.latestDisplayName || t('updates.banner.version-unknown')}) }}</span>
      <template #actions>
        <button type="button" class="px-btn" @click="dismissUpdateNotification">{{ t('close') }}</button>
        <button type="button" class="px-btn px-btn--primary" @click="openLatestRelease">{{ t('updates.actions.open') }}</button>
      </template>
    </UiNotice>

    <UiToast/>
    <UiConfirm/>
    <UiTooltip/>
  </div>
</template>


<script setup lang="ts">
import {useMenuStore} from "@/store/menuStore";
import {preloadBackgroundImage, analyzeImage, type ImageTheme} from "@/util/theme";
import {imageTheme, useAppTheme} from "@/composables/useAppTheme";
import IconDownload from "~icons/tabler/download";
import {getCachedBg, setCachedBg, clearCachedBg} from "@/util/bgCache";
import {getCachedLogo, setCachedLogo, clearCachedLogo} from "@/util/logoCache";
import DeepLinkImportOverlay from "@/components/DeepLinkImportOverlay.vue";
import HwidNotSupportedDialog from "@/components/HwidNotSupportedDialog.vue";
import HwidMaxDevicesDialog from "@/components/HwidMaxDevicesDialog.vue";
import SubscriptionAlertModal from "@/components/SubscriptionAlertModal.vue";
import {useUpdateStore} from "@/store/updateStore";
import {storeToRefs} from "pinia";
import {Browser, Events} from "@/runtime";
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {useWebStore} from "@/store/webStore";
import {useSettingStore} from "@/store/settingStore";
import {getRendererOrigin, normalizeCustomBackground} from "@/util/customBackground";
import {WS} from "@/util/ws";
import {formatDate} from "@/util/format";
import {logLevel} from "@/composables/logLevel";
import {BACKEND_CONN_UPDATED_EVENT} from "@/util/backendConn";
import {
  checkPendingSubscriptionAlerts,
  notifySubscriptionAlertClicked,
  formatAlertText,
  type SubscriptionAlert,
} from "@/util/subscriptionAlerts";

const menuStore = useMenuStore();
const updateStore = useUpdateStore();
const webStore = useWebStore();
const settingStore = useSettingStore();
const {t} = useI18n();
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const rendererOrigin = getRendererOrigin();

const {hasVisibleUpdate, latestUrl} = storeToRefs(updateStore);

// A new release is announced once in a dialog; closing it dismisses that
// version (the old sidebar banner did the same with its × button).
const showUpdateDialog = computed(() => hasVisibleUpdate.value);

useAppTheme();
const defaultTitle = "Prizrak-Box";
const defaultLogo = new URL("@/assets/images/appicon.png", import.meta.url).href;

const activeProfile = ref<any | null>((() => {
  // Hydrate from the synchronous logo cache so a custom logo/title shows
  // instantly on launch (no flash of the default), before loadProfiles() runs.
  const cached = getCachedLogo();
  return cached ? {id: cached.id, logo: cached.logo, headerTitle: cached.title} : null;
})());
const hasCustomLogo = computed(() => {
  const logo = activeProfile.value?.logo;
  return typeof logo === "string" && logo.trim() !== "";
});
const topTitle = computed(() => {
  if (!hasCustomLogo.value) {
    return defaultTitle;
  }

  const title = activeProfile.value?.headerTitle;
  const trimmed = typeof title === "string" ? title.trim() : "";
  return trimmed || defaultTitle;
});
const logoUrl = computed(() => {
  const logo = activeProfile.value?.logo;
  const trimmed = typeof logo === "string" ? logo.trim() : "";
  return trimmed;
});
const logoFallback = ref(false);
const topLogo = computed(() => {
  if (!logoUrl.value || logoFallback.value) {
    return defaultLogo;
  }
  return logoUrl.value;
});
const topIconStyle = computed(() => ({
  backgroundImage: `url(${topLogo.value})`
}));

const preloadLogo = (url: string) => new Promise<boolean>((resolve) => {
  const img = new Image();
  const done = (ok: boolean) => {
    img.onload = null;
    img.onerror = null;
    resolve(ok);
  };
  img.onload = () => done(true);
  img.onerror = () => done(false);
  img.src = url;
});

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
  updateStore.dismissCurrentUpdate();
};

const dismissUpdateNotification = () => {
  updateStore.dismissCurrentUpdate();
};

// 当前背景
// Empty until the stored background has loaded, so an upgraded install never
// flashes the default picture before its own one.
const currentBackground = ref("none");

// 切换背景: the image itself plus its analysis (accent, light/dark for "auto").
const changeBg = (bg: string, theme: ImageTheme | null) => {
  currentBackground.value = bg;
  imageTheme.value = theme;
}

function isExternalBg(bg: string): boolean {
  const url = bg.match(/url\(['"]?(.*?)['"]?\)/)?.[1] ?? '';
  return url.startsWith('http://') || url.startsWith('https://');
}

// Capture the already-loaded img element via canvas — no second network request,
// so we always store the exact image that was displayed on screen.
function captureAndCache(img: HTMLImageElement, storageKey: string): void {
  if (!img.naturalWidth || !img.naturalHeight) return;
  try {
    const MAX_DIM = 1920;
    const scale = Math.min(1, MAX_DIM / Math.max(img.naturalWidth, img.naturalHeight));
    const w = Math.round(img.naturalWidth * scale);
    const h = Math.round(img.naturalHeight * scale);
    const canvas = document.createElement('canvas');
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.drawImage(img, 0, 0, w, h);
    const dataUrl = canvas.toDataURL('image/jpeg', 0.85);
    if (menuStore.background === storageKey) {
      setCachedBg(storageKey, dataUrl);
    }
  } catch (e) {
    console.warn('[bg-cache] canvas capture failed:', e);
  }
}

const applyBackground = (value: string) => {
  const normalized = normalizeCustomBackground(value, rendererOrigin);

  if (normalized && normalized.storageValue !== value) {
    menuStore.setBackground(normalized.storageValue);
  }

  const storageKey = normalized?.storageValue ?? value;
  const cssValue = normalized?.cssValue ?? value;

  const loadExternal = () => {
    preloadBackgroundImage(cssValue, (bg: string, theme: ImageTheme | null, img?: HTMLImageElement) => {
      changeBg(bg, theme);
      // img is the already-loaded element — capture it without a second request
      if (img && isExternalBg(bg)) {
        captureAndCache(img, storageKey);
      }
    });
  };

  // In-memory cache pre-loaded from userData/px-bg-cache.json during bootstrap
  const cachedDataUrl = getCachedBg(storageKey);
  if (cachedDataUrl) {
    const img = new Image();
    img.onload = () => {
      let theme: ImageTheme | null = null;
      try {
        theme = analyzeImage(img);
      } catch (e) {
        console.warn('[bg-cache] theme analysis failed for cached image:', e);
      }
      changeBg(`url('${cachedDataUrl}')`, theme);
    };
    img.onerror = () => {
      clearCachedBg();
      loadExternal();
    };
    img.src = cachedDataUrl;
    return;
  }

  loadExternal();
};

onMounted(() => {
  applyBackground(menuStore.background);
});

const applyProfile = (data: any | null) => {
  activeProfile.value = data;
  // Keep the logo cache in sync: store on apply/refresh, clear on rollback
  // (profile without a custom logo) so the next launch shows the right thing.
  const logo = typeof data?.logo === "string" ? data.logo.trim() : "";
  if (logo) {
    setCachedLogo(String(data?.id ?? ""), logo, typeof data?.headerTitle === "string" ? data.headerTitle : "");
  } else {
    clearCachedLogo();
  }
};

watch(logoUrl, async (url) => {
  logoFallback.value = false;
  if (!url) {
    return;
  }
  const ok = await preloadLogo(url);
  if (!ok) {
    logoFallback.value = true;
  }
});

const pickSelectedProfile = (list: any[]) => {
  if (!Array.isArray(list) || list.length === 0) {
    applyProfile(null);
    return;
  }

  const primary = list.find(item => item?.primary);
  const selected = primary ?? list.find(item => item?.selected);
  applyProfile(selected ?? list[0]);
};

const handleVueProfilesUpdate = (event: Event) => {
  const customEvent = event as CustomEvent;
  const detail = customEvent.detail;

  if (!detail || !Array.isArray(detail.profiles)) {
    return;
  }

  pickSelectedProfile(detail.profiles);
};

// Fires the native OS notification for one subscription alert. The Wails
// path (window.pxNotifyAlert) round-trips the click through Go — see
// SubscriptionAlertModal.vue's onBackendEvent — while Electron's Web
// Notification API fires the click callback directly in this same renderer,
// so it dispatches the click event itself.
function notifySubscriptionAlertOs(profileData: any, alert: SubscriptionAlert) {
  const title = formatAlertText(t, alert);
  const body = profileData?.title || profileData?.headerTitle || '';
  const clickDetail = {
    profileId: profileData?.id,
    kind: alert.kind,
    days: alert.days,
    percent: alert.percent,
  };

  // @ts-ignore
  const pxNotifyAlert = window.pxNotifyAlert;
  if (typeof pxNotifyAlert === 'function') {
    pxNotifyAlert(title, body, clickDetail);
    return;
  }

  const NotificationCtor = window.Notification;
  if (typeof NotificationCtor !== 'function') {
    return;
  }

  const show = () => {
    try {
      const notification = new NotificationCtor(title, {body});
      notification.onclick = () => {
        if (typeof window.focus === 'function') {
          window.focus();
        }
        notifySubscriptionAlertClicked(clickDetail);
      };
    } catch (error) {
      console.error('Failed to display subscription alert notification', error);
    }
  };

  if (NotificationCtor.permission === 'granted') {
    show();
  } else if (NotificationCtor.permission === 'default' && typeof NotificationCtor.requestPermission === 'function') {
    NotificationCtor.requestPermission().then((permission) => {
      if (permission === 'granted') show();
    }).catch(() => { /* ignore */ });
  }
}

const loadProfiles = async () => {
  try {
    const list = await api.getProfileList();
    if (Array.isArray(list)) webStore.profileList = list;
    pickSelectedProfile(list);
    void checkPendingSubscriptionAlerts(list, {
      enabled: settingStore.notifySubscriptionAlerts,
      notify: notifySubscriptionAlertOs,
      ack: (id: string) => api.ackSubscriptionAlert(id),
    });
    // On startup fProfile is empty (no id), so Proxies.vue skips fetching
    // proxy groups. Set it from the active profile so groups load correctly
    // on re-launch without requiring the user to manually re-select a profile.
    // Guard prevents re-entry: if fProfile already has an id, do nothing.
    if (!webStore.fProfile?.id) {
      const primary = list.find((item: any) => item?.primary);
      const selected = primary ?? list.find((item: any) => item?.selected);
      const profile = selected ?? list[0];
      if (profile?.id) {
        webStore.fProfile = toRaw(profile);
      }
    }
  } catch (error) {
    console.error("Failed to load profiles", error);
  }
};

watch(
    () => webStore.fProfile,
    async (data: any) => {
      if (data && Object.keys(data).length > 0) {
        await loadProfiles();
      }
    }
);

// 监控背景切换
watch(() => menuStore.background, (nextBackground) => {
  applyBackground(nextBackground);
});

let logWs: WS | null = null;

function buildLogUrl() {
  const level = logLevel.value;
  const levelParam = level ? `&level=${encodeURIComponent(level)}` : '';
  return webStore.wsUrl + "/logs?token=" + webStore.secret + levelParam;
}

function connectLog(clearOnReconnect = false) {
  if (logWs) {
    logWs.close();
    logWs = null;
  }
  if (clearOnReconnect) {
    webStore.clearLogs();
  }
  logWs = new WS(buildLogUrl(), null, (ev: MessageEvent) => {
    const parsedData = JSON.parse(ev.data);
    webStore.addLog({
      time: formatDate(new Date()),
      type: parsedData["type"].toUpperCase(),
      payload: parsedData["payload"],
    });
  });
}

watch(logLevel, (newVal, oldVal) => {
  if (newVal !== oldVal) {
    connectLog(true);
  }
});

// px can be restarted while the app keeps running (TUN service install /
// uninstall) and comes back on a different port/secret, which leaves this
// long-lived socket pointing at a process that no longer exists.
function reconnectLogAfterBackendChange() {
  connectLog(false);
}

onMounted(async () => {
  await loadProfiles();
  Events.On("profiles", (list: any[]) => {
    pickSelectedProfile(list);
  });
  window.addEventListener('vue-profiles-updated', handleVueProfilesUpdate as EventListener);
  window.addEventListener(BACKEND_CONN_UPDATED_EVENT, reconnectLogAfterBackendChange);

  connectLog(false);
});

onBeforeUnmount(() => {
  window.removeEventListener('vue-profiles-updated', handleVueProfilesUpdate as EventListener);
  window.removeEventListener(BACKEND_CONN_UPDATED_EVENT, reconnectLogAfterBackendChange);
});

</script>

<style scoped>
.app-root {
  display: flex;
  height: 100vh;
  width: 100%;
  overflow: hidden;
  position: relative;
  color: var(--text);
  background: var(--app-bg);
}

.app-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
}

.app-bg__image {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  background-color: var(--panel-soft);
  transition: background-image .3s;
}

.app-bg__overlay {
  position: absolute;
  inset: 0;
  background: var(--overlay);
}

.app-top {
  position: absolute;
  top: 0;
  left: 228px;
  right: 0;
  height: 64px;
  z-index: 6;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 14px 0 0;
}

.app-side {
  position: relative;
  z-index: 1;
  width: 228px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 52px 14px 16px;
  gap: 14px;
  overflow-y: auto;
  scrollbar-width: none;
  user-select: none;
}

.app-side::-webkit-scrollbar {
  display: none;
}

.app-side__spacer {
  flex: 1;
}

.app-brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.app-brand__logo {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  background-size: contain;
  background-position: center;
  background-repeat: no-repeat;
  filter: var(--logo-shadow);
}

.app-brand__title {
  max-width: 100%;
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
  text-shadow: var(--title-shadow);
  padding: 3px 12px;
  border-radius: 999px;
  background: var(--title-bg);
  backdrop-filter: var(--side-blur);
}

.app-main {
  position: relative;
  z-index: 1;
  flex: 1;
  min-width: 0;
  margin: 64px 14px 14px 0;
  border-radius: 12px;
  background: var(--panel-bg);
  border: 1px solid var(--border);
  backdrop-filter: blur(var(--ui-blur));
  box-shadow: 0 20px 50px rgba(0, 0, 0, .25);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
