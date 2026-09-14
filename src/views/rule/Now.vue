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

    <div class="px-surface px-table rule-table">
      <div class="px-table__head">
        <div>{{ $t('rule.now.type') }}</div>
        <div>{{ $t('rule.now.payload') }}</div>
        <div>{{ $t('rule.now.proxy') }}</div>
      </div>
      <div class="px-table__body" @scroll="handleScroll">
        <div
            class="px-table__row"
            v-for="(item, i) in paginatedData"
            :key="i"
        >
          <div class="px-table__cell">{{ item.type }}</div>
          <div class="px-table__cell">{{ item.payload }}</div>
          <div class="px-table__cell">{{ item.proxy }}</div>
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
  margin-left: 0;
  margin-top: 5px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.search {
  margin-top: 6px;
}

/* Радиус, отступ и позиция кнопки очистки — теперь значения по умолчанию
   в MySimpleInput.vue самого поля, а не переопределение здесь. */

/* .px-table (tokens.css) — общий стол для правил/провайдеров: шапка,
   строки через разделитель, слабая зебра через nth-child(even), скроллбар
   на тех же токенах. Раньше это была рамка 2px var(--text-color) — вдвое
   толще и другого цвета, чем 1px var(--sub-card-border), которым обведена
   любая другая карточка/панель в приложении. */
.rule-table {
  grid-template-columns: 5fr 14fr 5fr;
  /* Тот же зазор тулбар→контент, что у карточек и стола провайдеров
     (var(--px-space-5), 20px) — здесь раньше был свой 25px. */
  margin-top: var(--px-space-5);
  flex: 1;
  min-height: 0;
}
</style>
