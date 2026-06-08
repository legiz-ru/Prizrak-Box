import {AxiosRequestConfig} from "axios";
import {Profile, ProfileSelectionPayload} from "@/types/profile";

export interface HwidStatus {
    hwidNotSupported: boolean;
    hwidMaxDevicesReached: boolean;
    supportUrl: string;
}

export type ProfileRefreshResult = Profile & {
    hwidNotSupported?: boolean;
    hwidMaxDevicesReached?: boolean;
}

function parseHwidFromError(error: any): HwidStatus | null {
    if (!error || typeof error !== 'object') return null;
    if (error.hwidNotSupported || error.hwidMaxDevicesReached) {
        return {
            hwidNotSupported: !!error.hwidNotSupported,
            hwidMaxDevicesReached: !!error.hwidMaxDevicesReached,
            supportUrl: typeof error.supportUrl === 'string' ? error.supportUrl : '',
        };
    }
    return null;
}

// 添加配置从input — при HWID-ошибке бросает { hwidNotSupported?, hwidMaxDevicesReached?, supportUrl? }
const addProfileFromInput = (proxy: any) => async function (profile: Profile, config?: AxiosRequestConfig): Promise<Profile[]> {
    return await proxy.$http.post('/profile', profile, config);
}

// 添加配置从文件
const addProfileFromFile = (proxy: any) => async function (profile: Profile) {
    return await proxy.$http.post('/profile/file', profile);
}

// 删除配置
const deleteProfile = (proxy: any) => async function (profile: Profile) {
    return await proxy.$http.post('/profile/delete', profile);
}

// 修改配置
const updateProfile = (proxy: any) => async function (profile: Profile) {
    return await proxy.$http.put('/profile', profile);
}

// 获取配置列表
const getProfileList = (proxy: any) => async function (): Promise<Profile[]> {
    return await proxy.$http.get('/profile');
}

// 刷新配置 — возвращает ProfileRefreshResult с транзитными HWID-полями
const refreshProfile = (proxy: any) => async function (profile: Profile): Promise<ProfileRefreshResult> {
    return await proxy.$http.put('/profile/refresh', profile);
}

// 切换配置
const switchProfile = (proxy: any) => async function (payload: Profile | ProfileSelectionPayload) {
    return await proxy.$http.patch('/profile', payload);
}


export { parseHwidFromError };

export default function createProfilesApi(proxy: any) {
    return {
        addProfileFromInput: addProfileFromInput(proxy),
        addProfileFromFile: addProfileFromFile(proxy),
        deleteProfile: deleteProfile(proxy),
        updateProfile: updateProfile(proxy),
        getProfileList: getProfileList(proxy),
        refreshProfile: refreshProfile(proxy),
        switchProfile: switchProfile(proxy),
    }
}
