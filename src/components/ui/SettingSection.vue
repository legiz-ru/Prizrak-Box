<script setup lang="ts">
// Группа настроек: подложка + заголовок + строки.
//
// Подложка (.px-surface) нужна, чтобы текст не лежал прямо на обоях. Её
// плотность считает theme.ts под конкретную картинку: на тёмных обоях ноль,
// и группа выглядит ровно как прежняя карточка.
defineProps<{
  title: string;
  /** Короткая приписка справа от заголовка — например статус обновления. */
  note?: string;
  noteType?: 'info' | 'success' | 'warning' | 'danger';
}>();
</script>

<template>
  <section class="px-surface setting-section">
    <header class="px-section__head">
      <h3 class="px-section__title">{{ title }}</h3>
      <span
          v-if="note"
          class="px-section__note"
          :class="noteType && `px-section__note--${noteType}`"
      >{{ note }}</span>
      <slot name="head"/>
    </header>
    <div class="setting-section__rows">
      <slot/>
    </div>
  </section>
</template>

<style scoped>
.setting-section {
  /* Отступы снаружи не задаются: расстояние между группами держит родитель
     через gap, иначе поля складываются и разъезжаются от экрана к экрану. */
  overflow: hidden;
}

.setting-section__rows {
  display: flex;
  flex-direction: column;
}

.px-section__note--success {
  color: var(--px-ok);
}

.px-section__note--warning {
  color: var(--px-warn);
}

.px-section__note--danger {
  color: var(--px-danger);
}
</style>
