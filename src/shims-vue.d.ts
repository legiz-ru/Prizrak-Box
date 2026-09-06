declare module "*.vue" {
    import { DefineComponent } from "vue";
    const component: DefineComponent<{}, {}, any>;
    export default component;
}

// unplugin-icons virtual modules. The auto-imported <icon-*/> components are
// declared in components.d.ts; this covers importing an icon directly, which is
// how util/proxyType.ts builds its type-to-icon map for <component :is>.
declare module "~icons/*" {
    import { DefineComponent } from "vue";
    const component: DefineComponent<{}, {}, any>;
    export default component;
}
