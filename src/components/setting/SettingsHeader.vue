<script setup lang="ts">
// Settings page header: [← back on sub-pages] [section pills] [section title]
// [section-specific controls from the default slot].
import {useI18n} from "vue-i18n";
import {useRouter} from "vue-router";
import {useMenuStore} from "@/store/menuStore";
import type {UiPillOption} from "@/components/ui";
import IconSettings from "~icons/tabler/settings";
import IconCpu from "~icons/tabler/cpu";
import IconGitBranch from "~icons/tabler/git-branch";
import IconWifi from "~icons/tabler/wifi";
import IconFileText from "~icons/tabler/file-text";

const props = defineProps<{
  /** dns / shortcut sub-pages show a back button and their own title. */
  sub?: 'dns' | 'shortcut';
}>();

const {t} = useI18n();
const router = useRouter();
const menuStore = useMenuStore();

const tabs = computed<UiPillOption<string>[]>(() => [
  {value: 'app', icon: IconSettings, tip: t('setting.tab.app')},
  {value: 'core', icon: IconCpu, tip: t('setting.tab.core')},
  {value: 'rule', icon: IconGitBranch, tip: t('sec-nav.rule')},
  {value: 'connection', icon: IconWifi, tip: t('sec-nav.conn')},
  {value: 'log', icon: IconFileText, tip: t('sec-nav.log')},
]);

const current = computed({
  get: () => props.sub ? (props.sub === 'dns' ? 'core' : 'app') : menuStore.settingTab,
  set: (value: string) => {
    menuStore.setSettingTab(value);
    if (props.sub) router.push('/Setting');
  },
});

const title = computed(() => t('setting.section.' + (props.sub ?? menuStore.settingTab)));

function back() {
  menuStore.setSettingTab(props.sub === 'dns' ? 'core' : 'app');
  router.push('/Setting');
}
</script>

<template>
  <div class="px-page-head settings-head">
    <UiIconButton v-if="sub" round soft :size="34" :label="t('connections.back')" @click="back">
      <icon-tabler-arrow-left width="17" height="17"/>
    </UiIconButton>
    <UiPillTabs v-model="current" :options="tabs" :aria-label="t('setting.title')"/>
    <h1 class="px-page-title settings-title ellipsis">{{ title }}</h1>
    <slot/>
  </div>
</template>

<style scoped>
.settings-head {
  flex-wrap: wrap;
}

.settings-title {
  min-width: 0;
}
</style>
