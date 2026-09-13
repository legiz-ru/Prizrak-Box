<script setup lang="ts">
import {computed, ref} from "vue";
import MySimpleInput from "@/components/MySimpleInput.vue";
import {useWebStore} from "@/store/webStore";

const webStore = useWebStore();
const search = ref("");

function handleInputChange(value: any) {
  search.value = value;
}

// Фильтрация в computed, а не вызовом функции прямо в шаблоне: иначе список
// пересчитывался на каждый рендер, в том числе на каждое новое сообщение.
const rows = computed(() => {
  const needle = search.value.trim().toLowerCase();
  if (!needle) return webStore.logs;
  return webStore.logs.filter((item: any) =>
      item.payload?.toLowerCase().includes(needle) ||
      item.type?.toLowerCase().includes(needle));
});

// Уровень задаётся и цветом, и словом: цвет один смысл не несёт.
const levelClass = (type: string) => {
  switch ((type || '').toUpperCase()) {
    case 'ERROR':
      return 'log-level--error';
    case 'WARNING':
    case 'WARN':
      return 'log-level--warn';
    case 'INFO':
      return 'log-level--info';
    default:
      return 'log-level--debug';
  }
};
</script>

<template>
  <div class="log">
    <div class="log__toolbar">
      <MySimpleInput
          class="log__search"
          :onInputChange="handleInputChange"
          :placeholder="$t('log.search')"
      />
      <span class="log__count px-num">{{ rows.length }}</span>
    </div>

    <div class="log__list px-surface">
      <p v-if="rows.length === 0" class="log__empty">{{ $t('connections.noData') }}</p>
      <div v-for="item in rows" :key="item.id" class="log__row">
        <span class="log__time px-num">{{ item.time }}</span>
        <span class="log__level" :class="levelClass(item.type)">{{ item.type }}</span>
        <span class="log__payload">{{ item.payload }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.log {
  display: flex;
  flex-direction: column;
  gap: var(--px-space-3);
  min-height: 0;
  flex: 1;
}

.log__toolbar {
  display: flex;
  align-items: center;
  gap: var(--px-space-3);
}

.log__search {
  flex: 1;
}

.log__search :deep(.custom-input) {
  border-radius: var(--px-r-pill);
  padding-left: var(--px-space-4);
}

.log__search :deep(.clear-button) {
  right: 14px;
}

.log__count {
  font-size: var(--px-fs-caption);
  color: var(--px-text-muted);
}

.log__list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  /* Рамка приходит из .px-surface: раньше здесь была сплошная линия в два
     пикселя цветом текста, которая спорила с содержимым сильнее, чем сам лог. */
}

.log__empty {
  margin: 0;
  padding: var(--px-space-6);
  text-align: center;
  color: var(--px-text-muted);
  font-size: var(--px-fs-small);
}

.log__row {
  display: grid;
  grid-template-columns: max-content 74px minmax(0, 1fr);
  gap: var(--px-space-3);
  align-items: baseline;
  padding: var(--px-space-2) var(--px-row-pad-x);
  border-top: 1px solid var(--sub-card-border);
  /* Строку лога нужно уметь выделить и скопировать: глобально по приложению
     стоит user-select: none. */
  user-select: text;
}

.log__row:first-of-type {
  border-top: 0;
}

.log__time {
  font-size: var(--px-fs-caption);
  color: var(--px-text-muted);
  white-space: nowrap;
}

.log__level {
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: .06em;
  text-align: center;
  padding: 1px 0;
  border-radius: var(--px-r-pill);
  border: 1px solid currentColor;
}

.log-level--error {
  color: var(--px-danger);
}

.log-level--warn {
  color: var(--px-warn);
}

.log-level--info {
  color: var(--px-info);
}

.log-level--debug {
  color: var(--px-text-muted);
}

.log__payload {
  font-size: var(--px-fs-small);
  line-height: var(--px-lh-body);
  overflow-wrap: anywhere;
}

.log__list::-webkit-scrollbar {
  width: 5px;
}

.log__list::-webkit-scrollbar-track {
  background: transparent;
}

.log__list::-webkit-scrollbar-thumb {
  background: var(--scrollbar-bg);
  border-radius: 2px;
}

.log__list::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-hover-bg);
}
</style>
