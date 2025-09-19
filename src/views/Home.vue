<script setup lang="ts">
import {computed, getCurrentInstance, onMounted, ref, watch} from "vue";
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {useWebStore} from "@/store/webStore";
import {prettyBytes} from "@/util/format";
import {Browser} from "@/runtime";

const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const {t} = useI18n();
const webStore = useWebStore();

const currentProfile = ref<Record<string, any> | null>(null);

const missingValue = "—";

const profileName = computed(() => {
  if (!currentProfile.value) {
    return "";
  }

  return currentProfile.value.title || currentProfile.value.name || "";
});

const hasProfileStats = computed(() => {
  if (!currentProfile.value) {
    return false;
  }

  const profile = currentProfile.value;
  return [profile.used, profile.available, profile.expire, profile.update].some((item) =>
    hasValue(item)
  );
});

function normalizeUrl(raw: unknown) {
  if (typeof raw !== "string") {
    return "";
  }

  return raw.trim();
}

const supportUrl = computed(() => normalizeUrl(currentProfile.value?.support));
const subscriptionUrl = computed(() => normalizeUrl(currentProfile.value?.home));

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

async function loadSelectedProfile() {
  try {
    const list = await api.getProfileList();
    if (Array.isArray(list)) {
      const selected = list.find((item: any) => item.selected);
      currentProfile.value = selected ?? null;
    } else {
      currentProfile.value = null;
    }
  } catch (e) {
    currentProfile.value = null;
  }
}

function openExternal(url: string) {
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

function openSupport() {
  openExternal(supportUrl.value);
}

function openSubscription() {
  openExternal(subscriptionUrl.value);
}

watch(
    () => webStore.fProfile,
    (data: any) => {
      if (data && Object.keys(data).length) {
        currentProfile.value = {...data};
      } else {
        currentProfile.value = null;
      }
    },
    {deep: true}
);

onMounted(async () => {
  if (webStore.fProfile && Object.keys(webStore.fProfile).length) {
    currentProfile.value = {...webStore.fProfile};
    return;
  }

  await loadSelectedProfile();
});
</script>

<template>
  <MyLayout>
    <template #bottom>
      <div class="home-content">
        <MyChart class="home-row-chart"></MyChart>
        <div class="home-second-row">
          <section class="profile-card">
            <div class="profile-card-header">
              <span class="profile-name" :title="currentProfile ? profileName : ''">
                {{ currentProfile ? (profileName || missingValue) : missingValue }}
              </span>
              <div class="profile-links" v-if="supportUrl || subscriptionUrl">
                <el-tooltip
                    v-if="supportUrl"
                    :content="$t('profiles.support')"
                    placement="top"
                >
                  <el-icon
                      size="20"
                      class="profile-link"
                      @click="openSupport"
                  >
                    <icon-mdi-face-agent/>
                  </el-icon>
                </el-tooltip>
                <el-tooltip
                    v-if="subscriptionUrl"
                    :content="$t('profiles.home')"
                    placement="top"
                >
                  <el-icon
                      size="20"
                      class="profile-link"
                      @click="openSubscription"
                  >
                    <icon-mdi-home-import-outline/>
                  </el-icon>
                </el-tooltip>
              </div>
            </div>
            <hr class="profile-divider"/>
            <div v-if="currentProfile" class="profile-stats">
              <template v-if="hasProfileStats">
                <div class="profile-stat-row" v-if="hasValue(currentProfile.used)">
                  <el-icon size="18" class="profile-stat-icon">
                    <icon-mdi-chart-timeline-variant/>
                  </el-icon>
                  <span class="profile-stat-label">{{ $t('profiles.use') }}</span>
                  <span class="profile-stat-value">{{ formatTrafficValue(currentProfile.used) }}</span>
                </div>
                <div class="profile-stat-row" v-if="hasValue(currentProfile.available)">
                  <el-icon size="18" class="profile-stat-icon">
                    <icon-mdi-database-check/>
                  </el-icon>
                  <span class="profile-stat-label">{{ $t('profiles.available') }}</span>
                  <span class="profile-stat-value">{{ formatTrafficValue(currentProfile.available) }}</span>
                </div>
                <div class="profile-stat-row" v-if="hasValue(currentProfile.expire)">
                  <el-icon size="18" class="profile-stat-icon">
                    <icon-mdi-calendar-alert/>
                  </el-icon>
                  <span class="profile-stat-label">{{ $t('profiles.expire') }}</span>
                  <span class="profile-stat-value">{{ formatDateValue(currentProfile.expire) }}</span>
                </div>
                <div class="profile-stat-row" v-if="hasValue(currentProfile.update)">
                  <el-icon size="18" class="profile-stat-icon">
                    <icon-mdi-update/>
                  </el-icon>
                  <span class="profile-stat-label">{{ $t('profiles.update') }}</span>
                  <span class="profile-stat-value">{{ formatDateValue(currentProfile.update) }}</span>
                </div>
              </template>
            </div>
            <div v-else class="profile-empty">
              {{ $t('home.profile.empty') }}
            </div>
          </section>
          <div class="test-wrapper">
            <MyTest></MyTest>
          </div>
        </div>
        <MyIp class="home-row-ip"></MyIp>
      </div>
    </template>
  </MyLayout>
</template>

<style scoped>
.home-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding-right: 18px;
}

.home-row-chart :deep(.spark) {
  width: 100%;
}

.home-second-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
  align-items: stretch;
  width: 100%;
}

.profile-card {
  padding: 10px;
  border-radius: 8px;
  text-align: left;
  box-shadow: var(--right-box-shadow);
  background: transparent;
  color: var(--text-color);
  display: flex;
  flex-direction: column;
}

.profile-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.profile-name {
  font-size: 18px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1 1 auto;
}

.profile-links {
  display: flex;
  align-items: center;
  gap: 10px;
}

.profile-link {
  color: var(--text-color);
  cursor: pointer;
  transition: color 0.2s ease;
}

.profile-link:hover {
  color: var(--hr-color);
}

.profile-divider {
  border: none;
  height: 1px;
  background-color: var(--hr-color);
  margin: 10px 0;
}

.profile-stats {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 120px;
}

.profile-stat-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.profile-stat-icon {
  color: var(--text-color);
}

.profile-stat-label {
  flex: 1;
  font-weight: 500;
  min-width: 0;
}

.profile-stat-value {
  font-weight: 600;
  text-align: right;
  white-space: nowrap;
}

.profile-empty {
  color: var(--text-color);
  font-size: 14px;
  opacity: 0.7;
}

.test-wrapper {
  min-width: 0;
  display: flex;
}

.test-wrapper :deep(.t-card) {
  margin-left: 0 !important;
  margin-top: 0;
  width: 100% !important;
  flex: 1;
  height: 100%;
}

.test-wrapper :deep(.t-card hr) {
  margin: 10px 0;
}

.home-row-ip :deep(.spark) {
  width: 100%;
}
</style>
