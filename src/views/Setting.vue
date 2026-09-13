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
// Иконки разделов выбираются в коде, поэтому импортируются явно: авто-импорт
// по употреблению в шаблоне динамический :is не видит.
import IconSettings from '~icons/tabler/settings';
import IconGitBranch from '~icons/tabler/git-branch';
import IconNetwork from '~icons/tabler/network';
import IconFileText from '~icons/tabler/file-text';

const {t} = useI18n();
const menuStore = useMenuStore();
const connectionStore = useConnectionStore();

const settingTab = computed({
  get: () => menuStore.settingTab,
  set: (val: string) => menuStore.setSettingTab(val),
});

const ruleSubTab = computed({
  get: () => menuStore.ruleMenu,
  set: (val: string) => menuStore.setRuleMenu(val),
});

const providersView = computed({
  get: () => menuStore.providersView,
  set: (v: 'cards' | 'table') => menuStore.setProvidersView(v),
});

// Разделы настроек. Короткое название показывается только у активного —
// длинное («Настройки приложения») в пилюле не помещается, а в подсказке
// остаётся полным.
const settingTabs = computed(() => [
  {key: 'app', title: t('setting.tab.app'), short: t('setting.tab.appShort'), icon: IconSettings},
  {key: 'core', title: t('setting.tab.core'), short: t('setting.tab.coreShort'), icon: null},
  {key: 'rule', title: t('sec-nav.rule'), short: t('sec-nav.rule'), icon: IconGitBranch},
  {key: 'connection', title: t('sec-nav.conn'), short: t('sec-nav.conn'), icon: IconNetwork},
  {key: 'log', title: t('sec-nav.log'), short: t('sec-nav.log'), icon: IconFileText},
]);

const ruleSubComponents: Record<string, any> = {
  Now: RuleNow,
  Group: RuleGroup,
  Providers: RuleProviders,
  Ignore: RuleIgnore,
};
</script>

