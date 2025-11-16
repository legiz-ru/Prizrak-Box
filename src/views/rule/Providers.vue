<script setup lang="ts">
import {getCurrentInstance, onMounted, reactive, ref, watch} from "vue";
import createApi from "@/api";
import {useI18n} from "vue-i18n";
import {pError, pSuccess} from "@/util/pLoad";
import {format} from "date-fns";
import {useWebStore} from "@/store/webStore";

interface RuleProviderItem {
  name: string;
  behavior?: string;
  ruleCount?: number;
  type?: string;
  updatedAt?: string;
  vehicleType?: string;
  path?: string;
}

const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);
const {t} = useI18n();
const webStore = useWebStore();

const providers = ref<RuleProviderItem[]>([]);
const loading = ref(false);
const updatingAll = ref(false);
const updatingProviders = reactive<Record<string, boolean>>({});

function getErrorMessage(error: unknown): string {
  if (typeof error === "string") {
    return error;
  }

  if (error && typeof error === "object" && "message" in error) {
    const message = (error as {message?: unknown}).message;
    if (typeof message === "string") {
      return message;
    }
  }

  try {
    return JSON.stringify(error);
  } catch (e) {
    console.error("Failed to stringify error", e);
    return String(error);
  }
}

function normalizeProvider(raw: Record<string, any>, name: string): RuleProviderItem {
  return {
    name,
    behavior: raw?.behavior,
    ruleCount: typeof raw?.ruleCount === "number" ? raw.ruleCount : undefined,
    type: raw?.type,
    updatedAt: typeof raw?.updatedAt === "string" ? raw.updatedAt : undefined,
    vehicleType: raw?.vehicleType,
    path: typeof raw?.path === "string" ? raw.path : undefined,
  };
}

const loadProviders = async () => {
  loading.value = true;
  try {
    const response = await api.getRuleProviders();
    const providerMap = response?.providers ?? {};
    const names = Object.keys(providerMap);
    names.sort((a, b) => a.localeCompare(b));
    providers.value = names.map((name) => normalizeProvider(providerMap[name], name));
  } catch (error) {
    pError(getErrorMessage(error));
  } finally {
    loading.value = false;
  }
};

const refreshProviders = async () => {
  await loadProviders();
};

const updateProvider = async (name: string) => {
  if (updatingProviders[name]) {
    return;
  }
  updatingProviders[name] = true;
  try {
    await api.updateRuleProvider(name);
    await loadProviders();
    pSuccess(t("rule.providers.updateSuccess", {name}));
  } catch (error) {
    pError(getErrorMessage(error));
  } finally {
    updatingProviders[name] = false;
  }
};

const updateAllProviders = async () => {
  if (!providers.value.length || updatingAll.value) {
    return;
  }
  updatingAll.value = true;
  const errors: string[] = [];
  try {
    for (const provider of providers.value) {
      updatingProviders[provider.name] = true;
      try {
        await api.updateRuleProvider(provider.name);
      } catch (error) {
        errors.push(`${provider.name}: ${getErrorMessage(error)}`);
      } finally {
        updatingProviders[provider.name] = false;
      }
    }
  } finally {
    updatingAll.value = false;
    await loadProviders();
    if (errors.length) {
      pError(errors.join(""));
    } else {
      pSuccess(t("rule.providers.updateAllSuccess"));
    }
  }
};

