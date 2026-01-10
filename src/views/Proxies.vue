<script setup lang="ts">
import createApi from "@/api";
import {useProxiesStore, type ProxyViewMode} from "@/store/proxiesStore";
import {useMenuStore} from "@/store/menuStore";
import {useSettingStore} from "@/store/settingStore";
import {useI18n} from "vue-i18n";
import {pError, pLoad} from "@/util/pLoad";
import {useWebStore} from "@/store/webStore";
import {changeProxyAndCloseConnections} from "@/util/proxy";

const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 当前页面双向绑定对象
const groupList = ref<string[]>([]);
const nodeList = ref<any[]>([]);
const fullViewNodes = ref<Record<string, any[]>>({});
const groupIcons = ref<Record<string, string>>({});
const nestedGroupSelections = ref<Record<string, string>>({});
const handleIconError = (event: Event) => {
  const target = event.target as HTMLImageElement | null;
  if (target) {
    target.style.display = 'none';
  }
};

// 当前页面使用store
const proxiesStore = useProxiesStore();
const menuStore = useMenuStore();
const settingStore = useSettingStore();
const webStore = useWebStore();

const expandedGroups = computed(() => proxiesStore.groupExpansion ?? {});
const selectedProxies = computed<Record<string, string>>(() => {
  const selections: Record<string, string> = {};
  const groups = fullViewNodes.value;
  Object.keys(groups).forEach((group) => {
    const nodes = groups[group];
    if (!Array.isArray(nodes)) {
      return;
    }
    const current = nodes.find((node) => node?.now);
    if (current?.name) {
      selections[group] = current.name;
    }
  });
  return selections;
});

// 获取分组
async function groups() {
  // 活跃分组
  const active = proxiesStore.active;

  const rawGroups = await api.getGroups();
  const normalizedInput = Array.isArray(rawGroups) ? rawGroups : [];
  const normalized = normalizedInput
      .map((item: any) => {
        if (typeof item === 'string') {
          return {name: item};
        }
        if (item && typeof item.name === 'string') {
          return {name: item.name, icon: item.icon};
        }
        return null;
      })
      .filter((item) => item && item.name) as {name: string; icon?: string}[];

  const icons: Record<string, string> = {};
  normalized.forEach(({name, icon}) => {
    if (icon) {
      icons[name] = icon;
    }
  });
  groupIcons.value = icons;
  const temp = normalized.map((item) => item.name);
  switch (menuStore.rule) {
    case "rule":
      groupList.value = temp;
      const hasActive = temp.includes(active);
      if (!hasActive || active === "GLOBAL") {
        if (temp[0]) {
          proxiesStore.setActive(temp[0]);
        }
      }
      break;
    case "global":
      groupList.value = temp.concat(["GLOBAL"]);
      if (!active && temp[0]) {
        proxiesStore.setActive(temp[0]);
      }
      break;
    case "direct":
      groupList.value = [];
      break;
  }
}

// Update active connections for nested groups (URLTest, Selector, etc.)
async function updateNestedGroupSelections() {
  const groupTypes = ['urltest', 'selector', 'fallback', 'loadbalance', 'relay'];
  const nestedGroups: string[] = [];

  // Collect all nodes that are groups themselves
  Object.values(fullViewNodes.value).forEach((nodes) => {
    if (Array.isArray(nodes)) {
      nodes.forEach((node) => {
        if (node.type && groupTypes.includes(node.type.toLowerCase())) {
          nestedGroups.push(node.name);
        }
      });
    }
  });

  // Also check current nodeList for non-full view modes
  if (Array.isArray(nodeList.value)) {
    nodeList.value.forEach((node) => {
      if (node.type && groupTypes.includes(node.type.toLowerCase()) && !nestedGroups.includes(node.name)) {
        nestedGroups.push(node.name);
      }
    });
  }

  // Request active connection for each nested group (without hidden filter)
  const selections: Record<string, string> = {};
  await Promise.all(
      nestedGroups.map(async (groupName) => {
        try {
          const proxies = await api.getProxies(groupName, false, false);
          const current = proxies.find((node) => node?.now);
          if (current?.name) {
            selections[groupName] = current.name;
          }
        } catch (e) {
          // Ignore errors for groups that don't exist
        }
      })
  );

  nestedGroupSelections.value = selections;
}