<template>
  <MyLayout>
    <template #top>
      <div class="setting-tabs-wrap">
        <!-- Переключатель разделов. Активный пункт подписан: пять одинаковых
             иконок подряд не дают понять, где ты находишься, без наведения. -->
        <div class="section-tabs" role="tablist" :aria-label="t('setting.title')">
          <el-tooltip
              v-for="tab in settingTabs"
              :key="tab.key"
              :content="tab.title"
              placement="bottom"
              :show-after="300"
              :disabled="settingTab === tab.key"
          >
            <button
                type="button"
                role="tab"
                :aria-selected="settingTab === tab.key"
                :class="['section-tabs__btn', { 'is-active': settingTab === tab.key }]"
                @click="settingTab = tab.key"
            >
              <svg v-if="tab.key === 'core'" class="core-svg" viewBox="0 0 50 50" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path fill-rule="evenodd" clip-rule="evenodd" d="M4.99239 5.21742C4.0328 5.32232 3.19446 5.43999 3.12928 5.47886C2.94374 5.58955 2.96432 33.4961 3.14997 33.6449C3.2266 33.7062 4.44146 34.002 5.84976 34.3022C7.94234 34.7483 8.60505 34.8481 9.47521 34.8481C10.3607 34.8481 10.5706 34.8154 10.7219 34.6541C10.8859 34.479 10.9066 33.7222 10.9338 26.9143L10.9638 19.3685L11.2759 19.1094C11.6656 18.7859 12.1188 18.7789 12.5285 19.0899C12.702 19.2216 14.319 20.624 16.1219 22.2061C17.9247 23.7883 19.5136 25.1104 19.6527 25.144C19.7919 25.1777 20.3714 25.105 20.9406 24.9825C22.6144 24.6221 23.3346 24.5424 24.9233 24.5421C26.4082 24.5417 27.8618 24.71 29.2219 25.0398C29.6074 25.1333 30.0523 25.1784 30.2107 25.1399C30.369 25.1016 31.1086 24.5336 31.8543 23.8777C33.3462 22.5653 33.6461 22.3017 35.4359 20.7293C36.1082 20.1388 36.6831 19.6313 36.7137 19.6017C37.5681 18.7742 38.0857 18.6551 38.6132 19.1642L38.9383 19.478V34.5138L39.1856 34.6809C39.6343 34.9843 41.2534 34.9022 43.195 34.4775C44.1268 34.2737 45.2896 34.0291 45.779 33.9339C46.2927 33.8341 46.7276 33.687 46.8079 33.5861C47.0172 33.3228 47.0109 5.87708 46.8014 5.6005C46.6822 5.4431 46.2851 5.37063 44.605 5.1996C43.477 5.08482 42.2972 5.00505 41.983 5.02223L41.4121 5.05368L35.4898 10.261C27.3144 17.4495 27.7989 17.0418 27.5372 16.9533C27.4148 16.912 26.1045 16.8746 24.6253 16.8702C22.0674 16.8626 21.9233 16.8513 21.6777 16.6396C21.0693 16.115 17.2912 12.8028 14.5726 10.4108C12.9548 8.98729 10.9055 7.18761 10.0186 6.41134L8.40584 5L7.5715 5.01331C7.11256 5.02072 5.95198 5.11252 4.99239 5.21742Z" fill="currentColor"/>
                <path d="M25.572 37.9556C25.3176 38.3822 24.6815 38.3822 24.427 37.9556L23.4728 36.3558C23.2184 35.9292 23.5364 35.396 24.0453 35.396H25.9537C26.4626 35.396 26.7807 35.9292 26.5262 36.3558L25.572 37.9556Z" fill="currentColor"/>
                <path d="M3 37.3157C3 36.9034 3.3453 36.5691 3.77126 36.5691H14.3485C14.7745 36.5691 15.1198 36.9034 15.1198 37.3157C15.1198 37.728 14.7745 38.0623 14.3485 38.0623H3.77126C3.3453 38.0623 3 37.728 3 37.3157Z" fill="currentColor"/>
                <path d="M3.58851 44.5029C3.44604 44.1144 3.65596 43.6876 4.05738 43.5497L14.0254 40.1251C14.4269 39.9872 14.8678 40.1904 15.0102 40.5789C15.1527 40.9675 14.9428 41.3943 14.5414 41.5322L4.57331 44.9568C4.17189 45.0947 3.73098 44.8915 3.58851 44.5029Z" fill="currentColor"/>
                <path d="M47 37.3157C47 36.9034 46.6547 36.5691 46.2287 36.5691H35.6515C35.2255 36.5691 34.8802 36.9034 34.8802 37.3157C34.8802 37.728 35.2255 38.0623 35.6515 38.0623H46.2287C46.6547 38.0623 47 37.728 47 37.3157Z" fill="currentColor"/>
                <path d="M46.4115 44.5029C46.554 44.1144 46.344 43.6876 45.9426 43.5497L35.9746 40.1251C35.5731 39.9872 35.1322 40.1904 34.9898 40.5789C34.8473 40.9675 35.0572 41.3943 35.4586 41.5322L45.4267 44.9568C45.8281 45.0947 46.269 44.8915 46.4115 44.5029Z" fill="currentColor"/>
              </svg>
              <el-icon v-else size="19"><component :is="tab.icon"/></el-icon>
              <span v-if="settingTab === tab.key" class="section-tabs__label">{{ tab.short }}</span>
            </button>
          </el-tooltip>
        </div>
        <LogLevelSelect v-if="settingTab === 'log'" class="log-level-select-wrap" />

        <!-- Active/Closed toggle: shown when connection tab is open and mode is list or process -->
        <div
            v-if="settingTab === 'connection' && (connectionStore.viewMode === 'list' || connectionStore.viewMode === 'process')"
            class="px-seg"
        >
          <el-tooltip :content="t('connections.active')" placement="bottom" :show-after="300">
            <button
                :class="['px-seg__btn', 'px-seg__btn--icon', { 'is-active': !connectionStore.showClosed }]"
                type="button"
                @click="connectionStore.setShowClosed(false)"
            >
              <el-icon size="18"><icon-tabler-bolt/></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip :content="t('connections.closed')" placement="bottom" :show-after="300">
            <button
                :class="['px-seg__btn', 'px-seg__btn--icon', { 'is-active': connectionStore.showClosed }]"
                type="button"
                @click="connectionStore.setShowClosed(true)"
            >
              <el-icon size="18"><icon-tabler-history/></el-icon>
            </button>
          </el-tooltip>
        </div>

        <!-- Providers view toggle: shown only when rule → Providers is active -->
        <div
            v-if="settingTab === 'rule' && ruleSubTab === 'Providers'"
            class="px-seg providers-view-toggle"
        >
          <el-tooltip :content="t('rule.providers.viewCards')" placement="bottom" :show-after="300">
            <button
                :class="['px-seg__btn', 'px-seg__btn--icon', { 'is-active': providersView === 'cards' }]"
                type="button"
                @click="providersView = 'cards'"
            >
              <el-icon size="18"><icon-tabler-layout-grid/></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip :content="t('rule.providers.viewTable')" placement="bottom" :show-after="300">
            <button
                :class="['px-seg__btn', 'px-seg__btn--icon', { 'is-active': providersView === 'table' }]"
                type="button"
                @click="providersView = 'table'"
            >
              <el-icon size="18"><icon-tabler-layout-list/></el-icon>
            </button>
          </el-tooltip>
        </div>
      </div>
    </template>

    <template #bottom>
      <!-- Rule sub-tab navigation -->
      <div v-if="settingTab === 'rule'" class="rule-nav">
        <div class="px-seg px-seg--text">
          <button :class="['px-seg__btn', 'px-seg__btn--text', { 'is-active': ruleSubTab === 'Now' }]"
                  @click="ruleSubTab = 'Now'">{{ $t('rule.now.title') }}</button>
          <button :class="['px-seg__btn', 'px-seg__btn--text', { 'is-active': ruleSubTab === 'Group' }]"
                  @click="ruleSubTab = 'Group'">{{ $t('rule.group.title') }}</button>
          <button :class="['px-seg__btn', 'px-seg__btn--text', { 'is-active': ruleSubTab === 'Providers' }]"
                  @click="ruleSubTab = 'Providers'">{{ $t('rule.providers.title') }}</button>
          <button :class="['px-seg__btn', 'px-seg__btn--text', { 'is-active': ruleSubTab === 'Ignore' }]"
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
.setting-tabs-wrap {
  /* Без собственного margin-left: шапка встаёт на ту же вертикаль, что и
     карточки настроек ниже. Сам переключатель не тронут. */
  margin-top: var(--px-space-5);
  display: flex;
  align-items: center;
  gap: var(--px-space-3);
}

