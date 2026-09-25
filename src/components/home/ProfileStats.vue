<script setup lang="ts">
import {computed} from 'vue';
import {useI18n} from 'vue-i18n';
import {formatDateValue, formatExact, formatTrafficValue, hasValue, relativeDate} from '@/util/profileView';

const {t} = useI18n();

interface Props {
  profile: any;
}

const props = defineProps<Props>();

// Every field is optional in a subscription: a missing value hides its row.
const shouldShowStats = computed(() => [
  props.profile?.used,
  props.profile?.available,
  props.profile?.expire,
  props.profile?.update,
].some(hasValue));
</script>

<template>
  <div v-if="shouldShowStats" class="profile-stats">
    <div v-if="hasValue(profile?.used)" class="stat-item">
      <icon-tabler-chart-line class="stat-icon" width="15" height="15"/>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.used') }}</span>
      <span class="stat-value tabular">{{ formatTrafficValue(profile.used) }}</span>
    </div>

    <div v-if="hasValue(profile?.available)" class="stat-item">
      <icon-tabler-database class="stat-icon" width="15" height="15"/>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.available') }}</span>
      <span class="stat-value tabular">{{ formatTrafficValue(profile.available) }}</span>
    </div>

    <div v-if="hasValue(profile?.expire)" class="stat-item">
      <icon-tabler-calendar-exclamation class="stat-icon" width="15" height="15"/>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.expire') }}</span>
      <span class="stat-value">{{ formatDateValue(profile.expire) }}</span>
    </div>

    <div v-if="hasValue(profile?.update)" class="stat-item">
      <icon-tabler-clock-check class="stat-icon" width="15" height="15"/>
      <span class="stat-label">{{ t('onboarding.active-profile.stats.update') }}</span>
      <span class="stat-value" v-tip="formatExact(profile.update)">{{ relativeDate(t, profile.update) }}</span>
    </div>
  </div>
</template>

<style scoped>
.profile-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
  min-height: 56px;
}

.stat-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
}

.stat-icon {
  color: var(--text-2);
  flex-shrink: 0;
}

/* The label shrinks first; the value always stays whole. */
.stat-label {
  font-size: 12px;
  color: var(--text-2);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stat-value {
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
  flex-shrink: 0;
}
</style>
