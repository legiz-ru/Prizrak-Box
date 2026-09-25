<script setup lang="ts">
import {computed, getCurrentInstance, onMounted, reactive, ref, watch} from "vue";
import {VAceEditor} from "vue3-ace-editor";
import "ace-builds/src-noconflict/ace";
import "ace-builds/src-noconflict/mode-yaml";
import "ace-builds/src-noconflict/theme-tomorrow_night";
import "ace-builds/src-noconflict/theme-tomorrow";
import "ace-builds/src-noconflict/ext-searchbox";
import createApi from "@/api";
import {useI18n} from "vue-i18n";
import {pError, pSuccess} from "@/util/pLoad";
import {format} from "date-fns";
import {useWebStore} from "@/store/webStore";
import {useMenuStore} from "@/store/menuStore";
import {formatExact, relativeDate} from "@/util/profileView";
import IconList from "~icons/tabler/list";

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
const menuStore = useMenuStore();

const providers = ref<RuleProviderItem[]>([]);
const loading = ref(false);
const firstLoad = ref(true);
const editorTheme = computed(() => menuStore.useWhite ? 'tomorrow_night' : 'tomorrow');
const updatingAll = ref(false);
const updatingProviders = reactive<Record<string, boolean>>({});

const viewMode = computed({
  get: () => menuStore.providersView,
  set: (v: 'cards' | 'table') => menuStore.setProvidersView(v),
});

function getErrorMessage(error: unknown): string {
  if (typeof error === "string") return error;
  if (error && typeof error === "object" && "message" in error) {
    const message = (error as {message?: unknown}).message;
    if (typeof message === "string") return message;
  }
  try { return JSON.stringify(error); } catch (e) { return String(error); }
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
    firstLoad.value = false;
  }
};

const updatedLabel = (value?: string) => value ? relativeDate(t, value) : t("rule.providers.never");

const refreshProviders = async () => { await loadProviders(); };

const updateProvider = async (name: string) => {
  if (updatingProviders[name]) return;
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
  if (!providers.value.length || updatingAll.value) return;
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
    if (errors.length) pError(errors.join(""));
    else pSuccess(t("rule.providers.updateAllSuccess"));
  }
};

function formatUpdatedAt(value?: string) {
  if (!value) return t("rule.providers.never");
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return format(date, "yyyy-MM-dd HH:mm:ss");
}

const isUpdating = (name: string) => Boolean(updatingProviders[name] || updatingAll.value);

const handleUpdateClick = (name: string) => {
  if (isUpdating(name)) return;
  void updateProvider(name);
};

const viewingProvider = ref<RuleProviderItem | null>(null);
const providerContent = ref('');
const loadingRules = ref(false);
const contentSearch = ref('');
const matchCount = ref<number | null>(null);
const aceEditorInstance = ref<any>(null);

const editorOptions = {
  showPrintMargin: false,
  readOnly: true,
  highlightActiveLine: false,
  fontSize: 13,
};

const onEditorInit = (editor: any) => { aceEditorInstance.value = editor; };

function countMatches(content: string, query: string): number {
  if (!query.trim()) return 0;
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  try { return (content.match(new RegExp(escaped, 'gi')) ?? []).length; } catch { return 0; }
}

const searchInEditor = () => {
  const editor = aceEditorInstance.value;
  if (!editor) return;
  const q = contentSearch.value;
  if (!q.trim()) { editor.clearSelection(); matchCount.value = null; return; }
  editor.find(q, {caseSensitive: false, regExp: false, wholeWord: false, wrap: true, backwards: false});
  matchCount.value = countMatches(providerContent.value, q);
};

const findNext = () => {
  const editor = aceEditorInstance.value;
  if (!editor || !contentSearch.value.trim()) return;
  editor.find(contentSearch.value, {caseSensitive: false, regExp: false, wholeWord: false, wrap: true, backwards: false});
};

const findPrev = () => {
  const editor = aceEditorInstance.value;
  if (!editor || !contentSearch.value.trim()) return;
  editor.find(contentSearch.value, {caseSensitive: false, regExp: false, wholeWord: false, wrap: true, backwards: true});
};

const onSearchClear = () => { aceEditorInstance.value?.clearSelection(); matchCount.value = null; };

const rulesDialogOpen = computed({
  get: () => !!viewingProvider.value,
  set: (open: boolean) => {
    if (!open) closeRulesDialog();
  },
});

