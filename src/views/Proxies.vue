<script setup lang="ts">
import createApi from "@/api";
import {useProxiesStore, type ProxyViewMode} from "@/store/proxiesStore";
import {useMenuStore} from "@/store/menuStore";
import {useSettingStore} from "@/store/settingStore";
import {useI18n} from "vue-i18n";
import {pError, pSuccess, pWarning} from "@/util/pLoad";
import {useWebStore} from "@/store/webStore";
import {changeProxyAndCloseConnections} from "@/util/proxy";
import {proxyNodeDelayLimiter} from "@/util/pLimit";
import {proxyTypeIcon, proxyTypeTooltip} from "@/util/proxyType";

// Delay-test timeouts (ms). Kept distinct: a single node's dial should fail
// fast, while a group-level /group/:name/delay call needs enough budget for
// Mihomo to test every member server-side in that one request.
const NODE_TEST_TIMEOUT = 2000;
const GROUP_TEST_TIMEOUT = 5000;

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

// --- Type badge -------------------------------------------------------------
// The type is shown as an icon, the panel's server description (when it sends
// one) as text. They used to share one text slot, where the description won and
// the type disappeared entirely.

// A node that is itself a group. Read from groupTypeMap, which is keyed by every
// group name the core reports, so a group type this build has never heard of is
// still recognised as a group.
const isGroupNode = (node: any) => !!groupTypeMap.value[node?.name];

const typeIcon = (node: any) => proxyTypeIcon(node?.type, isGroupNode(node));
const typeTooltip = (node: any) => proxyTypeTooltip(node?.type, isGroupNode(node), t);

// getDisplayType() returns the server description when the panel sends one and
// the plain type otherwise; only the former is worth printing next to the icon.
const serverDescription = (node: any): string | undefined =>
    node?.displayType && node.displayType !== node.type ? node.displayType : undefined;

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

// Тест задержки для всей страницы (кнопка у заголовка "Прокси").
// Не блокирует интерфейс — крутится только иконка кнопки, можно продолжать
// пользоваться приложением (включая переход на другие вкладки) пока тест идёт.
const bulkTestRunning = ref(false);

function testDelay() {
  if (bulkTestRunning.value) return;
  bulkTestRunning.value = true;

  if (proxiesStore.viewMode === 'full') {
    // Full view: test ALL groups concurrently (max 3 at a time), like Zashboard's allProxiesLatencyTest
    (async () => {
      try {
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
      } finally {
        bulkTestRunning.value = false;
      }
    })();
    return;
  }
  // Horizontal / dropdown: test only the active group
  (async () => {
    try {
      if (settingStore.independentDelayTest) {
        await testGroupDelay(proxiesStore.active);
      } else {
        await api.getDelay(proxiesStore.active, settingStore.testUrl, GROUP_TEST_TIMEOUT);
        await nodes();
        fetchSmartWeights();
      }
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    } finally {
      bulkTestRunning.value = false;
    }
  })();
}

// Тихий тест задержек без индикатора загрузки (для автозапуска после смены профиля)
async function runDelayTestSilent() {
  if (!proxiesStore.active) return;
  try {
    if (settingStore.independentDelayTest) {
      await testGroupDelay(proxiesStore.active);
    } else {
      await api.getDelay(proxiesStore.active, settingStore.testUrl, GROUP_TEST_TIMEOUT);
      await nodes();
      fetchSmartWeights();
    }
  } catch (_) {
    // silently ignore
  }
}

