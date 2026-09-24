<template>
  <button type="button"
          class="side-round"
          :aria-label="t('theme.label')"
          v-tip="t('theme.label')"
          @click="dialogOpen = true">
    <icon-tabler-palette width="17" height="17"/>
  </button>

  <UiModal v-model="dialogOpen"
           :width="580"
           :z-index="65"
           clear-overlay
           :aria-label="t('theme.label')"
           body-class="theme-body">
    <template #header>
      <icon-tabler-palette width="18" height="18" style="color:var(--accent);flex-shrink:0"/>
      <span class="theme-title">{{ t('theme.label') }}</span>
    </template>

    <div class="theme-row">
      <span class="theme-row__label">{{ t('theme.mode') }}</span>
      <UiPillTabs v-model="menuStore.themePref"
                  :options="modeOptions"
                  size="md"
                  stretch
                  class="theme-mode"
                  :aria-label="t('theme.mode')"/>
    </div>

    <div class="px-divider"></div>

    <div class="theme-row">
      <div class="theme-row__text">
        <span class="theme-row__title">{{ t('theme.use-image') }}</span>
        <span class="theme-row__desc">{{ t('theme.use-image-desc') }}</span>
      </div>
      <UiSwitch v-model="menuStore.useBgImage" :aria-label="t('theme.use-image')"/>
    </div>

    <template v-if="menuStore.useBgImage">
      <div class="theme-tiles" role="radiogroup" :aria-label="t('theme.use-image')">
        <div v-for="item in tiles"
             :key="item.id"
             role="radio"
             tabindex="0"
             class="theme-tile"
             :class="{ 'is-on': menuStore.bgTheme === item.id, 'is-custom-empty': item.custom && !item.thumb }"
             :style="item.thumb ? { backgroundImage: `url('${item.thumb}')` } : undefined"
             :aria-checked="menuStore.bgTheme === item.id ? 'true' : 'false'"
             :aria-label="item.label"
             v-tip="item.label"
             @click="changeBackground(item.option)"
             @keydown.enter.prevent="changeBackground(item.option)"
             @keydown.space.prevent="changeBackground(item.option)">
          <icon-tabler-photo-plus v-if="!item.thumb && item.custom" class="theme-tile__icon" width="18" height="18"/>
          <icon-tabler-dice-5 v-else-if="!item.thumb" class="theme-tile__icon" width="18" height="18"/>
          <span class="theme-tile__label" :class="{ 'has-thumb': !!item.thumb }">{{ item.label }}</span>
          <button v-if="item.custom"
                  type="button"
                  class="theme-tile__upload"
                  :aria-label="t('bg.upload')"
                  v-tip="t('bg.upload')"
                  @click.stop="triggerUpload(item.option)">
            <icon-tabler-upload width="13" height="13"/>
          </button>
        </div>
      </div>
      <span v-if="bgError" class="theme-error" role="alert">{{ bgError }}</span>

      <div class="theme-sliders">
        <label class="theme-slider">
          <span>{{ t('theme.transparency') }}</span>
          <input v-model.number="menuStore.uiTrans" type="range" min="5" max="85" step="1">
          <span class="theme-slider__val tabular">{{ menuStore.uiTrans }}%</span>
        </label>
        <label class="theme-slider">
          <span>{{ t('theme.blur') }}</span>
          <input v-model.number="menuStore.uiBlur" type="range" min="0" max="30" step="1">
          <span class="theme-slider__val tabular">{{ menuStore.uiBlur }} px</span>
        </label>
        <label class="theme-slider">
          <span>{{ t('theme.dim') }}</span>
          <input v-model.number="menuStore.bgDim" type="range" min="0" max="80" step="1">
          <span class="theme-slider__val tabular">{{ menuStore.bgDim }}%</span>
        </label>
      </div>

      <div class="theme-accent-note">
        <span class="theme-accent-chip"></span>
        <span>{{ imageTheme ? t('theme.accent-from-image') : t('theme.accent-fallback') }}</span>
      </div>
    </template>

    <div v-else class="theme-row">
      <span class="theme-row__label">{{ t('theme.accent') }}</span>
      <div class="theme-swatches" role="radiogroup" :aria-label="t('theme.accent')">
        <button v-for="color in swatches"
                :key="color"
                type="button"
                role="radio"
                class="theme-swatch"
                :class="{ 'is-on': menuStore.accent === color }"
                :style="{ background: color }"
                :aria-checked="menuStore.accent === color ? 'true' : 'false'"
                :aria-label="color"
                @click="menuStore.accent = color"></button>
      </div>
    </div>

    <input ref="fileInput"
           class="file-input"
           type="file"
           accept="image/*"
           @change="handleFileChange"/>

    <template #footer>
      <button type="button" class="px-btn px-btn--pill" @click="resetTheme">{{ t('theme.reset') }}</button>
      <button type="button" class="px-btn px-btn--primary px-btn--pill" @click="dialogOpen = false">{{ t('theme.done') }}</button>
    </template>
  </UiModal>