const openRulesDialog = async (provider: RuleProviderItem) => {
  viewingProvider.value = provider;
  providerContent.value = '';
  contentSearch.value = '';
  matchCount.value = null;
  aceEditorInstance.value = null;
  loadingRules.value = true;
  try {
    providerContent.value = await api.getRuleProviderRules(provider.name);
  } catch (error) {
    pError(getErrorMessage(error));
  } finally {
    loadingRules.value = false;
  }
};

const closeRulesDialog = () => {
  viewingProvider.value = null;
  providerContent.value = '';
  contentSearch.value = '';
  matchCount.value = null;
  aceEditorInstance.value = null;
};

onMounted(async () => { await refreshProviders(); });

watch(() => webStore.fProfile, async () => {
  await api.waitRunning();
  await refreshProviders();
});
</script>

<template>
  <div class="providers">
    <div class="providers-bar">
      <button type="button" class="px-btn px-btn--input px-btn--pill" :disabled="loading" @click="refreshProviders">
        <UiSpinner v-if="loading" :size="13"/>
        <icon-tabler-refresh v-else width="15" height="15"/>
        {{ t('rule.providers.refresh') }}
      </button>
      <button type="button" class="px-btn px-btn--input px-btn--pill" :disabled="updatingAll || !providers.length" @click="updateAllProviders">
        <UiSpinner v-if="updatingAll" :size="13"/>
        <icon-tabler-cloud-download v-else width="15" height="15"/>
        {{ t('rule.providers.updateAll') }}
      </button>
    </div>

    <div class="providers-body">
      <div v-if="firstLoad" class="providers-grid">
        <UiSkeleton :count="3" :min-height="140"/>
      </div>
      <UiEmpty v-else-if="!providers.length"
               :icon="IconList"
               :title="t('empty.providers.title')"
               :text="t('empty.providers.text')"/>

      <div v-else-if="viewMode === 'cards'" class="providers-grid">
        <article v-for="provider in providers" :key="provider.name" class="px-card provider">
          <div class="provider-head">
            <UiIconButton :size="26" :label="t('rule.providers.viewRules')" @click="openRulesDialog(provider)">
              <icon-tabler-eye width="15" height="15"/>
            </UiIconButton>
            <span class="provider-name ellipsis" v-tip="provider.name">{{ provider.name }}</span>
            <UiIconButton :size="26" :label="t('rule.providers.update')" :loading="isUpdating(provider.name)" @click="handleUpdateClick(provider.name)">
              <UiSpinner v-if="isUpdating(provider.name)" :size="12"/>
              <icon-tabler-refresh v-else width="14" height="14"/>
            </UiIconButton>
          </div>
          <div class="provider-tags">
            <span v-if="provider.vehicleType" class="px-tag">{{ provider.vehicleType }}</span>
            <span v-if="provider.behavior" class="px-tag px-tag--success">{{ provider.behavior }}</span>
          </div>
          <div class="px-divider provider-divider"></div>
          <div class="provider-row">
            <span>{{ t('rule.providers.ruleCount') }}</span>
            <span class="provider-value tabular">{{ provider.ruleCount ?? '—' }}</span>
          </div>
          <div class="provider-row">
            <span>{{ t('rule.providers.lastUpdate') }}</span>
            <span class="provider-value" v-tip="provider.updatedAt ? formatExact(provider.updatedAt) : ''">{{ updatedLabel(provider.updatedAt) }}</span>
          </div>
        </article>
      </div>

      <div v-else class="providers-table" role="table" :aria-label="t('rule.providers.title')">
        <div class="pt-row pt-row--head" role="row">
          <span role="columnheader">{{ t('rule.providers.name') }}</span>
          <span role="columnheader"></span>
          <span role="columnheader">{{ t('rule.providers.ruleCountShort') }}</span>
          <span role="columnheader">{{ t('rule.providers.updatedAt') }}</span>
          <span role="columnheader"></span>
        </div>
        <div v-for="provider in providers" :key="provider.name" class="pt-row" role="row">
          <span class="ellipsis pt-name" role="cell" v-tip="provider.name">{{ provider.name }}</span>
          <span class="pt-tags" role="cell">
            <span v-if="provider.vehicleType" class="px-tag">{{ provider.vehicleType }}</span>
            <span v-if="provider.behavior" class="px-tag px-tag--success">{{ provider.behavior }}</span>
          </span>
          <span class="tabular" role="cell">{{ provider.ruleCount ?? '—' }}</span>
          <span role="cell" v-tip="provider.updatedAt ? formatExact(provider.updatedAt) : ''">{{ updatedLabel(provider.updatedAt) }}</span>
          <span class="pt-actions" role="cell">
            <UiIconButton :size="26" :label="t('rule.providers.viewRules')" @click="openRulesDialog(provider)">
              <icon-tabler-eye width="14" height="14"/>
            </UiIconButton>
            <UiIconButton :size="26" :label="t('rule.providers.update')" :loading="isUpdating(provider.name)" @click="handleUpdateClick(provider.name)">
              <UiSpinner v-if="isUpdating(provider.name)" :size="12"/>
              <icon-tabler-refresh v-else width="14" height="14"/>
            </UiIconButton>
          </span>
        </div>
      </div>
    </div>
  </div>

  <UiModal v-model="rulesDialogOpen" :width="640" :aria-label="viewingProvider?.name ?? ''" body-class="rules-body">
    <template #header>
      <span class="rules-title ellipsis">{{ viewingProvider?.name }}</span>
      <div class="px-search rules-search">
        <icon-tabler-search width="14" height="14"/>
        <input v-model="contentSearch"
               class="px-input px-input--sm"
               type="search"
               :placeholder="t('rule.providers.rulesSearch')"
               :aria-label="t('rule.providers.rulesSearch')"
               @input="searchInEditor"
               @search="!contentSearch && onSearchClear()"
               @keydown.enter.exact.prevent="findNext"
               @keydown.shift.enter.prevent="findPrev">
      </div>
      <span v-if="matchCount !== null" class="rules-count tabular" role="status">{{ t('ui.found', {n: matchCount}) }}</span>
      <UiIconButton :size="28" :label="t('rule.providers.prevMatch')" :disabled="!contentSearch.trim()" @click="findPrev">
        <icon-tabler-chevron-up width="15" height="15"/>
      </UiIconButton>
      <UiIconButton :size="28" :label="t('rule.providers.nextMatch')" :disabled="!contentSearch.trim()" @click="findNext">
        <icon-tabler-chevron-down width="15" height="15"/>
      </UiIconButton>
    </template>
    <div v-if="loadingRules" class="rules-loading"><UiSpinner :size="18"/></div>
    <div v-else-if="!providerContent" class="rules-loading">{{ t('rule.providers.rulesEmpty') }}</div>
    <VAceEditor
        v-else
        :value="providerContent"
        lang="yaml"
        :theme="editorTheme"
        :options="editorOptions"
        class="rules-editor"
        @init="onEditorInit"
    />
  </UiModal>
