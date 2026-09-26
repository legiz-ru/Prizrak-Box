import {defineStore} from 'pinia';

export interface CustomDashboard {
    name: string;
    url: string;
}

export const useWebStore = defineStore('web', {
    state: () => ({
        host: '127.0.0.1', // 默认值
        port: '9686',       // 默认端口
        secret: 'Y8IUaPeFLTRvsrdf2mUJkLMBuphVZRE5',         // 默认密钥
        logs: [] as any[],         // 日志
        dnd: false,         // 拖拽显示
        dProfile: [] as any[],         // 传输文件 拖拽添加文件用
        fProfile: {} as Record<string, any>, // 更新profile 配置切换用
        profileList: [] as any[], // кэш списка профилей для мгновенного показа при навигации
        customDashboards: [] as CustomDashboard[],
    }),
    getters: {
        // 确保使用 state 参数引用正确
        baseUrl: (state) => `http://${state.host}:${state.port}`,
        wsUrl: (state) => `ws://${state.host}:${state.port}`,
    },
    actions: {
        setHost(host: string) {
            if (host) this.host = host;
        },
        setPort(port: string) {
            if (port) this.port = port;
        },
        setSecret(secret: string) {
            if (secret) this.secret = secret;
        },
        addLog(log: any) {
            // Oldest first; keep the most recent 1000 entries (also persisted).
            this.logs.push(log);
            if (this.logs.length > 1000) {
                this.logs.splice(0, this.logs.length - 1000);
            }
        },
        clearLogs() {
            this.logs = [];
        },
        addCustomDashboard(dashboard: CustomDashboard) {
            this.customDashboards.push(dashboard);
        },
        updateCustomDashboard(index: number, dashboard: CustomDashboard) {
            if (index < 0 || index >= this.customDashboards.length) {
                return;
            }

            this.customDashboards.splice(index, 1, dashboard);
        },
        removeCustomDashboard(index: number) {
            if (index < 0 || index >= this.customDashboards.length) {
                return;
            }

            this.customDashboards.splice(index, 1);
        },
    },
    persist: true,
});
