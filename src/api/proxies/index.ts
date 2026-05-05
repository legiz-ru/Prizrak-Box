// 排除的分组名字
const excludeGroupName: any = {
    DIRECT: true,
    REJECT: true,
    "REJECT-DROP": true,
    PASS: true,
    COMPATIBLE: true,
    GLOBAL: true
}

// 不排除的分组类型
const includeGroup: any = {
    Selector: true,
    URLTest: true,
    Fallback: true,
    LoadBalance: true,
    Smart: true,
    Relay: true
}

// 不排除的节点类型
const includeProxy: any = {
    Direct: true,
    Reject: true,
    RejectDrop: true,
    URLTest: true,
    LoadBalance: true,
    Selector: true,
    Pass: true,
    Relay: true,
    Fallback: true,
    Smart: true,
}

let proxyOriginCache: Record<string, string> | null = null;
let proxyOriginFetchedAt = 0;

export const resetProxyOriginCache = () => {
    proxyOriginCache = null;
    proxyOriginFetchedAt = 0;
}

const fetchProxyOrigins = async (proxy: any) => {
    const now = Date.now();
    if (proxyOriginCache && now - proxyOriginFetchedAt < 2000) {
        return proxyOriginCache;
    }

    try {
        const data = await proxy.$http.get('/profile/proxy-origins');
        if (data && typeof data === 'object') {
            proxyOriginCache = data as Record<string, string>;
        } else {
            proxyOriginCache = {};
        }
    } catch {
        if (!proxyOriginCache) {
            proxyOriginCache = {};
        }
    }

    proxyOriginFetchedAt = now;
    return proxyOriginCache;
}

const formatDisplayName = (name: string, origin?: string) => {
    if (typeof name !== 'string') {
        return name as any;
    }
    if (!origin) {
        return name;
    }
    const suffix = ` [${origin}]`;
    if (name.endsWith(suffix)) {
        return name.slice(0, -suffix.length).trim();
    }
    return name;
}

const parseOriginFromName = (name: string) => {
    if (typeof name !== 'string') {
        return undefined;
    }
    if (!name.endsWith(']')) {
        return undefined;
    }
    const start = name.lastIndexOf(' [');
    if (start === -1) {
        return undefined;
    }
    const origin = name.slice(start + 2, -1).trim();
    return origin || undefined;
}

// 计算类名
const getClass = (delay: any) => {
    if (delay === 99999) {
        return 'toHidden'
    }

    if (delay <= 300) {
        return 'toLow'
    } else if (delay <= 600) {
        return 'toMiddle'
    } else {
        return 'toHigh'
    }
}

// 获取节点延迟
// testUrl: when independent delay test is enabled, Mihomo stores per-URL results
// in proxy.extra[testUrl].history rather than proxy.history
const getDelay = (proxy: any, testUrl?: string | null) => {
    // Independent mode: prefer extra[testUrl].history (set by Mihomo for custom URLs).
    // Checked BEFORE the alive guard so that manually-triggered test results persist
    // even when Mihomo's internal URLTest interval later marks the group as alive=false
    // (e.g. a nested group whose URLTest selects DIRECT which can't reach the test URL).
    if (testUrl) {
        const extraHistory = proxy['extra']?.[testUrl]?.['history']
        if (Array.isArray(extraHistory) && extraHistory.length > 0) {
            const d = extraHistory[extraHistory.length - 1]['delay']
            return d > 0 ? d : 99999;
        }
    }

    if (!proxy['alive']) {
        return 99999;
    }

    const history = proxy['history']
    if (!history || history.length === 0) {
        return 99999;
    }

    return history[history.length - 1]['delay']
}

