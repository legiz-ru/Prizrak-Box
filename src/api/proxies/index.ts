// 排除的分组名字
const excludeGroupName: any = {
    DIRECT: true,
    REJECT: true,
    "REJECT-DROP": true,
    PASS: true,
    COMPATIBLE: true
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
const getDelay = (proxy: any) => {
    if (!proxy['alive']) {
        return 99999;
    }

    const history = proxy['history']
    if (!history || history.length === 0) {
        return 99999;
    }

    return history[history.length - 1]['delay']
}

const getProxyDelay = (proxy: any, proxiesMap: Record<string, any>) => {
    const type = proxy?.['type'];
    const now = proxy?.['now'];
    if (includeGroup[type] && typeof now === 'string' && proxiesMap?.[now]) {
        return getDelay(proxiesMap[now]);
    }

    return getDelay(proxy);
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
}

export default function createProxiesApi(proxy: any) {
    return {
        // 获取分组延迟
        async getDelay(group: any, url: any, timeout: any) {
            await proxy.$http.get('/group/' + group + '/delay?timeout=' + timeout + "&url=" + url);
        },
        // 获取分组列表
        async getGroups(): Promise<ProxyGroupInfo[]> {
            // 获取所有节点分组列表
            const data = await proxy.$http.get('/proxies');
            const proxies = data['proxies']

            // 判空
            if (!proxies['GLOBAL']) {
                return []
            }

            // 获取分组
            const proxyGroup: ProxyGroupInfo[] = []
            for (const name of proxies['GLOBAL']['all']) {
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
                })
            }

            return proxyGroup
        },
        // 获取相应的分组节点列表
        async getProxies(active: string, isHide: boolean, isSort: boolean) {
            // 获取所有节点分组列表
            const data = await proxy.$http.get('/proxies')
            const proxies = data['proxies']
            let serverDescriptions: Record<string, string> = {}
            try {
                const descriptions = await proxy.$http.get('/profile/serverDescriptions')
                if (descriptions && typeof descriptions === 'object') {
                    serverDescriptions = descriptions
                }
            } catch (e) {
                serverDescriptions = {}
            }

            // 判空
            if (!proxies[active]) {
                return []
            }

            // 获取分组节点列表
            const originMap = await fetchProxyOrigins(proxy);
            const hasOriginMap = originMap && Object.keys(originMap).length > 0;

            const proxiesNames = proxies[active]['all']
            const nowName = proxies[active]['now']

            // 获取节点延迟
            const activeProxies = []
            const inProxies = []
            for (const name of proxiesNames) {
                const proxy = proxies[name]
                const type = proxy['type'];
                const displayType = getDisplayType(proxy, serverDescriptions[name]);
                const icon = typeof proxy?.['icon'] === 'string' ? proxy['icon'] : undefined;
                const delay = getProxyDelay(proxy, proxies)
                let origin = originMap ? originMap[name] : undefined;
                if (!origin && hasOriginMap) {
                    origin = parseOriginFromName(name);
                }
                const displayName = formatDisplayName(name, origin);
                if (includeProxy[type]) {
                    inProxies.push({
                        name,
                        type,
                        displayType,
                        icon,
                        delay: delay,
                        now: name === nowName,
                        toClass: getClass(delay),
                        displayName,
                        origin,
                    })
                } else {
                    activeProxies.push({
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
            }

            // 获取显示的节点
            let showProxies = []
            if (isHide) {
                for (const proxy of activeProxies) {
                    if (proxy['delay'] != 99999) {
                        showProxies.push(proxy)
                    }
                }
            } else {
                showProxies = activeProxies
            }

            // 构建哈希表
            const GLOBAL = proxies['GLOBAL']['all'];
            const map = new Map();
            GLOBAL.forEach((value: any, index: any) => {
                map.set(value, index);
            });

            // 进行排序
            if (isSort) {
                inProxies.sort((obj1, obj2) => {
                    if (obj1.delay != obj2.delay) {
                        return obj1.delay - obj2.delay
                    }

                    return map.get(obj1.name) - map.get(obj2.name)
                });
                showProxies.sort((obj1, obj2) => obj1.delay - obj2.delay);
            } else {
                showProxies.sort((obj1, obj2) => map.get(obj1.name) - map.get(obj2.name));
                inProxies.sort((obj1, obj2) => map.get(obj1.name) - map.get(obj2.name));
            }

            return inProxies.concat(showProxies);
        },
        // 设置代理
        async setProxy(group: any, name: any) {
            await proxy.$http.put("/proxies/" + group, name);
        },
    };
}
