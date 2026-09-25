<script setup lang="ts">
import createApi from "@/api";
import {useProxiesStore} from "@/store/proxiesStore";
import {useMenuStore} from "@/store/menuStore";
import {useSettingStore} from "@/store/settingStore";
import {useI18n} from "vue-i18n";
import {pError, pSuccess, pWarning} from "@/util/pLoad";
import {useWebStore} from "@/store/webStore";
import {changeProxyAndCloseConnections} from "@/util/proxy";
import {proxyNodeDelayLimiter} from "@/util/pLimit";
import {proxyTypeIcon, proxyTypeTooltip} from "@/util/proxyType";
import {withFlag} from "@/util/flag";
import {delayColor, delayLabel} from "@/util/delay";
import UiDropdown from "@/components/ui/UiDropdown.vue";
import IconArrowRight from "~icons/tabler/arrow-right";
import IconRocket from "~icons/tabler/rocket";

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
  if (!webStore.fProfile || !(webStore.fProfile as any)['id']) {
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
  if (!webStore.fProfile || !(webStore.fProfile as any)['id']) {
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

const toggleGroup = (group: string) => {
  const next = !expandedGroups.value[group];
  proxiesStore.setGroupExpansionState(group, next);
};

// --- Full view presentation -------------------------------------------------

// First load: skeletons until the group list has been fetched once.
const loaded = ref(false);

const anyGroupOpen = computed(() => groupList.value.some(group => expandedGroups.value[group]));

function toggleAllGroups() {
  const open = !anyGroupOpen.value;
  const result: Record<string, boolean> = {};
  groupList.value.forEach(group => {
    result[group] = open;
  });
  proxiesStore.replaceGroupExpansions(result);
}

// An expanded group is capped at the visible height minus ~150 px and scrolls
// inside itself, so the next group header is always a short scroll away.
const scrollBox = ref<HTMLElement | null>(null);
const viewHeight = ref(640);
let resizeObserver: ResizeObserver | null = null;
const groupMaxHeight = computed(() => Math.max(180, viewHeight.value - 150) + 'px');

// Nodes of an expanded group are rendered in chunks of 60; more are added when
// the inner list is scrolled near its end or via "Show N more".
const CHUNK = 60;
const renderLimit = ref<Record<string, number>>({});
const limitOf = (group: string) => renderLimit.value[group] ?? CHUNK;
const visibleNodes = (group: string) => (fullViewNodes.value[group] ?? []).slice(0, limitOf(group));
const hiddenCount = (group: string) => Math.max(0, (fullViewNodes.value[group]?.length ?? 0) - limitOf(group));

function showMore(group: string) {
  renderLimit.value = {...renderLimit.value, [group]: limitOf(group) + CHUNK * 2};
}

function onGroupScroll(group: string, e: Event) {
  const el = e.currentTarget as HTMLElement;
  if (hiddenCount(group) > 0 && el.scrollTop + el.clientHeight > el.scrollHeight - 200) {
    showMore(group);
  }
}

// "Jump to group" is only worth it when there are more than 7 visible groups.
const showJump = computed(() => groupList.value.length > 7);

async function jumpToGroup(group: string) {
  proxiesStore.setGroupExpansionState(group, true);
  await nextTick();
  const box = scrollBox.value;
  const el = box?.querySelector<HTMLElement>(`[data-group="${CSS.escape(group)}"]`);
  if (box && el) {
    box.scrollTo({top: el.offsetTop - 8, behavior: 'smooth'});
    el.querySelector<HTMLElement>('.group-head')?.focus({preventScroll: true});
  }
}

const nodeName = (node: any) => withFlag(node?.displayName ?? node?.name);

// Weight badge: rank of a node inside a Smart group, or the "weights ready"
// summary for a node that is itself a Smart group.
function weightInfo(group: string, node: any): { icon: 'most' | 'occasional' | 'rarely' | 'none' | 'ready'; tip: string; tone: string } | null {
  if (groupTypeMap.value[group] === 'Smart') {
    const info = getNodeWeightInfo(group, node?.name);
    switch (info?.rank) {
      case 'MostUsed':
        return {icon: 'most', tone: 'var(--success)', tip: t('proxies.smart.most-used-tip', {weight: info.weight})};
      case 'OccasionalUsed':
        return {icon: 'occasional', tone: 'var(--warning)', tip: t('proxies.smart.occasional-used-tip', {weight: info.weight})};
      case 'RarelyUsed':
        return {icon: 'rarely', tone: 'var(--text-3)', tip: t('proxies.smart.rarely-used-tip', {weight: info.weight})};
      default:
        return {icon: 'none', tone: 'var(--text-3)', tip: t('proxies.smart.no-data')};
    }
  }
  if (node?.type === 'Smart') {
    const data = smartGroupWeights.value[node.name];
    if (!data?.hasData) {
      return {icon: 'none', tone: 'var(--text-3)', tip: t('proxies.smart.no-data')};
    }
    const tip = data.weights.map(w => `${w.Name}: ${rankLabel(w.Rank)} (${w.Weight})`).join('\n');
    return {icon: 'ready', tone: 'var(--accent)', tip};
  }
  return null;
}

let fresh: any = null;
let weightsInterval: any = null;
let earlyRetryInterval: any = null;
onMounted(async () => {
  // Only the full view exists in the redesign.
  if (proxiesStore.viewMode !== 'full') {
    proxiesStore.setViewMode('full');
  }
  if (scrollBox.value) {
    viewHeight.value = scrollBox.value.clientHeight;
    resizeObserver = new ResizeObserver(() => {
      if (scrollBox.value) viewHeight.value = scrollBox.value.clientHeight;
    });
    resizeObserver.observe(scrollBox.value);
  }
  try {
    await groups();
    await nodes();
  } finally {
    loaded.value = true;
  }
  runDelayTestSilent(); // fire-and-forget: auto-test on initial mount

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
  resizeObserver?.disconnect();
});

// 监听具体状态
watch(() => menuStore.rule, // 监听 store 中的某个状态
    async () => {
      await groups();
      await nodes();
    }
);

watch(() => webStore.fProfile, async () => {
  await groups();
  await nodes();
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
  <div class="px-page">
    <div class="px-page-head proxies-head">
      <h1 class="px-page-title">{{ $t("proxies.title") }}</h1>
      <div class="proxies-tools">
        <UiIconButton :label="anyGroupOpen ? $t('proxies.collapse-all') : $t('proxies.expand-all')" @click="toggleAllGroups">
          <icon-tabler-fold v-if="anyGroupOpen" width="17" height="17"/>
          <icon-tabler-selector v-else width="17" height="17"/>
        </UiIconButton>
        <UiIconButton :label="$t('proxies.test')" :loading="bulkTestRunning" :active="bulkTestRunning" @click="testDelay">
          <UiSpinner v-if="bulkTestRunning"/>
          <icon-tabler-bolt v-else width="17" height="17"/>
        </UiIconButton>
        <UiIconButton :label="proxiesStore.isHide ? $t('proxies.hide-on') : $t('proxies.hide-off')"
                      :active="proxiesStore.isHide"
                      :pressed="proxiesStore.isHide"
                      @click="setHide">
          <icon-tabler-eye-off v-if="proxiesStore.isHide" width="17" height="17"/>
          <icon-tabler-eye v-else width="17" height="17"/>
        </UiIconButton>
        <UiIconButton :label="proxiesStore.isSort ? $t('proxies.sort-on') : $t('proxies.sort-off')"
                      :active="proxiesStore.isSort"
                      :pressed="proxiesStore.isSort"
                      @click="setSort">
          <icon-tabler-sort-descending v-if="proxiesStore.isSort" width="17" height="17"/>
          <icon-tabler-arrows-sort v-else width="17" height="17"/>
        </UiIconButton>
        <UiDropdown v-if="showJump" align="right" :min-width="240" :max-width="320" :max-height="360">
          <template #trigger="{ open, toggle, attrs }">
            <button type="button" v-bind="attrs" class="jump-btn" :class="{ 'is-open': open }" @click="toggle">
              <icon-tabler-list width="15" height="15"/>
              {{ $t('proxies.jump-to-group') }}
              <icon-tabler-chevron-down class="jump-chev" :class="{ 'is-open': open }" width="13" height="13"/>
            </button>
          </template>
          <template #default="{ close }">
            <button v-for="group in groupList"
                    :key="group + '-jump'"
                    type="button"
                    role="option"
                    data-dd-item
                    aria-selected="false"
                    class="px-dd-item"
                    @click="close(); jumpToGroup(group)">
              <span class="ellipsis" style="flex:1" v-tip="withFlag(group)">{{ withFlag(group) }}</span>
              <span class="jump-count tabular">{{ fullViewNodes[group]?.length ?? '' }}</span>
            </button>
          </template>
        </UiDropdown>
      </div>
    </div>

    <div ref="scrollBox" class="proxies-body">
      <UiSkeleton v-if="!loaded && menuStore.rule != 'direct'" variant="group" :count="3"/>

      <div v-else-if="menuStore.rule == 'direct'" class="proxies-direct">
        <UiEmpty :icon="IconArrowRight" :title="$t('proxies.direct')"/>
      </div>

      <UiEmpty v-else-if="groupList.length === 0"
               :icon="IconRocket"
               :title="$t('empty.proxies.title')"
               :text="$t('empty.proxies.text')"/>

      <section v-for="group in groupList"
               v-else
               :key="group + '-full'"
               class="group"
               :data-group="group">
        <div class="group-head"
             role="button"
             tabindex="0"
             :aria-expanded="expandedGroups[group] ? 'true' : 'false'"
             @click="toggleGroup(group)">
          <div class="group-head__main">
            <div class="group-icon">
              <img v-if="groupIcons[group]" :src="groupIcons[group]" alt="" @error="handleIconError">
              <span v-else class="group-type" v-tip="proxyTypeTooltip(groupTypeMap[group], true, t)">
                <component :is="proxyTypeIcon(groupTypeMap[group], true)" width="17" height="17"/>
              </span>
            </div>
            <div class="group-text">
              <span class="group-name ellipsis" v-tip="withFlag(group)">{{ withFlag(group) }}</span>
              <span v-if="selectedProxies[group] && groupTypeMap[group] !== 'Smart' && groupTypeMap[group] !== 'LoadBalance'"
                    class="group-selected ellipsis">
                {{ $t('proxies.selected-label') }}: {{ withFlag(selectedProxies[group]) }}
              </span>
            </div>
          </div>
          <div class="group-head__actions">
            <UiIconButton :size="26"
                          :label="$t('proxies.test-group')"
                          :loading="groupLatencyTesting[group]"
                          :active="groupLatencyTesting[group]"
                          @click.stop="testGroupDelay(group)">
              <UiSpinner v-if="groupLatencyTesting[group]"/>
              <icon-tabler-bolt v-else width="16" height="16"/>
            </UiIconButton>
            <icon-tabler-chevron-down class="group-chev" :class="{ 'is-collapsed': !expandedGroups[group] }" width="16" height="16"/>
          </div>
        </div>

        <!-- Collapsed groups render no nodes at all. -->
        <template v-if="expandedGroups[group]">
          <div v-if="!fullViewNodes[group]" class="group-loading">
            <UiSpinner/>
            {{ $t('proxies.loading') }}
          </div>
          <div v-else
               class="group-nodes"
               :style="{ maxHeight: groupMaxHeight }"
               @scroll.passive="onGroupScroll(group, $event)">
            <button v-for="node in visibleNodes(group)"
                    :key="group + '-' + node['name']"
                    type="button"
                    class="node"
                    :class="{ 'is-selected': node['now'] }"
                    :aria-pressed="node['now'] ? 'true' : 'false'"
                    @click="setProxy(node['now'], node['name'], group)">
              <span class="node-name ellipsis" v-tip="nodeName(node)">{{ nodeName(node) }}</span>
              <span class="node-meta">
                <span class="node-type" v-tip="typeTooltip(node)">
                  <component :is="typeIcon(node)" width="14" height="14"/>
                </span>
                <span v-if="serverDescription(node) || node['origin'] || (nestedGroupSelections[node['name']] && node['type']?.toLowerCase() !== 'smart' && node['type']?.toLowerCase() !== 'loadbalance')"
                      class="node-desc ellipsis"
                      v-tip="[serverDescription(node), node['origin'], nestedGroupSelections[node['name']]].filter(Boolean).join(' • ')">
                  {{ [serverDescription(node), node['origin'], (node['type']?.toLowerCase() !== 'smart' && node['type']?.toLowerCase() !== 'loadbalance') ? withFlag(nestedGroupSelections[node['name']]) : ''].filter(Boolean).join(' • ') }}
                </span>
                <span class="node-spacer"></span>
                <template v-for="w in [weightInfo(group, node)]" :key="'w'">
                  <span v-if="w" class="node-weight" :style="{ color: w.tone }" v-tip="w.tip">
                    <icon-tabler-shield-half v-if="w.icon === 'occasional'" width="13" height="13"/>
                    <icon-tabler-shield-question v-else-if="w.icon === 'none'" width="13" height="13"/>
                    <icon-tabler-shield-check v-else-if="w.icon === 'ready'" width="13" height="13"/>
                    <icon-tabler-shield v-else width="13" height="13"/>
                  </span>
                </template>
                <span class="node-delay tabular" :style="{ color: delayColor(node['delay']) }">{{ delayLabel(node['delay']) }}</span>
              </span>
            </button>
            <button v-if="hiddenCount(group) > 0" type="button" class="show-more" @click="showMore(group)">
              {{ $t('proxies.show-more', {n: hiddenCount(group)}) }}
            </button>
          </div>
        </template>
      </section>
    </div>
  </div>
</template>

<style scoped>
.proxies-head {
  justify-content: space-between;
}

.proxies-tools {
  display: flex;
  gap: 6px;
  align-items: center;
}

.jump-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 10px 0 12px;
  border-radius: 999px;
  border: none;
  background: var(--panel-soft);
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}

.jump-btn:hover, .jump-btn.is-open {
  background: var(--hover-bg);
}

.jump-chev {
  opacity: .7;
  transition: transform .15s;
}

.jump-chev.is-open {
  transform: rotate(180deg);
}

.jump-count {
  font-size: 12px;
  color: var(--text-3);
  flex-shrink: 0;
}

.proxies-body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 8px 28px 28px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.group {
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--input-bg);
  overflow: hidden;
  flex-shrink: 0;
}

.group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 16px;
  cursor: pointer;
}

.group-head:hover {
  background: var(--hover-bg);
}

.group-head:focus-visible {
  outline-offset: -2px;
}

.group-head__main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.group-icon {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--panel-soft);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.group-icon img {
  width: 22px;
  height: 22px;
  object-fit: contain;
}

.group-type {
  display: flex;
  color: var(--text-2);
}

.group-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.group-name {
  font-size: 14px;
  font-weight: 700;
}

.group-selected {
  font-size: 12px;
  color: var(--text-2);
}

.group-head__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.group-chev {
  transition: transform .15s;
  flex-shrink: 0;
  color: var(--text-2);
}

.group-chev.is-collapsed {
  transform: rotate(-90deg);
}

.group-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 16px 16px;
  font-size: 13px;
  color: var(--text-2);
}

.group-nodes {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  align-content: start;
  gap: 10px;
  padding: 2px 6px 16px 16px;
  overflow-y: auto;
  scrollbar-gutter: stable;
}

.node {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  text-align: left;
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 11px 13px;
  cursor: pointer;
  background: var(--input-bg);
  color: var(--text);
}

.node:hover {
  border-color: color-mix(in srgb, var(--accent) 50%, var(--border));
}

.node.is-selected {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 14%, var(--input-bg));
}

.node-name {
  font-size: 13px;
  font-weight: 600;
}

.node-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.node-type {
  display: flex;
  color: var(--text-3);
  flex-shrink: 0;
  cursor: help;
}

.node-desc {
  font-size: 11px;
  color: var(--text-3);
}

.node-spacer {
  flex: 1;
}

.node-weight {
  display: flex;
  flex-shrink: 0;
  cursor: help;
}

.node-delay {
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
  flex-shrink: 0;
}

.show-more {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 36px;
  border-radius: 8px;
  border: 1px dashed var(--border);
  background: transparent;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-2);
  cursor: pointer;
}

.show-more:hover {
  background: var(--hover-bg);
  color: var(--text);
}

.proxies-direct {
  display: flex;
  justify-content: center;
}
</style>
