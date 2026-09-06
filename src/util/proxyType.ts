// Maps a proxy adapter type to the icon and tooltip shown in the proxy list.
//
// The badge in the Proxies list used to print the type as text, but that slot is
// shared: getDisplayType() (src/api/proxies/index.ts) substitutes the panel's
// serverDescription when it sends one, which pushed the type out entirely. The
// icon now always carries the type and the description keeps the text, so both
// survive.
//
// Types come from the core's AdapterType.String() (constant/adapters.go). Three
// families share the slot, because a group's members can themselves be groups:
//   - group types    Selector, URLTest, Fallback, LoadBalance, Relay, Smart
//   - service types  Direct, Reject, Dns, ...
//   - protocols      Vless, Trojan, Hysteria2, ...
//
// Icons are MDI except for the few protocols with a usable brand mark; see
// src/assets/icons/proto/ATTRIBUTION.md.

import IconGestureTap from '~icons/mdi/gesture-tap';
import IconClockFast from '~icons/mdi/clock-fast';
import IconBackupRestore from '~icons/mdi/backup-restore';
import IconScaleBalance from '~icons/mdi/scale-balance';
import IconTransitConnection from '~icons/mdi/transit-connection-variant';
import IconBrain from '~icons/mdi/brain';

import IconArrowRightBold from '~icons/mdi/arrow-right-bold-outline';
import IconCancel from '~icons/mdi/cancel';
import IconCloseOctagon from '~icons/mdi/close-octagon-outline';
import IconForward from '~icons/mdi/forward';
import IconDebugStepOver from '~icons/mdi/debug-step-over';
import IconSync from '~icons/mdi/sync';
import IconDns from '~icons/mdi/dns-outline';
import IconCog from '~icons/mdi/cog-outline';

import IconAirplane from '~icons/mdi/airplane';
import IconAlphaV from '~icons/mdi/alpha-v-box-outline';
import IconLightningBolt from '~icons/mdi/lightning-bolt-outline';
import IconRocket from '~icons/mdi/rocket-launch-outline';
import IconWeb from '~icons/mdi/web';
import IconConsole from '~icons/mdi/console';
import IconNetwork from '~icons/mdi/network-outline';

import IconDominoMask from '~icons/mdi/domino-mask';
import IconSemanticWeb from '~icons/mdi/semantic-web';

import IconShieldLock from '~icons/mdi/shield-lock-outline';
import IconSourceBranch from '~icons/mdi/source-branch';
import IconHelpCircle from '~icons/mdi/help-circle-outline';

import IconXray from '~icons/proto/xray';
import IconTrojan from '~icons/proto/trojan';
import IconOpenVpn from '~icons/proto/openvpn';
import IconTailscale from '~icons/proto/tailscale';
import IconWireGuard from '~icons/proto/wireguard';
import IconTrustTunnel from '~icons/proto/trusttunnel';
import IconSudoku from '~icons/proto/sudoku';
import IconGostRelay from '~icons/proto/gost-relay';

type IconComponent = any;

// Group types. Keyed lowercase so a core that changes casing cannot break the
// lookup silently.
const GROUP_ICONS: Record<string, IconComponent> = {
    selector: IconGestureTap,
    urltest: IconClockFast,
    fallback: IconBackupRestore,
    loadbalance: IconScaleBalance,
    relay: IconTransitConnection,
    smart: IconBrain,
};

const SERVICE_ICONS: Record<string, IconComponent> = {
    direct: IconArrowRightBold,
    reject: IconCancel,
    rejectdrop: IconCloseOctagon,
    pass: IconForward,
    passrule: IconDebugStepOver,
    rematch: IconSync,
    dns: IconDns,
    compatible: IconCog,
};

// Only protocols with a non-arbitrary mapping are named. Inventing a glyph for
// each of the remaining ones would read as noise, so they fall through to the
// protocol default below.
const PROTOCOL_ICONS: Record<string, IconComponent> = {
    vless: IconXray,
    trojan: IconTrojan,
    trusttunnel: IconTrustTunnel,
    vmess: IconAlphaV,
    shadowsocks: IconAirplane,
    shadowsocksr: IconAirplane,
    hysteria: IconLightningBolt,
    hysteria2: IconLightningBolt,
    tuic: IconRocket,
    wireguard: IconWireGuard,
    tailscale: IconTailscale,
    openvpn: IconOpenVpn,
    http: IconWeb,
    socks5: IconNetwork,
    ssh: IconConsole,
    // A domino mask and a semantic-web glyph aren't literal depictions of
    // either protocol — chosen deliberately (per product decision) over a
    // brand mark, since neither Masque nor AnyTLS has one that reduces to a
    // legible monochrome badge.
    masque: IconDominoMask,
    anytls: IconSemanticWeb,
    sudoku: IconSudoku,
    gostrelay: IconGostRelay,
};

export type ProxyTypeKind = 'group' | 'service' | 'protocol' | 'unknown';

/**
 * Classifies a node by the shape of its data rather than by a hardcoded list of
 * type names: anything the core reports with an `all` member list is a group.
 * A future group type therefore lands in the group branch on its own.
 */
export function proxyTypeKind(type: string | undefined, isGroup: boolean): ProxyTypeKind {
    const key = (type ?? '').toLowerCase();
    if (!key) return 'unknown';
    if (key === 'unknown') return 'unknown';
    if (isGroup || key in GROUP_ICONS) return 'group';
    if (key in SERVICE_ICONS) return 'service';
    return 'protocol';
}

/** The icon for a type, falling back per family so new types stay sensible. */
export function proxyTypeIcon(type: string | undefined, isGroup = false): IconComponent {
    const key = (type ?? '').toLowerCase();
    switch (proxyTypeKind(type, isGroup)) {
        case 'group':
            return GROUP_ICONS[key] ?? IconSourceBranch;
        case 'service':
            return SERVICE_ICONS[key] ?? IconCog;
        case 'protocol':
            return PROTOCOL_ICONS[key] ?? IconShieldLock;
        default:
            return IconHelpCircle;
    }
}

/**
 * Tooltip text for a type. Group and service types get a written explanation of
 * what they do; protocols get the name, since there is nothing to explain.
 *
 * `t` is the vue-i18n translate function, passed in so this module stays free of
 * an i18n instance and works from any component.
 */
export function proxyTypeTooltip(type: string | undefined, isGroup: boolean, t: (k: string, v?: any) => string): string {
    const raw = (type ?? '').trim();
    const key = raw.toLowerCase();
    const kind = proxyTypeKind(raw, isGroup);

    if (kind === 'unknown') {
        return t('proxies.type.unknown');
    }
    // A named group or service type has its own description; an unrecognised one
    // (a type the core gained after this build) has none, so it falls back to
    // naming itself rather than showing a missing-translation key.
    if ((kind === 'group' && key in GROUP_ICONS) || (kind === 'service' && key in SERVICE_ICONS)) {
        return t(`proxies.type.${key}`);
    }
    if (kind === 'protocol') {
        return t('proxies.type.protocol', { type: raw });
    }
    return raw;
}
