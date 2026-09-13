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
    <div class="px-toolbar">
      <button class="px-btn" @click="savaIgnore">
        <el-icon><icon-tabler-device-floppy/></el-icon>
        {{ $t('save') }}
      </button>
      <span class="px-toolbar__sep"></span>
      <span class="px-toolbar__note">{{ $t('rule.ignore.tip') }}</span>
      <!-- Подсказка раньше была вшита в шаблон по-русски и не переводилась. -->
      <PxInfo :content="$t('rule.ignore.hint')"/>
    </div>
    <div class="px-surface ignore__content">
      <textarea
          v-model="bypass"
          class="custom-textarea"
          :placeholder="$t('rule.ignore.place')"
          :aria-label="$t('rule.ignore.title')"
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
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.ignore__content {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}

.custom-textarea {
  flex: 1;
  min-height: 0;
  padding: var(--px-space-3) var(--px-row-pad-x);
  border: none;
  background: transparent;
  color: var(--text-color);
  font-family: var(--px-font-num);
  font-size: var(--px-fs-small);
  line-height: var(--px-lh-body);
  resize: none;
  outline: none;
  user-select: text;
}

.custom-textarea::placeholder {
  color: var(--placeholder-color);
}
</style>
