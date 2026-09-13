<script setup lang="ts">
// Переключатель настроек.
//
// До этого он был `div` с @click, продублированным одиннадцать раз в MyConfig
// и один раз в Profiles: с клавиатуры недоступен, для скринридера невидим.
// Здесь это настоящая кнопка с role="switch", поэтому Tab/Space/Enter работают
// сами собой, а состояние объявляется вслух.
//
// Цвет включённого состояния — тот же --left-item-selected-bg, которым боковое
// меню красит выбранный пункт, а раздел настроек — активную кнопку Tun Stack.
const model = defineModel<boolean>({default: false});

defineProps<{
  /** Доступное имя. Обязательно, когда рядом нет видимой метки. */
  label?: string;
  disabled?: boolean;
}>();
</script>

<template>
  <button
      type="button"
      role="switch"
      class="px-toggle"
      :class="{ 'is-on': model }"
      :aria-checked="model"
      :aria-label="label"
      :disabled="disabled"
      @click="model = !model"
  >
    <span class="px-toggle__thumb"></span>
  </button>
</template>

<style scoped>
.px-toggle {
  position: relative;
  display: inline-block;
  flex-shrink: 0;
  width: 46px;
  height: 26px;
  padding: 0;
  border: none;
  border-radius: var(--px-r-pill);
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--px-elev-1);
  cursor: pointer;
  transition: background-color var(--px-dur) var(--px-ease),
  box-shadow var(--px-dur) var(--px-ease);
}

.px-toggle:hover:not(:disabled) {
  box-shadow: var(--px-elev-2);
}

.px-toggle.is-on {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--px-elev-2);
}

.px-toggle:disabled {
  opacity: .45;
  cursor: not-allowed;
}

.px-toggle__thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background-color: var(--text-color);
  transition: left var(--px-dur) var(--px-ease),
  background-color var(--px-dur) var(--px-ease);
}

.px-toggle.is-on .px-toggle__thumb {
  left: 23px;
  /* На акценте белый кружок читается в обеих темах, в отличие от --text-color,
     который на светлых обоях становится чёрным и сливается с заливкой. */
  background-color: #fff;
}
</style>
