<script setup lang="ts">
// Dialog shell: blurred overlay, 12px card, header "icon + title + ×",
// optional footer with the actions on the right. Esc and overlay clicks close it
// unless disabled (the multi-profile warning must be answered explicitly).
import type {Component} from "vue";
import {registerModal} from "./services";

const props = withDefaults(defineProps<{
  modelValue: boolean;
  title?: string;
  /** Tabler icon shown in a tinted tile before the title. */
  icon?: Component;
  tone?: 'accent' | 'warning' | 'error' | 'info' | 'success';
  width?: number;
  closeOnEsc?: boolean;
  closeOnOverlay?: boolean;
  showClose?: boolean;
  /** Header and footer separated from the body by hairlines. */
  divided?: boolean;
  /** No overlay blur (the theme dialog must show the background it edits). */
  clearOverlay?: boolean;
  bodyClass?: string;
  zIndex?: number;
  ariaLabel?: string;
}>(), {
  title: '',
  tone: 'accent',
  width: 480,
  closeOnEsc: true,
  closeOnOverlay: true,
  showClose: true,
  divided: false,
  clearOverlay: false,
  bodyClass: '',
  zIndex: 60,
  ariaLabel: '',
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'close'): void;
}>();

const titleId = `px-modal-${Math.random().toString(36).slice(2, 9)}`;
const slots = useSlots();

function close() {
  emit('update:modelValue', false);
  emit('close');
}

let unregister: (() => void) | null = null;
watch(() => props.modelValue, (open) => {
  unregister?.();
  unregister = open ? registerModal(close, () => props.closeOnEsc) : null;
}, {immediate: true});
onBeforeUnmount(() => unregister?.());

// Only a click that both starts and ends on the overlay closes the dialog, so
// selecting text in an input and releasing outside does not dismiss it.
let downOnOverlay = false;
const onOverlayDown = (e: MouseEvent) => {
  downOnOverlay = e.target === e.currentTarget;
};
const onOverlayUp = (e: MouseEvent) => {
  if (downOnOverlay && e.target === e.currentTarget && props.closeOnOverlay) close();
  downOnOverlay = false;
};

const hasHeader = computed(() => !!(props.title || props.icon || slots.header));
</script>

<template>
  <Teleport to="body">
    <Transition name="px-modal">
      <div v-if="modelValue"
           class="px-modal-overlay"
           :class="{ 'is-clear': clearOverlay }"
           :style="{ zIndex }"
           @mousedown="onOverlayDown"
           @mouseup="onOverlayUp">
        <div class="px-modal"
             :class="{ 'is-divided': divided }"
             role="dialog"
             aria-modal="true"
             tabindex="-1"
             :aria-labelledby="title ? titleId : undefined"
             :aria-label="!title ? (ariaLabel || undefined) : undefined"
             :style="{ maxWidth: width + 'px' }">
          <div v-if="hasHeader" class="px-modal__head">
            <slot name="header">
              <div v-if="icon" class="px-modal__icon" :class="`tone-${tone}`">
                <component :is="icon" width="20" height="20"/>
              </div>
              <span :id="titleId" class="px-modal__title ellipsis">{{ title }}</span>
            </slot>
            <button v-if="showClose"
                    type="button"
                    class="px-modal__close"
                    :aria-label="$t('close')"
                    v-tip="$t('close')"
                    @click="close">
              <icon-tabler-x width="16" height="16"/>
            </button>
          </div>
          <div class="px-modal__body" :class="bodyClass">
            <slot/>
          </div>
          <div v-if="slots.footer || slots['footer-left']" class="px-modal__foot">
            <div class="px-modal__foot-left">
              <slot name="footer-left"/>
            </div>
            <div class="px-modal__foot-right">
              <slot name="footer"/>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.px-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, .5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  -webkit-app-region: no-drag;
  --wails-draggable: no-drag;
}

.px-modal-overlay.is-clear {
  background: rgba(0, 0, 0, .28);
  backdrop-filter: none;
}

.px-modal {
  width: 100%;
  max-height: min(780px, calc(100vh - 40px));
  background: var(--dialog-bg);
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, .4);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  user-select: text;
}

.px-modal__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 18px 16px 4px 22px;
  flex-shrink: 0;
  min-width: 0;
}

.is-divided .px-modal__head {
  padding: 16px 16px 14px 22px;
  border-bottom: 1px solid var(--border);
}

.px-modal__icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tone-accent { background: color-mix(in srgb, var(--accent) 16%, transparent); color: var(--accent); }
.tone-warning { background: color-mix(in srgb, var(--warning) 16%, transparent); color: var(--warning); }
.tone-error { background: color-mix(in srgb, var(--error) 16%, transparent); color: var(--error); }
.tone-info { background: color-mix(in srgb, var(--info) 16%, transparent); color: var(--info); }
.tone-success { background: color-mix(in srgb, var(--success) 16%, transparent); color: var(--success); }

.px-modal__title {
  flex: 1;
  font-size: 16px;
  font-weight: 700;
}

.px-modal__close {
  width: 32px;
  height: 32px;
  border-radius: 999px;
  border: none;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-2);
  cursor: pointer;
  flex-shrink: 0;
  margin-left: auto;
}

.px-modal__close:hover {
  background: var(--hover-bg);
  color: var(--accent);
}

.px-modal__body {
  padding: 14px 22px 4px;
  overflow-y: auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.is-divided .px-modal__body {
  padding: 20px 22px;
}

.px-modal__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 18px 22px 20px;
  flex-shrink: 0;
}

.is-divided .px-modal__foot {
  padding: 14px 22px;
  border-top: 1px solid var(--border);
}

.px-modal__foot-left, .px-modal__foot-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.px-modal__foot-right {
  margin-left: auto;
}

.px-modal-enter-active, .px-modal-leave-active {
  transition: opacity .15s ease;
}

.px-modal-enter-active .px-modal, .px-modal-leave-active .px-modal {
  transition: transform .15s ease;
}

.px-modal-enter-from, .px-modal-leave-to {
  opacity: 0;
}

.px-modal-enter-from .px-modal, .px-modal-leave-to .px-modal {
  transform: scale(.97);
}
</style>