// Показывает итог теста группы: сколько узлов ответили, сколько по таймауту
function reportGroupTestResult(groupName: string, total: number, failed: number) {
  if (total === 0) return;
  const succeeded = total - failed;
  const message = t('proxies.test-result', {name: groupName, succeeded, total});
  if (failed === 0) {
    pSuccess(message);
  } else {
    pWarning(message);
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
      // Route every per-node request through the shared proxyNodeDelayLimiter
      // (max 5 in flight app-wide) instead of firing them all at once. When
      // several groups are tested together (testDelay's outer p-limit(3)),
      // an unbounded Promise.all here would stack into hundreds of
      // simultaneous dials through the local backend, starving the
      // connection pool until requests queue past the shared axios timeout
      // and fail with a generic Network Error.
      const results = await Promise.allSettled(
          nodeNames.map((nodeName: string) =>
              proxyNodeDelayLimiter(() => api.testProxyLatency(nodeName, url, NODE_TEST_TIMEOUT))
          )
      );
      const failed = results.filter((r) => r.status !== 'fulfilled' || r.value === false).length;
      reportGroupTestResult(groupName, nodeNames.length, failed);
    } else {
      await api.getDelay(groupName, url, GROUP_TEST_TIMEOUT);
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
  <MyLayout>
    <template #top>
      <el-space class="space">
        <div class="title">
          {{ $t("proxies.title") }}
        </div>
        <div class="proxy-option">
          <el-tooltip :content="$t('proxies.test')" placement="top">
            <el-icon
                @click="testDelay"
                class="proxy-option-btn"
                :class="{ 'proxy-option-btn--testing': bulkTestRunning }"
            >
              <icon-tabler-loader-2 v-if="bulkTestRunning"/>
              <icon-tabler-gauge v-else/>
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
              <icon-tabler-eye-off v-if="proxiesStore.isHide"/>
              <icon-tabler-eye v-else/>
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
              <icon-tabler-sort-ascending v-if="proxiesStore.isSort"/>
              <icon-tabler-arrows-sort v-else/>
            </el-icon>
          </el-tooltip>

          <el-tooltip
              :content="viewModeTooltip"
              placement="top"
          >
            <el-icon @click="cycleViewMode" class="proxy-option-btn">
              <icon-tabler-arrows-horizontal v-if="proxiesStore.viewMode === 'horizontal'"/>
              <icon-tabler-arrows-vertical v-else-if="proxiesStore.viewMode === 'dropdown'"/>
              <icon-tabler-list v-else/>
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
            <el-icon class="dropdown-chevron" :class="{ 'dropdown-chevron--open': isDropdownOpen }">
              <icon-tabler-chevron-down/>
            </el-icon>
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
              :class="{ 'dropdown-item--active': item === proxiesStore.active }"
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
          <icon-tabler-chevron-left/>
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
          <icon-tabler-chevron-right/>
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
            <icon-tabler-check v-if="node['now']" class="proxy-node-check"/>
            <span class="proxy-node-name" :title="node['name']">
              {{ node["displayName"] ?? node["name"] }}
            </span>
            <span v-if="node['origin']" class="proxy-origin" :title="node['origin']">
              {{ node["origin"] }}
            </span>
          </div>
          <div class="proxy-nodes-tags">
            <span class="proxy-nodes-tags-left">
              <el-tooltip :content="typeTooltip(node)" placement="top">
                <el-icon class="proxy-type-icon">
                  <component :is="typeIcon(node)"/>
                </el-icon>
              </el-tooltip>
              <span v-if="serverDescription(node)" class="proxy-type-desc">{{ serverDescription(node) }}</span>
              <template v-if="nestedGroupSelections[node['name']] && node['type']?.toLowerCase() !== 'smart' && node['type']?.toLowerCase() !== 'loadbalance'">
                <span v-if="serverDescription(node)" class="proxy-selected-separator">•</span>
                <span class="proxy-selected-name" :title="nestedGroupSelections[node['name']]">
                  {{ nestedGroupSelections[node['name']] }}
                </span>
              </template>
            </span>
            <span class="proxy-nodes-tags-right">
              <!-- Иконка ранга: прокси внутри Smart-группы -->
              <template v-if="groupTypeMap[proxiesStore.active] === 'Smart'">
                <el-tooltip v-if="getNodeWeightInfo(proxiesStore.active, node['name'])?.rank === 'MostUsed'" :content="t('proxies.smart.most-used-tip', { weight: getNodeWeightInfo(proxiesStore.active, node['name'])?.weight })" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-tabler-shield-filled/></el-icon>
                </el-tooltip>
                <el-tooltip v-else-if="getNodeWeightInfo(proxiesStore.active, node['name'])?.rank === 'OccasionalUsed'" :content="t('proxies.smart.occasional-used-tip', { weight: getNodeWeightInfo(proxiesStore.active, node['name'])?.weight })" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-tabler-shield-half/></el-icon>
                </el-tooltip>
                <el-tooltip v-else-if="getNodeWeightInfo(proxiesStore.active, node['name'])?.rank === 'RarelyUsed'" :content="t('proxies.smart.rarely-used-tip', { weight: getNodeWeightInfo(proxiesStore.active, node['name'])?.weight })" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-tabler-shield/></el-icon>
                </el-tooltip>
                <el-tooltip v-else :content="t('proxies.smart.no-data')" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-tabler-shield-question/></el-icon>
                </el-tooltip>
              </template>
              <!-- Иконка сводки: сам прокси является Smart-группой -->
              <template v-else-if="node['type'] === 'Smart'">
                <el-tooltip v-if="!smartGroupWeights[node['name']]?.hasData" :content="t('proxies.smart.no-data')" placement="top">
                  <el-icon class="proxy-weight-icon"><icon-tabler-shield-question/></el-icon>
                </el-tooltip>
                <el-tooltip v-else placement="top">
                  <template #content>
                    <div v-for="w in smartGroupWeights[node['name']].weights" :key="w.Name" class="weight-tooltip-row">
                      {{ w.Name }}: {{ rankLabel(w.Rank) }} ({{ w.Weight }})
                    </div>
                  </template>
                  <el-icon class="proxy-weight-icon"><icon-tabler-shield-check/></el-icon>
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
                  <icon-tabler-loader-2 v-if="groupLatencyTesting[group]"/>
                  <icon-tabler-gauge v-else/>
                </el-icon>
              </el-tooltip>
              <el-icon class="full-view-toggle">
                <icon-tabler-chevron-up v-if="expandedGroups[group]"/>
                <icon-tabler-chevron-down v-else/>
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
                  <!-- Выбранный узел помечен не только заливкой: цвет не должен
                       быть единственным носителем состояния. -->
                  <icon-tabler-check v-if="node['now']" class="proxy-node-check"/>
                  <span class="proxy-node-name" :title="node['name']">
                    {{ node["displayName"] ?? node["name"] }}
                  </span>
                  <span v-if="node['origin']" class="proxy-origin" :title="node['origin']">
                    {{ node["origin"] }}
                  </span>
                </div>
                <div class="proxy-nodes-tags">
                  <span class="proxy-nodes-tags-left">
                    <el-tooltip :content="typeTooltip(node)" placement="top">
                      <el-icon class="proxy-type-icon">
                        <component :is="typeIcon(node)"/>
                      </el-icon>
                    </el-tooltip>
                    <span v-if="serverDescription(node)" class="proxy-type-desc">{{ serverDescription(node) }}</span>
                    <template v-if="nestedGroupSelections[node['name']] && node['type']?.toLowerCase() !== 'smart' && node['type']?.toLowerCase() !== 'loadbalance'">
                      <span v-if="serverDescription(node)" class="proxy-selected-separator">•</span>
                      <span class="proxy-selected-name" :title="nestedGroupSelections[node['name']]">
                        {{ nestedGroupSelections[node['name']] }}
                      </span>
                    </template>
                  </span>
                  <span class="proxy-nodes-tags-right">
                    <!-- Иконка ранга: прокси внутри Smart-группы -->
                    <template v-if="groupTypeMap[group] === 'Smart'">
                      <el-tooltip v-if="getNodeWeightInfo(group, node['name'])?.rank === 'MostUsed'" :content="t('proxies.smart.most-used-tip', { weight: getNodeWeightInfo(group, node['name'])?.weight })" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-tabler-shield-filled/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else-if="getNodeWeightInfo(group, node['name'])?.rank === 'OccasionalUsed'" :content="t('proxies.smart.occasional-used-tip', { weight: getNodeWeightInfo(group, node['name'])?.weight })" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-tabler-shield-half/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else-if="getNodeWeightInfo(group, node['name'])?.rank === 'RarelyUsed'" :content="t('proxies.smart.rarely-used-tip', { weight: getNodeWeightInfo(group, node['name'])?.weight })" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-tabler-shield/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else :content="t('proxies.smart.no-data')" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-tabler-shield-question/></el-icon>
                      </el-tooltip>
                    </template>
                    <!-- Иконка сводки: сам прокси является Smart-группой -->
                    <template v-else-if="node['type'] === 'Smart'">
                      <el-tooltip v-if="!smartGroupWeights[node['name']]?.hasData" :content="t('proxies.smart.no-data')" placement="top">
                        <el-icon class="proxy-weight-icon"><icon-tabler-shield-question/></el-icon>
                      </el-tooltip>
                      <el-tooltip v-else placement="top">
                        <template #content>
                          <div v-for="w in smartGroupWeights[node['name']].weights" :key="w.Name" class="weight-tooltip-row">
                            {{ w.Name }}: {{ rankLabel(w.Rank) }} ({{ w.Weight }})
                          </div>
                        </template>
                        <el-icon class="proxy-weight-icon"><icon-tabler-shield-check/></el-icon>
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
  margin-top: var(--px-space-4);
}

.title {
  font-size: var(--px-fs-display);
  font-weight: bold;
}

.proxy-option {
  font-size: 30px;
  padding-top: var(--px-space-3);
  margin-left: var(--px-space-3);
}

.proxy-option-btn {
  margin-right: var(--px-space-4);
}

.proxy-option-btn:hover {
  cursor: pointer;
  color: var(--hr-color);
}

.proxy-option-btn--testing {
  animation: spin 1s linear infinite;
  opacity: 0.4;
  pointer-events: none;
}

/* Без min-height: высоту ряда задают отступы .proxy-group — те же, что у
   .dropdown, поэтому шапка не прыгает при смене режима группировки. */
.button-container {
  display: flex;
  align-items: center;
  width: 100%;
}

.proxy-group {
  display: flex;
  flex: 1;
  min-width: 0;
  gap: var(--px-space-2);
  margin: var(--px-space-3) 0 var(--px-space-1);
  overflow-x: hidden;
  scroll-behavior: smooth;
}

/* Стрелки прокрутки — та же круглая кнопка-иконка, что и в остальных
   разделах: без рамки, подсветка по наведению. */
.scroll-left,
.scroll-right {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--px-r-pill);
  color: var(--text-color);
  opacity: .72;
  cursor: pointer;
  transition: background-color var(--px-dur) var(--px-ease),
  opacity var(--px-dur) var(--px-ease);
}

