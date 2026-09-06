// @ts-nocheck

import {clipboard, contextBridge, ipcRenderer, shell} from 'electron';
import os from 'os';

// tray相关
contextBridge.exposeInMainWorld('pxTray', {
    on: (name, callback) => {
        const eventName = 'px_' + name;
        // 移除旧监听器，确保只注册一次
        ipcRenderer.removeAllListeners(eventName);
        ipcRenderer.on(eventName, (_event, ...value) => callback(...value));
    },
    emit: (name, ...value) => ipcRenderer.send('px_' + name, ...value)
});


// 深度链接相关
contextBridge.exposeInMainWorld('pxDeepLink', {
    onImportProfile: (callback) => {
        ipcRenderer.removeAllListeners('import-profile-from-deeplink');
        ipcRenderer.on('import-profile-from-deeplink', (_event, data) => callback(data));
    },
    notifyReady: () => {
        ipcRenderer.send('deeplink-handler-ready');
    }
});


// 缓存接口
contextBridge.exposeInMainWorld('pxStore', {
    get: (key) => ipcRenderer.invoke('store:get', key),
    set: (key, value) => ipcRenderer.invoke('store:set', key, value)
});

// Кэш фонового изображения (файл userData/px-bg-cache.json, не зависит от порта)
contextBridge.exposeInMainWorld('pxBgCache', {
    read: () => ipcRenderer.invoke('bgcache:read'),
    write: (forBg: string, dataUrl: string) => ipcRenderer.invoke('bgcache:write', forBg, dataUrl),
    clear: () => ipcRenderer.invoke('bgcache:clear'),
});

// 获取系统信息
contextBridge.exposeInMainWorld('pxOs', () => {
    switch (os.type()) {
        case 'Darwin':
            return "MacOS " + os.arch()
        case 'Linux':
            return "Linux " + os.arch()
        case 'Windows_NT':
            return "Windows " + os.arch()
        default:
            return "Unknown";
    }
});

// Получить имя текущего пользователя
contextBridge.exposeInMainWorld('pxUsername', () => {
    try {
        return os.userInfo().username;
    } catch (e) {
        console.error('Failed to get username:', e);
        return '';
    }
});

// 打开配置目录
contextBridge.exposeInMainWorld('pxConfigDir', (url: string) => shell.openPath(url));

// 获取剪贴板内容
contextBridge.exposeInMainWorld('pxClipboard', () => clipboard.readText());

// 打开外部URL地址
contextBridge.exposeInMainWorld('pxOpen', (url: string) => shell.openExternal(url));

// Показать файл в проводнике
contextBridge.exposeInMainWorld('pxShowInFolder', (path: string) => shell.showItemInFolder(path));

// 控制标题栏
if (process.platform !== 'darwin') {
    contextBridge.exposeInMainWorld('pxShowBar', () => console.log('pxShowBar'));
}

// Service API для управления сервисом TUN
contextBridge.exposeInMainWorld('pxService', {
    getStatus: () => ipcRenderer.invoke('service:getStatus'),
    install: () => ipcRenderer.invoke('service:install'),
    uninstall: () => ipcRenderer.invoke('service:uninstall'),
    isRunning: () => ipcRenderer.invoke('service:isRunning'),
    restartBackend: () => ipcRenderer.invoke('service:restartBackend'),
    showInstallDialog: () => ipcRenderer.invoke('service:showInstallDialog'),
});

// Получить текущую директорию конфигурации
contextBridge.exposeInMainWorld('pxPreConfigDir', () => ipcRenderer.invoke('pre-config-dir'));

// Изменить директорию конфигурации
contextBridge.exposeInMainWorld('pxChangeConfigDir', (dir: string) => ipcRenderer.invoke('change-config-dir', dir));

// Добавить electron объект для вызова IPC
contextBridge.exposeInMainWorld('electron', {
    invoke: (channel: string, ...args: any[]) => ipcRenderer.invoke(channel, ...args)
});
