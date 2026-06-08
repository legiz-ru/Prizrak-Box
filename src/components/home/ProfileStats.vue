<script setup lang="ts">
import { computed, withDefaults } from 'vue';
import { useI18n } from 'vue-i18n';
import { prettyBytes } from '@/util/format';

const { t } = useI18n();

interface Props {
  profile: any;
  embedded?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  embedded: false,
});

// Проверка наличия значения
function hasValue(value: any) {
  return value !== undefined && value !== null && value !== '';
}

// Форматирование трафика
function formatTrafficValue(value: any) {
  if (!hasValue(value)) {
    return '';
  }
  const num = Number(value);
  if (Number.isFinite(num)) {
    return prettyBytes(num);
  }
  return String(value);
}

// Форматирование даты
function formatDateValue(value: any) {
  if (!hasValue(value)) {
    return '';
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    const match = trimmed.match(/^(\d{4})[-/.](\d{2})[-/.](\d{2})$/);
    if (match) {
      return `${match[3]}.${match[2]}.${match[1]}`;
    }

    const parsed = Date.parse(trimmed);
    if (!Number.isNaN(parsed)) {
      const date = new Date(parsed);
      const day = String(date.getDate()).padStart(2, '0');
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const year = date.getFullYear();
      return `${day}.${month}.${year}`;
    }

    return trimmed;
  }

  if (typeof value === 'number') {
    const timestamp = value > 1e12 ? value : value * 1000;
    const date = new Date(timestamp);
    if (!Number.isNaN(date.getTime())) {
      const day = String(date.getDate()).padStart(2, '0');
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const year = date.getFullYear();
      return `${day}.${month}.${year}`;
    }
  }

  if (value instanceof Date && !Number.isNaN(value.getTime())) {
    const day = String(value.getDate()).padStart(2, '0');
    const month = String(value.getMonth() + 1).padStart(2, '0');
    const year = value.getFullYear();
    return `${day}.${month}.${year}`;
  }

  return String(value);
}

// Проверка, нужно ли показывать панель статистики
const shouldShowStats = computed(() => {
  return hasValue(props.profile?.used) ||
         hasValue(props.profile?.available) ||
         hasValue(props.profile?.expire) ||
         hasValue(props.profile?.update);
});
</script>

<template>
  <div v-if="shouldShowStats" class="profile-stats" :class="{ 'profile-stats--embedded': embedded }">
    <!-- Использованный трафик -->
    <div v-if="hasValue(profile?.used)" class="stat-item">
      <el-icon class="stat-icon" size="18">
        <icon-mdi-chart-timeline-variant />
      </el-icon>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.used') }}</span>
      <span class="stat-value">{{ formatTrafficValue(profile.used) }}</span>
    </div>

    <!-- Доступный трафик -->
    <div v-if="hasValue(profile?.available)" class="stat-item">
      <el-icon class="stat-icon" size="18">
        <icon-mdi-database-check />
      </el-icon>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.available') }}</span>
      <span class="stat-value">{{ formatTrafficValue(profile.available) }}</span>
    </div>

    <!-- Дата истечения -->
    <div v-if="hasValue(profile?.expire)" class="stat-item">
      <el-icon class="stat-icon" size="18">
        <icon-mdi-calendar-alert />
      </el-icon>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.expire') }}</span>
      <span class="stat-value">{{ formatDateValue(profile.expire) }}</span>
    </div>

    <!-- Дата обновления -->
    <div v-if="hasValue(profile?.update)" class="stat-item">
      <el-icon class="stat-icon" size="18">
        <icon-mdi-update />
      </el-icon>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.update') }}</span>
      <span class="stat-value">{{ formatDateValue(profile.update) }}</span>
    </div>
  </div>
</template>

<style scoped>
.profile-stats {
  width: 95%;
  margin: 15px auto 0;
  padding: 15px 20px;
  border-radius: 20px;
  background: var(--sub-card-bg);
  box-shadow: var(--right-box-shadow);
  display: flex;
  justify-content: center;
  gap: 30px;
  flex-wrap: wrap;
}

.profile-stats--embedded {
  width: 100%;
  margin: 0;
  padding: 0 30px;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px 18px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  min-width: 230px;
  max-width: 240px;
}

.profile-stats--embedded .stat-item {
  flex: 1 1 220px;
  max-width: 260px;
  justify-content: center;
  text-align: center;
}

.stat-icon {
  color: var(--text-color);
  opacity: 0.6;
}

.stat-label {
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.7;
}

.stat-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
}
</style>
