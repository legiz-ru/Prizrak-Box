<template>
  <div v-if="isWindows">
    <el-tooltip
        :content="$t('minus')"
        placement="bottom">
      <span class="bar" @click="minus2tray">
          <el-icon>
              <icon-mdi-card-minus-outline/>
          </el-icon>
      </span>
    </el-tooltip>
    <el-tooltip
        :content="$t('mini')"
        placement="bottom">
      <span class="bar ncr-min" @click="minus">
          <el-icon>
              <icon-mdi-minus/>
          </el-icon>
      </span>
    </el-tooltip>
    <el-tooltip
        v-if="isMaximized"
        :content="$t('restore')"
        placement="bottom">
      <span class="bar ncr-max" @click="max">
          <el-icon>
              <icon-mdi-window-restore/>
          </el-icon>
      </span>
    </el-tooltip>
    <el-tooltip
        v-else
        :content="$t('max')"
        placement="bottom">
      <span class="bar ncr-max" @click="max">
          <el-icon>
              <icon-mdi-window-maximize/>
          </el-icon>
      </span>
    </el-tooltip>
    <el-tooltip
        :content="$t('close')"
        placement="bottom">
      <span class="" @click="close">
          <el-icon>
              <icon-mdi-window-close/>
          </el-icon>
      </span>
    </el-tooltip>
  </div>
  <div v-else>
    <el-tooltip
        :content="$t('minus')"
        placement="left">
      <span class="" @click="minus2tray">
          <el-icon>
              <icon-mdi-card-minus-outline/>
          </el-icon>
      </span>
    </el-tooltip>
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
