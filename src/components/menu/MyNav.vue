<script setup lang="ts">
import {h} from "vue";
import type {Component} from "vue";
import {useMenuStore} from "@/store/menuStore";
import {changeMenu} from "@/util/menu";
import {useRouter} from "vue-router";
import {useI18n} from "vue-i18n";
import type {MenuOption} from "naive-ui";
import IconHome from "~icons/tabler/home";
import IconSettings from "~icons/tabler/settings";
import IconRocket from "~icons/tabler/rocket";
import IconUserCog from "~icons/tabler/user-cog";
import IconChevronsLeft from "~icons/tabler/chevrons-left";
import IconChevronsRight from "~icons/tabler/chevrons-right";

const menuStore = useMenuStore();
const router = useRouter();
const {t} = useI18n();

function renderIcon(icon: Component) {
  return () => h(icon);
}

const options = computed<MenuOption[]>(() => [
  {key: "Home", label: t("nav.home"), icon: renderIcon(IconHome)},
  {key: "Setting", label: t("nav.setting"), icon: renderIcon(IconSettings)},
  {key: "Proxies", label: t("nav.proxies"), icon: renderIcon(IconRocket)},
  {key: "Profiles", label: t("nav.profiles"), icon: renderIcon(IconUserCog)},
]);

function handleUpdateValue(key: string) {
  changeMenu(key, router);
}

function toggleCollapsed() {
  menuStore.setNavCollapsed(!menuStore.navCollapsed);
}
</script>

<template>
  <div class="nav" :class="{ 'nav--collapsed': menuStore.navCollapsed }">
    <n-menu
        :value="menuStore.menu"
        :options="options"
        :collapsed="menuStore.navCollapsed"
        :collapsed-width="56"
        :collapsed-icon-size="20"
        :indent="18"
        :root-indent="18"
        @update:value="handleUpdateValue"
    />
    <n-tooltip trigger="hover" placement="right">
      <template #trigger>
        <div class="nav-collapse-toggle" @click="toggleCollapsed">
          <n-icon>
            <IconChevronsRight v-if="menuStore.navCollapsed"/>
            <IconChevronsLeft v-else/>
          </n-icon>
        </div>
      </template>
      {{ menuStore.navCollapsed ? $t('nav.expand') : $t('nav.collapse') }}
    </n-tooltip>
  </div>
</template>

<style scoped>
.nav {
  margin-top: 20px;
  margin-left: 22px;
  margin-right: 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* :collapsed-width only matters to n-layout-sider, which resizes its own box
   to match — a bare n-menu never shrinks itself, so without this the "icon
   only" state still rendered at the full 172px item width with the icon
   pinned to the left and the label's now-empty space just sitting there. */
.nav--collapsed {
  align-items: center;
}

.nav--collapsed :deep(.n-menu) {
  width: 56px;
}

.nav :deep(.n-menu-item) {
  margin-bottom: 8px;
}

/* Naive paints hover/selected as a separately-inset, actually-rounded
   ::before layer (inset: 0 8px) — mismatched against a plain background
   painted on this element (radius 0, edge-to-edge), which left sharp
   corners and their box-shadow peeking out from behind the rounded
   highlight. Rounding this element itself and pulling the ::before out to
   the same edges (inset: 0 below) makes both layers occupy the exact same
   box, so there is only ever one pill shape on screen. */
.nav :deep(.n-menu-item-content) {
  border-radius: 999px;
  box-shadow: var(--left-nav-shadow);
  background-color: var(--left-nav-btn-bg);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.nav :deep(.n-menu-item-content::before) {
  inset: 0 !important;
}

.nav :deep(.n-menu-item-content:hover),
.nav :deep(.n-menu-item-content--selected) {
  box-shadow: var(--left-nav-hover-shadow);
}

.nav :deep(.n-menu-item-content .n-menu-item-content__icon) {
  font-size: 18px;
}

.nav :deep(.n-menu-item-content .n-menu-item-content-header) {
  font-size: 14px;
}

.nav-collapse-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  align-self: flex-start;
  width: 32px;
  height: 32px;
  margin-left: 4px;
  border-radius: 999px;
  cursor: pointer;
  color: var(--text-color);
  opacity: 0.6;
  transition: opacity 0.2s ease, background-color 0.2s ease;
}

.nav-collapse-toggle:hover {
  opacity: 1;
  background-color: var(--left-nav-btn-hover-bg);
}

.nav--collapsed .nav-collapse-toggle {
  align-self: center;
}
</style>
