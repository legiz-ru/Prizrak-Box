<template>
  <UiDropdown hover placement="top" :min-width="160">
    <template #trigger="{ toggle, attrs }">
      <button type="button"
              class="side-round"
              v-bind="attrs"
              :aria-label="$t('ui.language')"
              v-tip="$t('ui.language')"
              @click="toggle">
        <icon-tabler-language width="17" height="17"/>
      </button>
    </template>
    <template #default="{ close }">
      <button v-for="lang in languages"
              :key="lang.id"
              type="button"
              role="option"
              data-dd-item
              class="px-dd-item"
              :class="{ 'is-selected': menuStore.language === lang.id }"
              :aria-selected="menuStore.language === lang.id ? 'true' : 'false'"
              @click="changeLang(lang.id); close()">
        <span style="flex:1;white-space:nowrap">{{ lang.name }}</span>
        <icon-tabler-check v-if="menuStore.language === lang.id" width="14" height="14" style="color:var(--accent)"/>
      </button>
    </template>
  </UiDropdown>
</template>

<script setup lang="ts">
import {useI18n} from 'vue-i18n';
import {useMenuStore} from "@/store/menuStore";
import {Events} from "@/runtime"
import UiDropdown from "@/components/ui/UiDropdown.vue";

// 存储语言
const menuStore = useMenuStore()

// 国际化
const {locale, t} = useI18n();

const languages = [
  {id: 'zh', name: '简体中文'},
  {id: 'en', name: 'English'},
  {id: 'ru', name: 'Русский'},
]

// tray 翻译id
const trayMenuId = [
  'tray.show',
  'tray.rule',
  'tray.global',
  'tray.direct',
  'tray.profiles',
  'tray.proxyGroups',
  'tray.dashboard',
  'tray.proxy',
  'tray.tun',
  'tray.quit'
]

// 发送 tray 翻译
function sendTranslation() {
  const translate: any = {}
  trayMenuId.forEach(item => {
    translate[item] = t(item)
  })
  Events.Emit({
    name: "translate",
    data: translate
  })
  Events.Emit({
    name: "tunAuthTip",
    data: t('tun-auth-tip')
  })
}


// 切换语言
const changeLang = (value: any) => {
  locale.value = value
  menuStore.setLanguage(value)
  sendTranslation()
}

// Keep i18n locale and tray labels in sync with the stored preference at all times.
// Using a watcher (instead of a one-shot onMounted) ensures the locale stays correct
// even if menuStore.language is changed after the component has already mounted.
watch(() => menuStore.language, (lang) => {
  if (lang) {
    locale.value = lang
    sendTranslation()
  }
}, { immediate: true })
</script>
