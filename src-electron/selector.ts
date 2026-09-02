import {dialog} from 'electron';

/**
 * 选择目录
 */
export async function selectDirectory(): Promise<string | null> {
    const result = await dialog.showOpenDialog({
        properties: ['openDirectory', 'createDirectory']
    });

    if (result.canceled || result.filePaths.length === 0) {
        return null;
    }

    return result.filePaths[0];
}
