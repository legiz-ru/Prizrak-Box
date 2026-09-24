import type {Directive} from "vue";

declare module "vue" {
    export interface GlobalDirectives {
        vTip: Directive<HTMLElement, unknown>;
    }
}

export {};
