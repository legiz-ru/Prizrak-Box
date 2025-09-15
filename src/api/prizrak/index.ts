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

// 更新HTTP客户端配置
const updateHTTPClientConfig = (proxy: any) => async function (config: any) {
    return await proxy.$http.put('/prizrak/httpClientConfig', config);
}

export default function createPrizrakApi(proxy: any) {
    return {
        enableProxy: enableProxy(proxy),
        disableProxy: disableProxy(proxy),
        checkAddressPort: checkAddressPort(proxy),
        configDir: configDir(proxy),
        exit: exit(proxy),
        updateHTTPClientConfig: updateHTTPClientConfig(proxy),
    }
}