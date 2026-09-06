import {app, BrowserWindow, globalShortcut} from 'electron'
import {onMsg, sendMsg} from "./common";

const scs: Record<string, (w: BrowserWindow) => void> = {
    "showOrHide": showOrHide
}

function replaceCtrlCmd(key: string): string {
    return key.replaceAll("Ctrl", "CommandOrControl").replaceAll("Cmd", "CommandOrControl")
}

function registerOne(acc: any, mainWindow: BrowserWindow | null): void {
    if (!scs[acc.name]) {
        return
    }
    if (acc.old) {
        globalShortcut.unregister(replaceCtrlCmd(acc.old))
    }
    globalShortcut.unregister(replaceCtrlCmd(acc.key))
    sendMsg(mainWindow, "shortcut:result",
        globalShortcut.register(replaceCtrlCmd(acc.key), () => scs[acc.name](mainWindow!))
    )
}

export function initShortcut(mainWindow: BrowserWindow) {
    onMsg('shortcut:register', (acc) => {
        registerOne(acc, mainWindow)
    })
    onMsg('shortcut:unregister-all', () => {
        globalShortcut.unregisterAll()
    })
}

function showOrHide(mainWindow: BrowserWindow) {
    if (mainWindow.isVisible() && mainWindow.isFocused()) {
        app.dock?.hide();
        mainWindow.hide()
    } else {
        mainWindow.show();
        mainWindow.focus();
        app.dock?.show();
    }
}
