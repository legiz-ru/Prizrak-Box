import {defineStore} from 'pinia';

interface DeepLinkImportState {
    isImporting: boolean;
    message: string;
    cancelLabel: string;
    cancelHandler: (() => void) | null;
}

interface StartImportPayload {
    message: string;
    cancelLabel?: string;
    onCancel?: () => void;
}

export const useDeepLinkImportStore = defineStore('deepLinkImport', {
    state: (): DeepLinkImportState => ({
        isImporting: false,
        message: '',
        cancelLabel: '',
        cancelHandler: null,
    }),
    actions: {
        startImport(payload: StartImportPayload) {
            this.isImporting = true;
            this.message = payload.message;
            this.cancelLabel = payload.cancelLabel ?? '';
            this.cancelHandler = payload.onCancel ?? null;
        },
        finishImport() {
            this.isImporting = false;
            this.message = '';
            this.cancelLabel = '';
            this.cancelHandler = null;
        },
        cancelImport() {
            if (!this.isImporting) {
                return;
            }
            const handler = this.cancelHandler;
            this.finishImport();
            if (handler) {
                handler();
            }
        },
    },
});