</template>

<script setup lang="ts">
import {useI18n} from 'vue-i18n';
import {useMenuStore} from "@/store/menuStore";
import {imageTheme} from "@/composables/useAppTheme";
import type {UiPillOption} from "@/components/ui";
import IconCircleHalf from "~icons/tabler/circle-half-2";
import IconSun from "~icons/tabler/sun";
import IconMoon from "~icons/tabler/moon";
import {
  buildRendererUrl,
  createStorageValue,
  ensureRelativeStorageValue,
  extractUrlFromCssValue,
  getRendererOrigin,
  getRelativeUserImagePath,
  normalizeCustomBackground,
  normalizeResponsePath,
} from "@/util/customBackground";

interface ThemeOption {
  id: string;
  bg?: string | string[];
  rand?: boolean;
}

const MAX_UPLOAD_BYTES = 1024 * 1024;
const uploadableThemeIds = new Set(['custom']);
const supportsUpload = (id: string) => uploadableThemeIds.has(id);
const swatches = ['#5b67e8', '#1f9e7a', '#c9484f', '#c98a2e', '#2f7fbf'];

const rendererOrigin = getRendererOrigin();
const customBackgroundApiUrl = buildRendererUrl('/api/custom-background', rendererOrigin);

// 存储背景主题
const menuStore = useMenuStore()

// 国际化
const {t} = useI18n();

const dialogOpen = ref(false);
const bgError = ref('');

const modeOptions = computed<UiPillOption<'auto' | 'light' | 'dark'>[]>(() => [
  {value: 'auto', label: t('theme.auto'), icon: IconCircleHalf, tip: t('theme.auto-tip')},
  {value: 'light', label: t('theme.light'), icon: IconSun},
  {value: 'dark', label: t('theme.dark'), icon: IconMoon},
]);

const getCustomBackgroundKey = (id: string) => `custom-bg-${id}`;
const customVersion = ref(0);

const readStoredCustom = (themeId: string) => {
  void customVersion.value;
  try {
    const stored = localStorage.getItem(getCustomBackgroundKey(themeId));
    const normalized = normalizeCustomBackground(stored, rendererOrigin);
    return normalized ? extractUrlFromCssValue(normalized.cssValue) : null;
  } catch {
    return null;
  }
};

const applyStoredCustomBackground = (themeId: string) => {
  const key = getCustomBackgroundKey(themeId);
  const stored = localStorage.getItem(key);
  if (!stored) {
    return false;
  }

  const normalized = normalizeCustomBackground(stored, rendererOrigin);
  if (!normalized) {
    return false;
  }

  if (normalized.storageValue !== stored) {
    try {
      localStorage.setItem(key, normalized.storageValue);
    } catch (error) {
      console.error('Failed to normalize stored custom background', error);
    }
  }

  menuStore.setBackground(normalized.storageValue);
  menuStore.bgTheme = themeId;
  return true;
};

// 获取随机元素
function getRandom(arr: any[]) {
  if (arr.length === 1) return arr[0];
  return arr[Math.floor(Math.random() * arr.length)];
}

// Local (bundled) images get a thumbnail; random remote sources don't, since
// each request returns a different picture.
const thumbOf = (item: ThemeOption): string | null => {
  if (supportsUpload(item.id)) return readStoredCustom(item.id);
  if (item.rand) return null;
  const first = Array.isArray(item.bg) ? item.bg[0] : item.bg;
  if (!first) return null;
  const url = extractUrlFromCssValue(first) ?? first;
  return url.startsWith('http') ? null : url;
};

const theme = ref<ThemeOption[]>([]);
const tiles = computed(() => theme.value.map(option => ({
  id: option.id,
  option,
  label: t('bg.' + option.id),
  custom: supportsUpload(option.id),
  thumb: thumbOf(option),
})));

// 切换背景
const changeBackground = (item: ThemeOption) => {
  bgError.value = '';
  if (supportsUpload(item.id)) {
    if (applyStoredCustomBackground(item.id)) {
      return;
    }

    if (!item.bg) {
      triggerUpload(item);
      return;
    }
  }

  let url: string;
  if (Array.isArray(item.bg)) {
    url = getRandom(item.bg);
    if (item["rand"]) {
      url = "url('" + url + "&date=" + Date.now() + "')";
    }
  } else if (typeof item.bg === "string") {
    url = item.bg;
  } else {
    console.warn(`Theme "${item.id}" is missing a background definition.`);
    return;
  }
  menuStore.setBackground(url);
  menuStore.bgTheme = item.id;
};

const resetTheme = () => {
  bgError.value = '';
  menuStore.resetThemeTweaks();
};

const fileInput = ref<HTMLInputElement | null>(null);
const pendingThemeId = ref<string | null>(null);

