// 获取Mihomo
const getMihomo = (proxy: any) => async function () {
    return await proxy.$http.get('/mihomo');
}

// 更新Mihomo
const updateMihomo = (proxy: any) => async function (configs: any) {
    return await proxy.$http.put('/mihomo', configs);
}

// 等待 Mihomo 切换完成
// NOTE: the backend has no '/wait' route, so this GET 404s and axios rejects.
// It must NEVER break its callers — switchProfile() awaits it right after the
// switch, and a thrown error here used to abort the rest (the local `selected`
// update that drives the active-card colour and webStore.fProfile that drives
// the profile-logo / header-title). Swallow any error and resolve.
const waitRunning = (proxy: any) => async function () {
    try {
        return await proxy.$http.get('/wait');
    } catch {
        return null;
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