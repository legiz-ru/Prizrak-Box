<script setup lang="ts">
import MyHotkeyInput from "@/components/setting/MyHotkeyInput.vue";
import {useSettingStore} from "@/store/settingStore";
import {useI18n} from "vue-i18n";

const {t} = useI18n();
const settingStore = useSettingStore();
</script>

<template>
  <el-row :gutter="20" class="spark" style="margin-left: 0; margin-top: 2px; margin-right: 0;">
    <el-col :span="24">
      <div class="box">
        <div class="title">
          {{ t('setting.shortcut.title') }}
        </div>
        <hr/>
        <ul class="info-list">
          <li>
            <strong>{{ t('setting.shortcut.showHide') }} :</strong>
            <el-switch
                v-model="settingStore.sc_switch"
                class="set-switch"
            />
            <MyHotkeyInput
                v-model="settingStore.sc_switch_key"
                style="margin-left: 12px"
            />
          </li>
        </ul>
      </div>
    </el-col>
  </el-row>
</template>

<style scoped>
.spark {
  max-width: 95%;
}

.box {
  padding: 10px;
  border-radius: 8px;
  text-align: left;
  box-shadow: var(--right-box-shadow);
}

.box hr {
  border: none;
  height: 1px;
  background-color: var(--hr-color);
  margin: 10px 0;
}

.info-list {
  list-style: none;
  padding: 0;
}

.info-list li {
  font-size: 18px;
  margin: 8px 0;
  display: flex;
  align-items: center;
}

.set-switch {
  margin-left: 10px;
  --el-switch-border-color: var(--text-color);
  --el-switch-on-color: var(--left-item-selected-bg);
  --el-switch-off-color: transparent;
}

:deep(.el-switch__core) {
  width: 46px;
  height: 26px;
  border-radius: 12px;
  border: 2px solid var(--text-color);
}

:deep(.el-switch__core .el-switch__action) {
  margin-left: 2px;
  background-color: var(--text-color);
}

:deep(.el-switch.is-checked .el-switch__core .el-switch__action) {
  left: calc(100% - 21px);
}
</style>
