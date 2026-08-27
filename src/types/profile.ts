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
    hwidActive?: boolean;
    ageSecretKey?: string;
    fallbackUrl?: string;
    fallbackDomain?: string;
    globalModeDisabled?: boolean;
    renewUrl?: string; // 可选 — subscription-renew-url
    notifyExpireDays?: number[]; // 可选 — notify-expire-days
    notifyTrafficPercent?: number[]; // 可选 — notify-traffic-percent
    pendingAlerts?: SubscriptionAlert[]; // 可选 — ещё не показанные напоминания
    clockSkewSeconds?: number; // 可选 — рассинхрон часов панели/устройства (заголовок Date)
    clockSkewAtSeconds?: number; // 可选 — когда измерено (unix seconds устройства)
}

export interface SubscriptionAlert {
    kind: 'expired' | 'expires_in' | 'traffic_used';
    days?: number; // при kind === 'expires_in'
    percent?: number; // при kind === 'traffic_used'
}

export interface ProfileSelectionPayload {
    id: string;
    selected?: boolean;
    exclusive?: boolean;
}
