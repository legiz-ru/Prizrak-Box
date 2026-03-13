export class Profile {
    id!: string;
    type!: number; // 1: 远程订阅, 2: 本地配置, 3: 爬取合并
    title?: string; // 可选
    headerTitle?: string; // 仅当 profile-title 标头存在时
    order!: string;
    primary?: boolean;
    selectionOrder?: number;
    selected?: boolean; // 可选
    path!: string;
    content?: string | ArrayBuffer; // 可选
    used?: bigint; // 可选
    available?: bigint; // 可选
    total?: bigint; // 可选
    expire?: string; // 可选
    interval?: string; // 可选
    home?: string; // 可选
    support?: string; // 可选
    logo?: string; // 可选
    announce?: string; // 可选
    announceUrl?: string; // 可选
    update?: string; // 可选
    template?: string; // 可选
    pxdTemplateUrl?: string;
    pxdTemplateScheme?: string;
}

export interface ProfileSelectionPayload {
    id: string;
    selected?: boolean;
    exclusive?: boolean;
}
