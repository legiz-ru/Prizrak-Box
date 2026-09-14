<script setup lang="ts">
import {useMenuStore} from "@/store/menuStore";
import {changeMenu} from "@/util/menu";
import {useRouter} from "vue-router";

const menuStore = useMenuStore()
const router = useRouter()
</script>

<template>
  <div class="nav">
    <div
        :class="menuStore.menu == 'Home' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Home', router)"
    >
      <n-icon>
        <icon-tabler-home/>
      </n-icon>
      <span class="nav-label">{{ $t('nav.home') }}</span>
    </div>

    <div
        :class="menuStore.menu == 'Setting' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Setting', router)"
    >
      <n-icon>
        <icon-tabler-settings/>
      </n-icon>
      <span class="nav-label">{{ $t('nav.setting') }}</span>
    </div>

    <div
        :class="menuStore.menu == 'Proxies' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Proxies', router)"
    >
      <n-icon>
        <icon-tabler-rocket/>
      </n-icon>
      <span class="nav-label">{{ $t('nav.proxies') }}</span>
    </div>

    <div
        :class="menuStore.menu == 'Profiles' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Profiles', router)"
    >
      <n-icon>
        <icon-tabler-user-cog/>
      </n-icon>
      <span class="nav-label">{{ $t('nav.profiles') }}</span>
    </div>
  </div>
</template>

<style scoped>
.nav {
  margin-top: 20px;
  margin-left: 22px;
  width: 185px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.nav-btn {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  border-radius: 999px;
  cursor: pointer;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
  font-size: 18px;
  color: var(--text-color);
}

.nav-btn:hover {
  background-color: var(--left-nav-btn-hover-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.nav-btn-select,
.nav-btn-select:hover {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.nav-label {
  font-size: 14px;
  margin-left: 12px;
}
</style>
