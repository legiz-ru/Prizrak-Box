import path from "node:path";
import net from "node:net";
import fs from "node:fs";
import {spawn} from "child_process";
import {app, dialog} from "electron";
import log from './log';
import {storeGet, storeSet} from "./store";

// Путь к сокету/пайпу
const WINDOWS_PIPE_NAME = '\\\\.\\pipe\\prizrak-box-service';
const UNIX_SOCKET_PATH = '/tmp/prizrak-box-service.sock';

// Путь к сервису
const isDev = !app.isPackaged;

function getServicePath(): string {
    const execName = process.platform === 'win32' ? 'px-service.exe' : 'px-service';
    return isDev
        ? path.join(__dirname, '../../src-service', execName)
        : path.join(process.resourcesPath, execName);
}

// IPC запрос
interface IPCRequest {
    command: string;
    data?: any;
}

// IPC ответ
interface IPCResponse {
    success: boolean;
    data?: any;
    error?: string;
}

// Статус сервиса
export interface ServiceStatus {
    installed: boolean;
    running: boolean;
    isAdmin: boolean;
    version?: string;
}

/**
 * Отправляет IPC запрос к сервису
 */
async function sendIPCRequest(request: IPCRequest, timeout: number = 5000): Promise<IPCResponse> {
    return new Promise((resolve, reject) => {
        const socketPath = process.platform === 'win32' ? WINDOWS_PIPE_NAME : UNIX_SOCKET_PATH;

        const client = net.createConnection(socketPath, () => {
            const requestStr = JSON.stringify(request) + '\n';
            client.write(requestStr);
        });

        let responseData = '';

        client.setTimeout(timeout);

        client.on('data', (data) => {
            responseData += data.toString();
            if (responseData.includes('\n')) {
                try {
                    const response = JSON.parse(responseData.trim());
                    client.end();
                    resolve(response);
                } catch (e) {
                    client.end();
                    reject(new Error('Invalid response format'));
                }
            }
        });

        client.on('timeout', () => {
            client.destroy();
            reject(new Error('Connection timeout'));
        });

        client.on('error', (err) => {
            reject(err);
        });
    });
}

/**
 * Проверяет запущен ли сервис
 */
export async function isServiceRunning(): Promise<boolean> {
    try {
        const response = await sendIPCRequest({command: 'ping'}, 2000);
        return response.success && response.data === 'pong';
    } catch {
        return false;
    }
}

/**
 * Проверяет, установлен ли бинарник сервиса на диске (без попытки соединения).
 */
export function isServiceBinaryPresent(): boolean {
    try {
        return fs.existsSync(getServicePath());
    } catch {
        return false;
    }
}

/**
 * Ожидает готовности сервиса (пинг), опрашивая его с заданным шагом до общего
 * бюджета времени. Используется при автозапуске, чтобы переждать гонку, когда
 * приложение стартует раньше, чем сервис успел поднять свой IPC-канал.
 */
export async function waitForServiceReady(totalMs: number, stepMs: number): Promise<boolean> {
    const deadline = Date.now() + totalMs;
    for (;;) {
        if (await isServiceRunning()) {
            return true;
        }
        if (Date.now() >= deadline) {
            return false;
        }
        await new Promise((r) => setTimeout(r, stepMs));
    }
}

/**
 * Проверяет установлен ли сервис
 */
export async function isServiceInstalled(): Promise<boolean> {
    const servicePath = getServicePath();
    if (!fs.existsSync(servicePath)) {
        return false;
    }

    // Пробуем подключиться к сервису
    return await isServiceRunning();
}

/**
 * Получает статус сервиса
 */
