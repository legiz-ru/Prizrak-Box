import * as os from 'os';
import {lookup} from 'dns/promises';
import {app} from 'electron';
// @ts-ignore
import AutoLaunch from 'auto-launch';
import log from './log';
import {storeGet, storeSet} from './store';

const APP_NAME = 'Prizrak-Box';
const BOOT_FLAG = '--boot-launch';

let autoLauncher = createAutoLauncher();

/**
 * 创建 AutoLaunch 实例
 */
function createAutoLauncher(): AutoLaunch {
    return new AutoLaunch({
        name: APP_NAME,
        path: app.getPath('exe'),
        args: [BOOT_FLAG],
    });
}

/**
 * 等待网络就绪（能解析指定域名），超时返回 false
 * @param timeout 超时时间（默认 30 秒）
 * @param host 检测的主机（默认 bing.com）
 */
export async function waitForNetworkReady(timeout = 30000, host = 'bing.com'): Promise<boolean> {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
        try {
            await lookup(host);
            return true;
        } catch {
            await new Promise(res => setTimeout(res, 1000));
        }
    }
    return false;
}

/**
 * 判断当前是否由开机自启启动
 */
export async function isBootAutoLaunch(): Promise<boolean> {
    const uptime = os.uptime();
    const launchedByFlag = process.argv.includes(BOOT_FLAG);
    const launchedSoonAfterBoot = uptime < 30;

    let wasOpenedAtLogin = false;
    try {
        wasOpenedAtLogin = app.getLoginItemSettings?.().wasOpenedAtLogin ?? false;
    } catch {
        // 忽略不支持的平台
    }

    log.info('process.argv is', process.argv);
    return launchedByFlag || wasOpenedAtLogin || launchedSoonAfterBoot;
}

/**
 * 启用开机自启
 */
export async function enableAutoLaunch(): Promise<void> {
    try {
        if (!(await autoLauncher.isEnabled())) {
            await autoLauncher.enable();
            storeSet('autoLaunch.lastRegisteredExe', app.getPath('exe'));
            log.info('✅ Auto-launch enabled');
        } else {
            log.info('Auto-launch already enabled');
        }
    } catch (err) {
        log.error('Failed to enable auto-launch:', err);
    }
}

/**
 * 禁用开机自启
 */
export async function disableAutoLaunch(): Promise<void> {
    try {
        if (await autoLauncher.isEnabled()) {
            await autoLauncher.disable();
            log.info('🛑 Auto-launch disabled');
        }
    } catch (err) {
        log.error('Failed to disable auto-launch:', err);
    }
}

/**
 * 查询开机自启状态
 */
export async function isAutoLaunchEnabled(): Promise<boolean> {
    try {
        return await autoLauncher.isEnabled();
    } catch (err) {
        log.error('Failed to query auto-launch status:', err);
        return false;
    }
}

/**
 * 更新开机自启注册项路径（如当前 exe 路径发生变化）
 */
export async function updateAutoLaunchRegistration(): Promise<void> {
    try {
        const currentExe = app.getPath('exe');
        const lastRegistered = storeGet('autoLaunch.lastRegisteredExe') as string | undefined;

        if ((await autoLauncher.isEnabled()) && currentExe !== lastRegistered) {
            await autoLauncher.disable();

            autoLauncher = createAutoLauncher();
            await autoLauncher.enable();

            storeSet('autoLaunch.lastRegisteredExe', currentExe);
            log.info(`🆕 Auto-launch path updated: ${currentExe}`);
        } else {
            log.info('Auto-launch registration does not need updating');
        }
    } catch (err) {
        log.error('Failed to update auto-launch registration:', err);
    }
}