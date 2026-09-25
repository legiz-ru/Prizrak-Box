<script setup lang="ts">
import {useI18n} from "vue-i18n";
import {useMenuStore} from "@/store/menuStore";
import {useConnectionStore} from "@/store/connectionStore";
import MyConfig from "@/components/setting/MyConfig.vue";
import ConnectionTab from "@/components/setting/ConnectionTab.vue";
import LogTab from "@/components/setting/LogTab.vue";
import RuleNow from "@/views/rule/Now.vue";
import RuleGroup from "@/views/rule/Group.vue";
import RuleProviders from "@/views/rule/Providers.vue";
import RuleIgnore from "@/views/rule/Ignore.vue";
import LogLevelSelect from "@/components/LogLevelSelect.vue";
import SettingsHeader from "@/components/setting/SettingsHeader.vue";
import type {UiPillOption} from "@/components/ui";
import IconBolt from "~icons/tabler/bolt";
import IconHistory from "~icons/tabler/history";
import IconLayoutGrid from "~icons/tabler/layout-grid";
import IconList from "~icons/tabler/list";

const {t} = useI18n();
const menuStore = useMenuStore();
const connectionStore = useConnectionStore();

const settingTab = computed(() => menuStore.settingTab);

const ruleSubTab = ref('Now');

const providersView = computed({
  get: () => menuStore.providersView,
  set: (v: 'cards' | 'table') => menuStore.setProvidersView(v),
});

const ruleSubComponents: Record<string, any> = {
  Now: RuleNow,
  Group: RuleGroup,
  Providers: RuleProviders,
  Ignore: RuleIgnore,
};

const ruleTabs = [
  {key: 'Now', label: 'rule.now.title'},
  {key: 'Group', label: 'rule.group.title'},
  {key: 'Providers', label: 'rule.providers.title'},
  {key: 'Ignore', label: 'rule.ignore.title'},
];

const connectionScope = computed({
  get: () => connectionStore.showClosed,
  set: (value: boolean) => connectionStore.setShowClosed(value),
});

const scopeOptions = computed<UiPillOption<boolean>[]>(() => [
  {value: false, icon: IconBolt, tip: t('connections.active')},
  {value: true, icon: IconHistory, tip: t('connections.closed')},
]);

const providersViewOptions = computed<UiPillOption<'cards' | 'table'>[]>(() => [
  {value: 'cards', icon: IconLayoutGrid, tip: t('rule.providers.viewCards')},
  {value: 'table', icon: IconList, tip: t('rule.providers.viewTable')},
]);
</script>

<template>
  <div class="px-page">
    <SettingsHeader>
      <LogLevelSelect v-if="settingTab === 'log'"/>
      <UiPillTabs v-if="settingTab === 'connection' && (connectionStore.viewMode === 'list' || connectionStore.viewMode === 'process')"
                  v-model="connectionScope"
                  :options="scopeOptions"
                  :aria-label="t('connections.title')"/>
    </SettingsHeader>

    <template v-if="settingTab === 'rule'">
      <div class="rule-tabs" role="tablist" :aria-label="t('sec-nav.rule')">
        <button v-for="tab in ruleTabs"
                :key="tab.key"
                type="button"
                role="tab"
                class="rule-tab"
                :class="{ 'is-active': ruleSubTab === tab.key }"
                :aria-selected="ruleSubTab === tab.key ? 'true' : 'false'"
                @click="ruleSubTab = tab.key">
          <icon-tabler-list-details v-if="tab.key === 'Now'" width="15" height="15"/>
          <icon-tabler-adjustments v-else-if="tab.key === 'Group'" width="15" height="15"/>
          <icon-tabler-list v-else-if="tab.key === 'Providers'" width="15" height="15"/>
          <icon-tabler-ban v-else width="15" height="15"/>
          {{ $t(tab.label) }}
        </button>
        <UiPillTabs v-if="ruleSubTab === 'Providers'"
                    v-model="providersView"
                    class="rule-tabs__end"
                    :options="providersViewOptions"
                    :aria-label="t('rule.providers.title')"/>
      </div>
      <component :is="ruleSubComponents[ruleSubTab]" :key="ruleSubTab"/>
    </template>

    <div v-else class="setting-body" :class="{ 'is-fill': settingTab === 'log' || settingTab === 'connection' }">
      <MyConfig v-if="settingTab === 'app'" section="app"/>
      <MyConfig v-else-if="settingTab === 'core'" section="core"/>
      <ConnectionTab v-else-if="settingTab === 'connection'"/>
      <LogTab v-else-if="settingTab === 'log'"/>
    </div>
  </div>
</template>

<style scoped>
.rule-tabs {
  padding: 0 28px 14px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.rule-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid var(--border);
  background: var(--input-bg);
  color: var(--text-2);
  white-space: nowrap;
}

.rule-tab:hover {
  color: var(--text);
}

.rule-tab.is-active {
  border-color: transparent;
  background: var(--accent);
  color: var(--on-accent);
}

.rule-tabs__end {
  margin-left: auto;
}

.setting-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 28px 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Log and connection lists manage their own scrolling. */
.setting-body.is-fill {
  overflow: hidden;
}
</style>
