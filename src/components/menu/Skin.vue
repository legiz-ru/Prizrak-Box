<template>
  <div class="dropdown-container"
       @mouseenter="showDropdown"
       @mouseleave="hideDropdown">
    <el-icon class="dropdown-button">
      <icon-mdi-tshirt-crew-outline/>
    </el-icon>
    <div class="dropdown-content"
         v-show="isDropdownVisible"
         @mouseenter="cancelHide">
      <div class="dropdown-item"
           v-for="(item,index) in theme"
           :key="index">
        <button class="dropdown-label"
                type="button"
                @click="changeBackground(item)">
          {{ t("bg." + item.id) }}
        </button>
        <button v-if="supportsUpload(item.id)"
                class="dropdown-upload"
                type="button"
                :title="t('bg.upload')"
                :aria-label="t('bg.upload')"
                @click.stop="triggerUpload(item)">
          <el-icon aria-hidden="true">
            <icon-mdi-upload/>
          </el-icon>
        </button>
      </div>
    </div>
    <input ref="fileInput"
           class="file-input"
           type="file"
           accept="image/*"
           @change="handleFileChange"/>
  </div>
</template>

<script setup lang="ts">
import {onMounted, ref} from 'vue';
import {useI18n} from 'vue-i18n';
import {ElMessage} from 'element-plus';
import {useMenuStore} from "@/store/menuStore";
import {
  buildRendererUrl,
  createStorageValue,
  ensureRelativeStorageValue,
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

const uploadableThemeIds = new Set(['custom']);

const supportsUpload = (id: string) => uploadableThemeIds.has(id);

const rendererOrigin = getRendererOrigin();

const customBackgroundApiUrl = buildRendererUrl('/api/custom-background', rendererOrigin);

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
  return true;
};

// 存储背景主题
const menuStore = useMenuStore()

// 国际化
const {t} = useI18n();

// 下拉框
const isDropdownVisible = ref(false);
let hideTimeout: any;

// 显示下拉框
const showDropdown = () => {
  clearTimeout(hideTimeout);
  isDropdownVisible.value = true;
};

// 隐藏下拉框（带延迟）
const hideDropdown = () => {
  hideTimeout = setTimeout(() => {
    isDropdownVisible.value = false;
  }, 200); // 延迟200ms隐藏
};

// 鼠标进入下拉框内容时取消隐藏
const cancelHide = () => {
  clearTimeout(hideTimeout);
};

// 获取随机元素
function getRandom(arr: any[]) {
  if (arr.length === 1) return arr[0];
  return arr[Math.floor(Math.random() * arr.length)];
}

// 切换背景
const changeBackground = (item: ThemeOption) => {
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
};

const theme = ref<ThemeOption[]>([]);
const fileInput = ref<HTMLInputElement | null>(null);
const pendingThemeId = ref<string | null>(null);

const getCustomBackgroundKey = (id: string) => `custom-bg-${id}`;

const triggerUpload = (item: ThemeOption) => {
  pendingThemeId.value = item.id;
  fileInput.value?.click();
};

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) {
    target.value = '';
    return;
  }
  const themeId = pendingThemeId.value;
  if (!themeId) {
    target.value = '';
    pendingThemeId.value = null;
    return;
  }
  const reader = new FileReader();
  reader.onload = async () => {
    const result = reader.result;
    if (typeof result !== 'string') {
      ElMessage.error(t('bg.upload-failed'));
      target.value = '';
      pendingThemeId.value = null;
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

    const previousPath = getRelativeUserImagePath(normalizedPrevious ?? previousValue ?? null);

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

      const data = await response.json() as {url?: string};
      if (!data.url) {
        throw new Error('Missing url in response');
      }

      const relativePath = normalizeResponsePath(data.url);
      const storageValue = createStorageValue(relativePath);
      menuStore.setBackground(storageValue);
      try {
        localStorage.setItem(key, storageValue);
      } catch (error) {
        console.error('Failed to save custom background', error);
        ElMessage.error(t('bg.storage-failed'));
      }
    } catch (error) {
      console.error('Failed to upload custom background', error);
      ElMessage.error(t('bg.upload-failed'));
    } finally {
      target.value = '';
      pendingThemeId.value = null;
    }
  };
  reader.onerror = () => {
    console.error('Failed to read custom background file', reader.error);
    ElMessage.error(t('bg.upload-failed'));
    target.value = '';
    pendingThemeId.value = null;
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
.dropdown-container {
  position: relative;
  display: inline-block;
}

.dropdown-button {
  margin-left: 20px;
  font-size: 20px;
  color: var(--text-color);
  border: none;
  border-radius: 5px;
  cursor: pointer;
}

.dropdown-content {
  font-size: 14px;
  min-width: 80px;
  position: absolute;
  bottom: 32px;
  margin-left: 30px;
  transform: translateX(-50%);
  background-color: var(--skin-bg-color);
  color: var(--text-color);
  padding: 10px;
  border-radius: 5px;
  box-shadow: var(--skin-box-shadow);
  text-align: center;
  z-index: 1;
  transition: all 0.3s ease;
}

.dropdown-item {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 4px;
}

.dropdown-label {
  padding: 5px 10px;
  border: none;
  border-radius: 3px;
  background: transparent;
  font: inherit;
  color: var(--text-color);
  cursor: pointer;
  transition: background-color 0.3s ease;
  width: 100%;
}

.dropdown-label:hover,
.dropdown-label:focus-visible {
  background-color: var(--skin-hover-color);
  outline: none;
}

.dropdown-upload {
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--text-color);
  font: inherit;
  font-size: 16px;
  cursor: pointer;
  opacity: 0.8;
  transition: opacity 0.3s ease;
  width: 100%;
}

.dropdown-upload .el-icon {
  font-size: 1em;
}

.dropdown-upload:hover,
.dropdown-upload:focus-visible {
  opacity: 1;
  outline: none;
}

.file-input {
  display: none;
}
</style>
