import {defineStore} from 'pinia';
import {defaultPersist} from '@/types/persist';
import packageInfo from '../../package.json';
import {compareVersions, resolveVersionLabel} from '@/util/version';

const GITHUB_LATEST_RELEASE_API = 'https://api.github.com/repos/legiz-ru/Prizrak-Box/releases/latest';

type CheckStatus = 'idle' | 'checking' | 'update-available' | 'up-to-date' | 'error';

interface UpdateState {
    currentVersion: string;
    latestTag: string | null;
    latestName: string | null;
    latestUrl: string | null;
    publishedAt: string | null;
    lastChecked: number | null;
    updateAvailable: boolean;
    dismissedTag: string | null;
    notifiedTag: string | null;
    checking: boolean;
    lastCheckStatus: CheckStatus;
    lastError: string | null;
}

export const useUpdateStore = defineStore('updates', {
    state: (): UpdateState => ({
        currentVersion: typeof packageInfo.version === 'string' ? packageInfo.version : '',
        latestTag: null,
        latestName: null,
        latestUrl: null,
        publishedAt: null,
        lastChecked: null,
        updateAvailable: false,
        dismissedTag: null,
        notifiedTag: null,
        checking: false,
        lastCheckStatus: 'idle',
        lastError: null,
    }),
    getters: {
        hasVisibleUpdate(state): boolean {
            return state.updateAvailable && !!state.latestTag && state.latestTag !== state.dismissedTag;
        },
        latestDisplayName(state): string {
            return resolveVersionLabel(state.latestTag, state.latestName);
        },
        shouldNotify(state): boolean {
            return state.updateAvailable
                && !!state.latestTag
                && state.latestTag !== state.dismissedTag
                && state.latestTag !== state.notifiedTag;
        },
    },
    actions: {
        async checkForUpdates(): Promise<boolean> {
            if (this.checking) {
                return this.updateAvailable;
            }

            const previousTag = this.latestTag;
            this.checking = true;
            this.lastError = null;
            this.lastCheckStatus = 'checking';

            try {
                const response = await fetch(GITHUB_LATEST_RELEASE_API, {
                    headers: {
                        'Accept': 'application/vnd.github+json',
                        'User-Agent': 'Prizrak-Box',
                    },
                });

                if (!response.ok) {
                    throw new Error(`GitHub responded with status ${response.status}`);
                }

                const data = await response.json();
                const tagName = typeof data?.tag_name === 'string' ? data.tag_name : '';
                const htmlUrl = typeof data?.html_url === 'string' ? data.html_url : '';
                const name = typeof data?.name === 'string' ? data.name : '';
                const publishedAt = typeof data?.published_at === 'string' ? data.published_at : null;

                this.latestTag = tagName || null;
                this.latestName = name || null;
                this.latestUrl = htmlUrl || null;
                this.publishedAt = publishedAt;
                this.lastChecked = Date.now();

                const hasTag = !!tagName;
                const isNewer = hasTag && compareVersions(tagName, this.currentVersion) > 0;
                this.updateAvailable = Boolean(isNewer);

                if (!this.updateAvailable) {
                    this.dismissedTag = null;
                    this.notifiedTag = null;
                } else if (previousTag !== this.latestTag) {
                    this.dismissedTag = null;
                    this.notifiedTag = null;
                }

                this.lastCheckStatus = this.updateAvailable ? 'update-available' : 'up-to-date';
            } catch (error: any) {
                this.lastCheckStatus = 'error';
                this.lastError = typeof error?.message === 'string' ? error.message : String(error);
            } finally {
                this.checking = false;
            }

            return this.updateAvailable;
        },
        dismissCurrentUpdate() {
            if (this.latestTag) {
                this.dismissedTag = this.latestTag;
            }
        },
        markNotified() {
            if (this.latestTag) {
                this.notifiedTag = this.latestTag;
            }
        },
        resetNotificationState() {
            this.notifiedTag = null;
            this.dismissedTag = null;
        },
        clearStatus() {
            this.lastCheckStatus = 'idle';
            this.lastError = null;
        },
    },
    persist: defaultPersist,
});
