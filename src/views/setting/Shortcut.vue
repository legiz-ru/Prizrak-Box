<script setup lang="ts">
import MyHotkeyInput from "@/components/setting/MyHotkeyInput.vue";
import SettingsHeader from "@/components/setting/SettingsHeader.vue";
import {useSettingStore} from "@/store/settingStore";
import {useI18n} from "vue-i18n";

const {t} = useI18n();
const settingStore = useSettingStore();
</script>

<template>
  <div class="px-page">
    <SettingsHeader sub="shortcut"/>
    <div class="px-page-body shortcut-body">
      <section class="px-card shortcut-card">
        <div class="px-card-title">{{ t('setting.shortcut.title') }}</div>
        <div class="px-divider"></div>
        <div class="shortcut-row">
          <span class="shortcut-label">{{ t('setting.shortcut.showHide') }}</span>
          <UiSwitch v-model="settingStore.sc_switch" :aria-label="t('setting.shortcut.showHide')"/>
          <MyHotkeyInput v-model="settingStore.sc_switch_key"/>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.shortcut-body {
  padding-top: 0;
}

.shortcut-card {
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.shortcut-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  flex-wrap: wrap;
}

.shortcut-label {
  font-size: 13px;
  font-weight: 600;
  width: 190px;
  padding-top: 7px;
}

.shortcut-row :deep(.px-switch) {
  margin-top: 4px;
}
</style>
