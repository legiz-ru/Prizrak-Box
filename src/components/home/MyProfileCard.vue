<script setup lang="ts">
import createApi from "@/api";
import {prettyBytes} from "@/util/format";
import {useI18n} from "vue-i18n";
import {Browser, Events} from "@/runtime";
import {useWebStore} from "@/store/webStore";

const {t} = useI18n();
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);
const webStore = useWebStore();

const currentProfile = ref<any | null>(null);

function hasValue(value: any) {
  return value !== undefined && value !== null && value !== "";
}

function formatTrafficValue(value: any) {
  if (!hasValue(value)) {
    return "";
  }
  const num = Number(value);
  if (Number.isFinite(num)) {
    return prettyBytes(num);
  }
  return String(value);
}

function formatDateValue(value: any) {
  if (!hasValue(value)) {
    return "";
  }

  if (typeof value === "string") {
    const trimmed = value.trim();
    const match = trimmed.match(/^(\d{4})[-/.](\d{2})[-/.](\d{2})$/);
    if (match) {
      return `${match[3]}.${match[2]}.${match[1]}`;
    }

    const parsed = Date.parse(trimmed);
    if (!Number.isNaN(parsed)) {
      const date = new Date(parsed);
      const day = String(date.getDate()).padStart(2, "0");
      const month = String(date.getMonth() + 1).padStart(2, "0");
      const year = date.getFullYear();
      return `${day}.${month}.${year}`;
    }

    return trimmed;
  }

  if (typeof value === "number") {
    const timestamp = value > 1e12 ? value : value * 1000;
    const date = new Date(timestamp);
    if (!Number.isNaN(date.getTime())) {
      const day = String(date.getDate()).padStart(2, "0");
      const month = String(date.getMonth() + 1).padStart(2, "0");
      const year = date.getFullYear();
      return `${day}.${month}.${year}`;
    }
  }

  if (value instanceof Date && !Number.isNaN(value.getTime())) {
    const day = String(value.getDate()).padStart(2, "0");
    const month = String(value.getMonth() + 1).padStart(2, "0");
    const year = value.getFullYear();
    return `${day}.${month}.${year}`;
  }

  return String(value);
}

const flagEmojiRegex = /([\u{1F1E6}-\u{1F1FF}]{2}|\u{1F3F3}|\u{1F3F4}|\u{1F6A9})/u;

function containsFlagEmoji(value: any) {
  if (typeof value !== "string") {
    return false;
  }
  return flagEmojiRegex.test(value);
}

function getProfileDisplayTitle(profile: any) {
  const title = typeof profile?.title === "string" ? profile.title.trim() : "";
  const headerTitle = typeof profile?.headerTitle === "string" ? profile.headerTitle.trim() : "";

  if (title) {
    if (!headerTitle) {
      return title;
    }
    if (containsFlagEmoji(title) || !containsFlagEmoji(headerTitle)) {
      return title;
    }
  }

  return headerTitle || title || "";
}

function openExternalLink(raw: any) {
  if (typeof raw !== "string") {
    return;
  }

  const url = raw.trim();
  if (!url) {
    return;
  }

  try {
    Browser.OpenURL(url);
  } catch (error) {
    if (typeof window !== "undefined") {
      window.open(url, "_blank", "noopener");
    }
  }
}

function goHome(data: any) {
  openExternalLink(data?.home);
}

function goSupport(data: any) {
  openExternalLink(data?.support);
}

function applyProfile(data: any | null) {
  if (!data) {
    currentProfile.value = null;
    return;
  }

  currentProfile.value = {...data};
}

function pickSelectedProfile(list: any[]) {
  if (!Array.isArray(list) || list.length === 0) {
    applyProfile(null);
    return;
  }

  const primary = list.find(item => item?.primary);
  const selected = primary ?? list.find(item => item?.selected);
  applyProfile(selected ?? list[0]);
}

async function loadProfiles() {
  try {
    const list = await api.getProfileList();
    pickSelectedProfile(list);
  } catch (error) {
    console.error("Failed to load profiles", error);
  }
}

