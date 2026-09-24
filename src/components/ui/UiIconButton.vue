<script setup lang="ts">
// Square (r8) or round icon button with an accessible name that doubles as its
// tooltip.
const props = withDefaults(defineProps<{
  label: string;
  size?: number;
  round?: boolean;
  /** Accent-tinted pressed state (toggles such as "hide bad nodes"). */
  active?: boolean;
  /** Red background on hover (delete, quit, close). */
  danger?: boolean;
  /** Filled soft background (log/connection toolbars). */
  soft?: boolean;
  disabled?: boolean;
  loading?: boolean;
  /** Tooltip differs from the accessible name. */
  tip?: string;
  noTip?: boolean;
  pressed?: boolean | undefined;
}>(), {
  size: 32,
  round: false,
  active: false,
  danger: false,
  soft: false,
  disabled: false,
  loading: false,
  noTip: false,
  pressed: undefined,
});

defineEmits<{ (e: 'click', ev: MouseEvent): void }>();

const tipText = computed(() => props.noTip ? '' : (props.tip ?? props.label));
</script>

<template>
  <button type="button"
          class="px-icon-btn"
          :class="{ 'is-round': round, 'is-active': active, 'is-danger': danger, 'is-soft': soft, 'is-loading': loading }"
          :style="{ width: size + 'px', height: size + 'px' }"
          :aria-label="label"
          :aria-pressed="pressed === undefined ? undefined : (pressed ? 'true' : 'false')"
          :aria-busy="loading ? 'true' : undefined"
          :disabled="disabled"
          v-tip="tipText"
          @click="$emit('click', $event)">
    <slot/>
  </button>
</template>

<style scoped>
.px-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 0;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-2);
  cursor: pointer;
  transition: background .15s, color .15s;
}

.px-icon-btn:hover:not(:disabled) {
  background: var(--hover-bg);
  color: var(--text);
}

.px-icon-btn.is-round {
  border-radius: 999px;
}

.px-icon-btn.is-soft {
  background: var(--panel-soft);
  color: var(--text);
}

.px-icon-btn.is-active {
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 16%, transparent);
}

.px-icon-btn.is-danger:hover:not(:disabled) {
  background: var(--error);
  color: #fff;
}

.px-icon-btn:disabled {
  opacity: .45;
  cursor: not-allowed;
}

.px-icon-btn.is-loading {
  color: var(--accent);
  cursor: progress;
}
</style>
