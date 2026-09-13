<template>
  <!-- Только контрол: метку «Tun Stack» рисует SettingRow. -->
  <div class="px-seg" role="group" :aria-label="'Tun Stack'">
    <button
        v-for="opt in options"
        :key="opt"
        type="button"
        class="px-seg__btn"
        :class="{ 'is-active': settingStore.stack === opt }"
        :aria-pressed="settingStore.stack === opt"
        @click="settingStore.stack = opt"
    >{{ opt }}</button>
  </div>
</template>

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

const options = ['Mixed', 'gVisor', 'System']
</script>