.log-level-select-wrap {
  margin-top: 0;
}

/* Переключатель разделов настроек. */
.section-tabs {
  display: inline-flex;
  align-items: center;
  gap: var(--px-space-1);
  padding: var(--px-space-1);
  border-radius: var(--px-r-pill);
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--px-elev-1);
}

.section-tabs__btn {
  display: inline-flex;
  align-items: center;
  gap: var(--px-space-2);
  height: 36px;
  padding: 0 var(--px-space-2);
  border: none;
  border-radius: var(--px-r-pill);
  background: transparent;
  color: var(--text-color);
  font-family: inherit;
  font-size: var(--px-fs-small);
  font-weight: 600;
  cursor: pointer;
  transition: background-color var(--px-dur) var(--px-ease),
  box-shadow var(--px-dur) var(--px-ease);
}

.section-tabs__btn:hover {
  background-color: var(--left-nav-btn-hover-bg);
}

.section-tabs__btn.is-active {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--px-elev-2);
}

/* Подпись появляется только у активного пункта, поэтому неактивные остаются
   квадратными и весь ряд не расползается по ширине. */
.section-tabs__label {
  padding-right: var(--px-space-1);
  white-space: nowrap;
}

.section-tabs__btn:not(.is-active) {
  width: 36px;
  justify-content: center;
  padding: 0;
}

/* Core SVG icon */
.core-svg {
  width: 19px;
  height: 19px;
  display: block;
  color: var(--text-color);
}

/* Rule sub-tab text toggle */
.rule-nav {
  margin-bottom: var(--px-space-4);
  width: 100%;
}

/* Текстовый вариант сегментов — подвкладки правил. */
.px-seg--text {
  gap: var(--px-space-1);
}

.px-seg__btn--text {
  font-size: var(--px-fs-body);
  padding: var(--px-space-2) var(--px-space-3);
}
</style>
