<script lang="ts" setup>
import {useMenuStore} from "@/store/menuStore";
import {useSettingStore} from "@/store/settingStore";
import createApi from "@/api";
import {pUpdateMihomo} from "@/util/mihomo";

const menuStore = useMenuStore()
const settingStore = useSettingStore()

const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

watch(() => settingStore.stack, async () => {
  if (menuStore.tun) {
    await api.updateConfigs({tun: {enable: false}})
  }

  api.updateConfigs({
    tun: {
      enable: menuStore.tun,
      stack: settingStore.stack,
    },
  }).then(() => {
    pUpdateMihomo(menuStore, settingStore, api)
  });
});

const options = ['Mixed', 'gVisor', 'System', 'Mips'].map(value => ({value, label: value}))
</script>

<template>
  <div class="px-row px-row--wrap">
    <span class="px-row__label">Tun Stack</span>
    <UiPillTabs v-model="settingStore.stack" :options="options" aria-label="Tun Stack"/>
  </div>
</template>
