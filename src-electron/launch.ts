import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import {execFileSync} from 'child_process';
import {lookup} from 'dns/promises';
import {app} from 'electron';
// @ts-ignore
import AutoLaunch from 'auto-launch';
import log from './log';
import {storeGet, storeSet} from './store';

const APP_NAME = 'Prizrak-Box';
const BOOT_FLAG = '--boot-launch';

// ─── Linux: write .desktop directly ─────────────────────────────────────────
//
// auto-launch has three bugs on Linux system packages (Arch, Manjaro, etc.):
//
//  1. app.getPath('exe') returns the Electron runtime binary
//     (/usr/bin/electron41 or /usr/lib/electron41/electron), not the app
//     wrapper script (/usr/bin/prizrak-box). Autostart launches bare Electron.
//
//  2. auto-launch's fixOpts() overwrites appName with the last path segment,
//     so Name='Prizrak-Box' is silently replaced with Name='electron' and
//     the file is saved as ~/.config/autostart/electron.desktop.
//
//  3. custom args are ignored on Linux — --boot-launch never reaches Exec=.
//
// Fix: on Linux we skip auto-launch and write the .desktop file ourselves.

const AUTOSTART_DIR  = path.join(os.homedir(), '.config', 'autostart');
const DESKTOP_FILE   = path.join(AUTOSTART_DIR, `${APP_NAME}.desktop`);

// Wrapper script names and directories to search, in priority order.
const WRAPPER_NAMES  = ['prizrak-box', 'Prizrak-Box'];
const WRAPPER_DIRS   = ['/usr/bin', '/usr/local/bin', '/opt/prizrak-box'];

function detectLinuxExecPath(): string {
    // AppImage bundles set this env var automatically.
    if (process.env.APPIMAGE) return process.env.APPIMAGE;

    // Search known bin directories for the wrapper script.
    for (const dir of WRAPPER_DIRS) {
        for (const name of WRAPPER_NAMES) {
            const candidate = path.join(dir, name);
            try {
                fs.accessSync(candidate, fs.constants.X_OK);
                return candidate;
            } catch { /* keep searching */ }
        }
    }

    // Try `which` as a last resort.
    for (const name of WRAPPER_NAMES) {
        try {
            const result = execFileSync('which', [name], {encoding: 'utf8'}).trim();
            if (result) return result;
        } catch { /* keep searching */ }
    }

    log.warn('prizrak-box wrapper not found, falling back to app.getPath("exe")');
    return app.getPath('exe');
}

function writeDesktopFile(): void {
    const execPath = detectLinuxExecPath();
    log.info(`Linux autostart: Exec=${execPath}`);
    fs.mkdirSync(AUTOSTART_DIR, {recursive: true});
    fs.writeFileSync(DESKTOP_FILE, [
        '[Desktop Entry]',
        'Type=Application',
        'Version=1.0',
        `Name=${APP_NAME}`,
        `Comment=${APP_NAME} autostart`,
        `Exec=${execPath} ${BOOT_FLAG}`,
        'StartupNotify=false',
        'Terminal=false',
    ].join('\n') + '\n', 'utf8');
}

const linuxAutoLaunch = {
    isEnabled: () => fs.existsSync(DESKTOP_FILE),
    enable()  { writeDesktopFile(); },
    disable() { try { fs.unlinkSync(DESKTOP_FILE); } catch { /* already gone */ } },
};

// ─── Cross-platform launcher (Windows / macOS) ────────────────────────────────

let autoLauncher = createAutoLauncher();

function createAutoLauncher(): AutoLaunch {
    return new AutoLaunch({
        name: APP_NAME,
        path: app.getPath('exe'),
        args: [BOOT_FLAG],
    });
}

// ─── Public API ───────────────────────────────────────────────────────────────

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

export async function isBootAutoLaunch(): Promise<boolean> {
    const uptime = os.uptime();
    const launchedByFlag = process.argv.includes(BOOT_FLAG);
    const launchedSoonAfterBoot = uptime < 30;

    let wasOpenedAtLogin = false;
    try {
        wasOpenedAtLogin = app.getLoginItemSettings?.().wasOpenedAtLogin ?? false;
    } catch { /* platform does not support getLoginItemSettings */ }

    log.info('process.argv is', process.argv);
    return launchedByFlag || wasOpenedAtLogin || launchedSoonAfterBoot;
}

export async function enableAutoLaunch(): Promise<void> {
    try {
        if (process.platform === 'linux') {
            if (!linuxAutoLaunch.isEnabled()) {
                linuxAutoLaunch.enable();
                storeSet('autoLaunch.lastRegisteredExe', detectLinuxExecPath());
                log.info('✅ Auto-launch enabled (Linux .desktop)');
            } else {
                log.info('Auto-launch already enabled');
            }
            return;
        }
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

export async function disableAutoLaunch(): Promise<void> {
    try {
        if (process.platform === 'linux') {
            linuxAutoLaunch.disable();
            log.info('🛑 Auto-launch disabled (Linux .desktop)');
            return;
        }
        if (await autoLauncher.isEnabled()) {
            await autoLauncher.disable();
            log.info('🛑 Auto-launch disabled');
        }
    } catch (err) {
        log.error('Failed to disable auto-launch:', err);
    }
}

export async function isAutoLaunchEnabled(): Promise<boolean> {
    try {
        if (process.platform === 'linux') return linuxAutoLaunch.isEnabled();
        return await autoLauncher.isEnabled();
    } catch (err) {
        log.error('Failed to query auto-launch status:', err);
        return false;
    }
}

export async function updateAutoLaunchRegistration(): Promise<void> {
    try {
        if (process.platform === 'linux') {
            if (!linuxAutoLaunch.isEnabled()) return;
            const currentExec = detectLinuxExecPath();
            const lastRegistered = storeGet('autoLaunch.lastRegisteredExe') as string | undefined;
            if (currentExec !== lastRegistered) {
                linuxAutoLaunch.enable(); // rewrites file with updated path
                storeSet('autoLaunch.lastRegisteredExe', currentExec);
                log.info(`🆕 Auto-launch path updated (Linux): ${currentExec}`);
            } else {
                log.info('Auto-launch registration does not need updating');
            }
            return;
        }
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