export async function getServiceStatus(): Promise<ServiceStatus> {
    const servicePath = getServicePath();
    const installed = fs.existsSync(servicePath);

    if (!installed) {
        return {installed: false, running: false, isAdmin: false};
    }

    try {
        const running = await isServiceRunning();
        if (running) {
            const versionResponse = await sendIPCRequest({command: 'version'}, 2000);
            const adminResponse = await sendIPCRequest({command: 'is_admin'}, 2000);
            const isAdmin = adminResponse.success && adminResponse.data === true;
            return {
                installed: true,
                running: true,
                isAdmin: isAdmin,
                version: versionResponse.data
            };
        }
        return {installed: true, running: false, isAdmin: false};
    } catch {
        return {installed: true, running: false, isAdmin: false};
    }
}

/**
 * Устанавливает сервис (требует прав администратора)
 */
export async function installService(): Promise<boolean> {
    const servicePath = getServicePath();

    if (!fs.existsSync(servicePath)) {
        log.error('[Service] Service binary not found:', servicePath);
        return false;
    }

    log.info('[Service] Installing service...');

    return new Promise((resolve) => {
        switch (process.platform) {
            case 'win32': {
                // Windows: используем PowerShell с UAC
                const psArgs = [
                    '-Command',
                    `Start-Process -FilePath '${servicePath}' -ArgumentList '-install' -Verb RunAs -Wait`
                ];
                const ps = spawn('powershell.exe', psArgs);

                ps.on('exit', async (code) => {
                    if (code === 0) {
                        // Ждём запуска сервиса
                        await new Promise(r => setTimeout(r, 2000));
                        const running = await isServiceRunning();
                        if (running) {
                            log.info('[Service] Service installed and running');
                            storeSet('serviceMode', true);
                            resolve(true);
                        } else {
                            log.error('[Service] Service installed but not running');
                            resolve(false);
                        }
                    } else {
                        log.error('[Service] Installation failed with code:', code);
                        resolve(false);
                    }
                });

                ps.on('error', (err) => {
                    log.error('[Service] Installation error:', err);
                    resolve(false);
                });
                break;
            }

            case 'darwin': {
                // macOS: используем osascript для sudo
                const script = `do shell script "${servicePath} -install" with administrator privileges with prompt "Prizrak Box требует установки сервиса для TUN режима"`;
                const osa = spawn('osascript', ['-e', script]);

                osa.on('exit', async (code) => {
                    if (code === 0) {
                        await new Promise(r => setTimeout(r, 2000));
                        const running = await isServiceRunning();
                        storeSet('serviceMode', running);
                        resolve(running);
                    } else {
                        resolve(false);
                    }
                });

                osa.on('error', () => resolve(false));
                break;
            }

            case 'linux': {
                // Linux: используем pkexec
                const methods = ['/usr/bin/pkexec', 'pkexec', '/usr/bin/sudo', 'sudo'];

                const tryInstall = (index: number): Promise<boolean> => {
                    if (index >= methods.length) return Promise.resolve(false);

                    const method = methods[index];
                    return new Promise((res) => {
                        const proc = spawn(method, [servicePath, '-install'], {
                            env: {...process.env, DISPLAY: process.env.DISPLAY},
                        });

                        proc.on('exit', (code) => {
                            if (code === 0) {
                                setTimeout(() => {
                                    isServiceRunning().then(running => {
                                        if (running) {
                                            log.info('[Service] Service installed and running');
                                            storeSet('serviceMode', true);
                                        } else {
                                            log.error('[Service] Service installed but not running (check SELinux/systemd logs)');
                                        }
                                        res(running);
                                    });
                                }, 5000);
                            } else {
                                log.warn(`[Service] Install method ${method} failed with code: ${code}`);
                                tryInstall(index + 1).then(result => res(result));
                            }
                        });

                        proc.on('error', () => {
                            tryInstall(index + 1).then(result => res(result));
                        });
                    });
                };

                tryInstall(0).then((result) => {
                    storeSet('serviceMode', result);
                    resolve(result);
                });
                break;
            }

            default:
                resolve(false);
        }
    });
}

/**
 * Удаляет сервис (требует прав администратора)
 */
