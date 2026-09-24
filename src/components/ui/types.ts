import type {Component} from "vue";

export interface UiSelectOption<V> {
    value: V;
    label: string;
    disabled?: boolean;
}

export interface UiPillOption<V> {
    value: V;
    label?: string;
    icon?: Component;
    tip?: string;
}
