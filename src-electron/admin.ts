import path from "node:path";
import {spawn} from "child_process";
import {app, dialog} from "electron";
import fs from "node:fs";
import log from './log';
import {storeGet} from "./store";
import {
    isServiceRunning,
    startPxViaService,
    isServiceModeEnabled
} from "./service";

// 是否在开发模式
const isDev = !app.isPackaged;

// 获取提示词
function getAuthTip(): string {
    const tip = storeGet("tunAuthTip")
    if (tip) {
        return tip.toString();
    } else {
        return "Px 需要授权才能使用 TUN 模式。\n[Px requires authorization to enable TUN.]";
    }
}

// 获取px路径
function getBackendPath() {
    const execName = process.platform === 'win32' ? 'px.exe' : 'px';
    return isDev
        ? path.join(__dirname, '../../src-go', execName)
        : path.join(process.resourcesPath, execName);
}

// 检查是否有管理员权限
function checkAdminRights(callback: any) {
    const platform = process.platform;

    if (platform === 'win32') {
        // Windows 上检查管理员权限
        const command = spawn('net', ['session']);

        command.on('error', (error) => {
            log.info('net session :', error);
            callback(false);
        });

        command.on('exit', (code) => {
            if (code === 0) {
                callback(true);   // 如果退出码为 0，表示有管理员权限
            } else {
                callback(false);  // 否则没有管理员权限
            }
        });

    } else if (platform === 'darwin' || platform === 'linux') {
        // macOS 或 Linux 上检查是否为 root 用户
        if (process.getuid && process.getuid() === 0) {
            callback(true);  // 有管理员权限
        } else {
            callback(false); // 没有管理员权限
        }
    } else {
        // 其他平台默认认为没有管理员权限
        callback(false);
    }
}

// 开启后端
export async function startBackend(addr: string) {
    const backendPath = getBackendPath();
    const homeDir = encodeURIComponent(log.getHomeDir());
    const args = ['-addr=' + addr, '-home=' + homeDir];

    // Проверяем режим сервиса
    if (isServiceModeEnabled()) {
        log.info('[Backend] Service mode is enabled, checking service...');
        const serviceRunning = await isServiceRunning();

        if (serviceRunning) {
            log.info('[Backend] Service is running, starting px via service...');
            const started = await startPxViaService(backendPath, addr, homeDir);
            if (started) {
                log.info('[Backend] px started via service successfully');
                return;
            }
            log.warn('[Backend] Failed to start px via service, falling back to normal mode');
        } else {
            log.warn('[Backend] Service mode enabled but service not running, falling back to normal mode');
        }
    }

    // Стандартный режим запуска
    checkAdminRights((isAdmin: boolean) => {
        if (isAdmin) {
            log.info('Has administrator privileges');

            startNormally(backendPath, args);
        } else {
            log.info('No administrator privileges');

            // Check if elevation is disabled
            const setting = storeGet("setting")
            if (!!setting) {
                const set = JSON.parse(setting as string);
                if (set.hasOwnProperty("auth") && set["auth"]) {
                    log.info('Elevation prompt enabled');
                } else {
                    log.info('Elevation prompt disabled');
                    startNormally(backendPath, args);
                    return;
                }
            } else {
                log.info('No settings found, skipping elevation');
                startNormally(backendPath, args);
                return;
            }

            // 只在 Windows 和 Linux 平台上弹出提权提示，macOS 也需要显示提权提示
            if (process.platform !== 'darwin') {
                const confirmed = dialog.showMessageBoxSync({
                    type: 'info',
                    buttons: ['Continue', 'Cancel'],
                    defaultId: 0,
                    cancelId: 1,
                    title: 'Prizrak-Box',
                    message: getAuthTip(),
                });

                if (confirmed === 1) {
                    // User cancelled elevation → start in normal mode
                    log.info('User cancelled elevation, starting with normal privileges');
                    startNormally(backendPath, args);
                    return;
                }
            }

            // Try to run with admin privileges, fallback if failed
            tryRunAsAdmin(backendPath, args, (success) => {
                if (!success) {
                    log.info('Failed to start with admin privileges, starting in normal mode');
                    startNormally(backendPath, args);
                }
            });
        }
    });
}

