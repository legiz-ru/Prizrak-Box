<script setup lang="ts">
// Renders the request made through confirm() (see services.ts).
import IconTrash from '~icons/tabler/trash';
import UiNotice from "./UiNotice.vue";
import {confirmState, settleConfirm} from "./services";

const open = computed({
  get: () => !!confirmState.current,
  set: (value: boolean) => {
    if (!value) settleConfirm(false);
  },
});
const current = computed(() => confirmState.current);
</script>

<template>
  <UiNotice v-model="open"
            role="alertdialog"
            tone="error"
            :z-index="80"
            :title="current?.title ?? ''"
            :icon="current?.icon ?? IconTrash">
    <span v-if="current?.text">{{ current.text }}</span>
    <template #actions>
      <button type="button" class="px-btn" @click="settleConfirm(false)">
        {{ current?.cancelLabel || $t('cancel') }}
      </button>
      <button type="button" class="px-btn px-btn--danger" @click="settleConfirm(true)">
        {{ current?.okLabel || $t('confirm.label') }}
      </button>
    </template>
  </UiNotice>
</template>