export async function uninstallService(): Promise<boolean> {
    const servicePath = getServicePath();

    if (!fs.existsSync(servicePath)) {
        return true;
    }

    log.info('[Service] Uninstalling service...');

    return new Promise((resolve) => {
        switch (process.platform) {
            case 'win32': {
                // Останавливаем и удаляем сервис через sc.exe
                const psArgs = [
                    '-Command',
                    `Start-Process -FilePath 'powershell.exe' -ArgumentList @('-NoProfile', '-Command', 'sc.exe stop PrizrakBoxService; sc.exe delete PrizrakBoxService') -Verb RunAs -Wait`
                ];
                const ps = spawn('powershell.exe', psArgs);

                ps.on('exit', (code) => {
                    storeSet('serviceMode', false);
                    resolve(code === 0);
                });

                ps.on('error', () => {
                    resolve(false);
                });
                break;
            }

            case 'darwin': {
                const script = `do shell script "${servicePath} -uninstall" with administrator privileges`;
                const osa = spawn('osascript', ['-e', script]);

                osa.on('exit', (code) => {
                    storeSet('serviceMode', false);
                    resolve(code === 0);
                });

                osa.on('error', () => resolve(false));
                break;
            }

            case 'linux': {
                const methods = ['/usr/bin/pkexec', 'pkexec', '/usr/bin/sudo', 'sudo'];

                const tryUninstall = (index: number): Promise<boolean> => {
                    if (index >= methods.length) return Promise.resolve(false);

                    const method = methods[index];
                    return new Promise((res) => {
                        const proc = spawn(method, [servicePath, '-uninstall'], {
                            env: {...process.env, DISPLAY: process.env.DISPLAY},
                        });

                        proc.on('exit', (code) => {
                            if (code === 0) res(true);
                            else tryUninstall(index + 1).then(result => res(result));
                        });

                        proc.on('error', () => tryUninstall(index + 1).then(result => res(result)));
                    });
                };

                tryUninstall(0).then((result) => {
                    storeSet('serviceMode', false);
                    resolve(result);
                });
                break;
            }

            default:
                resolve(false);
        }
    });
}

/**
 * Запускает px через сервис
 */
export async function startPxViaService(pxPath: string, addr: string, homeDir: string): Promise<boolean> {
    try {
        const response = await sendIPCRequest({
            command: 'start_px',
            data: {
                pxPath: pxPath,
                addr: addr,
                homeDir: homeDir
            }
        });

        if (response.success) {
            log.info('[Service] px started via service');
            return true;
        } else {
            log.error('[Service] Failed to start px:', response.error);
            return false;
        }
    } catch (err) {
        log.error('[Service] Failed to communicate with service:', err);
        return false;
    }
}

/**
 * Останавливает px через сервис
 */
export async function stopPxViaService(): Promise<boolean> {
    try {
        const response = await sendIPCRequest({command: 'stop_px'});
        return response.success;
    } catch {
        return false;
    }
}

/**
 * Проверяет включен ли режим сервиса
 */
export function isServiceModeEnabled(): boolean {
    const setting = storeGet("serviceMode");
    return !!setting;
}

/**
 * Показывает диалог предложения установки сервиса
 */
export async function showServiceInstallDialog(): Promise<'install' | 'skip' | 'cancel'> {
    const result = await dialog.showMessageBox({
        type: 'question',
        buttons: ['Установить сервис', 'Пропустить', 'Отмена'],
        defaultId: 0,
        cancelId: 2,
        title: 'Prizrak Box - Режим TUN',
        message: 'Для использования TUN режима требуются права администратора',
        detail: 'Вы можете установить сервис, который позволит использовать TUN режим без постоянного запуска приложения от администратора.\n\nУстановка сервиса требует однократного подтверждения прав администратора.',
    });

    switch (result.response) {
        case 0:
            return 'install';
        case 1:
            return 'skip';
        default:
            return 'cancel';
    }
}
