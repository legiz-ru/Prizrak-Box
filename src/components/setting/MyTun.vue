<template>
  <div class="custom-style">
    <span class="liable">Tun Stack:</span>
    <div class="pill-toggle">
      <button
          v-for="opt in options"
          :key="opt"
          :class="['pill-toggle__btn', { 'is-active': settingStore.stack === opt }]"
          @click="settingStore.stack = opt"
      >{{ opt }}</button>
    </div>
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

<style scoped>
.custom-style {
  display: flex;
  align-items: center;
}

.liable {
  font-size: 18px;
  font-weight: bold;
  white-space: nowrap;
}

.pill-toggle {
  display: inline-flex;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  padding: 4px;
  gap: 4px;
  margin-left: 10px;
}

.pill-toggle:hover {
  box-shadow: var(--left-nav-hover-shadow);
}

.pill-toggle__btn {
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 14px;
  padding: 5px 12px;
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
