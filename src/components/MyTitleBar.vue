<template>
  <div class="titlebar no-drag" role="toolbar" :aria-label="$t('ui.window-controls')">
    <button type="button" class="titlebar__btn" :aria-label="$t('minus')" v-tip="$t('minus')" @click="minus2tray">
      <icon-tabler-layout-bottombar-collapse width="18" height="18" stroke-width="1.8"/>
    </button>
    <template v-if="isWindows">
      <span class="titlebar__sep"></span>
      <button type="button" class="titlebar__btn ncr-min" :aria-label="$t('mini')" v-tip="$t('mini')" @click="minus">
        <icon-tabler-minus width="18" height="18" stroke-width="1.8"/>
      </button>
      <span class="titlebar__sep"></span>
      <button type="button"
              class="titlebar__btn ncr-max"
              :aria-label="isMaximized ? $t('restore') : $t('max')"
              v-tip="isMaximized ? $t('restore') : $t('max')"
              @click="max">
        <icon-tabler-copy v-if="isMaximized" width="18" height="18" stroke-width="1.8"/>
        <icon-tabler-square v-else width="18" height="18" stroke-width="1.8"/>
      </button>
      <span class="titlebar__sep"></span>
      <button type="button" class="titlebar__btn titlebar__btn--close" :aria-label="$t('close')" v-tip="$t('close')" @click="close">
        <icon-tabler-x width="18" height="18" stroke-width="1.8"/>
      </button>
    </template>
  </div>
</template>

<script setup lang="ts">
import {Events} from "@/runtime";

const isMaximized = ref(false)
const isWindows = ref(false)

onMounted(() => {
  // @ts-ignore
  if (window["pxShowBar"]) {
    isWindows.value = true;
  }
  // Authoritative maximise state from the Wails shell (Windows). Needed
  // because with native non-client regions (--wails-non-client-region below)
  // Windows handles the maximize button itself and the @click above never
  // fires; also covers Win+Up and other native maximise paths.
  Events.On('maximized', (val: any) => {
    isMaximized.value = !!val;
  });
})

function close() {
  Events.Emit({name: "close", data: true});
}

function minus() {
  Events.Emit({name: "min", data: true});
}

function max() {
  isMaximized.value = !isMaximized.value;
  Events.Emit({name: "max", data: true});
}

// 最小化到托盘
function minus2tray() {
  Events.Emit({name: "hide", data: true});
}
</script>

<style scoped>
.titlebar {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-left: auto;
  flex-shrink: 0;
  height: 38px;
  padding: 3px;
  border-radius: 999px;
  background: var(--side-bg);
  backdrop-filter: var(--side-blur);
  border: 1px solid var(--border);
}

.titlebar__btn {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  border: none;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-2);
  cursor: pointer;
  padding: 0;
}

.titlebar__btn:hover {
  background: var(--hover-bg);
  color: var(--text);
}

.titlebar__btn--close:hover {
  background: var(--error);
  color: #fff;
}

.titlebar__sep {
  width: 1px;
  height: 18px;
  background: var(--border);
  flex-shrink: 0;
}

/* Native non-client regions (Wails v3 + WebView2CompositionHosting on
   Windows): Windows treats these HTML buttons as real caption buttons, which
   enables the Win11 Snap Layouts flyout over the maximize button and native
   press/hover handling. The @click handlers remain as the fallback when the
   feature is unavailable (macOS/Linux/Electron, old WebView2 runtime).
   The close button is deliberately NOT marked: its semantics are custom
   (quit via px:fe:close), while a native close would hide to tray. */
.ncr-min {
  --wails-non-client-region: minimize;
}

.ncr-max {
  --wails-non-client-region: maximize;
}
</style>
