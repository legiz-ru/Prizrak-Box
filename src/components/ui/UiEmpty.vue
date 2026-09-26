<script setup lang="ts">
// Empty state: icon, title, explanation and an optional action button.
import type {Component} from "vue";

defineProps<{
  icon?: Component;
  title: string;
  text?: string;
  actionLabel?: string;
  actionIcon?: Component;
}>();

defineEmits<{ (e: 'action'): void }>();
</script>

<template>
  <div class="px-empty">
    <div class="px-empty__icon">
      <component :is="icon" v-if="icon" width="26" height="26" stroke-width="1.8"/>
    </div>
    <span class="px-empty__title">{{ title }}</span>
    <span v-if="text" class="px-empty__text">{{ text }}</span>
    <button v-if="actionLabel" type="button" class="px-btn px-btn--primary px-btn--pill px-empty__action" @click="$emit('action')">
      <component :is="actionIcon" v-if="actionIcon" width="15" height="15"/>
      {{ actionLabel }}
    </button>
  </div>
</template>

<style scoped>
.px-empty {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  gap: 8px;
  padding: 40px 16px;
}

.px-empty__icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  background: var(--panel-soft);
  color: var(--text-3);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 4px;
}

.px-empty__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
}

.px-empty__text {
  font-size: 13px;
  color: var(--text-2);
  max-width: 320px;
  line-height: 1.5;
}

.px-empty__action {
  margin-top: 6px;
  padding: 8px 18px;
}
</style>
