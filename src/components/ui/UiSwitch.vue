<script setup lang="ts">
// role="switch" toggle; Space/Enter come for free from the native button.
const props = withDefaults(defineProps<{
  modelValue: boolean;
  disabled?: boolean;
  loading?: boolean;
  ariaLabel?: string;
}>(), {disabled: false, loading: false});

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'change', value: boolean): void;
}>();

function flip() {
  if (props.disabled || props.loading) return;
  emit('update:modelValue', !props.modelValue);
  emit('change', !props.modelValue);
}
</script>

<template>
  <button type="button"
          role="switch"
          class="px-switch"
          :class="{ 'is-on': modelValue, 'is-loading': loading }"
          :aria-checked="modelValue ? 'true' : 'false'"
          :aria-label="ariaLabel"
          :aria-busy="loading ? 'true' : undefined"
          :disabled="disabled"
          @click="flip">
    <span class="px-switch__knob"></span>
  </button>
</template>

<style scoped>
.px-switch {
  position: relative;
  width: 42px;
  height: 24px;
  padding: 0;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--input-bg);
  cursor: pointer;
  flex-shrink: 0;
  transition: background .15s, border-color .15s;
}

.px-switch.is-on {
  background: var(--accent);
  border-color: transparent;
}

.px-switch:disabled {
  opacity: .5;
  cursor: not-allowed;
}

.px-switch.is-loading {
  opacity: .7;
  cursor: progress;
}

.px-switch__knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, .35);
  transition: left .15s;
}

.is-on .px-switch__knob {
  left: 20px;
}
</style>
