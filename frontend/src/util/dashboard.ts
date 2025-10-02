import type {CustomDashboard} from "@/store/webStore";
import {isHttpOrHttps} from "@/util/format";

export interface DashboardTemplate {
    key: string;
    name: string;
    url: string;
}

export interface DashboardOption extends DashboardTemplate {
    isCustom?: boolean;
}

export interface DashboardLink {
    key: string;
    name: string;
    url: string;
}

export const defaultDashboards: DashboardTemplate[] = [
    {
        key: "metacubexd",
        name: "MetaCubeXD",
        url: "https://metacubex.github.io/metacubexd/#/setup?http=true&hostname=%host&port=%port&secret=%secret",
    },
    {
        key: "yacd",
        name: "Yacd",
        url: "https://yacd.metacubex.one/?hostname=%host&port=%port&secret=%secret",
    },
    {
        key: "zashboard",
        name: "Zashboard",
        url: "https://board.zash.run.place/#/setup?http=true&hostname=%host&port=%port&secret=%secret",
    },
];

export const createCustomDashboardOptions = (dashboards: CustomDashboard[]): DashboardOption[] =>
    dashboards
        .map((entry, index) => ({
            key: `custom-${index}`,
            name: entry.name?.trim() ?? "",
            url: entry.url?.trim() ?? "",
            isCustom: true,
        }))
        .filter((entry) => entry.name !== "" && entry.url !== "");

export const resolveDashboardOptions = (customDashboards: CustomDashboard[]): DashboardOption[] => [
    ...defaultDashboards,
    ...createCustomDashboardOptions(customDashboards),
];

export const formatDashboardUrl = (
    template: string,
    context: { host: string; port: string; secret: string },
): string => template
    .replace(/%host/g, context.host)
    .replace(/%port/g, context.port)
    .replace(/%secret/g, context.secret);

export const createDashboardLinks = (
    customDashboards: CustomDashboard[],
    context: { host: string; port: string; secret: string },
): DashboardLink[] =>
    resolveDashboardOptions(customDashboards)
        .map((dashboard, index) => ({
            key: dashboard.key || `dashboard-${index}`,
            name: dashboard.name,
            url: formatDashboardUrl(dashboard.url, context),
        }))
        .filter((link) => link.name !== "" && link.url !== "" && isHttpOrHttps(link.url));
