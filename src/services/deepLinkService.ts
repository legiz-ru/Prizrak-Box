import { Router } from 'vue-router';
import { AxiosRequest } from '@/util/axiosRequest';
import { Profile } from '@/types/profile';
import { ElMessage } from 'element-plus';

let httpInstance: AxiosRequest;
let routerInstance: Router;

// 检查URL是否是HTTP或HTTPS格式
function isHttpOrHttps(url: string): boolean {
    return /^https?:\/\//.test(url);
}

// 导入配置的核心函数
async function importProfile(url: string): Promise<boolean> {
    if (!url) {
        ElMessage.error('Invalid URL: URL is empty');
        return false;
    }

    // 验证URL格式
    if (!isHttpOrHttps(url)) {
        ElMessage.error('Invalid URL format: URL must start with http:// or https://');
        return false;
    }

    try {
        const profile = new Profile();
        profile.content = url;
        
        // 显示加载消息
        ElMessage.info('Importing profile...');
        
        // 调用API导入配置
        const pList = await httpInstance.post('/profile', profile);
        
        if (pList && pList.length > 0) {
            ElMessage.success('Profile imported successfully!');
            
            // 导航到 Profiles 页面以显示导入的配置
            if (routerInstance.currentRoute.value.path !== '/Profiles') {
                await routerInstance.push('/Profiles');
            }
            
            return true;
        } else {
            ElMessage.error('Failed to import profile: No profiles returned');
            return false;
        }
    } catch (error: any) {
        const errorMessage = error?.message || 'Failed to import profile';
        ElMessage.error(errorMessage);
        console.error('Profile import error:', error);
        return false;
    }
}

// 设置深度链接服务
export function setupDeepLinkService(http: AxiosRequest, router: Router) {
    httpInstance = http;
    routerInstance = router;
    
    // 设置深度链接监听器
    // @ts-ignore
    if (window.pxDeepLink) {
        // @ts-ignore
        window.pxDeepLink.onImportProfile(async (data: { url: string }) => {
            console.log('Received deeplink import request:', data);
            await importProfile(data.url);
        });
    }
}

// 导出导入函数供其他组件使用
export { importProfile };