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
      <el-icon>
        <icon-mdi-home v-if="menuStore.menu == 'Home'"/>
        <icon-mdi-home-outline v-else/>
      </el-icon>
      <span class="nav-label">{{ $t('nav.home') }}</span>
    </div>

    <div
        :class="menuStore.menu == 'Setting' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Setting', router)"
    >
      <el-icon>
        <icon-mdi-cog v-if="menuStore.menu == 'Setting'"/>
        <icon-mdi-cog-outline v-else/>
      </el-icon>
      <span class="nav-label">{{ $t('nav.setting') }}</span>
    </div>

    <div
        :class="menuStore.menu == 'Proxies' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Proxies', router)"
    >
      <el-icon>
        <icon-mdi-rocket-launch v-if="menuStore.menu == 'Proxies'"/>
        <icon-mdi-rocket-launch-outline v-else/>
      </el-icon>
      <span class="nav-label">{{ $t('nav.proxies') }}</span>
    </div>

    <div
        :class="menuStore.menu == 'Profiles' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Profiles', router)"
    >
      <el-icon>
        <icon-mdi-account-cog v-if="menuStore.menu == 'Profiles'"/>
        <icon-mdi-account-cog-outline v-else/>
      </el-icon>
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
