import { Profile } from "@/types/profile";
import createApi from "@/api";
import { pError, pLoad, pSuccess } from "@/util/pLoad";
import { isHttpOrHttps } from "@/util/format";
import { useWebStore } from "@/store/webStore";
import { getCurrentInstance, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import router from "@/router";

export class DeepLinkService {
    private static instance: DeepLinkService | null = null;
    private api: any = null;
    private t: any = null;

    private constructor() {
        // Initialize I18n in constructor if possible
        this.trySetupI18n();
    }

    static getInstance(): DeepLinkService {
        if (!DeepLinkService.instance) {
            DeepLinkService.instance = new DeepLinkService();
        }
        return DeepLinkService.instance;
    }

    private trySetupI18n() {
        try {
            const { t } = useI18n();
            this.t = t;
        } catch (error) {
            // Will be set up later when Vue instance is available
            this.t = null;
        }
    }

    private getTranslation(key: string, fallback: string): string {
        if (!this.t) {
            this.trySetupI18n();
        }
        return this.t ? this.t(key) : fallback;
    }

    private async initializeApi(): Promise<any> {
        if (this.api) {
            return this.api;
        }

        // Wait for Vue instance to be available
        let attempts = 0;
        while (attempts < 20) { // Max 10 seconds
            try {
                const instance = getCurrentInstance();
                if (instance && instance.proxy) {
                    this.api = createApi(instance.proxy);
                    return this.api;
                }
            } catch (error) {
                // Vue instance not ready yet
            }
            
            await new Promise(resolve => setTimeout(resolve, 500));
            attempts++;
        }
        
        throw new Error('Failed to initialize API - Vue instance not available');
    }

    async handleImportProfile(url: string): Promise<boolean> {
        if (!url) {
            pError(this.getTranslation('profiles.deeplink.invalid-url', 'Invalid URL'));
            return false;
        }

        // 验证URL格式
        if (!isHttpOrHttps(url)) {
            pError(this.getTranslation('profiles.deeplink.invalid-url-format', 'Invalid URL format'));
            return false;
        }

        try {
            const api = await this.initializeApi();
            
            const p = new Profile();
            p.content = url;

            await pLoad(this.getTranslation('profiles.deeplink.importing', 'Importing profile...'), async () => {
                const pList = await api.addProfileFromInput(p);
                if (pList && pList.length > 0) {
                    // Navigate to profiles page to show the imported profile
                    await router.push('/Profiles');
                    pSuccess(this.getTranslation('profiles.deeplink.import-success', 'Profile imported successfully'));
                }
            });
            return true;
        } catch (e) {
            // If API is not available, store for later processing
            if (e instanceof Error && e.message.includes('Vue instance not available')) {
                const webStore = useWebStore();
                webStore.setPendingDeepLinkUrl(url);
                
                // Navigate to profiles page where it will be processed
                await router.push('/Profiles');
                return true;
            } else {
                const errorMessage = e instanceof Error ? e.message : this.getTranslation('profiles.deeplink.import-failed', 'Import failed');
                pError(errorMessage);
                return false;
            }
        }
    }

    async processPendingDeepLink(): Promise<void> {
        const webStore = useWebStore();
        if (webStore.pendingDeepLinkUrl) {
            const url = webStore.pendingDeepLinkUrl;
            webStore.setPendingDeepLinkUrl(null);
            await this.handleImportProfile(url);
        }
    }
}

export default DeepLinkService;