.scroll-left:hover,
.scroll-right:hover {
  background-color: var(--left-nav-btn-hover-bg);
  opacity: 1;
}

.scroll-left {
  margin-right: var(--px-space-2);
}

.scroll-right {
  margin-left: var(--px-space-2);
}

.scroll-left[hidden],
.scroll-right[hidden] {
  display: none;
}

/* Пилюля группы — тот же контрол, что .px-btn: высота из общей шкалы,
   рамка в один пиксель вместо двух и цвет рамки из карточек, а не --hr-color,
   который делал из ряда групп забор из ярких обводок. */
.proxy-group-title {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  height: var(--px-control-h);
  padding: 0 var(--px-space-3);
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  border: 1px solid var(--sub-card-border);
  border-radius: var(--px-r-pill);
  font-size: var(--px-fs-small);
  font-weight: 600;
  font-family: inherit;
  text-align: center;
  cursor: pointer;
  box-shadow: var(--px-elev-1);
  white-space: nowrap;
  transition: background-color var(--px-dur) var(--px-ease),
  border-color var(--px-dur) var(--px-ease);
}

.proxy-group-title:hover,
.proxy-group-title-select {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--px-elev-2);
  border-color: var(--text-color);
}

.proxy-group-title-select:hover {
  cursor: default;
}

