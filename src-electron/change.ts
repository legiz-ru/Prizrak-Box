import * as fs from 'fs-extra';
import log from './log';
import {storeSet} from "./store";
import {app} from "electron";
import * as path from "path";

/**
 * 修改配置文件目录
 */
export async function doChange(data: string, addr: string) {
    let destDir = data
    if (!data.endsWith("Prizrak-Box-V3")) {
        destDir = path.join(data, 'Prizrak-Box-V3');
    }

    if (log.getAppConfigDir() !== destDir) {
        await fs.move(log.getAppConfigDir(), destDir, {overwrite: true});
        console.log('目录移动成功！新路径：', destDir);
    } else {
        const files = await fs.readdir(destDir);
        if (files.length === 0) {
            await fs.move(log.getAppConfigDir(), destDir, {overwrite: true});
            console.log('目录移动成功！新路径：', destDir);
        }
    }

    // 更新数据库
    storeSet("appConfigDir", path.dirname(destDir));

    // Перезапустить приложение для применения изменений
    app.relaunch();
    app.exit(0);
}