// 获取节点列表
async function nodes() {
  if (menuStore.rule == "direct") {
    nodeList.value = [];
    fullViewNodes.value = {};
    nestedGroupSelections.value = {};
    return;
  }

  if (proxiesStore.viewMode === 'full') {
    const groups = [...groupList.value];
    const pairs = await Promise.all(
        groups.map(async (group) => {
          const proxies = await api.getProxies(
              group,
              proxiesStore.isHide,
              proxiesStore.isSort
          );
          return [group, proxies] as const;
        })
    );
    const mapped: Record<string, any[]> = {};
    pairs.forEach(([group, proxies]) => {
      mapped[group] = proxies;
    });
    fullViewNodes.value = mapped;
    nodeList.value = mapped[proxiesStore.active] ?? [];

    // Update nested group selections
    await updateNestedGroupSelections();
    return;
  }

  nodeList.value = await api.getProxies(
      proxiesStore.active,
      proxiesStore.isHide,
      proxiesStore.isSort
  ); // 更新响应式数据
  fullViewNodes.value = {};

  // Update nested group selections for non-full view
  await updateNestedGroupSelections();
}

// 设置活跃分组
async function setActive(value: any) {
  if (proxiesStore.active == value) {
    return;
  }
  proxiesStore.setActive(value);
  await nodes();
}

// 设置隐藏
async function setHide() {
  proxiesStore.setHide(!proxiesStore.isHide);
  await nodes();
}

// 设置排序
async function setSort() {
  proxiesStore.setSort(!proxiesStore.isSort);
  await nodes();
}

// 设置分组
const viewModeOrder: Record<ProxyViewMode, ProxyViewMode> = {
  horizontal: 'dropdown',
  dropdown: 'full',
  full: 'horizontal',
};

const viewModeTooltip = computed(() => {
  switch (proxiesStore.viewMode) {
    case 'horizontal':
      return t('proxies.vertical-off');
    case 'dropdown':
      return t('proxies.full-view');
    case 'full':
      return t('proxies.vertical-on');
  }
  return t('proxies.vertical-off');
});

async function cycleViewMode() {
  const nextMode = viewModeOrder[proxiesStore.viewMode];
  proxiesStore.setViewMode(nextMode);
  if (nextMode !== 'horizontal') {
    atStart.value = true;
    atEnd.value = true;
  }
  if (nextMode !== 'dropdown') {
    isDropdownOpen.value = false;
  }
  setTimeout(() => {
    updateButtonVisibility();
  }, 200);
  await nodes();
}

// 设置代理
async function setProxy(now: any, name: string, groupName?: string) {
  if (now) {
    return;
  }
  const targetGroup = groupName ?? proxiesStore.active;
  if (!targetGroup) {
    return;
  }
  try {
    await changeProxyAndCloseConnections(
        api,
        targetGroup,
        name,
    );
    proxiesStore.setActive(targetGroup);
    proxiesStore.setNow(name);
  } catch (error) {
    if (error && typeof error === 'object' && 'message' in error) {
      const message = (error as {message?: unknown}).message;
      if (typeof message === 'string') {
        pError(message);
      } else {
        console.error(error);
      }
    } else {
      console.error(error);
    }
  }
}

// 测试延迟
function testDelay() {
  pLoad(t("proxies.loading"), async () => {
    try {
      await api.getDelay(proxiesStore.active, settingStore.testUrl, 3000);
      await nodes();
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    }
  });
}

const proxyGroup = ref(null);
const atStart = ref(true); // 标记是否在最左边
const atEnd = ref(true); // 标记是否在最右边

const updateButtonVisibility = () => {
  if (proxiesStore.viewMode !== 'horizontal' || !proxyGroup.value) {
    atStart.value = true;
    atEnd.value = true;
    return;
  }

  const scrollLeft = proxyGroup.value.scrollLeft;
  const scrollWidth = proxyGroup.value.scrollWidth;
  const clientWidth = proxyGroup.value.clientWidth;

  atStart.value = scrollLeft === 0;
  atEnd.value = scrollLeft + clientWidth >= scrollWidth;
};