// 尝试管理员启动
function tryRunAsAdmin(executable: string, args: string[], callback: (success: boolean) => void) {
    switch (process.platform) {
        case 'darwin': {
            // macOS 使用 AppleScript 提权
            const command = `${[executable, ...args].map(escapeShell).join(' ')}`;
            // 使用 `with prompt` 来直接在授权对话框中显示提示信息
            const script = `
                do shell script "${command}" with administrator privileges with prompt "${getAuthTip()}"
            `;
            const osa = spawn('osascript', ['-e', script]);
            log.info("[Admin] Starting PX command line:", osa.spawnargs);
            osa.on('exit', (code) => callback(code === 0));
            osa.on('error', () => callback(false));
            break;
        }

        case 'win32': {
            // Windows 使用 PowerShell 提权并隐藏窗口
            const psArgs = [
                '-Command',
                `Start-Process -FilePath '${executable}' -ArgumentList '${args.join(' ')}' -Verb RunAs -WindowStyle Hidden`
            ];
            const ps = spawn('powershell.exe', psArgs);
            log.info("[Admin] Starting PX command line:", ps.spawnargs);
            ps.on('exit', (code) => callback(code === 0));
            ps.on('error', () => callback(false));
            break;
        }

        case 'linux': {
            // Linux: 提权方式依次尝试 pkexec → gksudo → kdesudo → sudo
            const env = {
                ...process.env,
                PATH: process.env.PATH || "/usr/bin:/bin:/usr/sbin:/sbin",
                DISPLAY: process.env.DISPLAY,
                XAUTHORITY: process.env.XAUTHORITY,
            };

            const methods = [
                '/usr/bin/pkexec',
                '/usr/bin/gksudo',
                '/usr/bin/kdesudo',
                '/usr/bin/sudo',
                'pkexec',
                'gksudo',
                'kdesudo',
                'sudo'
            ];

            let tried = false;

            (function tryNext(index = 0) {
                if (index >= methods.length) {
                    log.error("No available elevation method succeeded.");
                    callback(false);
                    return;
                }

                const method = methods[index];
                if (!fs.existsSync(method) && !method.includes('/')) {
                    // Skip fallback names like 'sudo' if not full path
                    return tryNext(index + 1);
                }

                log.info(`Trying to elevate with: ${method}`);
                tried = true;

                const elevated = spawn(method, [executable, ...args], {
                    env,
                    stdio: 'inherit',
                });
                log.info("[Admin] Starting PX command line:", elevated.spawnargs);

                elevated.on('error', (err) => {
                    log.error(`Error using ${method}:`, err);
                    tryNext(index + 1);
                });

                elevated.on('exit', (code, signal) => {
                    if (code === 0) {
                        log.info(`${method} succeeded`);
                        callback(true);
                    } else {
                        log.warn(`${method} exited with code ${code} or signal ${signal}`);
                        tryNext(index + 1);
                    }
                });
            })();

            break;
        }


        default:
            log.error('Unsupported platform:', process.platform);
            callback(false);
    }
}

function startNormally(executable: string, args: string[]) {
    const backend = spawn(executable, args, {
        stdio: ['ignore', 'pipe', 'pipe']
    });

    log.info("[Normal] Starting PX command line:", backend.spawnargs);

    backend.stdout.on('data', () => {
        // log.info(`[backend stdout]: ${data}`);
    });

    backend.stderr.on('data', (data) => {
        log.error(`[backend stderr]: ${data}`);
    });

    backend.on('error', (err) => log.error('Backend error:', err));
    backend.on('exit', (code) => log.info('Backend exited with code:', code));
}

function escapeShell(cmd: string): string {
    return cmd.replace(/"/g, '\\"').replace(/\$/g, '\\$');
}
