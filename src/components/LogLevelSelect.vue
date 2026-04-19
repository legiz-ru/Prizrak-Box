<script setup lang="ts">
import createApi from "@/api";
import {logLevel} from "@/composables/logLevel";

const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const LOG_LEVELS = ['debug', 'info', 'warning', 'error', 'silent'] as const;

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
  <el-select v-model="displayLevel" class="level-select">
    <el-option
        v-for="level in LOG_LEVELS"
        :key="level"
        :label="level"
        :value="level"
    />
  </el-select>
</template>

<style scoped>
.level-select {
  width: 100px;
  flex-shrink: 0;
}

:deep(.el-select__wrapper) {
  height: 38px;
  border-radius: 999px;
  background: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  border: none;
  padding: 0 12px 0 16px;
}

:deep(.el-select__wrapper:hover) {
  box-shadow: var(--left-nav-hover-shadow);
}

:deep(.el-select__placeholder),
:deep(.el-select__selected-item) {
  color: var(--text-color);
  text-transform: capitalize;
}

:deep(.el-select__suffix .el-icon) {
  color: var(--text-color);
  opacity: 0.6;
}
</style>
