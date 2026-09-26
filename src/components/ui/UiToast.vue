<script setup lang="ts">
// Toast stack: top centre, at most four, each disappears after 2.6 s.
import {toastState, dismissToast, loadingState} from "./services";
import UiSpinner from "./UiSpinner.vue";
</script>

<template>
  <Teleport to="body">
    <div class="px-toasts" role="status" aria-live="polite">
      <TransitionGroup name="px-toast">
        <div v-for="item in toastState.items"
             :key="item.id"
             class="px-toast"
             @click="dismissToast(item.id)">
          <icon-tabler-circle-check v-if="item.type === 'success'" class="px-toast__icon tone-success" width="16" height="16"/>
          <icon-tabler-circle-x v-else-if="item.type === 'error'" class="px-toast__icon tone-error" width="16" height="16"/>
          <icon-tabler-alert-triangle v-else-if="item.type === 'warning'" class="px-toast__icon tone-warning" width="16" height="16"/>
          <icon-tabler-info-circle v-else class="px-toast__icon tone-info" width="16" height="16"/>
          <span class="px-toast__text">{{ item.text }}</span>
        </div>
      </TransitionGroup>
    </div>
    <Transition name="px-toast">
      <div v-if="loadingState.count" class="px-loading" aria-busy="true">
        <div class="px-loading__box" role="status">
          <UiSpinner :size="22"/>
          <span v-if="loadingState.text">{{ loadingState.text }}</span>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.px-toasts {
  position: fixed;
  top: 74px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9000;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  pointer-events: none;
  max-width: min(560px, calc(100vw - 32px));
}

.px-toast {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 14px;
  border-radius: 12px;
  background: var(--dialog-bg);
  border: 1px solid var(--border);
  box-shadow: 0 12px 30px rgba(0, 0, 0, .3);
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  pointer-events: auto;
  cursor: default;
  max-width: 100%;
}

.px-toast__text {
  overflow-wrap: anywhere;
  line-height: 1.4;
}

.px-toast__icon {
  flex-shrink: 0;
}

.tone-success { color: var(--success); }
.tone-error { color: var(--error); }
.tone-warning { color: var(--warning); }
.tone-info { color: var(--info); }

.px-toast-enter-active {
  animation: px-toast-in .18s ease-out;
}

.px-toast-leave-active {
  transition: opacity .15s ease;
}

.px-toast-leave-to {
  opacity: 0;
}

.px-loading {
  position: fixed;
  inset: 0;
  z-index: 8500;
  background: rgba(0, 0, 0, .2);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: progress;
}

.px-loading__box {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 20px;
  border-radius: 12px;
  background: var(--dialog-bg);
  border: 1px solid var(--border);
  box-shadow: 0 16px 40px rgba(0, 0, 0, .35);
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
</style>
