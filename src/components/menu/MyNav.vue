<script setup lang="ts">
import {useMenuStore} from "@/store/menuStore";
import {changeMenu} from "@/util/menu";
import {useRouter} from "vue-router";

const menuStore = useMenuStore()
const router = useRouter()

const items = [
  {key: 'Home', label: 'nav.home'},
  {key: 'Setting', label: 'nav.setting'},
  {key: 'Proxies', label: 'nav.proxies'},
  {key: 'Profiles', label: 'nav.profiles'},
]
</script>

<template>
  <nav class="nav no-drag" :aria-label="$t('ui.nav-label')">
    <button v-for="item in items"
            :key="item.key"
            type="button"
            class="nav-btn"
            :class="{ 'is-active': menuStore.menu == item.key }"
            :aria-current="menuStore.menu == item.key ? 'page' : undefined"
            @click="changeMenu(item.key, router)">
      <icon-tabler-home v-if="item.key === 'Home'" width="18" height="18"/>
      <icon-tabler-settings v-else-if="item.key === 'Setting'" width="18" height="18"/>
      <icon-tabler-rocket v-else-if="item.key === 'Proxies'" width="18" height="18"/>
      <icon-tabler-user-cog v-else width="18" height="18"/>
      <span class="ellipsis">{{ $t(item.label) }}</span>
    </button>
  </nav>
</template>

<style scoped>
.nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.nav-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 14px;
  border: none;
  border-radius: 999px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  text-align: left;
  color: var(--text);
  background: var(--side-bg);
  backdrop-filter: var(--side-blur);
  transition: background .15s;
}

.nav-btn:hover {
  background: var(--hover-bg);
}

.nav-btn svg {
  flex-shrink: 0;
}

.nav-btn.is-active,
.nav-btn.is-active:hover {
  background: var(--accent);
  color: var(--on-accent);
}
</style>
