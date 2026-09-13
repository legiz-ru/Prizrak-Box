<template>
  <template v-if="isEditing">
    <input
        class="px-value-input"
        type="text"
        v-model="bind"
        autocapitalize="off"
        autocomplete="off"
        autocorrect="off"
        spellcheck="false"
        :aria-label="$t('setting.mihomo.bindAddress')"
        @keyup.enter="saveBind"
        @keyup.esc="cancelEdit"
    />
    <button class="px-iconbtn" :aria-label="$t('save')" @click="saveBind">
      <el-icon><icon-tabler-check/></el-icon>
    </button>
    <button class="px-iconbtn" :aria-label="$t('cancel')" @click="cancelEdit">
      <el-icon><icon-tabler-x/></el-icon>
    </button>
  </template>
  <button v-else class="px-value" @click="toggleEditing">
    <span class="px-value__text">{{ settingStore.bindAddress }}</span>
    <el-icon class="px-value__icon"><icon-tabler-pencil/></el-icon>
  </button>
</template>

<script setup lang="ts">
import {ref} from "vue";
import {useSettingStore} from "@/store/settingStore";
import {useI18n} from "vue-i18n";
import {pError} from "@/util/pLoad";
import {pUpdateMihomo} from "@/util/mihomo";
import createApi from "@/api";
import {useMenuStore} from "@/store/menuStore";
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
const bind = ref("");

// 切换编辑模式
const toggleEditing = () => {
  isEditing.value = !isEditing.value;
};

// IPv4 的正则表达式
const ipv4Regex = /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;

// IPv6 的正则表达式
const ipv6Regex = /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|(::[0-9a-fA-F]{1,4}){1,7}[0-9a-fA-F]{0,4})$/;


// 保存监听地址
const saveBind = async () => {
  // 检测是否匹配 IPv4 或 IPv6
  if (!ipv4Regex.test(bind.value) && !ipv6Regex.test(bind.value)) {
    pError(t('setting.mihomo.bind-error'))
    return;
  }

  // 监听地址没有变化
  if (bind.value === settingStore.bindAddress) {
    isEditing.value = false;
    return;
  }

  // 检测地址是否可用，不可用直接报错
  try {
    await api.checkAddressPort({
      "bindAddress": bind.value,
      "port": settingStore.port,
    })
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
      return
    }
  }

  // 更新配置
  api.updateConfigs({
    "allow-lan": true,
    "bind-address": bind.value,
  }).then(() => {
    settingStore.setBindAddress(bind.value);
    isEditing.value = false; // 退出编辑模式
    // 同步 mihomo 配置
    pUpdateMihomo(menuStore, settingStore, api)

    if (menuStore.proxy) {
      updateSystemProxy(api, settingStore, settingStore.systemProxyMode);
    }
  });
};

// 取消编辑
const cancelEdit = () => {
  isEditing.value = false;
  bind.value = settingStore.bindAddress; // 恢复原始值
};


onMounted(() => {
  // 初始化端口值
  bind.value = settingStore.bindAddress;
});
</script>
