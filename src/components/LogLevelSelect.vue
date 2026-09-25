<script setup lang="ts">
import createApi from "@/api";
import {logLevel} from "@/composables/logLevel";
import {useI18n} from "vue-i18n";

const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);
const {t} = useI18n();

const LOG_LEVELS = ['debug', 'info', 'warning', 'error', 'silent'] as const;
const options = LOG_LEVELS.map(level => ({value: level as string, label: level}));

// Level from mihomo config — used as display default when user hasn't overridden
const configLevel = ref('info');

// Computed for the select: shows user's override or config default
const displayLevel = computed({
  get: () => logLevel.value || configLevel.value,
  set: (val: string) => {
    logLevel.value = val;
  }
});

onMounted(async () => {
  try {
    const configs = await api.getConfigs();
    const level = configs?.['log-level'];
    if (level && (LOG_LEVELS as readonly string[]).includes(level)) {
      configLevel.value = level;
    }
  } catch {
    // keep 'info' fallback
  }
});
</script>

<template>
  <UiSelect v-model="displayLevel"
            class="level-select"
            variant="pill"
            capitalize
            :options="options"
            :aria-label="t('logs.level')"
            :tip="t('logs.level')"/>
</template>

<style scoped>
.level-select {
  width: 130px;
  flex-shrink: 0;
}
</style>
