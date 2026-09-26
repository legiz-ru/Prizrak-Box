// Global UI services shared by the custom components in this folder:
// toasts, the danger-confirmation dialog, the blocking loading overlay and the
// stack of open modals (for Esc handling). State is module-level and reactive so
// any component or plain .ts module (util/pLoad.ts) can call into it.
import {reactive, markRaw, type Component} from "vue";

export type ToastType = 'success' | 'info' | 'warning' | 'error';

export interface ToastItem {
    id: number;
    type: ToastType;
    text: string;
}

const TOAST_DURATION = 2600;
const TOAST_LIMIT = 4;
let toastSeq = 0;

export const toastState = reactive({items: [] as ToastItem[]});

export function toast(type: ToastType, text: unknown) {
    const message = text === undefined || text === null ? '' : String(text);
    if (!message) return;
    // Same text already on screen: restart it instead of stacking duplicates
    // (Element Plus' `grouping` did the same).
    const existing = toastState.items.find(item => item.type === type && item.text === message);
    if (existing) {
        dismissToast(existing.id);
    }
    const id = ++toastSeq;
    toastState.items = [...toastState.items.slice(-(TOAST_LIMIT - 1)), {id, type, text: message}];
    window.setTimeout(() => dismissToast(id), TOAST_DURATION);
}

export function dismissToast(id: number) {
    toastState.items = toastState.items.filter(item => item.id !== id);
}

// ---- confirm ----

export interface ConfirmOptions {
    title: string;
    text?: string;
    okLabel?: string;
    cancelLabel?: string;
    /** A tabler icon component; defaults to the trash can. */
    icon?: Component;
}

interface ConfirmRequest extends ConfirmOptions {
    resolve: (ok: boolean) => void;
}

export const confirmState = reactive({current: null as ConfirmRequest | null});

export function confirm(options: ConfirmOptions): Promise<boolean> {
    // A second request while one is open answers the first with "cancel".
    confirmState.current?.resolve(false);
    return new Promise<boolean>((resolve) => {
        confirmState.current = {
            ...options,
            icon: options.icon ? markRaw(options.icon) : undefined,
            resolve,
        };
    });
}

export function settleConfirm(ok: boolean) {
    const request = confirmState.current;
    confirmState.current = null;
    request?.resolve(ok);
}

// ---- blocking loading overlay (replacement for ElLoading.service) ----

export const loadingState = reactive({count: 0, text: ''});

export function showLoading(text = ''): () => void {
    loadingState.count++;
    loadingState.text = text;
    let closed = false;
    return () => {
        if (closed) return;
        closed = true;
        loadingState.count = Math.max(0, loadingState.count - 1);
        if (!loadingState.count) loadingState.text = '';
    };
}

// ---- modal stack ----
// Each open UiModal registers here so Esc only closes the topmost one and a
// modal with closeOnEsc=false (multi-profile warning, welcome, first profile)
// swallows Esc instead of letting it reach the modal underneath.

interface ModalEntry {
    id: number;
    onEsc: () => void;
    escClosable: () => boolean;
}

let modalSeq = 0;
const modalStack: ModalEntry[] = [];

export function registerModal(onEsc: () => void, escClosable: () => boolean): () => void {
    const entry: ModalEntry = {id: ++modalSeq, onEsc, escClosable};
    modalStack.push(entry);
    return () => {
        const index = modalStack.findIndex(item => item.id === entry.id);
        if (index >= 0) modalStack.splice(index, 1);
    };
}

export function handleModalEscape(): boolean {
    const top = modalStack[modalStack.length - 1];
    if (!top) return false;
    if (top.escClosable()) top.onEsc();
    return true;
}

export function hasOpenModal() {
    return modalStack.length > 0;
}