const getProxyDelay = (proxy: any, proxiesMap: Record<string, any>, depth = 0, testUrl?: string | null): number => {
    // Ограничение глубины рекурсии на случай циклических ссылок
    if (depth > 5) return getDelay(proxy, testUrl);

    const type = proxy?.['type'];
    const now = proxy?.['now'];
    // Следуем по цепочке только для Smart-групп: бэкенд не вычисляет им задержку,
    // у Selector/URLTest/etc. есть собственная история — используем её напрямую.
    if (type === 'Smart' && typeof now === 'string' && proxiesMap?.[now]) {
        return getProxyDelay(proxiesMap[now], proxiesMap, depth + 1, testUrl);
    }

    return getDelay(proxy, testUrl);
}

const getDisplayType = (proxy: any, fallbackDescription?: string) => {
    const serverDescription = proxy?.['serverDescription']
        ?? proxy?.['server_description']
        ?? proxy?.['server-description']
        ?? proxy?.extra?.['serverDescription']
        ?? proxy?.extra?.['server_description']
        ?? proxy?.extra?.['server-description']
        ?? fallbackDescription;
    if (typeof serverDescription === 'string') {
        const trimmed = serverDescription.trim();
        if (trimmed.length > 0) {
            return trimmed.slice(0, 25);
        }
    }

    return proxy?.['type'];
}

export interface ProxyGroupInfo {
    name: string;
    icon?: string;
    type?: string;
}