const scrollLeft = () => {
  if (proxyGroup.value) {
    proxyGroup.value.scrollLeft -= proxyGroup.value.clientWidth + 15;
  }
};

const scrollRight = () => {
  if (proxyGroup.value) {
    proxyGroup.value.scrollLeft += proxyGroup.value.clientWidth - 15;
  }
};

let isScrolling: any;
const handleScroll = () => {
  clearTimeout(isScrolling);
  isScrolling = setTimeout(() => {
    updateButtonVisibility();
  }, 200); // 200ms 延迟
};

const isDropdownOpen = ref(false);
const toggleGroup = (group: string) => {
  const next = !expandedGroups.value[group];
  proxiesStore.setGroupExpansionState(group, next);
};

// 添加延时隐藏下拉菜单
let isOvering: any;
const hideDropdown = () => {
  isOvering = setTimeout(() => {
    isDropdownOpen.value = false;
  }, 200); // 延迟 200 毫秒
};

// 鼠标进入下拉菜单时，清除延时隐藏
const enterDropDown = () => {
  clearTimeout(isOvering);
  isDropdownOpen.value = true;
};

let fresh: any = null;
onMounted(async () => {
  await groups();
  await nodes();
  updateButtonVisibility();
  // 监听 resize 事件
  window.addEventListener("resize", updateButtonVisibility);
  // 创建刷新定时器
  fresh = setInterval(async () => {
    await nodes();
  }, 10000);
});

onBeforeUnmount(() => {
  // 清除定时器
  clearInterval(fresh);
  // 移除 resize 事件监听
  window.removeEventListener("resize", updateButtonVisibility);
});

// 监听具体状态
watch(() => menuStore.rule, // 监听 store 中的某个状态
    async () => {
      await groups();
      await nodes();
      updateButtonVisibility();
    }
);

watch(() => webStore.fProfile, async () => {
  await groups();
  await nodes();
  updateButtonVisibility();
})

watch(() => proxiesStore.now, async () => {
  await nodes();
})

watch(groupList, (list) => {
  if (!list.length) {
    proxiesStore.replaceGroupExpansions({});
    return;
  }
  const result: Record<string, boolean> = {};
  list.forEach((group, index) => {
    const previous = expandedGroups.value[group];
    result[group] = typeof previous === 'boolean' ? previous : index === 0;
  });
  proxiesStore.replaceGroupExpansions(result);
}, {immediate: true});

</script>

