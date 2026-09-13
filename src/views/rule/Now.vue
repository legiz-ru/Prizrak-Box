<script setup lang="ts">

import MySimpleInput from "@/components/MySimpleInput.vue";
import createApi from "@/api";
import {useWebStore} from "@/store/webStore";


// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);


// 原始大数据集合
const allData = ref([]);
const filterData = ref([]);

// 分页数据状态
const itemsPerPage = 50; // 每页加载50条数据
const currentPage = ref(1); // 当前页数
const paginatedData = ref([]);

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
    allData.value = res;
    filterData.value = res;
    // 初始化分页数据
    paginatedData.value = allData.value.slice(0, itemsPerPage);
  });
});

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
    <MySimpleInput
        :onInputChange="handleInputChange"
        :placeholder="$t('rule.now.search')"
        class="search"
    ></MySimpleInput>

    <div class="px-surface px-table rules-table">
      <div class="px-table__head">
        <span>{{ $t('rule.now.type') }}</span>
        <span>{{ $t('rule.now.payload') }}</span>
        <span>{{ $t('rule.now.proxy') }}</span>
      </div>
      <div class="px-table__body" @scroll="handleScroll">
        <p v-if="paginatedData.length === 0" class="px-table__empty">
          {{ $t('connections.noData') }}
        </p>
        <div class="px-table__row" v-for="(item, i) in paginatedData" :key="i">
          <span class="px-table__cell rules-table__type">{{ item.type }}</span>
          <span class="px-table__cell">{{ item.payload }}</span>
          <span class="px-table__cell">{{ item.proxy }}</span>
        </div>
      </div>
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

.now {
  width: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  gap: var(--px-space-3);
}

.search :deep(.custom-input) {
  border-radius: var(--px-r-pill);
  padding-left: var(--px-space-4);
}

.search :deep(.clear-button) {
  right: 14px;
}

/* Ширины колонок задаёт экран, механику строк — .px-table из tokens.css. */
.rules-table {
  grid-template-columns: 140px minmax(0, 1fr) 180px;
  flex: 1;
}

/* Тип правила — короткий машинный идентификатор, моноширинный он читается
   как столбец, а не как продолжение соседней ячейки. */
.rules-table__type {
  font-family: var(--px-font-num);
  font-size: var(--px-fs-caption);
  color: var(--px-text-muted);
}
</style>
