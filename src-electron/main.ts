import {app, BrowserWindow, BrowserWindowConstructorOptions, ipcMain, session} from 'electron';
import path from 'node:path';
import {startServer, storeInfo} from "./server";
import {doQuit, initTray, showWindow} from "./tray";
import {startBackend} from "./admin";
import log from './log';
import {initStore, storeGet} from "./store";
import {isBootAutoLaunch, updateAutoLaunchRegistration, waitForNetworkReady} from "./launch";

// 是否在开发模式
const isDev = !app.isPackaged;

// Set application name for notifications
app.name = 'Prizrak-Box';

// Set App User Model ID for Windows notifications
if (process.platform === 'win32') {
    app.setAppUserModelId('com.legiz-ru.prizrak-box');
}

// 主窗口
let mainWindow: BrowserWindow | null = null;

// Флаг для отслеживания первого запуска с startMinimized
let isFirstLaunchMinimized = false;

// 深度链接相关
const DEEP_LINK_SCHEME = 'prizrak-box';
const DEEP_LINK_HOST_INSTALL = 'install-config';
const DEEP_LINK_EVENT = 'import-profile-from-deeplink';
const DEEP_LINK_READY_EVENT = 'deeplink-handler-ready';
const pendingDeepLinks: string[] = [];
let deepLinkHandlerReady = false;
// 屏蔽安全警告
process.env["ELECTRON_DISABLE_SECURITY_WARNINGS"] = "true";
const createWindow = (isBoot: boolean) => {
    let windowOptions: BrowserWindowConstructorOptions = {
        title: 'Prizrak-Box',
        minWidth: 960,
        minHeight: 660,
        width: 1100,
        height: 760,
        show: false, // 先不显示窗口
        center: true,
        webPreferences: {
            preload: path.join(__dirname, 'preload.js'),
            contextIsolation: true,
            webSecurity: false,
            nodeIntegrationInWorker: true
        },
        ...(process.platform !== 'darwin' ? {
            titleBarStyle: 'hidden'
        } : {
            titleBarStyle: 'hiddenInset'
        })
    };

    // 恢复上次窗口位置
    const savedBounds: any = storeGet('windowBounds');
    if (savedBounds && savedBounds.x !== undefined && savedBounds.y !== undefined) {
        windowOptions = {
            ...windowOptions,
            ...savedBounds
        };
    }

    deepLinkHandlerReady = false;
    mainWindow = new BrowserWindow(windowOptions);

    // 隐藏菜单栏
    mainWindow.setMenu(null);

    // 托盘
    initTray(mainWindow);

    // 页面加载
    const listenAddr = storeInfo.listenAddr();
    const queryParams = new URLSearchParams();

    const port = storeInfo.port();
    const secret = storeInfo.secret();

    if (port) {
        queryParams.set('port', port);
    }

    if (secret) {
        queryParams.set('secret', secret);
    }

    if (listenAddr) {
        queryParams.set('frontendOrigin', `http://${listenAddr}`);
    }

    const queryString = queryParams.toString();
    const devBase = 'http://localhost:5173';
    const prodBase = listenAddr ? `http://${listenAddr}/index.html` : 'index.html';

    const filePath = isDev
        ? `${devBase}${queryString ? `?${queryString}` : ''}`
        : `${prodBase}${queryString ? `?${queryString}` : ''}`;

    log.info('Preparing to load page');
    mainWindow.loadURL(filePath).catch((err) => {
        log.error('Page loading failed:', err);
    });

    mainWindow.webContents.on('did-start-loading', () => {
        deepLinkHandlerReady = false;
    });

    mainWindow.webContents.on('did-finish-load', () => {
        processPendingDeepLinks();
    });

    // 页面加载完成再显示，避免白屏
    mainWindow.webContents.once('did-finish-load', () => {
        // Get settings
        let settings: any = storeGet('setting');
        // Parse settings if it's a JSON string
        if (typeof settings === 'string') {
            try {
                settings = JSON.parse(settings);
            } catch (e) {
                log.error('Settings parsing error:', e);
                settings = {};
            }
        }
        const startMinimized = settings?.startMinimized === true;

        log.info('Startup settings check - isBoot:', isBoot, ', settings:', JSON.stringify(settings), ', startMinimized:', startMinimized);

        if (isBoot) {
            log.info('Silent startup complete (isBoot:', isBoot, ')');
        } else if (startMinimized) {
            // Start minimized to tray: hide window and dock
            isFirstLaunchMinimized = true;
            mainWindow.hide();
            app.dock?.hide();
            log.info('Starting minimized to tray');

            // Reset flag after a short delay to prevent immediate window showing
            setTimeout(() => {
                isFirstLaunchMinimized = false;
                log.info('isFirstLaunchMinimized flag reset');
            }, 1000);
        } else {
            mainWindow.show();
            mainWindow.focus();
            log.info('Page loaded successfully');
        }
    });

    mainWindow.on('closed', () => {
        deepLinkHandlerReady = false;
        mainWindow = null;
    });
};

const isDeepLinkUrl = (arg: string | undefined): arg is string => {
    return typeof arg === 'string' && arg.startsWith(`${DEEP_LINK_SCHEME}://`);
};