<template>
  <MyLayout hr-show>
    <template #top>
      <el-space class="space">
        <div class="title">
          {{ $t("proxies.title") }}
        </div>
        <div class="proxy-option">
          <el-tooltip :content="$t('proxies.test')" placement="top">
            <el-icon @click="testDelay" class="proxy-option-btn">
              <icon-mdi-speedometer/>
            </el-icon>
          </el-tooltip>

          <el-tooltip
              :content="
              proxiesStore.isHide
                ? $t('proxies.hide-on')
                : $t('proxies.hide-off')
            "
              placement="top"
          >
            <el-icon @click="setHide" class="proxy-option-btn">
              <icon-mdi-eye-off v-if="proxiesStore.isHide"/>
              <icon-mdi-eye v-else/>
            </el-icon>
          </el-tooltip>

          <el-tooltip
              :content="
              proxiesStore.isSort
                ? $t('proxies.sort-on')
                : $t('proxies.sort-off')
            "
              placement="top"
          >
            <el-icon @click="setSort" class="proxy-option-btn">
              <icon-mdi-sort-ascending v-if="proxiesStore.isSort"/>
              <icon-mdi-sort v-else/>
            </el-icon>
          </el-tooltip>

          <el-tooltip
              :content="viewModeTooltip"
              placement="top"
          >
            <el-icon @click="cycleViewMode" class="proxy-option-btn">
              <icon-mdi-arrow-expand-horizontal v-if="proxiesStore.viewMode === 'horizontal'"/>
              <icon-mdi-arrow-expand-vertical v-else-if="proxiesStore.viewMode === 'dropdown'"/>
              <icon-mdi-format-list-bulleted v-else/>
            </el-icon>
          </el-tooltip>
        </div>
      </el-space>

      <div
          class="dropdown"
          v-if="proxiesStore.viewMode === 'dropdown' && menuStore.rule != 'direct' && groupList.length > 0"
      >
        <button
            class="dropdown-btn"
            @mouseenter="enterDropDown"
            @mouseleave="hideDropdown"
        >
            <span class="dropdown-btn-content">
            <span
                v-if="groupIcons[proxiesStore.active]"
                class="proxy-icon-wrapper proxy-icon-wrapper--dropdown"
            >
              <img
                  :src="groupIcons[proxiesStore.active]"
                  alt=""
                  class="dropdown-item-icon"
                  @error="handleIconError"
              />
            </span>
            <span class="dropdown-item-label">{{ proxiesStore.active }}</span>
          </span>
        </button>
        <ul
            v-if="isDropdownOpen"
            @mouseenter="enterDropDown"
            @mouseleave="hideDropdown"
            class="dropdown-list"
        >
          <li
              v-for="item in groupList"
              :key="item + '-gv'"
              @click="setActive(item)"
              class="dropdown-item"
          >
            <span class="dropdown-btn-content">
              <span
                  v-if="groupIcons[item]"
                  class="proxy-icon-wrapper proxy-icon-wrapper--dropdown"
              >
                <img
                    :src="groupIcons[item]"
                    alt=""
                    class="dropdown-item-icon"
                    @error="handleIconError"
                />
              </span>
              <span class="dropdown-item-label">{{ item }}</span>
            </span>
          </li>
        </ul>
      </div>

      <div
          class="button-container"
          v-if="proxiesStore.viewMode === 'horizontal' && menuStore.rule != 'direct' && groupList.length > 0"
      >
        <el-icon v-if="!atStart" @click="scrollLeft" class="scroll-left">
          <icon-mdi-arrow-expand-left/>
        </el-icon>
        <div @scroll="handleScroll" ref="proxyGroup" class="proxy-group">
          <button
              :class="
              proxiesStore.active == item
                ? 'proxy-group-title proxy-group-title-select'
                : 'proxy-group-title'
            "
              @click="setActive(item)"
              v-for="item in groupList"
              :key="item + '-g'"
          >
            <span class="proxy-group-content">
              <span
                  v-if="groupIcons[item]"
                  class="proxy-icon-wrapper proxy-icon-wrapper--button"
              >
                <img
                    :src="groupIcons[item]"
                    alt=""
                    class="proxy-group-icon"
                    @error="handleIconError"
                />
              </span>
              <span class="proxy-group-label">{{ item }}</span>
            </span>
          </button>
        </div>
        <el-icon v-if="!atEnd" class="scroll-right" @click="scrollRight">
          <icon-mdi-arrow-expand-right/>
        </el-icon>
      </div>
    </template>


    <template #bottom>
      <div class="proxy-nodes" v-if="proxiesStore.viewMode !== 'full'">
        <div
            :class="
            node['now']
              ? 'proxy-nodes-card proxy-node-select'
              : 'proxy-nodes-card'
          "
            v-for="node in nodeList"
            @click="setProxy(node['now'], node['name'])"
            :key="node['name']"
        >
          <div class="proxy-nodes-title">
            <span :title="node['name']">
              {{ node["name"] }}
            </span>

          </div>
          <div class="proxy-nodes-tags">
            <span class="proxy-nodes-tags-left">
              <span>{{ node["type"] }}</span>
              <template v-if="nestedGroupSelections[node['name']]">
                <span class="proxy-selected-separator">•</span>
                <span class="proxy-selected-name" :title="nestedGroupSelections[node['name']]">
                  {{ nestedGroupSelections[node['name']] }}
                </span>
              </template>
            </span>
            <span :class="'proxy-nodes-tags-right ' + node['toClass']">
              {{ node["delay"] }} ms
            </span>
          </div>
        </div>
      </div>

      <div
          class="full-view-groups"
          v-else-if="menuStore.rule != 'direct' && groupList.length > 0"
      >
        <div
            class="full-view-group"
            v-for="group in groupList"
            :key="group + '-full'"
        >
          <div class="full-view-header" @click="toggleGroup(group)">
            <div class="full-view-info">
              <span
                  v-if="groupIcons[group]"
                  class="proxy-icon-wrapper proxy-icon-wrapper--full"
              >
                <img
                    :src="groupIcons[group]"
                    alt=""
                    class="full-view-icon"
                    @error="handleIconError"
                />
              </span>
              <div class="full-view-text">
                <span class="full-view-title">
                  {{ group }}
                </span>
                <span v-if="selectedProxies[group]" class="full-view-selected">
                  {{ $t('proxies.selected-label') }}: {{ selectedProxies[group] }}
                </span>
              </div>
            </div>
            <el-icon class="full-view-toggle">
              <icon-ep-arrow-up v-if="expandedGroups[group]"/>
              <icon-ep-arrow-down v-else/>
            </el-icon>
          </div>
          <div class="full-view-content" v-show="expandedGroups[group]">
            <div v-if="!fullViewNodes[group]" class="full-view-loading">
              {{ $t('proxies.loading') }}
            </div>
            <div v-else class="proxy-nodes full-view-nodes">
              <div
                  :class="
                  node['now']
                    ? 'proxy-nodes-card proxy-node-select'
                    : 'proxy-nodes-card'
                "
                  v-for="node in fullViewNodes[group]"
                  @click="setProxy(node['now'], node['name'], group)"
                  :key="group + '-' + node['name']"
              >
                <div class="proxy-nodes-title">
                  <span :title="node['name']">
                    {{ node["name"] }}
                  </span>

                </div>
                <div class="proxy-nodes-tags">
                  <span class="proxy-nodes-tags-left">
                    <span>{{ node["type"] }}</span>
                    <template v-if="nestedGroupSelections[node['name']]">
                      <span class="proxy-selected-separator">•</span>
                      <span class="proxy-selected-name" :title="nestedGroupSelections[node['name']]">
                        {{ nestedGroupSelections[node['name']] }}
                      </span>
                    </template>
                  </span>
                  <span :class="'proxy-nodes-tags-right ' + node['toClass']">
                    {{ node["delay"] }} ms
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="proxy-nodes" v-if="menuStore.rule == 'direct'">
        {{ $t("proxies.direct") }}
      </div>
    </template>
  </MyLayout>
