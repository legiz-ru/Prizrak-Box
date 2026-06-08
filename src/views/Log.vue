<script setup lang="ts">
import MySimpleInput from "@/components/MySimpleInput.vue";
import LogLevelSelect from "@/components/LogLevelSelect.vue";
import {useWebStore} from "@/store/webStore";

// 获取Store
const webStore = useWebStore();

// 搜索框
const search = ref("");

function handleInputChange(value: any) {
  search.value = value;
}

// 过滤数据
function filterData() {
  return webStore.logs.filter((data: any) => {
    const searchLower = search.value.toLowerCase();
    return (
        !search.value ||
        data.payload.toLowerCase().includes(searchLower) || // 内容过滤
        data.type.toLowerCase().includes(searchLower) // 类型过滤
    );
  });
}

</script>

<template>
  <MyLayout>
    <template #top>
      <el-space class="space">
        <div class="title">
          {{ $t("log.title") }}
        </div>
      </el-space>
    </template>
    <template #bottom>
      <div class="conn">
        <div class="toolbar">
          <LogLevelSelect />
          <div class="search">
            <MySimpleInput
                :onInputChange="handleInputChange"
                :placeholder="$t('log.search')"
            ></MySimpleInput>
          </div>
        </div>
      </div>

      <div class="content">
        <div class="info-list">
          <el-row class="info" v-for="(item, i) in filterData()" :key="i">
            <el-col :span="24">
              <div>
                {{ item.time }}&emsp;[{{ item.type }}]
                <br>
                {{ item.payload }}
              </div>
            </el-col>
          </el-row>
        </div>
      </div>
    </template>
  </MyLayout>
</template>

<style scoped>
.space {
  margin-top: 20px;
}

:deep(.bottom) {
  padding-bottom: 0;
  overflow-y: hidden;
  display: flex;
  flex-direction: column;
}

.conn {
  width: 100%;
  margin-left: 0;
  margin-top: 2px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.title {
  font-size: 32px;
  font-weight: bold;
  margin-left: 10px;
}

.search {
  width: 360px;
}

.content {
  border: 2px solid var(--text-color);
  border-radius: 20px;
  overflow: hidden;
  margin-top: 20px;
  width: 100%;
  margin-left: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.info-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.info {
  border-bottom: 1px solid var(--sub-card-border);
  padding: 5px 10px;
  font-size: 14px;
  line-height: 1.5;
  user-select: text;
  background-color: var(--left-bg-color);
}

.info-list::-webkit-scrollbar {
  width: 5px;
  padding-bottom: 20px;
}

.info-list::-webkit-scrollbar-track {
  background: transparent;
}

.info-list::-webkit-scrollbar-thumb {
  background: var(--scrollbar-bg);
  border-radius: 2px;
  transition: background 0.3s ease, box-shadow 0.3s ease;
}

.info-list::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-hover-bg);
  box-shadow: var(--scrollbar-hover-shadow);
}
</style>
