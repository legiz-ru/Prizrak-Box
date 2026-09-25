<script setup lang="ts">
import {ref} from "vue";
import {useSettingStore} from "@/store/settingStore";
import {pError} from "@/util/pLoad";
import {useI18n} from "vue-i18n";
import {pUpdateMihomo} from "@/util/mihomo";
import {useMenuStore} from "@/store/menuStore";
import createApi from "@/api";
import {updateSystemProxy} from "@/util/systemProxy";

// 使用 store
const menuStore = useMenuStore()
const settingStore = useSettingStore()
const {t} = useI18n()

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 定义数据
const isEditing = ref(false);
const port = ref(0);

// 切换编辑模式
const toggleEditing = () => {
  isEditing.value = !isEditing.value;
};

// 检查端口值是否在有效范围内
function isValidIntegerRegex(str: any) {
  return /^[1-9]\d{0,4}$/.test(str) && Number(str) <= 65535;
}

// 保存端口值
const savePort = async () => {
  // 检查端口值是否在有效范围内
  if (!isValidIntegerRegex(port.value)) {
    pError(t('setting.mihomo.port-error'))
    return;
  }

  // 端口号没有变化
  if (port.value === settingStore.port) {
    isEditing.value = false;
    return;
  }

  try {
    // 检测端口是否被占用
    await api.checkAddressPort({
      "bindAddress": settingStore.bindAddress,
      "port": Number(port.value),
    })

    // 更新配置
    api.updateConfigs({
      "mixed-port": Number(port.value),
    }).then((res: any) => {
      settingStore.setPort(port.value);
      isEditing.value = false; // 退出编辑模式
      // 同步 mihomo 配置
      pUpdateMihomo(menuStore, settingStore, api)

      if (menuStore.proxy) {
        updateSystemProxy(api, settingStore, settingStore.systemProxyMode);
      }
    });

  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }

};

// 取消编辑
const cancelEdit = () => {
  isEditing.value = false;
  port.value = settingStore.port; // 恢复原始值
};


onMounted(() => {
  // 初始化端口值
  port.value = settingStore.port;
});
</script>

<template>
  <div class="px-row">
    <span class="px-row__label">{{ $t('setting.mihomo.port') }}</span>
    <template v-if="isEditing">
      <input
          v-model="port"
          class="edit-input tabular"
          inputmode="numeric"
          :aria-label="$t('setting.mihomo.port')"
          autocapitalize="off"
          autocomplete="off"
          autocorrect="off"
          spellcheck="false"
          @keydown.enter.prevent="savePort"
          @keydown.esc.stop.prevent="cancelEdit"
      />
      <UiIconButton class="edit-ok" :size="26" :label="$t('save')" @click="savePort">
        <icon-tabler-check width="15" height="15"/>
      </UiIconButton>
      <UiIconButton :size="26" :label="$t('cancel')" @click="cancelEdit">
        <icon-tabler-x width="15" height="15"/>
      </UiIconButton>
    </template>
    <template v-else>
      <span class="px-row__value tabular">{{ settingStore.port }}</span>
      <UiIconButton :size="26" :label="$t('edit')" @click="toggleEditing">
        <icon-tabler-edit width="14" height="14"/>
      </UiIconButton>
    </template>
  </div>
</template>

<style scoped>
.edit-input {
  width: 140px;
  padding: 6px 10px;
  border-radius: 8px;
  border: 1px solid var(--accent);
  background: var(--panel-soft);
  color: var(--text);
  font-size: 13px;
  outline: none;
}

.edit-ok {
  color: var(--success);
}
</style>
