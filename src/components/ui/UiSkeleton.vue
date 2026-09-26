<script setup lang="ts">
// Grey placeholder silhouettes shown on the first load of a list.
withDefaults(defineProps<{
  variant?: 'group' | 'card';
  count?: number;
  minHeight?: number;
}>(), {variant: 'card', count: 3, minHeight: 200});
</script>

<template>
  <template v-if="variant === 'group'">
    <div v-for="i in count" :key="i" class="px-skel-group" aria-hidden="true">
      <div class="px-skel" style="width:30px;height:30px"></div>
      <div class="px-skel-lines">
        <div class="px-skel" style="width:32%;height:12px"></div>
        <div class="px-skel" style="width:20%;height:10px"></div>
      </div>
      <div class="px-skel" style="width:26px;height:26px"></div>
    </div>
  </template>
  <template v-else>
    <div v-for="i in count" :key="i" class="px-skel-card" :style="{ minHeight: minHeight + 'px' }" aria-hidden="true">
      <div class="px-skel" style="width:55%;height:14px"></div>
      <div class="px-skel" style="width:35%;height:10px"></div>
      <div style="flex:1"></div>
      <div class="px-skel" style="width:100%;height:10px"></div>
      <div class="px-skel" style="width:80%;height:10px"></div>
      <div class="px-skel" style="width:60%;height:10px"></div>
    </div>
  </template>
</template>

<style scoped>
.px-skel {
  border-radius: 8px;
  background: var(--panel-soft);
  animation: px-skel 1.2s ease-in-out infinite;
  flex-shrink: 0;
}

.px-skel-group {
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--input-bg);
  padding: 14px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.px-skel-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.px-skel-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--input-bg);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
