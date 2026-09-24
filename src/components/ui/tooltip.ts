// Styled tooltips. `v-tip="text"` marks an element; so does a plain `title`
// attribute, which is converted to data-tip on first hover so the native
// tooltip never shows. One shared bubble (UiTooltipHost) appears after 300 ms
// above the element, or below it when there is no room at the top of the window.
import {reactive, type Directive} from "vue";

export const tooltipState = reactive({
    visible: false,
    text: '',
    x: 0,
    y: 0,
    below: false,
});

const DELAY = 300;
let currentEl: HTMLElement | null = null;
let timer: number | undefined;

function tipTarget(node: EventTarget | null): HTMLElement | null {
    const el = node as HTMLElement | null;
    if (!el?.closest) return null;
    const found = el.closest<HTMLElement>('[title],[data-tip]');
    if (!found) return null;
    if (found.hasAttribute('title')) {
        const text = found.getAttribute('title');
        found.removeAttribute('title');
        if (text) found.setAttribute('data-tip', text);
    }
    return found.getAttribute('data-tip') ? found : null;
}

function show(el: HTMLElement) {
    if (el === currentEl) return;
    hide();
    currentEl = el;
    timer = window.setTimeout(() => {
        if (currentEl !== el || !el.isConnected) return;
        const text = el.getAttribute('data-tip') || '';
        if (!text) return;
        const r = el.getBoundingClientRect();
        const below = r.top < 76;
        tooltipState.text = text;
        tooltipState.x = r.left + r.width / 2;
        tooltipState.y = below ? r.bottom + 9 : r.top - 9;
        tooltipState.below = below;
        tooltipState.visible = true;
    }, DELAY);
}

export function hideTooltip() {
    hide();
}

function hide() {
    window.clearTimeout(timer);
    currentEl = null;
    tooltipState.visible = false;
}

let installed = false;

export function installTooltips() {
    if (installed) return;
    installed = true;
    document.addEventListener('mouseover', (e) => {
        const el = tipTarget(e.target);
        if (el) show(el);
        else if (currentEl && !currentEl.contains(e.target as Node)) hide();
    });
    document.addEventListener('mouseout', (e) => {
        if (!currentEl) return;
        const next = e.relatedTarget as Node | null;
        if (next && currentEl.contains(next)) return;
        hide();
    });
    document.addEventListener('focusin', (e) => {
        const el = tipTarget(e.target);
        // Only for keyboard focus: a click also focuses the element.
        if (el && (e.target as HTMLElement).matches?.(':focus-visible')) show(el);
        else hide();
    });
    document.addEventListener('focusout', hide);
    document.addEventListener('mousedown', hide, true);
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') hide();
    }, true);
    window.addEventListener('scroll', hide, true);
    window.addEventListener('blur', hide);
}

function apply(el: HTMLElement, value: unknown) {
    const text = value === undefined || value === null || value === false ? '' : String(value);
    el.removeAttribute('title');
    if (text) el.setAttribute('data-tip', text);
    else el.removeAttribute('data-tip');
    if (el === currentEl && tooltipState.visible) tooltipState.text = text;
}

export const vTip: Directive<HTMLElement, unknown> = {
    mounted(el, binding) {
        apply(el, binding.value);
    },
    updated(el, binding) {
        if (binding.value !== binding.oldValue) apply(el, binding.value);
    },
    beforeUnmount(el) {
        if (el === currentEl) hide();
    },
};
