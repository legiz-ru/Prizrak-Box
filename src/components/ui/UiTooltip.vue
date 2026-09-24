<script setup lang="ts">
// The single tooltip bubble driven by tooltip.ts.
import {tooltipState} from "./tooltip";

const MAX_W = 280;
const box = ref<HTMLElement | null>(null);
const left = ref(0);

// Measure after render so the bubble never leaves the window horizontally.
watch(() => [tooltipState.visible, tooltipState.text, tooltipState.x], async () => {
  if (!tooltipState.visible) return;
  await nextTick();
  const w = box.value?.offsetWidth ?? Math.min(MAX_W, tooltipState.text.length * 6.7 + 24);
  left.value = Math.max(8, Math.min(tooltipState.x - w / 2, window.innerWidth - w - 8));
});
</script>

<template>
  <Teleport to="body">
    <div v-if="tooltipState.visible" class="px-tip-layer" aria-hidden="true">
      <div class="px-tip-arrow"
           :style="{ left: tooltipState.x - 5 + 'px', top: (tooltipState.below ? tooltipState.y - 4 : tooltipState.y - 6) + 'px' }"></div>
      <div ref="box"
           class="px-tip"
           :style="{
             left: left + 'px',
             top: tooltipState.y + 'px',
             transform: tooltipState.below ? 'none' : 'translateY(-100%)',
           }">{{ tooltipState.text }}</div>
    </div>
  </Teleport>
</template>

<style scoped>
.px-tip-layer {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 10000;
}

.px-tip, .px-tip-arrow {
  position: absolute;
  background: var(--tip-bg);
  animation: px-fade-in .12s ease-out;
}

.px-tip {
  max-width: 280px;
  width: max-content;
  padding: 7px 11px;
  border-radius: 8px;
  color: #f2f2f4;
  border: 1px solid rgba(255, 255, 255, .08);
  font-size: 13px;
  font-weight: 500;
  line-height: 1.5;
  white-space: pre-line;
  overflow-wrap: anywhere;
  box-shadow: 0 10px 28px rgba(0, 0, 0, .35);
}

.px-tip-arrow {
  width: 10px;
  height: 10px;
  transform: rotate(45deg);
  border-radius: 2px;
}
</style>