.proxy-group-content {
  display: inline-flex;
  align-items: center;
  gap: var(--px-space-2);
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
  /* Сетка, а не flex-wrap: 1fr раздаёт ровно остаток ширины, поэтому ряд
     заканчивается вровень с правым краем панели, а не оставляет пустую полосу
     справа от прежних `calc(33% - 41px)` + `max-width: 210px`. Та же раскладка,
     что у узлов в полном виде, — режим группировки не меняет ширину карточек. */
  display: grid;
  /* minmax(0, …), а не голый 1fr: `1fr` разворачивается в `minmax(auto, 1fr)`,
     и нижней границей дорожки становится min-content самой длинной карточки. */
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--px-space-3);
  padding: 0;
  color: var(--text-color);
  margin-left: 0;
  width: 100%;
}

/* Дальше трёх колонок карточка растягивается за ~400px, поэтому с этой ширины
   набираем столько ~260px колонок, сколько вмещает панель. auto-fill, а не
   auto-fit: пустые дорожки сохраняются, и группа из двух узлов даёт карточки
   такой же ширины, как группа из двадцати. */
@media (min-width: 1400px) {
  .proxy-nodes {
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  }
}

.proxy-nodes-card {
  /* Ширину задаёт дорожка сетки. min-width: 0 обязателен: иначе он остаётся
     `auto`, то есть min-content по имени узла с `white-space: nowrap`, и длинное
     имя распирает карточку за ширину дорожки, а ellipsis не срабатывает. */
  min-width: 0;
  border: 1px solid var(--sub-card-border);
  border-radius: var(--px-r-md);
  padding: var(--px-space-2) var(--px-space-3);
  background: var(--sub-card-bg);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  line-height: 1.3;
  box-shadow: var(--px-elev-1);
  transition: background-color var(--px-dur) var(--px-ease),
  border-color var(--px-dur) var(--px-ease);
}

