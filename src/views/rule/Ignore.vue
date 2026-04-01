<script setup lang="ts">

import createApi from "@/api";
import {pError, pSuccess} from "@/util/pLoad";
import {useI18n} from "vue-i18n";

// i18n
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);


const bypass = ref('')

onMounted(async () => {
  const ignore: string[] = await api.getIgnore()
  bypass.value = ignore.join("\n")
})

async function savaIgnore() {
  let value = bypass.value.trim();
  if (value === '') {
    return
  }

  const ignores = value.split("\n");

  try {
    await api.updateIgnore(ignores)
    pSuccess(t('rule.success'))
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }
}


</script>

<template>
  <div class="ignore">
    <el-space class="op">
      <button class="pill-btn" @click="savaIgnore">{{ $t('save') }}</button>
      <el-divider direction="vertical" border-style="dashed"/>
      <el-text class="st">{{ $t('rule.ignore.tip') }}</el-text>
    </el-space>
    <div class="content">
      <textarea
          v-model="bypass"
          class="custom-textarea"
          :placeholder="$t('rule.ignore.place')"
      ></textarea>
    </div>
  </div>
</template>

<style scoped>
:deep(.bottom) {
  padding-bottom: 0;
  overflow-y: hidden;
  display: flex;
  flex-direction: column;
}

.ignore {
  width: 100%;
  margin-left: 0;
  margin-top: 5px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.op {
  margin-top: 6px;
}

.pill-btn {
  border: none;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  padding: 6px 18px;
  font-size: 14px;
  cursor: pointer;
  box-shadow: var(--left-nav-shadow);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.pill-btn:hover {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.st {
  color: var(--text-color);
}

.content {
  margin-top: 25px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.custom-textarea {
  background-color: transparent; /* 背景透明 */
  border: 2px solid var(--text-color); /* 边界为 2px 的白色 */
  color: var(--text-color);
  padding: 8px 8px 8px 16px; /* 内间距，确保内容不贴边 */
  border-radius: 20px;
  font-size: 16px; /* 字体大小 */
  resize: none; /* 禁止调整大小（可选） */
  outline: none; /* 去掉点击时的默认高亮框 */
  width: 100%;
  box-sizing: border-box;
  flex: 1;
  min-height: 0;
}

.custom-textarea::placeholder {
  color: rgba(255, 255, 255, 0.6); /* 占位符文字颜色，设置为白色半透明 */
}

.custom-textarea:focus {
  box-shadow: var(--right-box-shadow); /* 焦点时添加阴影效果（可选） */
}
</style>