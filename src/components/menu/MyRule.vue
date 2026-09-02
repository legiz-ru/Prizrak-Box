<template>
  <div v-if="!hideModeSwitch" class="rule-pill">
    <button
        v-for="opt in options"
        :key="opt.value"
        type="button"
        :class="['rule-pill__btn', { 'is-active': menuStore.rule === opt.value }]"
        @click="menuStore.rule = opt.value"
    >
      {{ opt.label }}
    </button>
  </div>
</template>

<script lang="ts" setup>
import {useMenuStore} from "@/store/menuStore";
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {pSuccess} from "@/util/pLoad";
import {pUpdateMihomo} from "@/util/mihomo";
import {useSettingStore} from "@/store/settingStore";

// 使用store
const menuStore = useMenuStore();
const settingStore = useSettingStore();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 国际化
const {t} = useI18n();

// Активный профиль приходит из App.vue (primary > selected > первый в списке).
const props = defineProps<{ activeProfile?: any }>();

// HTTP-заголовок профиля `global-mode: false` (без учёта регистра) или `0`
// полностью скрывает переключатель режимов в левом боковом меню. Любое другое
// значение или отсутствие заголовка оставляет его видимым. Только для десктопа.
const hideModeSwitch = computed(() => props.activeProfile?.globalModeDisabled === true);

const options = computed((): any[] => {
  const modes = [
    {
      label: t("rules.rule"),
      value: "rule",
    },
    {
      label: t("rules.global"),
      value: "global",
    },
  ];

  if (t("lang") != "ru") {
    modes.push({
      label: t("rules.direct"),
      value: "direct",
    });
  }

  return modes
});

// Когда переключатель скрыт, а активным остаётся Global — возвращаемся в Rule
// (watch ниже сам применит режим в Mihomo), чтобы профиль не залип в Global.
watch(
    hideModeSwitch,
    (hidden) => {
      if (hidden && menuStore.rule === "global") {
        menuStore.rule = "rule";
      }
    },
    {immediate: true}
);

// 监听 store.rule 的变化
watch(
    () => menuStore.rule,
    (newValue) => {
      api.updateConfigs({
        mode: newValue,
      }).then((res: any) => {
        pSuccess(t("rules." + newValue + "-switch"));
        // 同步 mihomo 配置
        pUpdateMihomo(menuStore, settingStore, api)
      });
    }
);
</script>

<style scoped>
.rule-pill {
  margin-left: 22px;
  margin-top: 23px;
  width: 185px;
  box-sizing: border-box;
  display: flex;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  padding: 4px;
  gap: 4px;
}

.rule-pill:hover {
  box-shadow: var(--left-nav-hover-shadow);
}

.rule-pill__btn {
  flex: 1;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--left-nav-text);
  cursor: pointer;
  font-size: 13px;
  padding: 7px 4px;
  white-space: nowrap;
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.rule-pill__btn:hover {
  background-color: var(--left-nav-btn-hover-bg);
}

.rule-pill__btn.is-active {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
  color: var(--text-color);
}
</style>