watch(
    () => webStore.fProfile,
    (data: any) => {
      if (data && Object.keys(data).length > 0) {
        applyProfile(data);
      }
    }
);

onMounted(async () => {
  await loadProfiles();
  Events.On("profiles", (list: any[]) => {
    pickSelectedProfile(list);
  });
});

const profileTitle = computed(() => {
  if (!currentProfile.value) {
    return "";
  }
  return getProfileDisplayTitle(currentProfile.value) || currentProfile.value.name || "";
});

const hasStats = computed(() => {
  if (!currentProfile.value) {
    return false;
  }
  const profile = currentProfile.value;
  return [profile.used, profile.available, profile.expire, profile.update].some(value => hasValue(value));
});

</script>

<template>
  <div class="profile-card" v-if="currentProfile">
    <div class="profile-header">
      <div class="profile-name" :title="profileTitle">
        {{ profileTitle }}
      </div>
      <div class="profile-links">
        <el-tooltip
            v-if="currentProfile.support"
            :content="$t('profiles.support')"
            placement="top">
          <el-icon
              class="profile-link"
              size="20"
              @click.stop="goSupport(currentProfile)">
            <icon-mdi-face-agent/>
          </el-icon>
        </el-tooltip>
        <el-tooltip
            v-if="currentProfile.home"
            :content="$t('profiles.home')"
            placement="top">
          <el-icon
              class="profile-link"
              size="20"
              @click.stop="goHome(currentProfile)">
            <icon-mdi-home-import-outline/>
          </el-icon>
        </el-tooltip>
      </div>
    </div>
    <hr v-if="hasStats" class="profile-divider">
    <div class="profile-stat" v-if="hasValue(currentProfile.used)">
      <el-icon class="profile-stat-icon" size="18">
        <icon-mdi-chart-timeline-variant/>
      </el-icon>
      <span class="profile-stat-label">{{ $t('profiles.use') }}</span>
      <span class="profile-stat-value">{{ formatTrafficValue(currentProfile.used) }}</span>
    </div>
    <div class="profile-stat" v-if="hasValue(currentProfile.available)">
      <el-icon class="profile-stat-icon" size="18">
        <icon-mdi-database-check/>
      </el-icon>
      <span class="profile-stat-label">{{ $t('profiles.available') }}</span>
      <span class="profile-stat-value">{{ formatTrafficValue(currentProfile.available) }}</span>
    </div>
    <div class="profile-stat" v-if="hasValue(currentProfile.expire)">
      <el-icon class="profile-stat-icon" size="18">
        <icon-mdi-calendar-alert/>
      </el-icon>
      <span class="profile-stat-label">{{ $t('profiles.expire') }}</span>
      <span class="profile-stat-value">{{ formatDateValue(currentProfile.expire) }}</span>
    </div>
    <div class="profile-stat" v-if="hasValue(currentProfile.update)">
      <el-icon class="profile-stat-icon" size="18">
        <icon-mdi-update/>
      </el-icon>
      <span class="profile-stat-label">{{ $t('profiles.update') }}</span>
      <span class="profile-stat-value">{{ formatDateValue(currentProfile.update) }}</span>
    </div>
  </div>
  <div class="profile-card profile-card-empty" v-else>
    {{ $t('home.profile.empty') }}
  </div>
</template>

<style scoped>
.profile-card {
  width: 100%;
  color: var(--text-color);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.profile-card-empty {
  align-items: center;
  justify-content: center;
  text-align: center;
  min-height: 180px;
  font-size: 16px;
  display: flex;
}

.profile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.profile-name {
  font-size: 20px;
  font-weight: 600;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-variant-emoji: emoji;
}

.profile-links {
  display: flex;
  align-items: center;
  gap: 8px;
}

.profile-link {
  color: var(--text-color);
}

.profile-link:hover {
  cursor: pointer;
  color: var(--hr-color);
}

.profile-divider {
  border: none;
  height: 1px;
  background-color: var(--hr-color);
  margin: 10px 0;
}

.profile-stat {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.profile-stat-label {
  flex: 1;
}

.profile-stat-value {
  font-weight: 500;
}

.profile-stat-icon {
  color: var(--text-color);
}
</style>
