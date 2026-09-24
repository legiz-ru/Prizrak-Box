<template>
  <div v-if="!hideModeSwitch" class="rule-pill no-drag" role="group" :aria-label="$t('ui.mode-label')">
    <button
        v-for="opt in options"
        :key="opt.value"
        type="button"
        :class="['rule-pill__btn', { 'is-active': menuStore.rule === opt.value }]"
        :aria-pressed="menuStore.rule === opt.value ? 'true' : 'false'"
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
  display: flex;
  background: var(--side-bg);
  backdrop-filter: var(--side-blur);
  border-radius: 999px;
  padding: 3px;
  gap: 2px;
}

.rule-pill__btn {
  flex: 1;
  min-width: 0;
  padding: 7px 4px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-2);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rule-pill__btn:hover {
  color: var(--text);
}

.rule-pill__btn.is-active {
  background: var(--accent);
  color: var(--on-accent);
}
</style>
