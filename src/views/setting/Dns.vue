<script setup lang="ts">
import MyEditor from "@/components/MyEditor.vue";
import SettingsHeader from "@/components/setting/SettingsHeader.vue";
import createApi from "@/api";
import {pError, pSuccess} from "@/util/pLoad";
import {useI18n} from "vue-i18n";

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// i18n
const {t} = useI18n();

const load = async function (yamlContent: { value: string }) {
  const dns = await api.getDNS();
  yamlContent.value = dns.content;
}

const save = async function (yamlContent: string) {
  try {
    await api.updateDNS({
      data: yamlContent
    })
    pSuccess(t('dns.success'))
    return true
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
    return false
  }
}
</script>

<template>
  <div class="px-page">
    <SettingsHeader sub="dns"/>
    <div class="dns-body">
      <MyEditor :load="load" :save="save" :hint="t('dns.hint')"/>
    </div>
  </div>
</template>

<style scoped>
.dns-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0 28px 28px;
}
</style>
