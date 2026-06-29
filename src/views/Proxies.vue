<script setup lang="ts">
import createApi from "@/api";
import {useProxiesStore, type ProxyViewMode} from "@/store/proxiesStore";
import {useMenuStore} from "@/store/menuStore";
import {useSettingStore} from "@/store/settingStore";
import {useI18n} from "vue-i18n";
import {pError, pLoad, pWarning} from "@/util/pLoad";
import {useWebStore} from "@/store/webStore";
import {changeProxyAndCloseConnections} from "@/util/proxy";

const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 当前页面双向绑定对象
const groupList = ref<string[]>([]);
const groupTypeMap = ref<Record<string, string>>({});
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

// Smart-group weights
interface WeightEntry { Name: string; Rank: string; Weight: number }
interface GroupWeightData { weights: WeightEntry[]; hasData: boolean }
const smartGroupWeights = ref<Record<string, GroupWeightData>>({});

function getNodeWeightInfo(groupName: string, nodeName: string): { rank: string; weight: number } | null {
  const data = smartGroupWeights.value[groupName];
  if (!data?.hasData) return null;
  const entry = data.weights.find(w => w.Name === nodeName);
  return entry ? { rank: entry.Rank, weight: entry.Weight } : null;
}

function rankLabel(rank: string): string {
  switch (rank) {
    case 'MostUsed': return t('proxies.smart.most-used');
    case 'OccasionalUsed': return t('proxies.smart.occasional-used');
    case 'RarelyUsed': return t('proxies.smart.rarely-used');
    default: return rank;
  }
}