</template>

<style scoped>
.space {
  margin-top: 15px;
}

.title {
  font-size: 32px;
  font-weight: bold;
  margin-left: 10px;
}

.proxy-option {
  margin-left: 10px;
  font-size: 30px;
  padding-top: 10px;
}

.proxy-option-btn {
  margin-right: 15px;
}

.proxy-option-btn:hover {
  cursor: pointer;
  color: var(--hr-color);
}

.button-container {
  display: flex;
  align-items: center;
  width: 95%;
  margin-left: 10px;
  min-height: 50px;
}

.proxy-group {
  display: flex;
  gap: 10px;
  margin: 12px 0 3px 0;
  overflow-x: hidden;
  scroll-behavior: smooth;
}

.scroll-left {
  cursor: pointer;
  border: none;
  margin-right: 10px;
}

.scroll-right {
  cursor: pointer;
  border: none;
  margin-left: 10px;
}

.scroll-left[hidden],
.scroll-right[hidden] {
  display: none;
}

.proxy-group-title {
  background-color: transparent;
  color: var(--text-color);
  border: 2px solid var(--hr-color);
  border-radius: 8px;
  padding: 6px 10px;
  font-size: 15px;
  font-family: inherit;
  text-align: center;
  cursor: pointer;
  box-shadow: var(--left-nav-shadow);
  white-space: nowrap;
}

.proxy-group-title:hover,
.proxy-group-title-select {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
  border-color: var(--text-color);
}

.proxy-group-title-select:hover {
  cursor: default;
}

