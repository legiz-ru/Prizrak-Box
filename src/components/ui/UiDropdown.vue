<script setup lang="ts">
// Anchored popup list. The trigger slot receives `attrs` to spread onto its
// button (marks it as the list's trigger for focus return) and `toggle`.
// Items should carry `data-dd-item` so ArrowUp/ArrowDown walk them (a11y.ts).
// Closes on outside click, Esc (focus goes back to the trigger) and item pick.
import {focusables} from "./a11y";

const props = withDefaults(defineProps<{
  placement?: 'bottom' | 'top';
  align?: 'left' | 'right' | 'stretch';
  minWidth?: number | string;
  maxWidth?: number | string;
  maxHeight?: number;
  /** Open on hover (language menu in the sidebar). */
  hover?: boolean;
  disabled?: boolean;
  /** Extra classes for the panel. */
  panelClass?: string;
  role?: 'listbox' | 'menu';
}>(), {
  placement: 'bottom',
  align: 'left',
  maxHeight: 320,
  hover: false,
  disabled: false,
  panelClass: '',
  role: 'listbox',
});

const emit = defineEmits<{ (e: 'open'): void; (e: 'close'): void }>();

const open = ref(false);
const root = ref<HTMLElement | null>(null);
const panel = ref<HTMLElement | null>(null);
let hoverTimer: number | undefined;

const px = (v: number | string | undefined) => v === undefined ? undefined : (typeof v === 'number' ? v + 'px' : v);

const panelStyle = computed(() => ({
  [props.placement === 'top' ? 'bottom' : 'top']: 'calc(100% + 8px)',
  ...(props.align === 'right' ? {right: 0} : props.align === 'stretch' ? {left: 0, right: 0} : {left: 0}),
  minWidth: px(props.minWidth),
  maxWidth: px(props.maxWidth),
  maxHeight: props.maxHeight + 'px',
}));

function setOpen(value: boolean, focusItem = false) {
  if (props.disabled && value) return;
  if (open.value === value) return;
  open.value = value;
  if (value) {
    emit('open');
    if (focusItem) {
      nextTick(() => {
        if (!panel.value) return;
        const items = focusables(panel.value).filter(el => el.hasAttribute('data-dd-item'));
        const selected = items.find(el => el.getAttribute('aria-selected') === 'true');
        (selected ?? items[0])?.focus({preventScroll: false});
      });
    } else {
      nextTick(() => {
        panel.value?.querySelector<HTMLElement>('[aria-selected="true"]')?.scrollIntoView({block: 'nearest'});
      });
    }
  } else {
    emit('close');
  }
}

function toggle(e?: Event) {
  // A keyboard click (Enter/Space) has detail 0: move focus into the list.
  const viaKeyboard = !!e && (e as MouseEvent).detail === 0;
  setOpen(!open.value, viaKeyboard);
}

function close() {
  setOpen(false);
}

function focusTrigger() {
  root.value?.querySelector<HTMLElement>('[data-dd-trigger]')?.focus();
}

function onKeydown(e: KeyboardEvent) {
  if (!open.value) {
    if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && (e.target as HTMLElement).hasAttribute('data-dd-trigger')) {
      e.preventDefault();
      setOpen(true, true);
    }
    return;
  }
  if (e.key === 'Escape') {
    e.preventDefault();
    e.stopPropagation();
    setOpen(false);
    focusTrigger();
  } else if (e.key === 'Tab') {
    setOpen(false);
  }
}

function onDocDown(e: MouseEvent) {
  if (open.value && root.value && !root.value.contains(e.target as Node)) setOpen(false);
}

function onEnter() {
  if (!props.hover) return;
  window.clearTimeout(hoverTimer);
  setOpen(true);
}

function onLeave() {
  if (!props.hover) return;
  window.clearTimeout(hoverTimer);
  hoverTimer = window.setTimeout(() => setOpen(false), 200);
}

onMounted(() => document.addEventListener('mousedown', onDocDown, true));
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocDown, true);
  window.clearTimeout(hoverTimer);
});

const triggerAttrs = computed(() => ({
  'data-dd-trigger': '',
  'aria-haspopup': props.role,
  'aria-expanded': open.value,
}));

defineExpose({open: () => setOpen(true), close, toggle, isOpen: open});
</script>

<template>
  <div ref="root"
       class="px-dd"
       data-dd
       @keydown="onKeydown"
       @mouseenter="onEnter"
       @mouseleave="onLeave">
    <slot name="trigger" :open="open" :toggle="toggle" :attrs="triggerAttrs"/>
    <div v-if="open"
         ref="panel"
         class="px-dd-panel"
         :class="panelClass"
         :role="role"
         :style="panelStyle">
      <slot :close="close"/>
    </div>
  </div>
</template>

<style scoped>
.px-dd {
  position: relative;
  min-width: 0;
}
</style>