const triggerUpload = (item: ThemeOption) => {
  pendingThemeId.value = item.id;
  fileInput.value?.click();
};

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  const themeId = pendingThemeId.value;
  const reset = () => {
    target.value = '';
    pendingThemeId.value = null;
  };
  if (!file || !themeId) {
    reset();
    return;
  }
  if (file.size > MAX_UPLOAD_BYTES) {
    bgError.value = t('bg.too-large');
    reset();
    return;
  }
  bgError.value = '';
  const reader = new FileReader();
  reader.onload = async () => {
    const result = reader.result;
    if (typeof result !== 'string') {
      bgError.value = t('bg.upload-failed');
      reset();
      return;
    }

    const key = getCustomBackgroundKey(themeId);
    const previousValue = localStorage.getItem(key);
    const normalizedPrevious = ensureRelativeStorageValue(previousValue, rendererOrigin);
    if (normalizedPrevious && normalizedPrevious !== previousValue) {
      try {
        localStorage.setItem(key, normalizedPrevious);
      } catch (error) {
        console.error('Failed to normalize stored custom background before upload', error);
      }
    }

    const previousPath = getRelativeUserImagePath(normalizedPrevious ?? previousValue ?? null, rendererOrigin);

    try {
      const response = await fetch(customBackgroundApiUrl, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          themeId,
          dataUrl: result,
          fileName: file.name,
          previousPath: previousPath ?? undefined,
        }),
      });

      if (!response.ok) {
        throw new Error(`Upload failed with status ${response.status}`);
      }

      const data = await response.json() as { url?: string };
      if (!data.url) {
        throw new Error('Missing url in response');
      }

      const relativePath = normalizeResponsePath(data.url);
      const storageValue = createStorageValue(relativePath);
      menuStore.setBackground(storageValue);
      menuStore.bgTheme = themeId;
      try {
        localStorage.setItem(key, storageValue);
        customVersion.value++;
      } catch (error) {
        console.error('Failed to save custom background', error);
        bgError.value = t('bg.storage-failed');
      }
    } catch (error) {
      console.error('Failed to upload custom background', error);
      bgError.value = t('bg.upload-failed');
    } finally {
      reset();
    }
  };
  reader.onerror = () => {
    console.error('Failed to read custom background file', reader.error);
    bgError.value = t('bg.upload-failed');
    reset();
  };
  reader.readAsDataURL(file);
};

onMounted(async () => {
  try {
    const response = await fetch("/json/theme.json");
    theme.value = await response.json() as ThemeOption[];
  } catch (error) {
    console.error("获取 JSON 失败", error);
  }
});
</script>

<style scoped>
.theme-title {
  font-size: 16px;
  font-weight: 700;
  flex: 1;
}

:deep(.theme-body) {
  gap: 18px;
  padding: 6px 22px 18px;
}

.theme-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.theme-row__label {
  font-size: 13px;
  font-weight: 600;
  width: 170px;
}

.theme-mode {
  width: 340px;
  max-width: 100%;
}

.theme-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.theme-row__title {
  font-size: 13px;
  font-weight: 600;
}

.theme-row__desc {
  font-size: 12px;
  color: var(--text-2);
  line-height: 1.45;
}

.theme-tiles {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.theme-tile {
  position: relative;
  height: 52px;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--panel-soft) center / cover no-repeat;
  border: 1px solid var(--border);
}

.theme-tile.is-custom-empty {
  border: 1.5px dashed var(--border);
}

.theme-tile.is-on {
  box-shadow: 0 0 0 2px var(--accent);
}

.theme-tile__icon {
  color: var(--text-2);
  margin-bottom: 12px;
}

.theme-tile__label {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 10px 6px 4px;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.2;
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text);
}

.theme-tile__label.has-thumb {
  color: #fff;
  background: linear-gradient(transparent, rgba(0, 0, 0, .65));
}

.theme-tile__upload {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 22px;
  height: 22px;
  border-radius: 8px;
  border: none;
  background: rgba(0, 0, 0, .55);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
}

.theme-tile__upload:hover {
  background: var(--accent);
  color: var(--on-accent);
}

.theme-error {
  font-size: 12px;
  font-weight: 600;
  color: var(--error);
  line-height: 1.4;
}

.theme-sliders {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 12px;
  background: var(--panel-soft);
}

.theme-slider {
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr) 52px;
  align-items: center;
  gap: 12px;
  font-size: 13px;
  font-weight: 600;
}

.theme-slider input {
  width: 100%;
  accent-color: var(--accent);
  cursor: pointer;
}

.theme-slider__val {
  font-size: 12px;
  color: var(--text-2);
  text-align: right;
}

.theme-accent-note {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.45;
}

.theme-accent-chip {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--accent);
  box-shadow: 0 0 0 2px var(--dialog-bg), 0 0 0 3px var(--border);
}

.theme-swatches {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.theme-swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  cursor: pointer;
  border: 2px solid transparent;
  box-shadow: 0 0 0 1px var(--border);
  padding: 0;
}

.theme-swatch.is-on {
  border-color: var(--text);
}

.file-input {
  display: none;
}
</style>
