<template>
  <div v-if="isWindows">
    <n-tooltip trigger="hover" placement="bottom">
  <template #trigger>
      <span class="bar" @click="minus2tray">
          <n-icon>
              <icon-tabler-square-rounded-minus-2/>
          </n-icon>
      </span>
    </template>
  {{ $t('minus') }}
</n-tooltip>
    <n-tooltip trigger="hover" placement="bottom">
  <template #trigger>
      <span class="bar ncr-min" @click="minus">
          <n-icon>
              <icon-tabler-minus/>
          </n-icon>
      </span>
    </template>
  {{ $t('mini') }}
</n-tooltip>
    <n-tooltip trigger="hover" placement="bottom" v-if="isMaximized">
  <template #trigger>
      <span class="bar ncr-max" @click="max">
          <n-icon>
              <icon-tabler-window-minimize/>
          </n-icon>
      </span>
    </template>
  {{ $t('restore') }}
</n-tooltip>
    <n-tooltip trigger="hover" placement="bottom" v-else>
  <template #trigger>
      <span class="bar ncr-max" @click="max">
          <n-icon>
              <icon-tabler-window-maximize/>
          </n-icon>
      </span>
    </template>
  {{ $t('max') }}
</n-tooltip>
    <n-tooltip trigger="hover" placement="bottom">
  <template #trigger>
      <span class="" @click="close">
          <n-icon>
              <icon-tabler-x/>
          </n-icon>
      </span>
    </template>
  {{ $t('close') }}
</n-tooltip>
  </div>
  <div v-else>
    <n-tooltip trigger="hover" placement="left">
  <template #trigger>
      <span class="" @click="minus2tray">
          <n-icon>
              <icon-tabler-square-rounded-minus-2/>
          </n-icon>
      </span>
    </template>
  {{ $t('minus') }}
</n-tooltip>
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
.bar {
  margin-right: 15px;
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
