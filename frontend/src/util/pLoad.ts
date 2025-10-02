import {ElLoading, ElMessage} from "element-plus";

function translateErrorSegment(key: string, fallback: string): string {
    const translator = (window as any)?.pxTranslate;
    if (typeof translator === "function") {
        try {
            const result = translator(key);
            if (typeof result === "string" && result !== key) {
                return result;
            }
        } catch (error) {
            console.error("Failed to translate error segment", error);
        }
    }

    return fallback;
}

function normalizeErrorMessage(message: unknown): string {
    if (message === undefined || message === null) {
        return "";
    }

    const text = typeof message === "string" ? message : String(message);

    const replacements: Array<[RegExp, string, string]> = [
        [/请求失败/g, "errors.request-failed", "Request failed"],
        [/发送请求失败/g, "errors.request-send-failed", "Failed to send request"],
        [/创建请求失败/g, "errors.request-create-failed", "Failed to create request"],
        [/响应内容为空/g, "errors.response-empty", "Response body is empty"],
        [/未知原因/g, "errors.unknown-reason", "Unknown reason"],
    ];

    let translated = text;
    for (const [pattern, key, fallback] of replacements) {
        if (pattern.test(translated)) {
            const value = translateErrorSegment(key, fallback);
            translated = translated.replace(pattern, value);
        }
    }

    return translated;
}

export async function pLoad(tip: any, callback: any) {
    const loading = ElLoading.service({
        lock: true,
        text: tip,
        background: "rgba(0, 0, 0, 0.2)",
    });
    await callback();
    loading.close();
}


export async function copy(textToCopy: any, t: any) {
    try {
        await navigator.clipboard.writeText(textToCopy);
        pSuccess(t("copy.success"));
    } catch (error) {
        error(t("copy.fail"));
    }
}

export function pSuccess(msg: any) {
    ElMessage({
        message: msg,
        type: "success",
        grouping: true
    });
}

export function pError(msg: any) {
    ElMessage({
        message: normalizeErrorMessage(msg),
        type: "error",
        duration: 5000,
        grouping: true
    });
}

export function pWarning(msg: any) {
    ElMessage({
        message: msg,
        type: "warning",
        duration: 5000,
        grouping: true
    });
}