const processPendingDeepLinks = () => {
    if (!mainWindow || mainWindow.isDestroyed() || !deepLinkHandlerReady) {
        return;
    }

    if (pendingDeepLinks.length === 0) {
        return;
    }

    const queue = pendingDeepLinks.splice(0, pendingDeepLinks.length);
    showWindow();

    for (const url of queue) {
        if (!url) {
            continue;
        }

        log.info('Processing deep link queue:', url);
        mainWindow.webContents.send(DEEP_LINK_EVENT, {rawUrl: url});
    }
};

const enqueueDeepLink = (url: string) => {
    pendingDeepLinks.push(url);
    processPendingDeepLinks();
};

function handleDeepLink(url: string) {
    const trimmed = url?.trim();
    if (!trimmed) {
        return;
    }

    try {
        const parsedUrl = new URL(trimmed);
        if (parsedUrl.protocol !== `${DEEP_LINK_SCHEME}:`) {
            return;
        }

        const host = parsedUrl.hostname || parsedUrl.host;
        if (host && host.toLowerCase() === DEEP_LINK_HOST_INSTALL) {
            log.info('Received deep link:', trimmed);
            enqueueDeepLink(trimmed);
        } else {
            log.warn('Unknown deep link:', trimmed);
        }
    } catch (error) {
        log.error('Failed to parse deep link:', error);
    }
}

ipcMain.on(DEEP_LINK_READY_EVENT, (event) => {
    if (!mainWindow || event.sender !== mainWindow.webContents) {
        return;
    }

    deepLinkHandlerReady = true;
    processPendingDeepLinks();
});

// 等待 backend 传来的 port 和 secret
let resolveReady: () => void;
const waitForReady = new Promise<void>((resolve) => {
    resolveReady = resolve;
});

// 生成一个随机 UA
const version = Math.floor(Math.random() * 20 + 85); // 统一版本号
const agents = [
    {
        ua: `Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/${version}.0.0.0 Safari/537.36`,
        platform: `"Windows"`,
        secChUa: `"Google Chrome";v="${version}", "Chromium";v="${version}", "Not_A Brand";v="99"`
    },
    {
        ua: `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_${Math.floor(Math.random() * 9 + 10)}) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/${version}.0.0.0 Safari/537.36`,
        platform: `"macOS"`,
        secChUa: `"Google Chrome";v="${version}", "Chromium";v="${version}", "Not_A Brand";v="99"`
    },
    {
        ua: `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/${version}.0.0.0 Safari/537.36`,
        platform: `"Linux"`,
        secChUa: `"Google Chrome";v="${version}", "Chromium";v="${version}", "Not_A Brand";v="99"`
    }
];

const registerDeepLinkProtocol = () => {
    try {
        if (process.defaultApp && process.argv.length >= 2) {
            const exePath = process.execPath;
            const resolvedPath = path.resolve(process.argv[1]);
            app.setAsDefaultProtocolClient(DEEP_LINK_SCHEME, exePath, [resolvedPath]);
        } else if (!app.isDefaultProtocolClient(DEEP_LINK_SCHEME)) {
            app.setAsDefaultProtocolClient(DEEP_LINK_SCHEME);
        }
    } catch (error) {
        log.error('Failed to register deep link protocol:', error);
    }
};

for (const arg of process.argv) {
    if (isDeepLinkUrl(arg)) {
        pendingDeepLinks.push(arg);
    }
}

// 单例模式
const gotTheLock = app.requestSingleInstanceLock();
if (!gotTheLock) {
    doQuit()
} else {
    // 试图启动第二个应用实例
    app.on('second-instance', (_event, commandLine) => {
        const urls = commandLine.filter(isDeepLinkUrl);
        if (urls.length > 0) {
            urls.forEach(handleDeepLink);
        }
        showWindow();
    });

    // 监听应用被激活
    app.on('activate', showWindow);

    if (process.platform === 'darwin') {
        app.on('open-url', (event, url) => {
            event.preventDefault();
            handleDeepLink(url);
        });
    }

    app.whenReady().then(async () => {
        // Check if launched at boot
        const isBoot = await isBootAutoLaunch();
        log.info('Boot launch:', isBoot);

        // If launched at boot, wait for network to be ready (up to 30 seconds)
        if (isBoot) {
            // Hide dock first
            app.dock?.hide()

            log.info('Boot launch, waiting for network...');
            const networkReady = await waitForNetworkReady(30000, 'bing.com');
            if (!networkReady) {
                log.warn('Network detection timeout, continuing but network may be unavailable');
            } else {
                log.info('Network is ready');
            }
        }

        // Initialize frontend store
        initStore(log.getHomeDir())

        // Start frontend static server
        startServer(resolveReady, startBackend)

        // Wait for backend to start
        await waitForReady;

        registerDeepLinkProtocol();

        // Set request headers
        const agent = agents[Math.floor(Math.random() * agents.length)];
        session.defaultSession.webRequest.onBeforeSendHeaders((details, callback) => {
            details.requestHeaders['Referer'] = new URL(details.url).origin // Only send domain
            details.requestHeaders['User-Agent'] = agent.ua;
            details.requestHeaders['sec-ch-ua-platform'] = agent.platform;
            details.requestHeaders['sec-ch-ua'] = agent.secChUa;
            callback({requestHeaders: details.requestHeaders})
        })

        // Start UI
        log.info('Ready, starting window, port=', storeInfo.port(), ' secret=', storeInfo.secret());
        createWindow(isBoot);

        // Update auto-launch registration path
        await updateAutoLaunchRegistration()
    });
}