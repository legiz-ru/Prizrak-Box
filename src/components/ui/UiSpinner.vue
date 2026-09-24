<script setup lang="ts">
// Small ring spinner; sized to sit inside a button in place of its icon.
withDefaults(defineProps<{ size?: number }>(), {size: 14});
</script>

<template>
  <span class="px-spinner"
        aria-hidden="true"
        :style="{ width: size + 'px', height: size + 'px' }"></span>
</template>

<style scoped>
.px-spinner {
  display: inline-block;
  flex-shrink: 0;
  border-radius: 50%;
  border: 2px solid currentColor;
  border-right-color: transparent;
  animation: px-spin .7s linear infinite;
}
</style>
