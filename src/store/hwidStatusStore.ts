import {defineStore} from "pinia";

type HwidErrorType = 'not-supported' | 'max-devices-reached' | null;

export const useHwidStatusStore = defineStore('hwidStatus', {
    state: () => ({
        errorType: null as HwidErrorType,
        supportUrl: '',
    }),
    actions: {
        showNotSupported() {
            this.errorType = 'not-supported';
            this.supportUrl = '';
        },
        showMaxDevicesReached(supportUrl: string = '') {
            this.errorType = 'max-devices-reached';
            this.supportUrl = supportUrl;
        },
        clear() {
            this.errorType = null;
            this.supportUrl = '';
        },
    },
});
