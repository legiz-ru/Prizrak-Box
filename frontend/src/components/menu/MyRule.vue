<template>
  <div class="custom-style">
    <el-segmented v-model="menuStore.rule" :options="getOptions()">
      <template #default="scope">
        <div>
          {{ (scope as any).item["label"] }}
        </div>
      </template>
    </el-segmented>
  </div>
</template>

<script lang="ts" setup>
import {useMenuStore} from "@/store/menuStore";
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {pSuccess} from "@/util/pLoad";
import {pUpdateMihomo} from "@/util/mihomo";
import {useSettingStore} from "@/store/settingStore";

// 使用store
const menuStore = useMenuStore();
const settingStore = useSettingStore();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 国际化
const {t} = useI18n();
const getOptions = function (): any[] {
  const modes = [
    {
      label: t("rules.rule"),
      value: "rule",
    },
    {
      label: t("rules.global"),
      value: "global",
    },
  ];

  if (t("lang") != "ru") {
    modes.push({
      label: t("rules.direct"),
      value: "direct",
    });
  }

  return modes
};

// 监听 store.rule 的变化
watch(
    () => menuStore.rule,
    (newValue) => {
      api.updateConfigs({
        mode: newValue,
      }).then((res: any) => {
        pSuccess(t("rules." + newValue + "-switch"));
        // 同步 mihomo 配置
        pUpdateMihomo(menuStore, settingStore, api)
      });
    }
);
</script>

<style scoped>
.custom-style {
  margin-left: 22px;
  margin-top: 23px;
}

.custom-style .el-segmented {
  min-width: 185px;
  border: 1px solid var(--sub-card-border);
  background: var(--left-proxy-bg);
  box-shadow: var(--left-nav-shadow);
  --el-segmented-item-selected-color: var(--text-color);
  --el-segmented-item-selected-bg-color: var(--left-item-selected-bg);
  --el-border-radius-base: 5px;
  color: var(--text-color);
  font-size: 15px;
}

.custom-style .el-segmented:hover {
  box-shadow: var(--left-nav-hover-shadow);
}

:deep(.el-segmented__item) {
  padding: 0 8px;
}
</style>
