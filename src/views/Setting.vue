<script setup lang="ts">
import {useI18n} from "vue-i18n";
import {useMenuStore} from "@/store/menuStore";
import MyConfig from "@/components/setting/MyConfig.vue";
import ConnectionTab from "@/components/setting/ConnectionTab.vue";
import LogTab from "@/components/setting/LogTab.vue";
import RuleNow from "@/views/rule/Now.vue";
import RuleGroup from "@/views/rule/Group.vue";
import RuleProviders from "@/views/rule/Providers.vue";
import RuleIgnore from "@/views/rule/Ignore.vue";

const {t} = useI18n();
const menuStore = useMenuStore();

const settingTab = computed({
  get: () => menuStore.settingTab,
  set: (val: string) => menuStore.setSettingTab(val),
});

const ruleSubTab = ref('Now');

const ruleSubComponents: Record<string, any> = {
  Now: RuleNow,
  Group: RuleGroup,
  Providers: RuleProviders,
  Ignore: RuleIgnore,
};

const currentTabLabel = computed(() => {
  const labels: Record<string, string> = {
    app: t('setting.tab.app'),
    core: t('setting.tab.core'),
    rule: t('sec-nav.rule'),
    connection: t('sec-nav.conn'),
    log: t('sec-nav.log'),
  };
  return labels[settingTab.value] || t('setting.title');
});
</script>

<template>
  <MyLayout>
    <template #top>
      <el-space class="space">
        <el-dropdown trigger="click" @command="(cmd: string) => settingTab = cmd">
          <div class="title">
            {{ currentTabLabel }}
            <el-icon class="title-caret"><icon-mdi-menu-down/></el-icon>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="app">
                <span :class="{ 'menu-active': settingTab === 'app' }">{{ $t('setting.tab.app') }}</span>
              </el-dropdown-item>
              <el-dropdown-item command="core">
                <span :class="{ 'menu-active': settingTab === 'core' }">{{ $t('setting.tab.core') }}</span>
              </el-dropdown-item>
              <el-dropdown-item divided command="rule">
                <span :class="{ 'menu-active': settingTab === 'rule' }">{{ $t('sec-nav.rule') }}</span>
              </el-dropdown-item>
              <el-dropdown-item command="connection">
                <span :class="{ 'menu-active': settingTab === 'connection' }">{{ $t('sec-nav.conn') }}</span>
              </el-dropdown-item>
              <el-dropdown-item command="log">
                <span :class="{ 'menu-active': settingTab === 'log' }">{{ $t('sec-nav.log') }}</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-space>
    </template>

    <template #bottom>
      <!-- Rule sub-tab navigation -->
      <div v-if="settingTab === 'rule'" class="rule-nav">
        <div class="pill-toggle">
          <button :class="['pill-toggle__btn', { 'is-active': ruleSubTab === 'Now' }]"
                  @click="ruleSubTab = 'Now'">{{ $t('rule.now.title') }}</button>
          <button :class="['pill-toggle__btn', { 'is-active': ruleSubTab === 'Group' }]"
                  @click="ruleSubTab = 'Group'">{{ $t('rule.group.title') }}</button>
          <button :class="['pill-toggle__btn', { 'is-active': ruleSubTab === 'Providers' }]"
                  @click="ruleSubTab = 'Providers'">{{ $t('rule.providers.title') }}</button>
          <button :class="['pill-toggle__btn', { 'is-active': ruleSubTab === 'Ignore' }]"
                  @click="ruleSubTab = 'Ignore'">{{ $t('rule.ignore.title') }}</button>
        </div>
      </div>

      <MyConfig v-if="settingTab === 'app'" section="app"/>
      <MyConfig v-else-if="settingTab === 'core'" section="core"/>
      <component v-else-if="settingTab === 'rule'" :is="ruleSubComponents[ruleSubTab]" :key="ruleSubTab"/>
      <ConnectionTab v-else-if="settingTab === 'connection'"/>
      <LogTab v-else-if="settingTab === 'log'"/>
    </template>
  </MyLayout>
</template>

<style scoped>
:deep(.bottom) {
  padding-bottom: 0;
}

.space {
  margin-top: 20px;
}

.title {
  font-size: 32px;
  font-weight: bold;
  margin-left: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text-color);
  user-select: none;
}

.title:hover {
  opacity: 0.85;
}

.title-caret {
  font-size: 28px;
  opacity: 0.7;
}

.rule-nav {
  margin-top: 10px;
  margin-left: 10px;
  margin-bottom: 16px;
  width: 95%;
}

.pill-toggle {
  display: inline-flex;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  padding: 4px;
  gap: 4px;
}

.pill-toggle:hover {
  box-shadow: var(--left-nav-hover-shadow);
}

.pill-toggle__btn {
  flex: 1;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 14px;
  padding: 6px 14px;
  white-space: nowrap;
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.pill-toggle__btn:hover {
  background-color: var(--left-nav-btn-hover-bg);
}

.pill-toggle__btn.is-active {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}
</style>

<style>
.menu-active {
  color: var(--left-item-selected-bg);
  font-weight: 600;
}
</style>
