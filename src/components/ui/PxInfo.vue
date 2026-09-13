<script setup lang="ts">
// Иконка информации с подсказкой.
//
// Нужна там, где текст не является описанием настройки: диагностика (HWID),
// оговорка про условия работы (уведомления о подписке), версия ядра. Такой
// текст постоянной строкой раздувает ряд и спорит с меткой, поэтому живёт под
// иконкой.
//
// Триггер сделан фокусируемым: подсказка по наведению, недоступная с
// клавиатуры, для части пользователей означает, что текста нет вовсе.
withDefaults(defineProps<{
  /** Однострочная подсказка. */
  content?: string;
  /** Многострочная подсказка — каждая строка отдельным абзацем. */
  lines?: string[];
  placement?: string;
  /** Доступное имя, когда сам текст подсказки как имя не годится. */
  label?: string;
}>(), {placement: 'top'});
</script>

<template>
  <el-tooltip :placement="placement" effect="dark" :show-after="150">
    <template #content>
      <div class="px-info__body">
        <slot>
          <template v-if="lines">
            <div v-for="line in lines" :key="line">{{ line }}</div>
          </template>
          <template v-else>{{ content }}</template>
        </slot>
      </div>
    </template>
    <span class="px-info" tabindex="0" :aria-label="label ?? content">
      <el-icon><icon-tabler-info-circle/></el-icon>
    </span>
  </el-tooltip>
</template>

<style scoped>
.px-info {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: var(--px-r-pill);
  color: var(--text-color);
  opacity: .55;
  cursor: help;
  transition: opacity var(--px-dur) var(--px-ease);
}

.px-info:hover,
.px-info:focus-visible {
  opacity: 1;
}

.px-info :deep(svg) {
  width: 18px;
  height: 18px;
}

.px-info__body {
  max-width: 320px;
  line-height: var(--px-lh-body);
  /* Диагностика вроде HWID=… набирается моноширинным в самом слоте; здесь
     только перенос длинных значений, чтобы подсказка не уезжала за экран. */
  overflow-wrap: anywhere;
}
</style>
