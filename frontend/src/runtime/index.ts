import {
    BrowserOpenURL,
    ClipboardGetText,
    ClipboardSetText,
    EventsEmit,
    EventsOn,
    StoreGet,
    StoreSet
} from "@wailsapp/runtime";

export const Events = {
    Emit: ({name, data}: { name: string; data: any }) => {
        return EventsEmit(name, data);
    },
    On: (name: string, callback: (...args: any[]) => void) => {
        return EventsOn(name, callback);
    },
};

export const Clipboard = {
    Text: async () => ClipboardGetText(),
    SetText: async (value: string) => ClipboardSetText(value),
};

export const Browser = {
    OpenURL: (url: string) => BrowserOpenURL(url),
};

export const Store = {
    get: (key: string) => StoreGet(key),
    set: (key: string, value: unknown) => StoreSet(key, value),
};
