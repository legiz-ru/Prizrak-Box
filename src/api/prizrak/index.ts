// 开启代理
const enableProxy = (proxy: any) => async function (configs: any) {
    return await proxy.$http.put('/prizrak/enableProxy', configs);
}

// 关闭代理
const disableProxy = (proxy: any) => async function () {
    return await proxy.$http.get('/prizrak/disableProxy');
}

// 检测地址端口是否可用
const checkAddressPort = (proxy: any) => async function (configs: any) {
    return await proxy.$http.put('/prizrak/checkAddressPort', configs);
}

// 获取配置文件目录
const configDir = (proxy: any) => async function () {
    return await proxy.$http.get('/prizrak/configDir');
}

// 退出Px
const exit = (proxy: any) => async function () {
    return await proxy.$http.get('/prizrak/exit');
}

// 获取设置
const getSettings = (proxy: any) => async function () {
    return await proxy.$http.get('/prizrak/settings');
}

// 设置HWID开关
const setHWIDSetting = (proxy: any) => async function (hwid: boolean) {
    return await proxy.$http.put('/prizrak/settings/hwid', { hwid });
}

export default function createPrizrakApi(proxy: any) {
    return {
        enableProxy: enableProxy(proxy),
        disableProxy: disableProxy(proxy),
        checkAddressPort: checkAddressPort(proxy),
        configDir: configDir(proxy),
        exit: exit(proxy),
        getSettings: getSettings(proxy),
        setHWIDSetting: setHWIDSetting(proxy),
    }
}