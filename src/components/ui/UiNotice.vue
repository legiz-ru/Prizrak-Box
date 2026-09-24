<script setup lang="ts">
// Centered dialog: tinted icon tile, title, text and full-width actions.
// Used by UiConfirm and the service notices (HWID, update, first profile…).
import type {Component} from "vue";
import {registerModal} from "./services";

const props = withDefaults(defineProps<{
  modelValue: boolean;
  title: string;
  icon?: Component;
  tone?: 'accent' | 'warning' | 'error' | 'info' | 'success';
  width?: number;
  closeOnEsc?: boolean;
  closeOnOverlay?: boolean;
  role?: 'dialog' | 'alertdialog';
  zIndex?: number;
  iconSize?: 'md' | 'lg';
}>(), {
  tone: 'error',
  width: 400,
  closeOnEsc: true,
  closeOnOverlay: true,
  role: 'dialog',
  zIndex: 70,
  iconSize: 'md',
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'close'): void;
}>();

const titleId = `px-notice-${Math.random().toString(36).slice(2, 9)}`;

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

let downOnOverlay = false;
const onOverlayDown = (e: MouseEvent) => {
  downOnOverlay = e.target === e.currentTarget;
};
const onOverlayUp = (e: MouseEvent) => {
  if (downOnOverlay && e.target === e.currentTarget && props.closeOnOverlay) close();
  downOnOverlay = false;
};
</script>

<template>
  <Teleport to="body">
    <Transition name="px-notice">
      <div v-if="modelValue"
           class="px-notice-overlay"
           :style="{ zIndex }"
           @mousedown="onOverlayDown"
           @mouseup="onOverlayUp">
        <div class="px-notice"
             :role="role"
             aria-modal="true"
             tabindex="-1"
             :aria-labelledby="titleId"
             :style="{ maxWidth: width + 'px' }">
          <slot name="top"/>
          <div v-if="icon" class="px-notice__icon" :class="[`tone-${tone}`, `size-${iconSize}`]">
            <component :is="icon" :width="iconSize === 'lg' ? 30 : 24" :height="iconSize === 'lg' ? 30 : 24"/>
          </div>
          <span :id="titleId" class="px-notice__title">{{ title }}</span>
          <div class="px-notice__text">
            <slot/>
          </div>
          <div v-if="$slots.actions" class="px-notice__actions">
            <slot name="actions"/>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.px-notice-overlay {
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

.px-notice {
  width: 100%;
  background: var(--dialog-bg);
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, .4);
  padding: 24px 24px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 6px;
  user-select: text;
  max-height: calc(100vh - 40px);
  overflow-y: auto;
}

.px-notice__icon {
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 8px;
  flex-shrink: 0;
}

.size-md { width: 48px; height: 48px; }
.size-lg { width: 56px; height: 56px; }

.tone-accent { background: color-mix(in srgb, var(--accent) 16%, transparent); color: var(--accent); }
.tone-warning { background: color-mix(in srgb, var(--warning) 16%, transparent); color: var(--warning); }
.tone-error { background: color-mix(in srgb, var(--error) 16%, transparent); color: var(--error); }
.tone-info { background: color-mix(in srgb, var(--info) 16%, transparent); color: var(--info); }
.tone-success { background: color-mix(in srgb, var(--success) 16%, transparent); color: var(--success); }

.px-notice__title {
  font-size: 16px;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.px-notice__text {
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.5;
  overflow-wrap: anywhere;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.px-notice__actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
  width: 100%;
}

.px-notice__actions > :deep(*) {
  flex: 1;
}

.px-notice-enter-active, .px-notice-leave-active {
  transition: opacity .15s ease;
}

.px-notice-enter-from, .px-notice-leave-to {
  opacity: 0;
}
</style>
