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

const profileStats = computed(() => {
  if (!currentProfile.value) {
    return [] as { key: string; icon: string; label: string; value: string }[];
  }

  const profile = currentProfile.value;

  const stats = [
    {
      key: "used",
      icon: "icon-mdi-chart-timeline-variant",
      label: t("profiles.use"),
      value: formatTrafficValue(profile.used)
    },
    {
      key: "available",
      icon: "icon-mdi-database-check",
      label: t("profiles.available"),
      value: formatTrafficValue(profile.available)
    },
    {
      key: "expire",
      icon: "icon-mdi-calendar-alert",
      label: t("profiles.expire"),
      value: formatDateValue(profile.expire)
    },
    {
      key: "update",
      icon: "icon-mdi-update",
      label: t("profiles.update"),
      value: formatDateValue(profile.update)
    }
  ];

  return stats.map(stat => ({
    ...stat,
    value: stat.value || missingValue
  }));
});

const supportUrl = computed(() => currentProfile.value?.support ?? "");
const subscriptionUrl = computed(() => currentProfile.value?.home ?? "");

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

function openSupport() {
  if (supportUrl.value) {
    Browser.OpenURL(supportUrl.value);
  }
}

function openSubscription() {
  if (subscriptionUrl.value) {
    Browser.OpenURL(subscriptionUrl.value);
  }
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
              <h2 class="profile-card-title">
                {{ $t('home.profile.title') }}
              </h2>
            </div>
            <div v-if="currentProfile" class="profile-card-body">
              <div class="profile-name-block">
                <span class="profile-label">{{ $t('home.profile.name') }}</span>
                <div class="profile-name" :title="profileName">
                  {{ profileName || missingValue }}
                </div>
              </div>
              <div class="profile-stats-block">
                <span class="profile-label">{{ $t('home.profile.stats') }}</span>
                <div class="profile-stats">
                  <div class="profile-stat-row" v-for="stat in profileStats" :key="stat.key">
                    <el-icon size="18" class="profile-stat-icon">
                      <component :is="stat.icon"/>
                    </el-icon>
                    <span class="profile-stat-label">{{ stat.label }}</span>
                    <span class="profile-stat-value">{{ stat.value }}</span>
                  </div>
                </div>
              </div>
              <div class="profile-actions" v-if="supportUrl || subscriptionUrl">
                <el-button
                    v-if="supportUrl"
                    size="small"
                    type="primary"
                    plain
                    @click="openSupport"
                >
                  <el-icon class="profile-action-icon">
                    <icon-mdi-face-agent/>
                  </el-icon>
                  {{ $t('home.profile.support') }}
                </el-button>
                <el-button
                    v-if="subscriptionUrl"
                    size="small"
                    type="primary"
                    plain
                    @click="openSubscription"
                >
                  <el-icon class="profile-action-icon">
                    <icon-mdi-home-import-outline/>
                  </el-icon>
                  {{ $t('home.profile.subscription') }}
                </el-button>
              </div>
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
}

.profile-card {
  border: 2px solid var(--sub-card-border);
  background: var(--sub-card-bg);
  border-radius: 8px;
  color: var(--text-color);
  box-shadow: var(--left-nav-shadow);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.profile-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.profile-card-title {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
}

.profile-card-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.profile-name-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.profile-label {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-color);
  opacity: 0.7;
}

.profile-name {
  font-size: 18px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-stats-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.profile-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.profile-stat-row {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 8px;
}

.profile-stat-icon {
  color: var(--text-color);
}

.profile-stat-label {
  font-size: 14px;
}

.profile-stat-value {
  font-weight: 600;
}

.profile-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.profile-action-icon {
  margin-right: 6px;
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
