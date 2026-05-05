<script setup lang="ts">
import MyHr from "@/components/MyHr.vue";
import {useMenuStore} from "@/store/menuStore";
import {useRouter} from "vue-router";
import {computed} from "vue";

const menuStore = useMenuStore()
const router = useRouter()

const getActive = function (value: string): string {
  return menuStore.ruleMenu === value ? 'proxy-group-title proxy-group-title-select' : 'proxy-group-title';
}

const setActive = function (value: string) {
  router.push("/Rule/" + value);
}

const providersView = computed({
  get: () => menuStore.providersView,
  set: (v: 'cards' | 'table') => menuStore.setProvidersView(v),
})
</script>

<template>
  <MyLayout hr-show>
    <template #top>
      <el-space class="space">
        <div class="title">
          {{ $t('rule.title') }}
        </div>
      </el-space>

      <div class="proxy-group-row">
        <div class="proxy-group">
          <button
              :class="getActive('Now')"
              @click="setActive('Now')"
          >
            <icon-mdi-eye-arrow-right class="pre"/>
            <span class="suf">
              {{ $t('rule.now.title') }}
            </span>
          </button>
          <button
              :class="getActive('Group')"
              @click="setActive('Group')"
          >
            <icon-mdi-view-dashboard class="pre"/>
            <span class="suf">
              {{ $t('rule.group.title') }}
            </span>
          </button>
          <button
              :class="getActive('Providers')"
              @click="setActive('Providers')"
          >
            <icon-mdi-script-text-outline class="pre"/>
            <span class="suf">
              {{ $t('rule.providers.title') }}
            </span>
          </button>
          <button
              :class="getActive('Ignore')"
              @click="setActive('Ignore')"
          >
            <icon-mdi-cancel class="pre"/>
            <span class="suf">
              {{ $t('rule.ignore.title') }}
            </span>
          </button>
        </div>

        <div v-if="menuStore.ruleMenu === 'Providers'" class="view-toggle">
          <el-tooltip :content="$t('rule.providers.viewCards')" placement="bottom" :show-after="300">
            <button
                :class="['toggle-btn', { 'is-active': providersView === 'cards' }]"
                type="button"
                @click="providersView = 'cards'"
            >
              <el-icon size="18"><icon-mdi-view-module/></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip :content="$t('rule.providers.viewTable')" placement="bottom" :show-after="300">
            <button
                :class="['toggle-btn', { 'is-active': providersView === 'table' }]"
                type="button"
                @click="providersView = 'table'"
            >
              <el-icon size="18"><icon-mdi-view-list/></el-icon>
            </button>
          </el-tooltip>
        </div>
      </div>
    </template>
    <template #bottom>
      <router-view/>
    </template>
  </MyLayout>
</template>

<style scoped>
.space {
  margin-top: 20px;
}

.title {
  font-size: 32px;
  font-weight: bold;
  margin-left: 10px;
}

.proxy-group-row {
  display: flex;
  align-items: center;
  margin-top: 20px;
  margin-left: 10px;
  gap: 12px;
  width: 95%;
}

.proxy-group {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1;
}

.view-toggle {
  display: inline-flex;
  border-radius: 999px;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  padding: 4px;
  gap: 2px;
  flex-shrink: 0;
}

.view-toggle:hover {
  box-shadow: var(--left-nav-hover-shadow);
}

.toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  padding: 0;
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
  flex-shrink: 0;
}

.toggle-btn:hover {
  background-color: var(--left-nav-btn-hover-bg);
}

.toggle-btn.is-active {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.pre {
  position: absolute;
}

.suf {
  margin-left: 25px;
}

.proxy-group-title {
  background-color: transparent;
  color: var(--text-color);
  border: 2px solid var(--hr-color);
  border-radius: 20px;
  padding: 6px 8px 6px 8px;
  font-size: 16px;
  text-align: center;
  cursor: pointer;
  box-shadow: var(--left-nav-shadow);
}

.proxy-group-title:hover, .proxy-group-title-select {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
  border-color: var(--text-color);
}

:deep(.bottom) {
  display: flex;
  flex-direction: column;
  padding-bottom: 0;
}
</style>