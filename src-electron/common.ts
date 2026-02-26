import {app, BrowserWindow, ipcMain} from "electron";

export const AppName = "Prizrak-Box"
export const AppBaseDir = "Prizrak-Box-V3"

export const IsDev = !app.isPackaged

export const sendMsg = (mainWindow: BrowserWindow | null, name: string, ...args: any[]) => {
    mainWindow?.webContents.send(`px_${name}`, ...args);
}

export const onMsg = (name: string, cb: (val: any) => void) => {
    ipcMain.on(`px_${name}`, (_e, val) => cb(val));
};
