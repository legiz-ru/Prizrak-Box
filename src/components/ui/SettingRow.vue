<script setup lang="ts">
// Строка настройки: метка и пояснение слева, контрол справа.
//
// Ключевое отличие от прежней вёрстки (`<strong>Метка :</strong><контрол>`) —
// контрол не стоит сразу за меткой, а прижат к правому краю. За счёт этого все
// переключатели группы выстраиваются по одной вертикали, и раздел читается
// сверху вниз, а не зигзагом.
//
// Пояснение — постоянный текст, а не тултип: то, что раньше было доступно
// только по наведению на иконку, теперь видно сразу.
defineProps<{
  label?: string;
  hint?: string;
  /** Контрол занимает всю ширину и встаёт под меткой (списки, результаты). */
  stacked?: boolean;
}>();
</script>

<template>
  <div class="px-row" :class="{ 'px-row--stacked': stacked }">
    <div class="px-row__text">
      <div class="px-row__label">
        <slot name="label">{{ label }}</slot>
      </div>
      <p v-if="hint" class="px-row__hint">{{ hint }}</p>
      <slot name="hint"/>
    </div>
    <div class="px-row__control">
      <slot/>
    </div>
  </div>
</template>

<style scoped>
.px-row__text {
  min-width: 0;
}

.px-row--stacked {
  grid-template-columns: minmax(0, 1fr);
  align-items: stretch;
  gap: var(--px-space-3);
}

.px-row--stacked .px-row__control {
  justify-content: flex-start;
}
</style>