export default function createProxiesApi(proxy: any) {
    return {
        // 获取分组延迟
        async getDelay(group: any, url: any, timeout: any) {
            await proxy.$http.get('/group/' + encodeURIComponent(group) + '/delay?timeout=' + timeout + '&url=' + encodeURIComponent(url));
        },
        // 测试单个代理节点延迟 (for independent test: Selector/LoadBalance/Smart groups)
        // Results are stored by Mihomo in proxy.extra[url].history
        async testProxyLatency(proxyName: string, url: string, timeout: number): Promise<void> {
            try {
                await proxy.$http.get('/proxies/' + encodeURIComponent(proxyName) + '/delay?timeout=' + timeout + '&url=' + encodeURIComponent(url));
            } catch {
                // unreachable node — not an error
            }
        },
        // 获取 Smart 分组的权重信息
        async getGroupWeights(name: string): Promise<{ weights: Array<{ Name: string; Rank: string; Weight: number }>; hasData: boolean }> {
            try {
                const data = await proxy.$http.get('/group/' + encodeURIComponent(name) + '/weights');
                const weights = data?.['weights'];
                if (Array.isArray(weights) && weights.length > 0) {
                    return { weights, hasData: true };
                }
                return { weights: [], hasData: false };
            } catch {
                return { weights: [], hasData: false };
            }
        },
        // 获取分组配置的测试URL
        async getGroupTestUrl(name: string): Promise<string | null> {
            try {
                const data = await proxy.$http.get('/proxies/' + encodeURIComponent(name));
                // Mihomo stores the configured test URL in 'testUrl' field for URLTest/Fallback groups
                const url = data?.['testUrl'] || data?.['url'];
                return typeof url === 'string' && url.length > 0 ? url : null;
            } catch {
                return null;
            }
        },
        // 获取分组列表
        async getGroups(): Promise<ProxyGroupInfo[]> {
            // 获取所有节点分组列表
            const data = await proxy.$http.get('/proxies');
            const proxies = data?.['proxies']

            // 判空 — proxies может быть null пока Mihomo загружает pxd-template конфиг
            if (!proxies) {
                return []
            }

            // Sort order from GLOBAL.all (matches zashboard behaviour)
            const sortIndex: string[] = proxies['GLOBAL']?.['all'] ?? []

            // 获取分组
            const proxyGroup: ProxyGroupInfo[] = []
            for (const name in proxies) {
                if (excludeGroupName[name]) {
                    continue
                }
                const group:any = proxies[name]
                if (!includeGroup[group['type']]) {
                    continue
                }
                if (!!group['hidden']) {
                    continue
                }
                proxyGroup.push({
                    name,
                    icon: typeof group['icon'] === 'string' ? group['icon'] : undefined,
                    type: group['type'],
                })
            }

            proxyGroup.sort((prev, next) => {
                const prevIndex = sortIndex.indexOf(prev.name)
                const nextIndex = sortIndex.indexOf(next.name)
                if (prevIndex === -1 && nextIndex === -1) return 0
                if (prevIndex === -1) return 1
                if (nextIndex === -1) return -1
                return prevIndex - nextIndex
            })

            return proxyGroup
        },
        // 获取相应的分组节点列表
        // useIndependentUrl: when true, reads latency from proxy.extra[groupTestUrl].history
        // (the storage Mihomo uses when tested with a custom URL)
        // fallbackTestUrl: used when independentUrl mode is on but group has no configured testUrl
        // (prevents reading from proxy.history which may be contaminated by other groups' tests)
        async getProxies(active: string, isHide: boolean, isSort: boolean, useIndependentUrl = false, overrideTestUrl?: string | null, fallbackTestUrl?: string | null) {
            // 获取所有节点分组列表
            const data = await proxy.$http.get('/proxies')
            const proxies = data?.['proxies']
            let serverDescriptions: Record<string, string> = {}
            try {
                const descriptions = await proxy.$http.get('/profile/serverDescriptions')
                if (descriptions && typeof descriptions === 'object') {
                    serverDescriptions = descriptions
                }
            } catch (e) {
                serverDescriptions = {}
            }

            // 判空 — proxies может быть null пока Mihomo загружает pxd-template конфиг
            if (!proxies?.[active]) {
                return []
            }

            // Determine the test URL this group uses in independent mode.
            // Priority: overrideTestUrl (from user's groupTestUrls config) >
            //           Mihomo's testUrl/url field (for URLTest/Fallback/Smart groups) >
            //           null (fall back to proxy.history)
            const groupData = proxies[active]
            const groupTestUrl: string | null = useIndependentUrl
                ? (overrideTestUrl || groupData?.['testUrl'] || groupData?.['url'] || fallbackTestUrl || null)
                : null

            // 获取分组节点列表
            const originMap = await fetchProxyOrigins(proxy);
            const hasOriginMap = originMap && Object.keys(originMap).length > 0;

            const proxiesNames: string[] = proxies[active]['all']
            const nowName = proxies[active]['now']

            // Build all nodes preserving the original order from proxies[active]['all']
            const allProxies = []
            for (const name of proxiesNames) {
                const proxyNode = proxies[name]
                if (!proxyNode) continue
                const type = proxyNode['type'];
                const displayType = getDisplayType(proxyNode, serverDescriptions[name]);
                const icon = typeof proxyNode?.['icon'] === 'string' ? proxyNode['icon'] : undefined;
                const delay = getProxyDelay(proxyNode, proxies, 0, groupTestUrl)
                let origin = originMap ? originMap[name] : undefined;
                if (!origin && hasOriginMap) {
                    origin = parseOriginFromName(name);
                }
                const displayName = formatDisplayName(name, origin);
                allProxies.push({
                    name,
                    type,
                    displayType,
                    icon,
                    delay,
                    now: name === nowName,
                    toClass: getClass(delay),
                    displayName,
                    origin,
                })
            }

            // Apply hide filter (remove unreachable nodes when isHide is on)
            let visibleProxies = isHide
                ? allProxies.filter(p => p.delay !== 99999)
                : allProxies

            // Sort by latency when requested; otherwise original order is already preserved
            if (isSort) {
                visibleProxies = [...visibleProxies].sort((a, b) => {
                    if (a.delay !== b.delay) return a.delay - b.delay
                    return proxiesNames.indexOf(a.name) - proxiesNames.indexOf(b.name)
                })
            }

            return visibleProxies;
        },
        // 设置代理
        async setProxy(group: any, name: any) {
            await proxy.$http.put("/proxies/" + group, name);
        },
    };
}