async function fetchSmartWeights() {
  const toFetch = new Set<string>();
  for (const [name, type] of Object.entries(groupTypeMap.value)) {
    if (type === 'Smart') toFetch.add(name);
  }
  const allNodes = [
    ...nodeList.value,
    ...Object.values(fullViewNodes.value).flat(),
  ];
  for (const node of allNodes) {
    if (node?.type === 'Smart') toFetch.add(node.name);
  }
  if (toFetch.size === 0) return;
  const results = await Promise.allSettled(
    Array.from(toFetch).map(async (name) => {
      const data = await api.getGroupWeights(name);
      return { name, data };
    })
  );
  const newMap = { ...smartGroupWeights.value };
  for (const result of results) {
    if (result.status === 'fulfilled') {
      newMap[result.value.name] = result.value.data;
    }
  }
  smartGroupWeights.value = newMap;
}

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
  if (!webStore.fProfile || !webStore.fProfile['id']) {
    groupList.value = [];
    groupIcons.value = {};
    groupTypeMap.value = {};
    proxiesStore.setActive('');
    return;
  }
  // 活跃分组
  const active = proxiesStore.active;

  let rawGroups: any[];
  try {
    rawGroups = await api.getGroups();
  } catch (e) {
    // Mihomo ещё не готов (pxd-template загружается на старте) — молча выходим,
    // earlyRetry повторит попытку через 3 секунды
    return;
  }
  const normalizedInput = Array.isArray(rawGroups) ? rawGroups : [];
  const normalized = normalizedInput
      .map((item: any) => {
        if (typeof item === 'string') {
          return {name: item};
        }
        if (item && typeof item.name === 'string') {
          return {name: item.name, icon: item.icon, type: item.type};
        }
        return null;
      })
      .filter((item) => item && item.name) as {name: string; icon?: string; type?: string}[];

  const icons: Record<string, string> = {};
  const typeMap: Record<string, string> = {};
  normalized.forEach(({name, icon, type}) => {
    if (icon) icons[name] = icon;
    if (type) typeMap[name] = type;
  });
  groupIcons.value = icons;
  typeMap['GLOBAL'] = 'Selector'; // GLOBAL is always a Selector in Mihomo
  groupTypeMap.value = typeMap;
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
async function nodes(onlyGroup?: string) {
  if (!webStore.fProfile || !webStore.fProfile['id']) {
    nodeList.value = [];
    fullViewNodes.value = {};
    nestedGroupSelections.value = {};
    return;
  }
  if (menuStore.rule == "direct") {
    nodeList.value = [];
    fullViewNodes.value = {};
    nestedGroupSelections.value = {};
    return;
  }

  try {
  if (proxiesStore.viewMode === 'full') {
    if (onlyGroup) {
      // Only refresh the specific group that was tested
      const overrideUrl = settingStore.independentDelayTest
          ? (settingStore.groupTestUrls.find((x: {name: string; url: string}) => x.name === onlyGroup)?.url || null)
          : null;
      const proxies = await api.getProxies(
          onlyGroup,
          proxiesStore.isHide,
          proxiesStore.isSort,
          settingStore.independentDelayTest,
          overrideUrl,
          settingStore.independentDelayTest ? settingStore.testUrl : null
      );
      fullViewNodes.value = { ...fullViewNodes.value, [onlyGroup]: proxies };
      if (proxiesStore.active === onlyGroup) {
        nodeList.value = proxies;
      }
      return;
    }

    const groupsArr = [...groupList.value];
    const pairs = await Promise.all(
        groupsArr.map(async (group) => {
          const overrideUrl = settingStore.independentDelayTest
              ? (settingStore.groupTestUrls.find((x: {name: string; url: string}) => x.name === group)?.url || null)
              : null;
          const proxies = await api.getProxies(
              group,
              proxiesStore.isHide,
              proxiesStore.isSort,
              settingStore.independentDelayTest,
              overrideUrl,
              settingStore.independentDelayTest ? settingStore.testUrl : null
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
    fetchSmartWeights(); // fire-and-forget
    return;
  }

  const activeOverrideUrl = settingStore.independentDelayTest
      ? (settingStore.groupTestUrls.find((x: {name: string; url: string}) => x.name === proxiesStore.active)?.url || null)
      : null;
  nodeList.value = await api.getProxies(
      proxiesStore.active,
      proxiesStore.isHide,
      proxiesStore.isSort,
      settingStore.independentDelayTest,
      activeOverrideUrl,
      settingStore.independentDelayTest ? settingStore.testUrl : null
  ); // 更新响应式数据
  } catch (e) {
    // Mihomo ещё не готов — earlyRetry повторит попытку
    return;
  }
  fullViewNodes.value = {};

  // Update nested group selections for non-full view
  await updateNestedGroupSelections();
  fetchSmartWeights(); // fire-and-forget
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
  if (groupTypeMap.value[targetGroup] !== 'Selector') {
    pWarning(t('proxies.auto-group-no-manual-select'));
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
  if (proxiesStore.viewMode === 'full') {
    // Full view: test ALL groups concurrently (max 3 at a time), like Zashboard's allProxiesLatencyTest
    pLoad(t("proxies.loading"), async () => {
      const groups = [...groupList.value];
      if (groups.length === 0) return;
      // Simple p-limit(3): run at most 3 concurrent group tests
      const CONCURRENCY = 3;
      let active = 0;
      let idx = 0;
      await new Promise<void>((resolve) => {
        const next = () => {
          while (active < CONCURRENCY && idx < groups.length) {
            const g = groups[idx++];
            active++;
            testGroupDelay(g).finally(() => {
              active--;
              if (idx < groups.length) {
                next();
              } else if (active === 0) {
                resolve();
              }
            });
          }
          if (idx >= groups.length && active === 0) resolve();
        };
        next();
      });
    });
    return;
  }
  // Horizontal / dropdown: test only the active group
  pLoad(t("proxies.loading"), async () => {
    try {
      if (settingStore.independentDelayTest) {
        await testGroupDelay(proxiesStore.active);
      } else {
        await api.getDelay(proxiesStore.active, settingStore.testUrl, 3000);
        await nodes();
        fetchSmartWeights();
      }
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    }
  });
}

// Тихий тест задержек без индикатора загрузки (для автозапуска после смены профиля)
async function runDelayTestSilent() {
  if (!proxiesStore.active) return;
  try {
    if (settingStore.independentDelayTest) {
      await testGroupDelay(proxiesStore.active);
    } else {
      await api.getDelay(proxiesStore.active, settingStore.testUrl, 3000);
      await nodes();
      fetchSmartWeights();
    }
  } catch (_) {
    // silently ignore
  }
}

// Тест задержки для отдельной группы (в режиме Fullview)
const groupLatencyTesting = ref<Record<string, boolean>>({});

async function testGroupDelay(groupName: string) {
  if (groupLatencyTesting.value[groupName]) return;
  groupLatencyTesting.value = { ...groupLatencyTesting.value, [groupName]: true };
  try {
    // Determine test URL:
    // 1. User-configured per-group URL (groupTestUrls in settings)
    // 2. Group's own testUrl from Mihomo config (for URLTest/Fallback/Smart)
    // 3. Global testUrl fallback
    let url = settingStore.testUrl;
    let userOverrideUrl: string | null = null;
    if (settingStore.independentDelayTest) {
      const userEntry = settingStore.groupTestUrls.find((x: {name: string; url: string}) => x.name === groupName);
      if (userEntry?.url) {
        url = userEntry.url;
        userOverrideUrl = userEntry.url;
      } else {
        const groupUrl = await api.getGroupTestUrl(groupName);
        if (groupUrl) url = groupUrl;
      }
    }

    // For Selector/LoadBalance/Smart groups in independent mode:
    // test each node individually so results go into proxy.extra[url].history
    // This matches Zashboard behavior and allows per-URL accessibility detection
    // (e.g. DIRECT shows as unreachable for YouTube but reachable for Apple)
    const groupType = (groupTypeMap.value[groupName] || '').toLowerCase();
    const perNodeTypes = ['selector', 'loadbalance', 'smart'];
    if (settingStore.independentDelayTest && perNodeTypes.includes(groupType)) {
      const groupNodes = await api.getProxies(groupName, false, false, false, null);
      const nodeNames = groupNodes.map((n: any) => n.name);
      await Promise.all(nodeNames.map((nodeName: string) => api.testProxyLatency(nodeName, url, 3000)));
    } else {
      await api.getDelay(groupName, url, 3000);
    }

    await nodes(groupName);
    fetchSmartWeights();
  } catch (e) {
    if (e && typeof e === 'object' && 'message' in e) {
      const message = (e as {message?: unknown}).message;
      if (typeof message === 'string') pError(message);
    }
  } finally {
    groupLatencyTesting.value = { ...groupLatencyTesting.value, [groupName]: false };
  }
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

const scrollGroupIntoView = async (groupName: string) => {
  await nextTick();
  const container = proxyGroup.value as HTMLElement | null;
  if (!container) {
    return;
  }
  const buttons = Array.from(container.querySelectorAll<HTMLButtonElement>('button[data-group]'));
  const target = buttons.find((button) => button.dataset.group === groupName);
  if (!target) {
    return;
  }
  target.scrollIntoView({behavior: 'smooth', block: 'nearest', inline: 'center'});
};

let wheelAccumulator = 0;
let wheelResetTimer: ReturnType<typeof setTimeout> | null = null;
const handleGroupWheel = async (event: WheelEvent) => {
  if (proxiesStore.viewMode !== 'horizontal' || groupList.value.length === 0) {
    return;
  }
  if (event.deltaY === 0) {
    return;
  }
  wheelAccumulator += event.deltaY;
  if (wheelResetTimer) {
    clearTimeout(wheelResetTimer);
  }
  wheelResetTimer = setTimeout(() => {
    wheelAccumulator = 0;
  }, 150);
  if (Math.abs(wheelAccumulator) < 40) {
    return;
  }
  const direction = wheelAccumulator > 0 ? 1 : -1;
  wheelAccumulator = 0;
  const groups = groupList.value;
  const currentIndex = Math.max(0, groups.indexOf(proxiesStore.active));
  const nextIndex = Math.min(groups.length - 1, Math.max(0, currentIndex + direction));
  if (nextIndex === currentIndex) {
    return;
  }
  await setActive(groups[nextIndex]);
  await scrollGroupIntoView(groups[nextIndex]);
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
  if (!isDropdownOpen.value) {
    runDelayTestSilent(); // fire-and-forget: auto-test when dropdown first opens
  }
  isDropdownOpen.value = true;
};

let fresh: any = null;
let weightsInterval: any = null;
let earlyRetryInterval: any = null;
onMounted(async () => {
  await groups();
  await nodes();
  updateButtonVisibility();
  runDelayTestSilent(); // fire-and-forget: auto-test on initial mount
  // 监听 resize 事件
  window.addEventListener("resize", updateButtonVisibility);

  // earlyRetry: если Mihomo ещё не загрузил pxd-template конфиг при старте —
  // повторяем каждые 3 секунды до 30 секунд, пока группы не появятся.
  if (groupList.value.length === 0 && menuStore.rule !== 'direct') {
    let earlyRetryCount = 0;
    earlyRetryInterval = setInterval(async () => {
      earlyRetryCount++;
      if (groupList.value.length > 0 || earlyRetryCount >= 10) {
        clearInterval(earlyRetryInterval);
        earlyRetryInterval = null;
        return;
      }
      await groups();
      await nodes();
    }, 3000);
  }

  // 创建刷新定时器
  fresh = setInterval(async () => {
    if (groupList.value.length === 0 && menuStore.rule !== 'direct') {
      await groups();
    }
    await nodes();
  }, 10000);
  // Обновляем веса каждые 2 минуты
  weightsInterval = setInterval(() => {
    fetchSmartWeights();
  }, 120000);
});

onBeforeUnmount(() => {
  // 清除定时器
  clearInterval(fresh);
  clearInterval(weightsInterval);
  clearInterval(earlyRetryInterval);
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
  runDelayTestSilent(); // fire-and-forget: auto-test after profile switch
})

watch(() => proxiesStore.now, async () => {
  await nodes();
})

watch(groupList, (list) => {
  if (!list.length) return;
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
        <div
            @scroll="handleScroll"
            @wheel.prevent="handleGroupWheel"
            ref="proxyGroup"
            class="proxy-group"
        >
          <button
              :class="
              proxiesStore.active == item
                ? 'proxy-group-title proxy-group-title-select'
                : 'proxy-group-title'
            "
              @click="setActive(item)"
              v-for="item in groupList"
              :key="item + '-g'"
              :data-group="item"
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
            <span class="proxy-node-name" :title="node['name']">
              {{ node["displayName"] ?? node["name"] }}
            </span>
            <span v-if="node['origin']" class="proxy-origin" :title="node['origin']">
              {{ node["origin"] }}
            </span>
          </div>
          <div class="proxy-nodes-tags">
            <span class="proxy-nodes-tags-left">
              <span>{{ node["displayType"] ?? node["type"] }}</span>
              <template v-if="nestedGroupSelections[node['name']] && node['type']?.toLowerCase() !== 'smart' && node['type']?.toLowerCase() !== 'loadbalance'">
                <span class="proxy-selected-separator">•</span>
                <span class="proxy-selected-name" :title="nestedGroupSelections[node['name']]">
                  {{ nestedGroupSelections[node['name']] }}
                </span>
              </template>
            </span>
            <span class="proxy-nodes-tags-right">
              <!-- Иконка ранга: прокси внутри Smart-группы -->
              <template v-if="groupTypeMap[proxiesStore.active] === 'Smart'">
                <el-tooltip v-if="getNodeWeightInfo(proxiesStore.active, node['name'])?.rank === 'MostUsed'" :content="t('proxies.smart.most-used-tip', { weight: getNodeWeightInfo(proxiesStore.active, node['name'])?.weight })" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-mdi-shield/></el-icon>
                </el-tooltip>
                <el-tooltip v-else-if="getNodeWeightInfo(proxiesStore.active, node['name'])?.rank === 'OccasionalUsed'" :content="t('proxies.smart.occasional-used-tip', { weight: getNodeWeightInfo(proxiesStore.active, node['name'])?.weight })" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-mdi-shield-half-full/></el-icon>
                </el-tooltip>
                <el-tooltip v-else-if="getNodeWeightInfo(proxiesStore.active, node['name'])?.rank === 'RarelyUsed'" :content="t('proxies.smart.rarely-used-tip', { weight: getNodeWeightInfo(proxiesStore.active, node['name'])?.weight })" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-mdi-shield-outline/></el-icon>
                </el-tooltip>
                <el-tooltip v-else :content="t('proxies.smart.no-data')" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-mdi-shield-sync-outline/></el-icon>
                </el-tooltip>
              </template>
              <!-- Иконка сводки: сам прокси является Smart-группой -->
              <template v-else-if="node['type'] === 'Smart'">
                <el-tooltip v-if="!smartGroupWeights[node['name']]?.hasData" :content="t('proxies.smart.no-data')" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-mdi-shield-sync-outline/></el-icon>
                </el-tooltip>
                <el-tooltip v-else placement="top">
                  <template #content>
                    <div v-for="w in smartGroupWeights[node['name']].weights" :key="w.Name" class="weight-tooltip-row">
                      {{ w.Name }}: {{ rankLabel(w.Rank) }} ({{ w.Weight }})
                    </div>
                  </template>
                  <el-icon class="proxy-weight-icon"><icon-mdi-shield-check-outline/></el-icon>
                </el-tooltip>
              </template>
              <span :class="node['toClass']">{{ node["delay"] }} ms</span>
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
                <span v-if="selectedProxies[group] && groupTypeMap[group] !== 'Smart' && groupTypeMap[group] !== 'LoadBalance'" class="full-view-selected">
                  {{ $t('proxies.selected-label') }}: {{ selectedProxies[group] }}
                </span>
              </div>
            </div>
            <div class="full-view-header-actions">
              <el-tooltip :content="$t('proxies.test-group')" placement="top">
                <el-icon
                    class="full-view-test-btn"
                    :class="{ 'full-view-test-btn--testing': groupLatencyTesting[group] }"
                    @click.stop="testGroupDelay(group)"
                >
                  <icon-ep-loading v-if="groupLatencyTesting[group]"/>
                  <icon-mdi-speedometer v-else/>
                </el-icon>
              </el-tooltip>
              <el-icon class="full-view-toggle">
                <icon-ep-arrow-up v-if="expandedGroups[group]"/>
                <icon-ep-arrow-down v-else/>
              </el-icon>
            </div>
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
                  <span class="proxy-node-name" :title="node['name']">
                    {{ node["displayName"] ?? node["name"] }}
                  </span>
                  <span v-if="node['origin']" class="proxy-origin" :title="node['origin']">
                    {{ node["origin"] }}
                  </span>
                </div>
                <div class="proxy-nodes-tags">
                  <span class="proxy-nodes-tags-left">
                    <span>{{ node["displayType"] ?? node["type"] }}</span>
                    <template v-if="nestedGroupSelections[node['name']] && node['type']?.toLowerCase() !== 'smart' && node['type']?.toLowerCase() !== 'loadbalance'">
                      <span class="proxy-selected-separator">•</span>
                      <span class="proxy-selected-name" :title="nestedGroupSelections[node['name']]">
                        {{ nestedGroupSelections[node['name']] }}
                      </span>
                    </template>
                  </span>
                  <span class="proxy-nodes-tags-right">
                    <!-- Иконка ранга: прокси внутри Smart-группы -->
                    <template v-if="groupTypeMap[group] === 'Smart'">
                      <el-tooltip v-if="getNodeWeightInfo(group, node['name'])?.rank === 'MostUsed'" :content="t('proxies.smart.most-used-tip', { weight: getNodeWeightInfo(group, node['name'])?.weight })" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-mdi-shield/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else-if="getNodeWeightInfo(group, node['name'])?.rank === 'OccasionalUsed'" :content="t('proxies.smart.occasional-used-tip', { weight: getNodeWeightInfo(group, node['name'])?.weight })" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-mdi-shield-half-full/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else-if="getNodeWeightInfo(group, node['name'])?.rank === 'RarelyUsed'" :content="t('proxies.smart.rarely-used-tip', { weight: getNodeWeightInfo(group, node['name'])?.weight })" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-mdi-shield-outline/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else :content="t('proxies.smart.no-data')" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-mdi-shield-sync-outline/></el-icon>
                      </el-tooltip>
                    </template>
                    <!-- Иконка сводки: сам прокси является Smart-группой -->
                    <template v-else-if="node['type'] === 'Smart'">
                      <el-tooltip v-if="!smartGroupWeights[node['name']]?.hasData" :content="t('proxies.smart.no-data')" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-mdi-shield-sync-outline/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else placement="top">
                        <template #content>
                          <div v-for="w in smartGroupWeights[node['name']].weights" :key="w.Name" class="weight-tooltip-row">
                            {{ w.Name }}: {{ rankLabel(w.Rank) }} ({{ w.Weight }})
                          </div>
                        </template>
                        <el-icon class="proxy-weight-icon"><icon-mdi-shield-check-outline/></el-icon>
                      </el-tooltip>
                    </template>
                    <span :class="node['toClass']">{{ node["delay"] }} ms</span>
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
:deep(.bottom) {
  padding-bottom: 0;
}

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
  border-radius: 20px;
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
  margin-left: 0;
  width: 100%;
}

.proxy-nodes-card {
  width: calc(33% - 41px);
  max-width: 210px;
  border: 2px solid var(--sub-card-border);
  border-radius: 20px;
  padding: 8px 12px;
  background: var(--sub-card-bg);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  line-height: 1.3;
  box-shadow: var(--left-nav-shadow);
  margin-top: 3px;
  transition: background-color 0.15s, border-color 0.15s;
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
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.proxy-node-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.proxy-origin {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 999px;
  border: 1px solid var(--text-color);
  opacity: 0.7;
  white-space: nowrap;
  flex-shrink: 0;
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
  display: flex;
  align-items: center;
  gap: 4px;
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

.proxy-weight-icon {
  font-size: 13px;
  cursor: help;
  flex-shrink: 0;
}

.weight-tooltip-row {
  font-size: 13px;
  line-height: 1.6;
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
  border-radius: 20px;
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
  border-radius: 20px;
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
  border-radius: 12px;
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
  width: 100%;
  margin-left: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.full-view-group {
  border: 2px solid var(--sub-card-border);
  border-radius: 20px;
  background: var(--sub-card-bg);
  box-shadow: var(--left-nav-shadow);
  overflow: hidden;
}

.full-view-group:hover {
  box-shadow: var(--left-nav-hover-shadow);
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
  border-radius: 12px;
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

.full-view-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.full-view-test-btn {
  font-size: 18px;
  cursor: pointer;
  color: var(--text-color);
  opacity: 0.6;
  transition: opacity 0.2s, color 0.2s;
}

.full-view-test-btn:hover {
  opacity: 1;
  color: var(--hr-color);
}

.full-view-test-btn--testing {
  animation: spin 1s linear infinite;
  opacity: 0.4;
  pointer-events: none;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.full-view-toggle {
  font-size: 16px;
  transition: transform 0.2s;
  flex-shrink: 0;
}

.full-view-content {
  /* card(8) + padding(8) = group(16) */
  padding: 8px;
}

.full-view-loading {
  padding: 12px 0;
  color: var(--text-color);
  opacity: 0.8;
}

.full-view-nodes {
  gap: 8px; /* = padding контента */
  width: 100%;
  margin-left: 0;
}

.full-view-nodes .proxy-nodes-card {
  width: calc(33% - 41px);
  max-width: 210px;
  border-radius: 12px; /* concentric: group(20) - padding(8) = 12 */
  margin-top: 3px;
  box-sizing: border-box;
}
</style>
