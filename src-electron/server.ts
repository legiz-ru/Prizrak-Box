import express from "express";
import type {RequestHandler} from "express";
import path from "path";
import {existsSync, mkdirSync, promises as fsPromises} from "fs";
import {randomUUID} from "crypto";
import {app as electronApp} from "electron";
import log from './log';

const BODY_LIMIT = '20mb';

const app = express();

let storedPort: string | undefined;
let storedSecret: string | undefined;
let listenAddr: string | undefined;
let goFlag: Function;

// 解析请求参数
app.use(express.urlencoded({extended: true, limit: BODY_LIMIT}));
app.use(express.json({limit: BODY_LIMIT}));

// **提供静态文件服务**
app.use(express.static(path.join(__dirname, '../renderer/px_window')));

const sanitizeThemeId = (value: unknown) => {
    if (typeof value !== 'string') {
        return 'custom';
    }
    const cleaned = value.replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 40);
    return cleaned.length > 0 ? cleaned : 'custom';
};

const sanitizeExtension = (originalName: unknown) => {
    if (typeof originalName !== 'string') {
        return '.png';
    }
    const ext = path.extname(originalName).toLowerCase();
    if (!ext || !/^[.][a-z0-9]+$/.test(ext)) {
        return '.png';
    }
    return ext;
};

const ensureImagesDir = () => {
    const baseDir = path.join(electronApp.getPath('temp'), 'Prizrak-Box-V3', 'images');
    if (!existsSync(baseDir)) {
        mkdirSync(baseDir, {recursive: true});
    }
    return baseDir;
};

let cachedStaticMiddleware: RequestHandler | null = null;

const customImageMiddleware: RequestHandler = (req, res, next) => {
    try {
        if (!cachedStaticMiddleware) {
            cachedStaticMiddleware = express.static(ensureImagesDir());
        }
        return cachedStaticMiddleware(req, res, next);
    } catch (error) {
        log.error('Failed to serve custom background image', error);
        res.status(500).end();
    }
};

app.use('/user-images', customImageMiddleware);

const extractRelativePath = (value: unknown) => {
    if (typeof value !== 'string') {
        return null;
    }
    if (!value.startsWith('/user-images/')) {
        return null;
    }
    const normalized = path.normalize(value);
    if (!normalized.startsWith('/user-images/')) {
        return null;
    }
    return normalized;
};

const removePreviousImage = async (relativePath: string) => {
    const dir = ensureImagesDir();
    const fileName = relativePath.replace('/user-images/', '');
    if (!fileName) {
        return;
    }
    const targetPath = path.join(dir, fileName);
    if (!targetPath.startsWith(dir)) {
        log.warn('Attempt to delete file outside of custom images directory ignored:', targetPath);
        return;
    }
    try {
        await fsPromises.unlink(targetPath);
    } catch (error) {
        const err = error as NodeJS.ErrnoException;
        if (err.code !== 'ENOENT') {
            log.warn('Failed to remove previous custom background image', err);
        }
    }
};

app.post('/api/custom-background', async (req, res) => {
    const {themeId, dataUrl, fileName, previousPath} = req.body ?? {};

    if (typeof dataUrl !== 'string' || dataUrl.length === 0) {
        return res.status(400).json({error: 'INVALID_DATA'});
    }

    const dataUrlMatch = /^data:(.+);base64,(.+)$/i.exec(dataUrl);
    if (!dataUrlMatch) {
        return res.status(400).json({error: 'INVALID_DATA_URL'});
    }

    const buffer = Buffer.from(dataUrlMatch[2], 'base64');
    const safeThemeId = sanitizeThemeId(themeId);
    const extension = sanitizeExtension(fileName);
    const dir = ensureImagesDir();
    const targetName = `${safeThemeId}-${randomUUID()}${extension}`;
    const targetPath = path.join(dir, targetName);

    try {
        await fsPromises.writeFile(targetPath, buffer);
    } catch (error) {
        log.error('Failed to write custom background image', error);
        return res.status(500).json({error: 'WRITE_FAILED'});
    }

    const relative = `/user-images/${targetName}`;

    if (typeof previousPath === 'string') {
        const sanitizedPrevious = extractRelativePath(previousPath);
        if (sanitizedPrevious) {
            removePreviousImage(sanitizedPrevious).catch((error) => {
                log.warn('Failed to cleanup previous custom background image', error);
            });
        }
    }

    return res.status(200).json({url: relative});
});

// **检测 PX 后端是否存活**
app.get("/pxAlive", (req, res) => {
    res.status(200).send("alive");
});

// **存储 PX 后端的端口和密钥**
// @ts-ignore
app.get("/pxStore", (req, res) => {
    const {port, secret} = req.query;

    if (!port || !secret) {
        return res.status(400).json({error: "Missing port or secret parameter"});
    }

    storedPort = port as string;
    storedSecret = secret as string;

    log.info("Retrieved port:", storedPort);
    log.info("Retrieved secret:", storedSecret);
    res.status(200).send("ok");

    if (goFlag) {
        goFlag(); // 通知主流程可以继续了
    }
});

// **处理所有未匹配的请求，返回 index.html**
app.use((req, res) => {
    res.redirect(302, '/index.html');
});


// **启动服务器**
export const startServer = (c1: Function, c2: Function) => {
    goFlag = c1
    const server = app.listen(0, "127.0.0.1", () => {
        // @ts-ignore
        listenAddr = `127.0.0.1:${server.address().port}`;
        c2(listenAddr)
    });
}

// 获取端口密钥
export const storeInfo = {
    port: () => storedPort,
    secret: () => storedSecret,
    listenAddr: () => listenAddr,
}
