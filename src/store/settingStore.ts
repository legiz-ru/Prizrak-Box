import {defineStore} from 'pinia';
import {defaultPersist} from "@/types/persist";

export const useSettingStore = defineStore('setting', {
    state: () => ({
        testUrl: 'https://www.google.com/blank.html',
        port: 12345,
        bindAddress: "127.0.0.1",
        stack: 'Mixed',
        ipv6: false,
        dns: false,
        startup: false,
        startMinimized: false,
        auth: false,
        hwid: true,
        systemProxyMode: true,
        multiProfileEnabled: false,
        multiProfileHintShown: false,
        hwidHeaders: {
            hwid: '',
            os: '',
            osVersion: '',
            model: '',
        },
        sc_switch: false,
        sc_switch_key: 'Ctrl+Shift+X',
        independentDelayTest: true,
        groupTestUrls: [] as Array<{name: string; url: string}>,
    }),
    actions: {
        setTestUrl(testUrl: any) {
            this.testUrl = testUrl;
        },
        setPort(port: any) {
            this.port = Number(port);
        },
        setStack(stack: any) {
            this.stack = stack;
        },
        setIpv6(ipv6: any) {
            this.ipv6 = ipv6;
        },
        setDns(dns: any) {
            this.dns = dns;
        },
        setStartup(startup: any) {
            this.startup = startup;
        },
        setStartMinimized(startMinimized: any) {
            this.startMinimized = startMinimized;
        },
        setBindAddress(bindAddress: any) {
            this.bindAddress = bindAddress;
        },
        setAuth(auth: any) {
            this.auth = auth;
        },
        setHwid(hwid: any) {
            this.hwid = hwid;
        },
        setSystemProxyMode(systemProxyMode: any) {
            this.systemProxyMode = systemProxyMode;
        },
        setMultiProfileEnabled(enabled: any) {
            this.multiProfileEnabled = !!enabled;
        },
        setMultiProfileHintShown(shown: any) {
            this.multiProfileHintShown = !!shown;
        },
        setHwidHeaders(headers: { hwid?: string; os?: string; osVersion?: string; model?: string }) {
            this.hwidHeaders = {
                hwid: headers?.hwid ?? this.hwidHeaders.hwid,
                os: headers?.os ?? this.hwidHeaders.os,
                osVersion: headers?.osVersion ?? this.hwidHeaders.osVersion,
                model: headers?.model ?? this.hwidHeaders.model,
            };
        },
        setScSwitch(val: boolean) {
            this.sc_switch = val;
        },
        setScSwitchKey(val: string) {
            this.sc_switch_key = val;
        },
        setIndependentDelayTest(val: boolean) {
            this.independentDelayTest = val;
        },
        setGroupTestUrls(val: Array<{name: string; url: string}>) {
            this.groupTestUrls = val;
        },
    },
    persist: defaultPersist,
});
