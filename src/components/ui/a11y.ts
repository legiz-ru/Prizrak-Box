// Keyboard model for the whole app (ported from the mockup's _syncDialogs /
// _trap / _onFocusIn):
//  - every element with aria-modal="true" that appears gets focus on its first
//    field (or first button); when it disappears focus goes back to the element
//    that opened it — for an item of a [data-dd] list, to the list's trigger;
//  - Tab / Shift+Tab cycle inside the topmost modal;
//  - ArrowUp / ArrowDown move between [data-dd-item]s of the focused list;
//  - Enter / Space activate non-native [role=button] / [role=switch] elements;
//  - Esc closes the topmost modal (see services.handleModalEscape).
import {handleModalEscape} from "./services";

const FOCUSABLE = [
    'button:not([disabled])',
    '[role="button"]',
    '[role="switch"]',
    'input:not([disabled]):not([type="file"]):not([type="hidden"])',
    'textarea:not([disabled])',
    'select:not([disabled])',
    'a[href]',
    '[tabindex]:not([tabindex="-1"])',
].join(',');

export function focusables(root: Element): HTMLElement[] {
    return [...root.querySelectorAll<HTMLElement>(FOCUSABLE)]
        .filter(el => el.offsetParent !== null && !el.closest('[aria-hidden="true"]') && el.tabIndex !== -1);
}

function topDialog(): HTMLElement | null {
    const dialogs = [...document.querySelectorAll<HTMLElement>('[aria-modal="true"]')]
        .filter(d => d.offsetParent !== null || d.getClientRects().length);
    return dialogs[dialogs.length - 1] ?? null;
}

let installed = false;

export function installA11y() {
    if (installed) return;
    installed = true;

    const seen = new WeakSet<Element>();
    let stack: Array<{ el: HTMLElement; ret: HTMLElement | null }> = [];
    let lastFocus: HTMLElement | null = null;
    let lastTrigger: HTMLElement | null = null;

    const triggerOf = (el: HTMLElement): HTMLElement => {
        const dd = el.closest('[data-dd]');
        return (dd?.querySelector<HTMLElement>('[data-dd-trigger]')) ?? el;
    };

    const syncDialogs = () => {
        document.querySelectorAll<HTMLElement>('[aria-modal="true"]').forEach(dialog => {
            if (seen.has(dialog)) return;
            seen.add(dialog);
            let ret = document.activeElement as HTMLElement | null;
            if (!ret || ret === document.body) {
                ret = lastTrigger?.isConnected ? lastTrigger : lastFocus;
            }
            if (ret && !dialog.contains(ret)) {
                const dd = ret.closest('[data-dd]');
                if (dd && !dialog.contains(dd)) ret = triggerOf(ret);
            }
            stack.push({el: dialog, ret});
            requestAnimationFrame(() => {
                if (!dialog.isConnected || dialog.contains(document.activeElement)) return;
                const list = focusables(dialog);
                const first = list.find(el => /INPUT|TEXTAREA/.test(el.tagName) && !el.hasAttribute('readonly')) ?? list[0];
                (first ?? dialog).focus({preventScroll: true});
            });
        });
        stack = stack.filter(entry => {
            if (entry.el.isConnected) return true;
            if (entry.ret?.isConnected) entry.ret.focus({preventScroll: true});
            return false;
        });
    };

    new MutationObserver(syncDialogs).observe(document.body, {childList: true, subtree: true});

    document.addEventListener('focusin', (e) => {
        const target = e.target as HTMLElement | null;
        if (!target?.closest || target.closest('[aria-modal="true"]')) return;
        lastFocus = target;
        lastTrigger = triggerOf(target);
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Tab') {
            const dialog = topDialog();
            if (!dialog) return;
            const list = focusables(dialog);
            if (!list.length) {
                e.preventDefault();
                dialog.focus();
                return;
            }
            const index = list.indexOf(document.activeElement as HTMLElement);
            if (e.shiftKey && index <= 0) {
                e.preventDefault();
                list[list.length - 1].focus();
            } else if (!e.shiftKey && (index === -1 || index === list.length - 1)) {
                e.preventDefault();
                list[0].focus();
            }
            return;
        }
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
            const active = document.activeElement as HTMLElement | null;
            const dd = active?.closest?.('[data-dd]');
            if (!dd) return;
            const items = focusables(dd).filter(el => el.hasAttribute('data-dd-item'));
            if (!items.length) return;
            e.preventDefault();
            const index = items.indexOf(active!);
            const next = index === -1
                ? (e.key === 'ArrowDown' ? 0 : items.length - 1)
                : (index + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length;
            items[next].focus();
        }
    }, true);

    window.addEventListener('keydown', (e) => {
        if (e.defaultPrevented) return;
        if (e.key === 'Escape') {
            if (handleModalEscape()) e.preventDefault();
            return;
        }
        const target = e.target as HTMLElement | null;
        if ((e.key === 'Enter' || e.key === ' ') && target?.getAttribute
            && /^(button|switch)$/.test(target.getAttribute('role') || '')
            && !/^(BUTTON|INPUT|TEXTAREA|A)$/.test(target.tagName)) {
            e.preventDefault();
            target.click();
        }
    });
}
