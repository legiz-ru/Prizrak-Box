<script setup lang="ts">
// Группа настроек: подложка + заголовок + строки.
//
// Подложка (.px-surface) нужна, чтобы текст не лежал прямо на обоях. Её
// плотность считает theme.ts под конкретную картинку: на тёмных обоях ноль,
// и группа выглядит ровно как прежняя карточка.
//
// В заголовке два дополнительных места: слот title-after — для иконки
// информации рядом с названием, и actions — для действий всего раздела,
// прижатых к правому краю (обновление приложения, например). Такие действия
// не относятся ни к одной настройке по отдельности, поэтому отдельной строки
// внизу им не нужно.
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
      <slot name="title-after"/>
      <span
          v-if="note"
          class="px-section__note"
          :class="noteType && `px-section__note--${noteType}`"
      >{{ note }}</span>
      <div class="px-section__actions">
        <slot name="actions"/>
      </div>
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

.px-section__head {
  min-height: 44px;
}

.px-section__actions {
  display: flex;
  align-items: center;
  gap: var(--px-space-1);
  margin-left: auto;
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
