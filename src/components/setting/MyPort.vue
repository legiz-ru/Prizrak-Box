<template>
  <!-- Контрол без метки: её держит SettingRow, поэтому все значения группы
       выстраиваются по одной правой границе. -->
  <template v-if="isEditing">
    <input
        class="px-value-input"
        type="text"
        v-model="port"
        inputmode="numeric"
        autocapitalize="off"
        autocomplete="off"
        autocorrect="off"
        spellcheck="false"
        :aria-label="$t('setting.mihomo.port')"
        @keyup.enter="savePort"
        @keyup.esc="cancelEdit"
    />
    <button class="px-iconbtn" :aria-label="$t('save')" @click="savePort">
      <el-icon><icon-tabler-check/></el-icon>
    </button>
    <button class="px-iconbtn" :aria-label="$t('cancel')" @click="cancelEdit">
      <el-icon><icon-tabler-x/></el-icon>
    </button>
  </template>
  <button v-else class="px-value" @click="toggleEditing">
    <span class="px-value__text">{{ settingStore.port }}</span>
    <el-icon class="px-value__icon"><icon-tabler-pencil/></el-icon>
  </button>
</template>

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