.proxy-nodes-card:hover,
.proxy-node-select {
  background-color: var(--left-item-selected-bg);
  border-color: var(--text-color);
  cursor: pointer;
}

.proxy-node-select:hover {
  cursor: default;
}

.proxy-nodes-title {
  font-size: var(--px-fs-body);
  display: flex;
  align-items: center;
  gap: var(--px-space-2);
  min-width: 0;
}

.proxy-node-check {
  flex-shrink: 0;
  font-size: 15px;
  color: var(--text-color);
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
  border-radius: var(--px-r-pill);
  border: 1px solid var(--text-color);
  opacity: 0.7;
  white-space: nowrap;
  flex-shrink: 0;
}

.proxy-nodes-tags {
  font-size: var(--px-fs-body);
  display: flex;
  margin-top: var(--px-space-2);
  justify-content: space-between;
}

.proxy-nodes-tags-left {
  flex: 1;
  display: flex;
  align-items: center;
  gap: var(--px-space-1);
  overflow: hidden;
  min-width: 0;
}

/* Описание сервера от панели (serverDescription, до 25 символов). Обрезается,
   иначе распирает строку тегов и выдавливает задержку за край карточки. Сам тип
   теперь показывает иконка слева, а не этот текст. */
.proxy-type-desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

/* Иконка типа: не сжимается, когда описание длинное, и приглушена, чтобы не
   спорить с именем узла. */
.proxy-type-icon {
  flex-shrink: 0;
  font-size: 15px;
  opacity: 0.75;
  cursor: help;
}

.proxy-nodes-tags-right {
  display: flex;
  align-items: center;
  gap: var(--px-space-1);
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
  width: 100%;
  margin: var(--px-space-3) 0 var(--px-space-1);
}

/* Кнопка-селектор группы: высота и рамка те же, что у остальных контролов,
   чтобы ряд совпадал по высоте с пилюлями горизонтального режима. Заливка
   выбора остаётся — кнопка показывает активную группу. */
.dropdown-btn {
  display: inline-flex;
  align-items: center;
  height: var(--px-control-h);
  min-width: 220px;
  max-width: 100%;
  padding: 0 var(--px-space-3);
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--px-elev-2);
  border: 1px solid var(--text-color);
  border-radius: var(--px-r-pill);
  color: var(--text-color);
  font-family: inherit;
  font-size: var(--px-fs-small);
  font-weight: 600;
  cursor: pointer;
  outline: none;
  text-align: left;
  transition: background-color var(--px-dur) var(--px-ease);
}

