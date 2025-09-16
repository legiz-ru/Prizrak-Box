import { Router } from 'vue-router';
import { AxiosRequest } from '@/util/axiosRequest';
import { Profile } from '@/types/profile';
import { ElMessage } from 'element-plus';

let httpInstance: AxiosRequest;
let routerInstance: Router;
let isServiceReady = false;

// 检查URL是否是HTTP或HTTPS格式
function isHttpOrHttps(url: string): boolean {
    return /^https?:\/\//.test(url);
}

// 导入配置的核心函数
async function importProfile(url: string): Promise<boolean> {
    console.log('importProfile called with URL:', url);
    
    if (!isServiceReady) {
        console.error('Deeplink service not ready');
        ElMessage.error('Deeplink service not initialized');
        return false;
    }
    
    if (!url) {
        console.error('Invalid URL: URL is empty');
        ElMessage.error('Invalid URL: URL is empty');
        return false;
    }

    // 验证URL格式
    if (!isHttpOrHttps(url)) {
        console.error('Invalid URL format:', url);
        ElMessage.error('Invalid URL format: URL must start with http:// or https://');
        return false;
    }

    try {
        const profile = new Profile();
        profile.content = url;
        
        console.log('Creating profile with content:', profile.content);
        
        // 显示加载消息
        ElMessage.info('Importing profile...');
        
        // 调用API导入配置
        console.log('Calling API to import profile');
        const pList = await httpInstance.post('/profile', profile);
        console.log('API response:', pList);
        
        if (pList && pList.length > 0) {
            ElMessage.success('Profile imported successfully!');
            
            // 导航到 Profiles 页面以显示导入的配置
            if (routerInstance.currentRoute.value.path !== '/Profiles') {
                console.log('Navigating to Profiles page');
                await routerInstance.push('/Profiles');
            }
            
            return true;
        } else {
            console.error('No profiles returned from API');
            ElMessage.error('Failed to import profile: No profiles returned');
            return false;
        }
    } catch (error: any) {
        const errorMessage = error?.message || 'Failed to import profile';
        console.error('Profile import error:', error);
        ElMessage.error(errorMessage);
        return false;
    }
}

// 设置深度链接服务
export function setupDeepLinkService(http: AxiosRequest, router: Router) {
    console.log('Setting up deeplink service...');
    
    httpInstance = http;
    routerInstance = router;
    
    // 检查依赖是否正确设置
    if (!http) {
        console.error('HTTP instance not provided to deeplink service');
        return;
    }
    
    if (!router) {
        console.error('Router instance not provided to deeplink service');
        return;
    }
    
    // 检查 window.pxDeepLink 是否可用
    // @ts-ignore
    if (!window.pxDeepLink) {
        console.error('window.pxDeepLink not available');
        
        // 延迟重试设置
        setTimeout(() => {
            console.log('Retrying deeplink service setup...');
            setupDeepLinkService(http, router);
        }, 1000);
        return;
    }
    
    try {
        // 设置深度链接监听器
        // @ts-ignore
        window.pxDeepLink.onImportProfile(async (data: { url: string }) => {
            console.log('Received deeplink import request:', data);
            
            if (!data || !data.url) {
                console.error('Invalid deeplink data received:', data);
                ElMessage.error('Invalid deeplink data received');
                return;
            }
            
            await importProfile(data.url);
        });
        
        isServiceReady = true;
        console.log('Deeplink service setup completed successfully');
        
    } catch (error) {
        console.error('Error setting up deeplink service:', error);
        isServiceReady = false;
    }
}

// 导出导入函数供其他组件使用
export { importProfile };