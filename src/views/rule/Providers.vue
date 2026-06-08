<script setup lang="ts">
import {getCurrentInstance, onMounted, reactive, ref, watch} from "vue";
import {VAceEditor} from "vue3-ace-editor";
import "ace-builds/src-noconflict/ace";
import "ace-builds/src-noconflict/mode-yaml";
import "ace-builds/src-noconflict/theme-monokai";
import "ace-builds/src-noconflict/ext-searchbox";
import createApi from "@/api";
import {useI18n} from "vue-i18n";
import {pError, pSuccess} from "@/util/pLoad";
import {format} from "date-fns";
import {useWebStore} from "@/store/webStore";
import {useMenuStore} from "@/store/menuStore";

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
  }
};

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
  <div class="rule-providers">
    <div class="actions">
      <!-- Action buttons only; view toggle moved to Rule.vue / Setting.vue top bar -->
      <button
          :disabled="loading"
          :class="['pill-btn', {loading}]"
          type="button"
          @click="refreshProviders"
      >
        <icon-mdi-refresh :class="['btn-icon', {spin: loading}]"/>
        {{ $t('rule.providers.refresh') }}
      </button>
      <button
          :disabled="!providers.length || updatingAll"
          :class="['pill-btn', {loading: updatingAll, disabled: !providers.length && !updatingAll}]"
          type="button"
          @click="updateAllProviders"
      >
        <icon-mdi-sync :class="['btn-icon', {spin: updatingAll}]"/>
        {{ $t('rule.providers.updateAll') }}
      </button>
    </div>

    <!-- Skeleton -->
    <el-skeleton v-if="loading" :count="3" animated class="skeleton">
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
          </div>
        </div>
      </template>
    </el-skeleton>

    <div v-else-if="!providers.length" class="empty">
      <el-empty :description="$t('rule.providers.empty')"/>
    </div>

    <!-- Cards view -->
    <div v-else-if="viewMode === 'cards'" class="provider-list">
      <div v-for="provider in providers" :key="provider.name" class="provider-card">
        <div class="card-header">
          <el-tooltip :content="$t('rule.providers.viewRules')" placement="top">
            <el-icon class="view-btn" @click.stop="openRulesDialog(provider)" size="22">
              <icon-mdi-eye-outline/>
            </el-icon>
          </el-tooltip>
          <div class="provider-name" :title="provider.name">{{ provider.name }}</div>
          <div class="header-action">
            <el-tooltip :content="$t('rule.providers.update')" placement="top">
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
          <el-tag v-if="provider.vehicleType" size="small" type="info" class="provider-tag">{{ provider.vehicleType }}</el-tag>
          <el-tag v-if="provider.behavior" size="small" type="success" class="provider-tag">{{ provider.behavior }}</el-tag>
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
          <div v-if="provider.path" class="stat-row path-row">
            <el-icon size="18" class="stat-icon"><icon-mdi-folder-outline/></el-icon>
            <span class="stat-label">{{ $t('rule.providers.path') }}</span>
            <span class="stat-value path-text">{{ provider.path }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Table view -->
    <div v-else class="table-wrap">
      <div class="table-header">
        <div class="col-icon"></div>
        <div class="col-icon"></div>
        <div class="col-tags">{{ $t('rule.now.type') }}</div>
        <div class="col-name">{{ $t('rule.providers.name') }}</div>
        <div class="col-count">{{ $t('rule.providers.ruleCountShort') }}</div>
        <div class="col-date">{{ $t('rule.providers.updatedAt') }}</div>
      </div>
      <div class="table-body">
        <div
            v-for="(provider, i) in providers"
            :key="provider.name"
            :class="['table-row', { 'table-row--alt': i % 2 === 1 }]"
        >
          <div class="col-icon">
            <el-tooltip :content="$t('rule.providers.viewRules')" placement="top">
              <el-icon class="row-action-btn" size="18" @click.stop="openRulesDialog(provider)">
                <icon-mdi-eye-outline/>
              </el-icon>
            </el-tooltip>
          </div>
          <div class="col-icon">
            <el-tooltip :content="$t('rule.providers.update')" placement="top">
              <el-icon
                  :class="['row-action-btn', {disabled: isUpdating(provider.name)}]"
                  size="18"
                  @click.stop="handleUpdateClick(provider.name)"
              >
                <icon-mdi-refresh :class="{spin: isUpdating(provider.name)}"/>
              </el-icon>
            </el-tooltip>
          </div>
          <div class="col-tags">
            <el-tag v-if="provider.vehicleType" size="small" type="info" class="provider-tag">{{ provider.vehicleType }}</el-tag>
            <el-tag v-if="provider.behavior" size="small" type="success" class="provider-tag">{{ provider.behavior }}</el-tag>
          </div>
          <div class="col-name" :title="provider.name">{{ provider.name }}</div>
          <div class="col-count">{{ provider.ruleCount ?? 0 }}</div>
          <div class="col-date">{{ formatUpdatedAt(provider.updatedAt) }}</div>
        </div>
      </div>
    </div>
  </div>

  <!-- Rules dialog -->
  <el-dialog
      v-if="viewingProvider"
      class="provider-rules-dialog"
      :model-value="!!viewingProvider"
      :show-close="false"
      width="800px"
      destroy-on-close
      @close="closeRulesDialog"
  >
    <template #header="{close}">
      <div class="dialog-header">
        <span class="dialog-title" :title="viewingProvider.name">{{ viewingProvider.name }}</span>
        <div class="header-search" v-if="!loadingRules">
          <el-input
              v-model="contentSearch"
              :placeholder="$t('rule.providers.rulesSearch')"
              clearable
              size="small"
              class="header-search-input"
              @input="searchInEditor"
              @clear="onSearchClear"
              @keydown.enter.prevent="findNext"
              @keydown.shift.enter.prevent="findPrev"
          >
            <template #prefix><icon-mdi-magnify/></template>
          </el-input>
          <span :class="['match-badge', {zero: (matchCount ?? 0) === 0}]">{{ matchCount ?? 0 }}</span>
          <div class="nav-buttons">
            <button class="nav-btn" :disabled="!contentSearch.trim() || (matchCount ?? 0) === 0" @click="findPrev" :title="$t('rule.providers.prevMatch')">
              <icon-mdi-chevron-up/>
            </button>
            <button class="nav-btn" :disabled="!contentSearch.trim() || (matchCount ?? 0) === 0" @click="findNext" :title="$t('rule.providers.nextMatch')">
              <icon-mdi-chevron-down/>
            </button>
          </div>
        </div>
        <button class="dialog-close-btn" @click="close" :title="$t('common.close')">
          <icon-mdi-close/>
        </button>
      </div>
    </template>
    <div class="provider-content-body">
      <el-skeleton v-if="loadingRules" :rows="10" animated class="content-skeleton"/>
      <VAceEditor
          v-else
          v-model:value="providerContent"
          lang="yaml"
          theme="monokai"
          :options="editorOptions"
          class="content-editor"
          @init="onEditorInit"
      />
    </div>
  </el-dialog>
</template>

<style scoped>
.rule-providers {
  width: 100%;
  margin-left: 0;
  margin-top: 10px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* ── Toolbar ── */
.actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

/* Action buttons — same pill-btn as Group.vue */
.pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  padding: 9px 18px;
  font-size: 15px;
  cursor: pointer;
  box-shadow: var(--left-nav-shadow);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.pill-btn:hover,
.pill-btn.loading {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.pill-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}

.btn-icon {
  font-size: 16px;
  flex-shrink: 0;
}

/* ── Cards view ── */
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
  border-radius: 20px;
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

.view-btn {
  color: var(--text-color);
  cursor: pointer;
  opacity: 0.7;
  transition: opacity 0.2s ease;
  flex-shrink: 0;
}

.view-btn:hover {
  opacity: 1;
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

.provider-tag {
  border-radius: 999px;
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

.stat-line-label { font-weight: 600; }
.stat-line-value { font-weight: 500; }

.stat-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.stat-icon { color: var(--text-color); }
.stat-label { flex: 1; color: var(--text-color); }
.stat-value { font-weight: 500; color: var(--text-color); word-break: break-all; }
.path-row { align-items: flex-start; }
.path-text { word-break: break-all; }

/* ── Table view — same style as Now.vue ── */
.table-wrap {
  border: 2px solid var(--text-color);
  border-radius: 20px;
  overflow: hidden;
  margin-top: 20px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.table-header {
  display: flex;
  align-items: center;
  padding: 8px 10px 8px 16px;
  border-bottom: 1px solid var(--text-color);
  font-weight: bold;
  gap: 4px;
  flex-shrink: 0;
}

.table-body {
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.table-body::-webkit-scrollbar { width: 5px; }
.table-body::-webkit-scrollbar-track { background: transparent; }
.table-body::-webkit-scrollbar-thumb { background: var(--scrollbar-bg); border-radius: 2px; }
.table-body::-webkit-scrollbar-thumb:hover { background: var(--scrollbar-hover-bg); box-shadow: var(--scrollbar-hover-shadow); }

.table-row {
  display: flex;
  align-items: center;
  padding: 7px 10px 7px 16px;
  border-bottom: 1px solid var(--sub-card-border);
  gap: 4px;
  color: var(--text-color);
  line-height: 1.4;
}

.table-row--alt {
  background-color: var(--rule-list-bg);
}

.table-row:hover {
  background-color: var(--rule-list-hover);
}

/* Column widths */
.col-icon {
  width: 28px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.col-tags {
  width: 140px;
  flex-shrink: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.col-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-weight: 500;
}

.col-count {
  width: 90px;
  flex-shrink: 0;
  text-align: right;
}

.col-date {
  width: 175px;
  flex-shrink: 0;
  text-align: right;
}

.row-action-btn {
  color: var(--text-color);
  cursor: pointer;
  opacity: 0.7;
  transition: opacity 0.2s ease;
}

.row-action-btn:hover {
  opacity: 1;
}

.row-action-btn.disabled {
  opacity: 0.35;
  cursor: default;
  pointer-events: none;
}

/* ── Shared ── */
.empty { margin-top: 60px; }
.skeleton-card { pointer-events: none; }

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* ── Dialog ── */
:deep(.provider-rules-dialog) {
  background: #f7f7f9;
  border: 1px solid #d5d8df;
  color: #252b36;
}

:deep(.provider-rules-dialog .el-dialog__header) {
  margin-right: 0;
  padding-bottom: 10px;
}

:deep(.provider-rules-dialog .el-dialog__body) {
  padding-top: 4px;
}

:deep(.header-search-input .el-input__wrapper) {
  background: #ffffff;
  box-shadow: 0 0 0 1px #d7dce6 inset;
  border-radius: 8px;
}

:deep(.header-search-input .el-input__inner) { color: #252b36; }
:deep(.header-search-input .el-input__inner::placeholder) { color: #9aa3b2; }
:deep(.header-search-input .el-input__prefix-inner) { color: #7f8794; }

.dialog-header {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.dialog-title {
  flex: 0 1 auto;
  max-width: 170px;
  font-size: 16px;
  font-weight: 700;
  color: #202532;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.header-search {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  background: #eef1f6;
  border: 1px solid #d7dce6;
  border-radius: 12px;
  padding: 6px 8px;
}

.header-search-input { flex: 1; min-width: 0; }

.match-badge {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 600;
  min-width: 28px;
  text-align: center;
  padding: 3px 8px;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #d7dce6;
  color: #252b36;
  white-space: nowrap;
}

.match-badge.zero { color: #e06c75; }

.nav-buttons { display: flex; gap: 2px; flex-shrink: 0; }

.nav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #ffffff;
  border: 1px solid #d7dce6;
  border-radius: 8px;
  color: #252b36;
  cursor: pointer;
  width: 32px;
  height: 28px;
  line-height: 1;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

.nav-btn:hover { background: #e8edf7; border-color: #bcc6d8; }
.nav-btn:disabled { opacity: 0.45; cursor: not-allowed; }

.dialog-close-btn {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: #ffffff;
  border: 1px solid #d7dce6;
  border-radius: 8px;
  color: #646c78;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

.dialog-close-btn:hover { background: #e8edf7; border-color: #bcc6d8; color: #252b36; }

.provider-content-body {
  display: flex;
  flex-direction: column;
}

.content-skeleton { padding: 8px 0; }

.content-editor {
  width: 100%;
  height: 500px;
  border: 2px solid var(--text-color);
  border-radius: 12px;
  font: 13px "Monaco", "Menlo", "Ubuntu Mono", "Consolas", "Source Code Pro", monospace;
}

:deep(.ace_editor) { border-radius: 12px; }
:deep(.ace_gutter) { border-top-left-radius: 10px; border-bottom-left-radius: 10px; }
</style>
