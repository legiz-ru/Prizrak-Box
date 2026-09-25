<script setup lang="ts">

import createApi from "@/api";
import {useWebStore} from "@/store/webStore";


// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);


// 原始大数据集合
const allData = ref<any[]>([]);
const filterData = ref<any[]>([]);
const query = ref("");
const loading = ref(true);

// 分页数据状态
const itemsPerPage = 50; // 每页加载50条数据
const currentPage = ref(1); // 当前页数
const paginatedData = ref<any[]>([]);

// 加载下一页数据
function loadMore() {
  if (currentPage.value * itemsPerPage >= filterData.value.length) return; // 没有更多数据时停止加载
  currentPage.value++;
  const nextPageData = filterData.value.slice(
      (currentPage.value - 1) * itemsPerPage,
      currentPage.value * itemsPerPage
  );
  paginatedData.value = [...paginatedData.value, ...nextPageData];
}

// 监听滚动事件
function handleScroll(event: Event) {
  const target = event.target as HTMLElement;
  if (
      target.scrollTop + target.clientHeight >= target.scrollHeight - 10
  ) {
    loadMore(); // 滚动到底部时加载更多
  }
}


onMounted(() => {
  api.getRules().then((res) => {
    allData.value = Array.isArray(res) ? res : [];
    filterData.value = allData.value;
    // 初始化分页数据
    paginatedData.value = allData.value.slice(0, itemsPerPage);
  }).finally(() => {
    loading.value = false;
  });
});

watch(query, (value) => handleInputChange(value));

// 过滤数据
function handleInputChange(value: any) {
  if (value) {
    filterData.value = allData.value.filter((item: any) => {
      return (
          item.type.toLowerCase().includes(value.toLowerCase()) ||
          item.payload.toLowerCase().includes(value.toLowerCase()) ||
          item.proxy.toLowerCase().includes(value.toLowerCase())
      );
    });
  } else {
    filterData.value = allData.value;
  }
  // 重置分页数据
  currentPage.value = 1;
  paginatedData.value = filterData.value.slice(0, itemsPerPage);
}

// 监控配置切换
const webStore = useWebStore();
watch(() => webStore.fProfile, async () => {
  await api.waitRunning()
  api.getRules().then((res) => {
    allData.value = res;
    filterData.value = res;
    // 初始化分页数据
    paginatedData.value = allData.value.slice(0, itemsPerPage);
  });
})

</script>

<template>
  <div class="now">
    <div class="px-search now-search">
      <icon-tabler-search width="15" height="15"/>
      <input v-model="query" class="px-input" type="search" :placeholder="$t('rule.now.search')" :aria-label="$t('rule.now.search')">
    </div>

    <div class="now-table" role="table" :aria-label="$t('rule.now.title')">
      <div class="now-row now-row--head" role="row">
        <span role="columnheader">{{ $t('rule.now.type') }}</span>
        <span role="columnheader">{{ $t('rule.now.payload') }}</span>
        <span role="columnheader">{{ $t('rule.now.proxy') }}</span>
      </div>
      <div class="now-list" @scroll.passive="handleScroll">
        <div v-if="loading" class="now-loading"><UiSpinner/></div>
        <div v-for="(item, i) in paginatedData"
             :key="i"
             class="now-row"
             :class="{ 'is-odd': i % 2 === 1 }"
             role="row">
          <span class="ellipsis now-type" role="cell">{{ item.type }}</span>
          <span class="ellipsis mono" role="cell" v-tip="item.payload && item.payload.length > 40 ? item.payload : ''">{{ item.payload }}</span>
          <span class="ellipsis now-proxy" role="cell" v-tip="item.proxy">{{ item.proxy }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.now {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0 28px 28px;
  gap: 10px;
}

.now-search {
  max-width: 340px;
}

.now-table {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}

.now-row {
  display: grid;
  grid-template-columns: 1.4fr 2.4fr 1fr;
  gap: 12px;
  padding: 9px 14px;
  font-size: 13px;
  border-bottom: 1px solid var(--border);
  min-width: 0;
}

.now-row.is-odd {
  background: var(--input-bg);
}

.now-row:not(.now-row--head):hover {
  background: var(--hover-bg);
}

.now-row--head {
  padding: 10px 14px;
  font-size: 12px;
  font-weight: 700;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: .04em;
  background: var(--input-bg);
}

.now-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.now-type {
  color: var(--text-2);
}

.now-proxy {
  color: var(--accent);
  font-weight: 600;
}

.now-loading {
  display: flex;
  justify-content: center;
  padding: 24px;
  color: var(--text-2);
}
</style>
