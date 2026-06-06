// @ts-nocheck

import {app, BrowserWindow, ipcMain, Menu, nativeImage, Tray, shell} from 'electron';
import path from "node:path";
import {storeSet} from "./store";
import {disableAutoLaunch, enableAutoLaunch} from "./launch";

// 是否在开发模式
const isDev = !app.isPackaged;

// Matches flag pairs AND all emoji blocks up to U+1FAFF (includes 🪢🪆🥷 etc.)
const EMOJI_REGEX = /[\u{1F1E6}-\u{1F1FF}]{2}|[\u{1F300}-\u{1FAFF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/u;
const EMOJI_REGEX_GLOBAL = /[\u{1F1E6}-\u{1F1FF}]{2}|[\u{1F300}-\u{1FAFF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/gu;

function extractEmoji(text: string): string | null {
    const match = text.match(EMOJI_REGEX);
    return match ? match[0] : null;
}

// Returns all emoji in the text concatenated, e.g. "🇨🇭🇩🇪 Twisted" → "🇨🇭🇩🇪"
function extractAllEmoji(text: string): string {
    const matches = text.match(EMOJI_REGEX_GLOBAL);
    return matches ? matches.join('') : '';
}

function removeEmoji(text: string): string {
    return text.replace(EMOJI_REGEX_GLOBAL, '').trim();
}

// Cache for emoji icons
const emojiIconCache = new Map();

// Renders emoji via Chromium canvas in the renderer window — no CDN required.
// System emoji fonts (Apple Color Emoji / Segoe UI Emoji / Noto Color Emoji) are
// used automatically by Chromium, so all Unicode emoji including U+1FA00–U+1FAFF render.
async function createEmojiIcon(emoji: string): Promise<any> {
    if (emojiIconCache.has(emoji)) {
        return emojiIconCache.get(emoji);
    }
    try {
        if (!mainWindow || mainWindow.isDestroyed()) return null;
        const dataUrl: string | null = await mainWindow.webContents.executeJavaScript(
            `typeof window.pxDrawEmoji === 'function' ? window.pxDrawEmoji(${JSON.stringify(emoji)}) : null`
        );
        if (!dataUrl) return null;
        const base64 = dataUrl.replace(/^data:image\/png;base64,/, '');
        const raw = nativeImage.createFromBuffer(Buffer.from(base64, 'base64'));
        const { width, height } = raw.getSize();
        // Preserve aspect ratio: composite of N emoji is N×64 wide, 64 tall → N×16 × 16
        const iconH = 16;
        const iconW = Math.max(16, Math.round(width / height * iconH));
        const icon = raw.resize({ width: iconW, height: iconH });
        emojiIconCache.set(emoji, icon);
        return icon;
    } catch (e) {
        console.error('Failed to create emoji icon:', e);
        return null;
    }
}

// 托盘
let tray: Tray;
// 托盘菜单
let currentMenu: any
// 当前窗口
let mainWindow: BrowserWindow

// 退出app
let isQuiting = false;
export const doQuit = () => {
    // 保存窗口位置和大小
    if (mainWindow) {
        const bounds = mainWindow.getBounds();
        storeSet('windowBounds', bounds);
    }

    // 执行软件退出
    isQuiting = true;
    app.quit();
}
const readyToQuit = () => emitWindow("readyToQuit");
onWindow("doQuit", doQuit)

// 处理菜单
const createMenu = (menuTemplate: any) => {
    if (process.platform === 'darwin') {
        if (isDev) {
            menuTemplate.push(
                {
                    label: 'View',
                    submenu: [
                        {
                            label: 'Open Developer Tools',
                            click: () => {
                                // 获取当前聚焦的窗口
                                const win = BrowserWindow.getFocusedWindow();
                                if (win) win.webContents.openDevTools();
                            }
                        },
                        {
                            label: 'Reload',
                            click: () => {
                                const win = BrowserWindow.getFocusedWindow();
                                if (win) win.webContents.reload();
                            }
                        },
                        {
                            label: 'Force Reload',
                            click: () => {
                                const win = BrowserWindow.getFocusedWindow();
                                if (win) win.webContents.reloadIgnoringCache();
                            }
                        }
                    ]
                }
            )
        }
        const menu = Menu.buildFromTemplate(menuTemplate);
        Menu.setApplicationMenu(menu);
    }
};

ipcMain.on('update-menu', (event, menuTemplate) => {
    createMenu(menuTemplate);
});

const initMenu = () => createMenu([
    {
        label: 'Prizrak-Box', submenu: [
            {
                label: 'Quit', accelerator: 'Cmd+Q', click: readyToQuit
            }
        ]
    },
    {
        label: 'Edit',
        submenu: [
            {label: 'Undo', role: 'undo'},
            {label: 'Redo', role: 'redo'},
            {type: 'separator'},
            {label: 'Cut', role: 'cut'},
            {label: 'Copy', role: 'copy'},
            {label: 'Paste', role: 'paste'},
            {label: 'Delete', role: 'delete'},
            {type: 'separator'},
            {label: 'Select All', role: 'selectAll'}
        ]
    }
]);

// 显示窗口
export function showWindow() {
    if (mainWindow) {
        mainWindow.show();
        app.dock?.show();
        mainWindow.focus();
    }
}

// 切换规则
function switchMode(menuItem, mode) {
    if (!menuItem.checked) {
        menuItem.checked = true
        return
    }
    emitWindow("switchMode", mode);
}

// 切换配置
function switchProfiles(menuItem, profile) {
    if (!menuItem.checked) {
        menuItem.checked = true
        return
    }
    emitWindow("switchProfiles", {
        profile,
        selected: menuItem.checked,
        exclusive: false
    });
}

// Switch proxy in a group
function switchProxyInGroup(menuItem, groupName, proxyName) {
    // Always emit the event to switch proxy, Electron will handle checkbox state
    emitWindow("switchProxyInGroup", {group: groupName, proxy: proxyName});
}

const trayMap: Map<any, any> = new Map();
trayMap.set('tray.show', {
    id: 'tray.show',
    label: '显示窗口',
    type: 'normal',
    click: showWindow
});
trayMap.set('tray.rule', {
    id: 'tray.rule',
    label: '规则',
    type: 'checkbox',
    checked: false,
    click: (menuItem) => switchMode(menuItem, 'rule')
});
trayMap.set('tray.global', {
    id: 'tray.global',
    label: '全局',
    type: 'checkbox',
    checked: false,
    click: (menuItem) => switchMode(menuItem, 'global')
});
trayMap.set('tray.direct', {
    id: 'tray.direct',
    label: '直连',
    type: 'checkbox',
    checked: false,
    click: (menuItem) => switchMode(menuItem, 'direct')
});
trayMap.set('tray.profiles', {id: 'tray.profiles', label: '订阅', submenu: []});
trayMap.set('tray.proxyGroups', {id: 'tray.proxyGroups', label: 'Proxy Groups', submenu: []});
trayMap.set('tray.dashboard', {
    id: 'tray.dashboard',
    label: 'Open Dashboard',
    submenu: [],
    enabled: false,
});
trayMap.set('tray.proxy', {
    id: 'tray.proxy',
    label: '系统代理',
    type: 'checkbox',
    checked: false,
    click: () => emitWindow("switchProxy")
});
trayMap.set('tray.tun', {
    id: 'tray.tun',
    label: 'Tun模式',
    type: 'checkbox',
    checked: false,
    click: () => emitWindow("switchTun")
});
trayMap.set('tray.quit', {id: 'tray.quit', label: '退出', type: 'normal', click: readyToQuit});

const createTrayMenu = () => [
    trayMap.get('tray.show'),
    {type: 'separator'},
    trayMap.get('tray.rule'),
    trayMap.get('tray.global'),
    trayMap.get('tray.direct'),
    {type: 'separator'},
    // Profile selection from the tray is intentionally disabled.
    trayMap.get('tray.proxyGroups'),
    trayMap.get('tray.dashboard'),
    {type: 'separator'},
    trayMap.get('tray.proxy'),
    trayMap.get('tray.tun'),
    {type: 'separator'},
    trayMap.get('tray.quit'),
]

// 初始化托盘菜单
currentMenu = Menu.buildFromTemplate(createTrayMenu());

// Resolves the path to a file from the public/ directory.
// In dev mode: <project_root>/public/<filename>
// In packaged app: <app.asar>/.vite/renderer/px_window/<filename>
// This fixes the bug where path.join(__dirname, filename) pointed to
// .vite/build/ which does NOT contain public/ assets.
function getTrayIconPath(filename: string): string {
    if (isDev) {
        return path.join(app.getAppPath(), 'public', filename);
    }
    return path.join(app.getAppPath(), '.vite', 'renderer', 'px_window', filename);
}

// 初始化托盘
export function initTray(browserWindow: BrowserWindow): void {
    // 初始化左上角菜单
    initMenu()

    // 初始化窗口事件
    mainWindow = browserWindow
    mainWindow.on('close', (event) => {
        if (!isQuiting) {
            event.preventDefault();
            if (process.platform !== 'darwin') {
                mainWindow.minimize()
            } else {
                mainWindow.hide();
            }
        }
    });

    // 初始化tray
    let trayImage: any;
    if (process.platform === 'darwin') {
        // macOS: use monochrome template PNG (black on transparent) so the system
        // automatically adapts it to light/dark menu bar.
        // Electron auto-picks tray-macos@2x.png on Retina displays.
        trayImage = nativeImage.createFromPath(getTrayIconPath('tray-macos.png'));
        trayImage.setTemplateImage(true);
    } else {
        trayImage = nativeImage.createFromPath(getTrayIconPath('tray.png')).resize({width: 32, height: 32});
    }
    tray = new Tray(trayImage);
    tray.setToolTip('Prizrak-Box');
    tray.setContextMenu(Menu.buildFromTemplate(createTrayMenu()))

    // 左键点击时弹出菜单
    tray.on('click', () => {
        tray.popUpContextMenu();
    });
    // 右键点击时弹出菜单
    tray.on('right-click', () => {
        tray.popUpContextMenu();
    });
}


// 接收浏览器消息
function onWindow(name, cb) {
    ipcMain.on('px_' + name, (_event, value) => {
        if (cb) {
            cb(value)
        }
    })
}

// 发送消息到浏览器
function emitWindow(name: string, ...value: any[]) {
    if (mainWindow) {
        mainWindow.webContents.send('px_' + name, ...value);
    }
}

const openDashboardFromTray = (url: string) => {
    if (!url) {
        return;
    }

    void shell.openExternal(url).catch((error) => {
        console.error('Failed to open dashboard link from tray', error);
    });
};


// 监听消息
onWindow("translate", function (trayOptions) {
    for (const [key, value] of Object.entries(trayOptions)) {
        trayMap.get(key).label = value
    }
    currentMenu = Menu.buildFromTemplate(createTrayMenu());
    tray.setContextMenu(currentMenu);
})
onWindow("mode", function (value) {
    currentMenu.getMenuItemById('tray.rule').checked = false;
    currentMenu.getMenuItemById('tray.global').checked = false;
    currentMenu.getMenuItemById('tray.direct').checked = false;
    trayMap.get('tray.rule').checked = false;
    trayMap.get('tray.global').checked = false;
    trayMap.get('tray.direct').checked = false;
    const key = 'tray.' + value
    currentMenu.getMenuItemById(key).checked = true;
    trayMap.get(key).checked = true
})
onWindow("proxy", function (value) {
    const key = 'tray.proxy'
    currentMenu.getMenuItemById(key).checked = value;
    trayMap.get(key).checked = value
})
onWindow("tun", function (value) {
    const key = 'tray.tun'
    currentMenu.getMenuItemById(key).checked = value;
    trayMap.get(key).checked = value
})
onWindow("profiles", function (profiles) {
    const key = 'tray.profiles'
    const pList: any[] = []
    for (let profile of profiles) {
        pList.push({
            label: profile.title,
            type: 'checkbox',
            checked: !!profile.selected,
            click: (menuItem) => switchProfiles(menuItem, profile)
        })
    }
    trayMap.get(key).submenu = pList
    currentMenu = Menu.buildFromTemplate(createTrayMenu());
    tray.setContextMenu(currentMenu);
})

onWindow("dashboards", function (dashboards) {
    const key = 'tray.dashboard';
    const items = Array.isArray(dashboards)
        ? dashboards
            .filter((dashboard) => dashboard && dashboard.name && dashboard.url)
            .map((dashboard) => ({
                id: `${key}.${dashboard.key ?? dashboard.name}`,
                label: dashboard.name,
                type: 'normal',
                click: () => openDashboardFromTray(dashboard.url),
            }))
        : [];

    const menuItem = trayMap.get(key);
    menuItem.submenu = items;
    menuItem.enabled = items.length > 0;
    currentMenu = Menu.buildFromTemplate(createTrayMenu());
    tray.setContextMenu(currentMenu);
})

onWindow("proxyGroups", async function (proxyGroups) {
    const key = 'tray.proxyGroups';
    const groupMenus: any[] = [];

    if (Array.isArray(proxyGroups) && proxyGroups.length > 0) {
        for (const group of proxyGroups) {
            if (!group || !group.name || !Array.isArray(group.proxies) || group.proxies.length === 0) {
                continue;
            }

            // Create all proxy items with icons loaded in parallel
            const proxyItems = await Promise.all(group.proxies.map(async (proxy) => {
                const menuItem: any = {
                    label: proxy.name,
                    type: 'radio',
                    checked: proxy.now || false,
                    click: (menuItem) => switchProxyInGroup(menuItem, group.name, proxy.name)
                };

                // If emoji found, create icon from it and remove emoji from label
                const emoji = extractAllEmoji(proxy.name);
                if (emoji) {
                    const icon = await createEmojiIcon(emoji);
                    if (icon) {
                        menuItem.icon = icon;
                        menuItem.label = removeEmoji(proxy.name);
                    }
                }

                return menuItem;
            }));

            const groupItem: any = {
                label: group.name,
                submenu: proxyItems,
            };
            const groupEmoji = extractAllEmoji(group.name);
            if (groupEmoji) {
                const groupIcon = await createEmojiIcon(groupEmoji);
                if (groupIcon) {
                    groupItem.icon = groupIcon;
                    groupItem.label = removeEmoji(group.name);
                }
            }
            groupMenus.push(groupItem);
        }
    }

    trayMap.get(key).submenu = groupMenus;
    currentMenu = Menu.buildFromTemplate(createTrayMenu());
    tray.setContextMenu(currentMenu);
})

// 窗口控制
onWindow("hide", function () {
    mainWindow.hide();
    app.dock?.hide()
})
onWindow("close", function () {
    app.quit()
})
onWindow("max", function () {
    mainWindow.isMaximized() ? mainWindow.unmaximize() : mainWindow.maximize()
})
onWindow("min", function () {
    mainWindow.minimize()
})

// 开机自启
onWindow("boot", function (value) {
    if (value) {
        return enableAutoLaunch()
    } else {
        return disableAutoLaunch()
    }
})

// 授权提示
onWindow("tunAuthTip", function (tunAuthTip) {
    if (tunAuthTip) {
        storeSet("tunAuthTip", tunAuthTip);
    }
})





