<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { prettyBytes } from '@/util/format';

const { t } = useI18n();

interface Props {
  profile: any;
}

const props = defineProps<Props>();

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

// Сколько показателей реально отдала подписка: любое из четырёх полей
// необязательное, поэтому их бывает от одного до четырёх. Число уезжает в
// data-count и задаёт колонки сетки — иначе перенос оставляет последний
// показатель один на строке.
const visibleCount = computed(() => {
  return [
    props.profile?.used,
    props.profile?.available,
    props.profile?.expire,
    props.profile?.update,
  ].filter(hasValue).length;
});

// Проверка, нужно ли показывать панель статистики
const shouldShowStats = computed(() => visibleCount.value > 0);
</script>

<template>
  <div v-if="shouldShowStats" class="profile-stats" :data-count="visibleCount">
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
  width: 100%;
  padding: 0 30px;
  box-sizing: border-box;
  display: grid;
  gap: 8px 20px;
  justify-items: center;
}

/* Колонки задаёт число показателей, а не перенос. Раньше раскладка была
   flex-wrap с `flex: 1 1 220px`: при четырёх показателях суммарная база
   4 × 220 не влезала в карточку, и четвёртый уезжал один на отдельную
   строку. С явными колонками осиротевшая строка невозможна. */
.profile-stats[data-count="1"] { grid-template-columns: minmax(0, 1fr); }
.profile-stats[data-count="2"] { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.profile-stats[data-count="3"] { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.profile-stats[data-count="4"] { grid-template-columns: repeat(4, minmax(0, 1fr)); }

/* Средняя ширина: четыре показателя складываются 2 × 2. Три намеренно
   остаются в три колонки — делить их пришлось бы как 2 + 1, а это ровно
   тот же одинокий элемент, от которого мы уходим. */
@media (max-width: 980px) {
  .profile-stats[data-count="4"] { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 680px) {
  .profile-stats[data-count="2"],
  .profile-stats[data-count="3"],
  .profile-stats[data-count="4"] { grid-template-columns: minmax(0, 1fr); }
}

.stat-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-width: 0;
  white-space: nowrap;
}

.stat-icon {
  color: var(--text-color);
  opacity: 0.6;
  flex: 0 0 auto;
}

/* Если ячейка всё же уже содержимого, сокращается подпись, а не значение. */
.stat-label {
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.7;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.stat-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  flex: 0 0 auto;
}
</style>
