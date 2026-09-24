<template>
  <UiDropdown hover placement="top" align="right" role="menu">
    <template #trigger="{ attrs }">
      <button type="button"
              class="side-round side-round--quit"
              v-bind="attrs"
              :aria-label="t('quit.label')"
              v-tip="t('quit.label')"
              @click="askQuit">
        <icon-tabler-power width="17" height="17"/>
      </button>
    </template>
    <template #default="{ close }">
      <button type="button" role="menuitem" data-dd-item class="px-dd-item quit-item" @click="close(); askQuit()">
        {{ t('quit.label') }}
      </button>
    </template>
  </UiDropdown>

  <UiNotice v-model="dialogOpen"
            tone="error"
            :icon="IconPower"
            :title="t('quit.confirm.title')">
    <span>{{ t('quit.confirm.text') }}</span>
    <template #actions>
      <button type="button" class="px-btn" @click="dialogOpen = false">{{ t('cancel') }}</button>
      <button type="button" class="px-btn px-btn--danger" @click="dialogOpen = false; quit()">{{ t('quit.confirm.ok') }}</button>
    </template>
  </UiNotice>
</template>

<script setup lang="ts">
import {useI18n} from 'vue-i18n';
import {Events} from "@/runtime";
import createApi from "@/api";
import IconPower from "~icons/tabler/power";

// 国际化
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// Quitting stops the core, so the in-app button asks first (accepted product
// change). The tray's "Quit" (readyToQuit) still exits directly.
const dialogOpen = ref(false);
const askQuit = () => {
  dialogOpen.value = true;
};

// 退出
const quit = () => {
  api.exit().then(res => {
    if (res && res === "ok") {
      Events.Emit({name: "doQuit", data: true})
    }
  }).catch(() => {
    Events.Emit({name: "doQuit", data: false})
  })
}

// 监听准备退出
onMounted(() => Events.On("readyToQuit", quit))
</script>

<style scoped>
.side-round--quit:hover {
  background: var(--error) !important;
  color: #fff !important;
}

.quit-item {
  font-weight: 600;
  white-space: nowrap;
  padding: 7px 14px;
}
</style>