function formatUpdatedAt(value?: string) {
  if (!value) {
    return t("rule.providers.never");
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return format(date, "yyyy-MM-dd HH:mm:ss");
}

const isUpdating = (name: string) => Boolean(updatingProviders[name] || updatingAll.value);

const handleUpdateClick = (name: string) => {
  if (isUpdating(name)) {
    return;
  }
  void updateProvider(name);
};

onMounted(async () => {
  await refreshProviders();
});

watch(() => webStore.fProfile, async () => {
  await api.waitRunning();
  await refreshProviders();
});
</script>

<template>
  <div class="rule-providers">
    <div class="actions">
      <button
          :disabled="loading"
          :class="['action-button', {loading}]"
          type="button"
          @click="refreshProviders"
      >
        <span class="pre">
          <icon-mdi-refresh :class="{spin: loading}"/>
        </span>
        <span class="suf">{{ $t('rule.providers.refresh') }}</span>
      </button>
      <button
          :disabled="!providers.length || updatingAll"
          :class="['action-button', {loading: updatingAll, disabled: !providers.length && !updatingAll}]"
          type="button"
          @click="updateAllProviders"
      >
        <span class="pre">
          <icon-mdi-sync :class="{spin: updatingAll}"/>
        </span>
        <span class="suf">{{ $t('rule.providers.updateAll') }}</span>
      </button>
    </div>

    <el-skeleton
        v-if="loading"
        :count="3"
        animated
        class="skeleton"
    >
      <template #template>
        <div class="provider-card skeleton-card">
          <div class="card-header">
            <el-skeleton-item style="width: 70%" variant="text"/>
            <el-skeleton-item style="width: 36px" variant="circle"/>
          </div>
          <div class="tag-row">
            <el-skeleton-item variant="text" style="width: 40%"/>
            <el-skeleton-item variant="text" style="width: 30%"/>
          </div>
          <div class="stats">
            <el-skeleton-item variant="text" style="width: 60%"/>
            <el-skeleton-item variant="text" style="width: 70%"/>
            <el-skeleton-item variant="text" style="width: 80%"/>
          </div>
        </div>
      </template>
    </el-skeleton>

    <div v-else-if="!providers.length" class="empty">
      <el-empty :description="$t('rule.providers.empty')"/>
    </div>

    <div v-else class="provider-list">
      <div
          v-for="provider in providers"
          :key="provider.name"
          class="provider-card"
      >
        <div class="card-header">
          <div class="provider-name" :title="provider.name">{{ provider.name }}</div>
          <div class="header-action">
            <el-tooltip
                :content="$t('rule.providers.update')"
                placement="top"
            >
              <el-icon
                  :class="['refresh-btn', {disabled: isUpdating(provider.name)}]"
                  @click.stop="handleUpdateClick(provider.name)"
                  size="22"
              >
                <icon-mdi-refresh :class="{spin: isUpdating(provider.name)}"/>
              </el-icon>
            </el-tooltip>
          </div>
        </div>
        <div class="tag-row">
          <el-tag
              v-if="provider.vehicleType"
              size="small"
              type="info"
          >
            {{ provider.vehicleType }}
          </el-tag>
          <el-tag
              v-if="provider.behavior"
              size="small"
              type="success"
          >
            {{ provider.behavior }}
          </el-tag>
        </div>
        <div class="stats">
          <div class="stat-line">
            <span class="stat-line-label">{{ $t('rule.providers.ruleCount') }}:</span>
            <span class="stat-line-value">{{ provider.ruleCount ?? 0 }}</span>
          </div>
          <div class="stat-line">
            <span class="stat-line-label">{{ $t('rule.providers.lastUpdate') }}:</span>
            <span class="stat-line-value">{{ formatUpdatedAt(provider.updatedAt) }}</span>
          </div>
          <div
              v-if="provider.path"
              class="stat-row path-row"
          >
            <el-icon size="18" class="stat-icon">
              <icon-mdi-folder-outline/>
            </el-icon>
            <span class="stat-label">{{ $t('rule.providers.path') }}</span>
            <span class="stat-value path-text">{{ provider.path }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.rule-providers {
  width: 95%;
  margin-left: 10px;
  margin-top: 10px;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.action-button {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background-color: transparent;
  color: var(--text-color);
  border: 2px solid var(--hr-color);
  border-radius: 8px;
  padding: 6px 12px 6px 32px;
  font-size: 16px;
  cursor: pointer;
  box-shadow: var(--left-nav-shadow);
  transition: background-color 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.action-button .pre {
  position: absolute;
  left: 10px;
  display: flex;
  align-items: center;
}

.action-button .suf {
  font-weight: 500;
}

.action-button:hover,
.action-button.loading {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
  border-color: var(--text-color);
}

.action-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  box-shadow: none;
}

.skeleton {
  margin-top: 20px;
}

.provider-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
  margin-top: 20px;
}

.provider-card {
  padding: 8px 10px;
  border: 2px solid var(--sub-card-border);
  border-radius: 8px;
  background: var(--sub-card-bg);
  color: var(--text-color);
  box-shadow: var(--left-nav-shadow);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.provider-card:hover {
  background-color: var(--left-item-selected-bg);
  border: 2px solid var(--text-color);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.provider-name {
  flex: 1;
  font-size: 18px;
  font-weight: 600;
  text-align: center;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  color: var(--text-color);
}

.header-action {
  min-width: 24px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.refresh-btn {
  color: var(--text-color);
  cursor: pointer;
  transition: transform 0.2s ease;
}

.refresh-btn.disabled {
  opacity: 0.5;
  cursor: default;
  pointer-events: none;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 0 4px;
}

.stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 4px;
}

.stat-line {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  color: var(--text-color);
}

.stat-line-label {
  font-weight: 600;
}

.stat-line-value {
  font-weight: 500;
}

.stat-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.stat-icon {
  color: var(--text-color);
}

.stat-label {
  flex: 1;
  color: var(--text-color);
}

.stat-value {
  font-weight: 500;
  color: var(--text-color);
  word-break: break-all;
}

.path-row {
  align-items: flex-start;
}

.path-text {
  word-break: break-all;
}

.empty {
  margin-top: 60px;
}

.spin {
  animation: spin 1s linear infinite;
}

.action-button .pre .spin {
  animation: spin 1s linear infinite;
}

.skeleton-card {
  pointer-events: none;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
