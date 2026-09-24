<script setup lang="ts" generic="T extends string | number | null">
// Select built on UiDropdown: a button showing the current value with an
// explicit ⇅ (field) or chevron (pill) indicator, and a 12px-radius list where
// the chosen option is tinted with the accent and carries a check mark.
import UiDropdown from "./UiDropdown.vue";
import type {UiSelectOption} from "./types";


const props = withDefaults(defineProps<{
  modelValue: T;
  options: UiSelectOption<T>[];
  variant?: 'field' | 'pill';
  placeholder?: string;
  placement?: 'bottom' | 'top';
  align?: 'left' | 'right' | 'stretch';
  minWidth?: number | string;
  maxHeight?: number;
  disabled?: boolean;
  ariaLabel?: string;
  tip?: string;
  capitalize?: boolean;
}>(), {
  variant: 'field',
  placeholder: '—',
  placement: 'bottom',
  align: 'stretch',
  maxHeight: 260,
  disabled: false,
  capitalize: false,
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: T): void;
  (e: 'change', value: T): void;
}>();

const current = computed(() => props.options.find(o => o.value === props.modelValue));

function pick(option: UiSelectOption<T>, close: () => void) {
  if (option.disabled) return;
  close();
  if (option.value !== props.modelValue) {
    emit('update:modelValue', option.value);
    emit('change', option.value);
  }
}
</script>

<template>
  <UiDropdown :placement="placement"
              :align="align"
              :min-width="minWidth ?? '100%'"
              :max-height="maxHeight"
              :disabled="disabled">
    <template #trigger="{ open, toggle, attrs }">
      <button type="button"
              v-bind="attrs"
              class="px-select"
              :class="[`px-select--${variant}`, { 'is-open': open, 'is-capitalize': capitalize }]"
              :disabled="disabled"
              :aria-label="ariaLabel"
              v-tip="tip"
              @click="toggle">
        <slot name="prefix"/>
        <span class="px-select__value ellipsis" v-tip="current && current.label.length > 24 ? current.label : ''">
          <slot name="value" :option="current">{{ current?.label ?? placeholder }}</slot>
        </span>
        <icon-tabler-selector v-if="variant === 'field'" class="px-select__ind" width="15" height="15"/>
        <icon-tabler-chevron-down v-else class="px-select__chev" :class="{ 'is-open': open }" width="14" height="14"/>
      </button>
    </template>
    <template #default="{ close }">
      <button v-for="option in options"
              :key="String(option.value)"
              type="button"
              role="option"
              data-dd-item
              class="px-dd-item"
              :class="{ 'is-selected': option.value === modelValue, 'is-capitalize': capitalize }"
              :aria-selected="option.value === modelValue ? 'true' : 'false'"
              :disabled="option.disabled"
              @click="pick(option, close)">
        <span class="ellipsis" style="flex:1" v-tip="option.label.length > 28 ? option.label : ''">
          <slot name="option" :option="option">{{ option.label }}</slot>
        </span>
        <icon-tabler-check v-if="option.value === modelValue" class="px-select__check" width="14" height="14"/>
      </button>
    </template>
  </UiDropdown>
</template>

<style scoped>
.px-select {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
  cursor: pointer;
  color: var(--text);
  font-size: 13px;
  text-align: left;
}

.px-select:disabled {
  opacity: .55;
  cursor: not-allowed;
}

.px-select--field {
  padding: 9px 10px 9px 12px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--input-bg);
}

.px-select--field:hover:not(:disabled), .px-select--field.is-open {
  border-color: var(--accent);
}

.px-select--pill {
  height: 36px;
  padding: 0 12px 0 16px;
  border-radius: 999px;
  border: none;
  background: var(--panel-soft);
  font-weight: 600;
}

.px-select--pill:hover:not(:disabled), .px-select--pill.is-open {
  background: var(--hover-bg);
}

.px-select__value {
  flex: 1;
}

.is-capitalize {
  text-transform: capitalize;
}

.px-select__ind {
  flex-shrink: 0;
  color: var(--text-2);
}

.px-select__chev {
  flex-shrink: 0;
  opacity: .7;
  transition: transform .15s;
}

.px-select__chev.is-open {
  transform: rotate(180deg);
}

.px-select__check {
  flex-shrink: 0;
  color: var(--accent);
}
</style>
