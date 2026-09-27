<script setup lang="ts" generic="T extends string | number | boolean">
// Segmented pills: icon-only (with tooltip) or text, the active one filled with
// the accent.
import type {UiPillOption} from "./types";


const props = withDefaults(defineProps<{
  modelValue: T;
  options: UiPillOption<T>[];
  size?: 'sm' | 'md';
  ariaLabel?: string;
  stretch?: boolean;
  disabled?: boolean;
}>(), {size: 'sm', stretch: false, disabled: false});

const emit = defineEmits<{
  (e: 'update:modelValue', value: T): void;
  (e: 'change', value: T): void;
}>();

function pick(value: T) {
  if (props.disabled || value === props.modelValue) return;
  emit('update:modelValue', value);
  emit('change', value);
}
</script>

<template>
  <div class="px-seg" :class="{ 'is-md': size === 'md', 'is-stretch': stretch, 'is-disabled': disabled }" role="group" :aria-label="ariaLabel" :aria-disabled="disabled ? 'true' : undefined">
    <button v-for="option in options"
            :key="String(option.value)"
            type="button"
            class="px-seg__item"
            :class="{ 'is-active': option.value === modelValue, 'px-seg__item--icon': !option.label }"
            :aria-pressed="option.value === modelValue ? 'true' : 'false'"
            :disabled="disabled"
            :aria-label="option.label ? undefined : (option.tip || String(option.value))"
            v-tip="option.tip"
            @click="pick(option.value)">
      <component :is="option.icon" v-if="option.icon" :width="size === 'md' ? 16 : 14" :height="size === 'md' ? 16 : 14"/>
      <span v-if="option.label">{{ option.label }}</span>
    </button>
  </div>
</template>

<style scoped>
.is-md .px-seg__item:not(.px-seg__item--icon) {
  height: 30px;
  font-size: 13px;
}

.is-stretch .px-seg__item {
  flex: 1;
}

.is-disabled {
  opacity: .55;
}

.is-disabled .px-seg__item {
  cursor: not-allowed;
}
</style>
