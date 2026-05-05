<script setup lang="ts">
import {ref} from "vue";
import {EditPen} from "@element-plus/icons-vue";
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
  <div class="input-container">
    <span>{{ $t('setting.mihomo.port') }} :</span>
    <template v-if="isEditing">
      <input
          type="text"
          v-model="port"
          placeholder="请输入端口号"
          autocapitalize="off"
          autocomplete="off"
          autocorrect="off"
          spellcheck="false"
      />
    </template>
    <template v-else>
      <span class="content">{{ settingStore.port }}</span>
    </template>
    <button class="action-btn" @click="toggleEditing" v-if="!isEditing">
      <el-icon><EditPen/></el-icon>
    </button>
    <button class="action-btn" @click="savePort" v-if="isEditing">
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