</template>

<style scoped>
.providers {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.providers-bar {
  padding: 0 28px 14px;
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}

.providers-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 28px 28px;
}

.providers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
}

.provider {
  padding: 14px 16px;
  min-width: 0;
}

.provider-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.provider-name {
  flex: 1;
  font-size: 14px;
  font-weight: 700;
  text-align: center;
}

.provider-tags, .pt-tags {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  flex-wrap: wrap;
}

.pt-tags {
  margin-top: 0;
}

.provider-divider {
  margin: 10px 0;
}

.provider-row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--text-2);
  margin-top: 4px;
}

.provider-value {
  color: var(--text);
  font-weight: 600;
}

.providers-table {
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}

.pt-row {
  display: grid;
  grid-template-columns: 2fr 1.4fr .8fr 1.2fr auto;
  gap: 12px;
  align-items: center;
  padding: 8px 14px;
  font-size: 13px;
  border-bottom: 1px solid var(--border);
  min-width: 0;
}

.pt-row:last-child {
  border-bottom: none;
}

.pt-row--head {
  font-size: 12px;
  font-weight: 700;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: .04em;
  background: var(--input-bg);
}

.pt-name {
  font-weight: 600;
}

.pt-actions {
  display: flex;
  gap: 4px;
}

.rules-title {
  font-size: 14px;
  font-weight: 700;
  flex-shrink: 1;
  max-width: 35%;
}

.rules-search {
  flex: 1;
  min-width: 120px;
}

.rules-search > .px-input {
  padding-left: 30px;
}

.rules-count {
  font-size: 12px;
  font-weight: 700;
  color: var(--text-3);
  flex-shrink: 0;
}

:deep(.rules-body) {
  padding: 12px 20px 20px;
  min-height: 300px;
}

.rules-loading {
  display: flex;
  justify-content: center;
  padding: 40px 0;
  color: var(--text-2);
  font-size: 13px;
}

.rules-editor {
  width: 100%;
  height: min(60vh, 520px);
  border-radius: 12px;
  border: 1px solid var(--border);
}
</style>
