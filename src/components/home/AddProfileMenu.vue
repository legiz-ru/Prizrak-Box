<script setup lang="ts">
import {h} from 'vue';
import {useI18n} from 'vue-i18n';
import {NIcon} from 'naive-ui';
import IconPencil from '~icons/tabler/pencil';
import IconContentPaste from '~icons/tabler/clipboard';
import IconFolderOpen from '~icons/tabler/folder-open';

const {t} = useI18n();

const emit = defineEmits<{
  (e: 'add'): void;
  (e: 'paste'): void;
  (e: 'file'): void;
}>();

function renderIcon(icon: unknown) {
  return () => h(NIcon, null, {default: () => h(icon as any)});
}

const options = computed(() => [
  {label: t('profiles.add'), key: 'add', icon: renderIcon(IconPencil)},
  {label: t('profiles.paste'), key: 'paste', icon: renderIcon(IconContentPaste)},
  {label: t('profiles.open'), key: 'file', icon: renderIcon(IconFolderOpen)},
]);

function handleSelect(key: string) {
  if (key === 'add') emit('add');
  else if (key === 'paste') emit('paste');
  else if (key === 'file') emit('file');
}
</script>

<template>
  <n-dropdown trigger="click" :options="options" @select="handleSelect">
    <slot/>
  </n-dropdown>
</template>
