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
    <div class="ignore-bar">
      <button type="button" class="px-btn px-btn--primary" @click="savaIgnore">{{ $t('save') }}</button>
      <span class="px-kbd-hint">{{ $t('rule.ignore.tip') }}</span>
      <span class="px-info" tabindex="0" :aria-label="$t('rule.ignore.info')" v-tip="$t('rule.ignore.info')">
        <icon-tabler-info-circle width="15" height="15"/>
      </span>
    </div>
    <textarea
        v-model="bypass"
        class="px-textarea ignore-text"
        spellcheck="false"
        :aria-label="$t('rule.ignore.title')"
        :placeholder="$t('rule.ignore.place')"
    ></textarea>
  </div>
</template>

<style scoped>
.ignore {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 0 28px 28px;
}

.ignore-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.ignore-text {
  flex: 1;
  min-height: 280px;
  line-height: 1.7;
}
</style>
