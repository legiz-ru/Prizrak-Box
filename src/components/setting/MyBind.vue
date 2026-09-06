<script setup lang="ts">
import {ref} from "vue";
import {EditPen} from "@element-plus/icons-vue";
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

<template>
  <div class="input-container">
    <span>{{ $t('setting.mihomo.bindAddress') }} :</span>
    <template v-if="isEditing">
      <input
          type="text"
          v-model="bind"
          placeholder="请输入端口号"
          autocapitalize="off"
          autocomplete="off"
          autocorrect="off"
          spellcheck="false"
      />
    </template>
    <template v-else>
      <span class="content">{{ settingStore.bindAddress }}</span>
    </template>
    <button class="action-btn" @click="toggleEditing" v-if="!isEditing">
      <el-icon><EditPen/></el-icon>
    </button>
    <button class="action-btn" @click="saveBind" v-if="isEditing">
      <el-icon><icon-ep-select/></el-icon>
    </button>
    <button class="action-btn" @click="cancelEdit" v-if="isEditing">
      <el-icon><icon-ep-close-bold/></el-icon>
    </button>
  </div>
</template>

<style scoped>
.input-container {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 30px;
}

span {
  color: var(--text-color);
  font-size: 18px;
  font-weight: bold;
}

.content {
  font-weight: normal;
}

input {
  width: 100px;
  padding: 5px 8px;
  border: 1px solid var(--text-color);
  border-radius: 5px;
  background-color: rgba(255, 255, 255, 0.1);
  color: var(--text-color);
  font-size: 16px;
}

input:focus {
  outline: none;
}

.action-btn {
  height: 36px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  box-shadow: var(--left-nav-shadow);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  flex-shrink: 0;
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.action-btn:hover {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}
</style>
