import {AxiosRequest} from "@/util/axiosRequest";

// 为 '@/api' 模块提供类型声明
declare module '@/api' {
    // 定义一个类型接口，替代 `any`，根据项目实际的 API 结构定义
    export interface Api {
        proxies: () => Promise<any>;
    }
}

// 为 Vue 的全局属性添加类型声明
declare module '@vue/runtime-core' {
    export interface ComponentCustomProperties {
        $http: AxiosRequest; // 声明全局 $http 的类型
        $t: (key: string, values?: Record<string, unknown>) => string; // i18n
    }
}

// 绑定函数
declare global {
    /** Prizrak-Core version from src-go/go.mod, injected by vite.config.ts. */
    const __CORE_VERSION__: string;

    interface Window {
        pxOs: () => string;
        /** Set by src/wails-shim.ts when running in the Wails shell. */
        pxIsWails?: boolean;
        pxDeepLink?: {
            onImportProfile: (callback: (data: { rawUrl?: string; url?: string; name?: string } | string) => void) => void;
            notifyReady?: () => void | Promise<void>;
        };
    }
}

