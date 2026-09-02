// 获取Mihomo
const getMihomo = (proxy: any) => async function () {
    return await proxy.$http.get('/mihomo');
}

// 更新Mihomo
const updateMihomo = (proxy: any) => async function (configs: any) {
    return await proxy.$http.put('/mihomo', configs);
}

// 等待 Mihomo 切换完成
//
// Polls until px's control API answers, so callers that re-assert OS-level
// state against it (the system proxy and TUN on startup) do not fire at a
// backend that is not serving yet.
//
// This used to GET '/wait', a route the backend does not have: it 404'd, the
// error was swallowed, and every `await api.waitRunning()` resolved instantly —
// a gate in name only. '/version' is the cheapest route that actually exists.
//
// It must still NEVER break its callers — switchProfile() awaits it right after
// the switch, and a thrown error there used to abort the rest (the local
// `selected` update that drives the active-card colour, and webStore.fProfile
// that drives the profile-logo / header-title). Hence: resolve with null on
// timeout rather than reject, and keep the timeout short enough that a genuinely
// dead backend cannot stall the UI for long.
const waitRunning = (proxy: any) => async function (timeoutMs: number = 5000) {
    const deadline = Date.now() + timeoutMs;
    for (; ;) {
        try {
            return await proxy.$http.get('/version');
        } catch {
            if (Date.now() >= deadline) {
                return null;
            }
            await new Promise(resolve => setTimeout(resolve, 200));
        }
    }
}

// 获取Mihomo
const getAdmin = (proxy: any) => async function () {
    return await proxy.$http.get('/mihomo/admin');
}


export default function createMihomoApi(proxy: any) {
    return {
        getMihomo: getMihomo(proxy),
        updateMihomo: updateMihomo(proxy),
        waitRunning: waitRunning(proxy),
        getAdmin: getAdmin(proxy),
    }
}