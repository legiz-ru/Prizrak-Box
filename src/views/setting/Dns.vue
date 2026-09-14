<script setup lang="ts">
import MyEditor from "@/components/MyEditor.vue";
import createApi from "@/api";
import {pError, pSuccess} from "@/util/pLoad";
import {useI18n} from "vue-i18n";

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// i18n
const {t} = useI18n();

const load = function (yamlContent: any) {
  api.getDNS().then((dns) => {
    yamlContent.value = dns.content;
  })
}

const save = async function (yamlContent: any) {
  try {
    await api.updateDNS({
      data: yamlContent
    })
    pSuccess(t('dns.success'))
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }
}
</script>

<template>
  <MyLayout>
    <template #top>
      <div class="px-page-head">
        <div class="px-page-title">
          {{ $t("dns.title") }}
        </div>
      </div>
    </template>
    <template #bottom>
      <MyEditor :load="load" :save="save"></MyEditor>
    </template>
  </MyLayout>
</template>

