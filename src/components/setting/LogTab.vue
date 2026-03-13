<script setup lang="ts">
import {ref} from "vue";
import MySimpleInput from "@/components/MySimpleInput.vue";
import {useWebStore} from "@/store/webStore";

const webStore = useWebStore();
const search = ref("");

function handleInputChange(value: any) {
  search.value = value;
}

function filterData() {
  return webStore.logs.filter((data: any) => {
    const searchLower = search.value.toLowerCase();
    return (
        !search.value ||
        data.payload.toLowerCase().includes(searchLower) ||
        data.type.toLowerCase().includes(searchLower)
    );
  });
}
</script>

<template>
  <div class="conn">
    <div class="search">
      <MySimpleInput
          :onInputChange="handleInputChange"
          :placeholder="$t('log.search')"
          class="search"
      />
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

<style scoped>
.conn {
  width: 95%;
  margin-left: 10px;
  margin-top: 2px;
}

.search {
  width: 400px;
}

.search :deep(.custom-input) {
  border-radius: 999px;
  padding-left: 16px;
}

.search :deep(.clear-button) {
  right: 14px;
}

.content {
  border: 2px solid var(--text-color);
  border-radius: 20px;
  margin-top: 20px;
  width: 95%;
  margin-left: 10px;
  overflow: hidden;
}

.info-list {
  max-height: calc(100vh - 250px);
  overflow-y: auto;
}

.info {
  border-bottom: 1px solid var(--sub-card-border);
  padding: 5px 10px 5px 16px;
  font-size: 14px;
  line-height: 1.5;
  user-select: text;
  background-color: var(--left-bg-color);
}

.info-list::-webkit-scrollbar {
  width: 5px;
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