.dropdown-btn:hover {
  background-color: var(--left-nav-btn-hover-bg);
}

.dropdown-btn-content {
  display: inline-flex;
  align-items: center;
  gap: var(--px-space-2);
  justify-content: flex-start;
  width: 100%;
  min-width: 0;
}

/* Шеврон: у кнопки не было ни одного признака, что она раскрывает список.
   Поворот вниз-вверх повторяет состояние списка. */
.dropdown-chevron {
  flex-shrink: 0;
  margin-left: auto;
  font-size: 14px;
  opacity: .7;
  transition: transform var(--px-dur) var(--px-ease);
}

.dropdown-chevron--open {
  transform: rotate(180deg);
}

.dropdown-list {
  position: absolute;
  background: var(--dropdown-list-bg);
  border: 1px solid var(--sub-card-border);
  box-shadow: var(--px-elev-2);
  margin-top: var(--px-space-1);
  padding: var(--px-space-1);
  list-style: none;
  /* Та же ширина, что у кнопки: absolute-элемент ужимается по содержимому,
     поэтому список задаёт нижнюю границу, а не собственную ширину. */
  min-width: 220px;
  box-sizing: border-box;
  z-index: 20;
  border-radius: var(--px-r-md);
  font-size: var(--px-fs-small);
  text-align: left;
  max-height: calc(100vh - 230px);
  overflow-y: auto;
}

.dropdown-list::-webkit-scrollbar {
  width: 5px;
}

.dropdown-list::-webkit-scrollbar-track {
  background: transparent;
}

.dropdown-list::-webkit-scrollbar-thumb {
  background: var(--scrollbar-bg);
  border-radius: 2px;
}

.dropdown-item {
  color: var(--text-color);
  padding: var(--px-space-2) var(--px-space-3);
  border-radius: var(--px-r-sm);
  cursor: pointer;
  transition: background-color var(--px-dur-fast) var(--px-ease);
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
  background: var(--left-nav-btn-hover-bg);
}

/* Активная группа отмечена и в списке — иначе при раскрытии непонятно,
   на чём стоишь. */
.dropdown-item--active {
  background: var(--left-item-selected-bg);
  font-weight: 600;
}

.full-view-groups {
  width: 100%;
  margin-left: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.full-view-group {
  border: 1px solid var(--sub-card-border);
  border-radius: var(--px-r-lg);
  /* Подложка ложится поверх штатной заливки: при нулевой плотности (тёмные
     обои) карточка выглядит ровно как раньше. */
  background:
    linear-gradient(rgba(var(--px-scrim-rgb), var(--px-scrim-a)),
    rgba(var(--px-scrim-rgb), var(--px-scrim-a))),
    var(--sub-card-bg);
  box-shadow: var(--px-elev-1);
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
  /* Grid rather than flex-wrap: 1fr distributes the exact remaining space, so a
     row always ends flush with the right edge instead of leaving the gutter the
     old `calc(33% - 41px)` + `max-width: 210px` combination always left behind. */
  display: grid;
  /* minmax(0, …), а не голый 1fr: `1fr` разворачивается в `minmax(auto, 1fr)`,
     и нижней границей дорожки становится min-content самой длинной карточки —
     из-за чего три колонки суммарно вылезали за правый край панели. */
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px; /* = padding контента */
  width: 100%;
  margin-left: 0;
}

/* Past this width three columns would stretch every card beyond ~400px, so fit
   as many ~260px columns as the panel allows instead.
   auto-fill (never auto-fit) is deliberate: it keeps the empty tracks, so a
   group with two nodes gets cards exactly as wide as a group with twenty — the
   column width depends only on the panel, which is identical for every group. */
@media (min-width: 1400px) {
  .full-view-nodes {
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  }
}

.full-view-nodes .proxy-nodes-card {
  border-radius: var(--px-r-sm); /* concentric: group(20) - padding(8) */
  box-sizing: border-box;
}
</style>