.proxy-group-content {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.proxy-group-icon {
  width: 18px;
  height: 18px;
  object-fit: contain;
}

.proxy-icon-wrapper--button {
  padding: 2px;
}

.proxy-nodes {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 0;
  color: var(--text-color);
  margin-left: 12px;
  width: 95%;
}

.proxy-nodes-card {
  width: calc(33% - 41px);
  max-width: 210px;
  border: 2px solid var(--sub-card-border);
  border-radius: 8px;
  padding: 8px 12px;
  background: var(--sub-card-bg);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  line-height: 1.3;
  box-shadow: var(--left-nav-shadow);
  margin-top: 3px;
}

.proxy-nodes-card:hover,
.proxy-node-select {
  background-color: var(--left-item-selected-bg);
  border: 2px solid var(--text-color);
  cursor: pointer;
}

.proxy-node-select:hover {
  cursor: default;
}

.proxy-nodes-title {
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.proxy-nodes-tags {
  font-size: 14px;
  display: flex;
  margin-top: 10px;
  justify-content: space-between;
}

.proxy-nodes-tags-left {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
}

.proxy-nodes-tags-right {
  text-align: right;
  flex-shrink: 0;
}

.proxy-selected-separator {
  color: var(--text-color);
  opacity: 0.4;
  margin: 0 2px;
  flex-shrink: 0;
  font-size: 12px;
}

.proxy-selected-name {
  font-size: 13px;
  color: var(--text-color);
  opacity: 0.75;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.toHidden {
  display: none;
}

.dropdown {
  position: relative;
  display: inline-block;
  width: 95%;
  margin: 12px 10px 5px 10px;
}

.dropdown-btn {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
  border: 2px solid var(--text-color);
  color: var(--text-color);
  font-family: inherit;
  padding: 5px 10px;
  cursor: pointer;
  font-size: 15px;
  outline: none;
  border-radius: 8px;
  min-width: 204px;
  text-align: left;
}

.dropdown-btn:hover {
  opacity: 0.8;
}

.dropdown-btn-content {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  justify-content: flex-start;
  width: 100%;
}

.dropdown-list {
  position: absolute;
  background: var(--skin-bg-color);
  border: 2px solid var(--text-color);
  margin-top: 4px;
  padding: 0;
  list-style: none;
  min-width: 200px;
  z-index: 20;
  border-radius: 8px;
  font-size: 15px;
  text-align: left;
  max-height: calc(100vh - 230px);
  overflow-y: auto;
}

.dropdown-item {
  color: var(--text-color);
  padding: 8px;
  cursor: pointer;
}

.dropdown-item-icon {
  width: 20px;
  height: 20px;
  object-fit: contain;
}

.proxy-icon-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background-color: var(--left-nav-btn-active-bg);
  box-shadow: var(--left-nav-shadow);
  padding: 3px;
  line-height: 0;
}

.proxy-icon-wrapper--dropdown {
  padding: 4px;
}

.dropdown-item-label {
  display: inline-flex;
  align-items: center;
}

.dropdown-item:hover {
  background: var(--skin-hover-color);
}

.full-view-groups {
  width: 95%;
  margin-left: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.full-view-group {
  border: 2px solid var(--sub-card-border);
  border-radius: 10px;
  background: var(--sub-card-bg);
  box-shadow: var(--left-nav-shadow);
}

.full-view-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  color: var(--text-color);
  cursor: pointer;
}

.full-view-header:hover {
  background: var(--skin-hover-color);
}

.full-view-title {
  font-size: 16px;
  font-weight: 600;
}

.full-view-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.full-view-icon {
  width: 28px;
  height: 28px;
  object-fit: contain;
}

.proxy-icon-wrapper--full {
  padding: 5px;
  border-radius: 10px;
}

.full-view-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.full-view-selected {
  font-size: 13px;
  color: var(--text-color);
  opacity: 0.75;
}

.full-view-toggle {
  font-size: 16px;
}

.full-view-content {
  padding: 0 12px 12px 12px;
}

.full-view-loading {
  padding: 12px 0;
  color: var(--text-color);
  opacity: 0.8;
}

.full-view-nodes {
  width: 100%;
  margin-left: 0;
}
</style>
