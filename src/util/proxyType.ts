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
// Icons are Tabler (@iconify-json/tabler).

import IconPointer from '~icons/tabler/pointer';
import IconClock from '~icons/tabler/clock';
import IconReload from '~icons/tabler/reload';
import IconScale from '~icons/tabler/scale';
import IconRoute from '~icons/tabler/route';
import IconBrain from '~icons/tabler/brain';

import IconArrowRight from '~icons/tabler/arrow-right';
import IconBan from '~icons/tabler/ban';
import IconCircleX from '~icons/tabler/circle-x';
import IconTrackNext from '~icons/tabler/player-track-next';
import IconForwardUp from '~icons/tabler/arrow-forward-up';
import IconRefresh from '~icons/tabler/refresh';
import IconServer from '~icons/tabler/server-2';
import IconSettings from '~icons/tabler/settings';

import IconActivity from '~icons/tabler/activity';
import IconBolt from '~icons/tabler/bolt';
import IconLetterV from '~icons/tabler/letter-v';
import IconRocket from '~icons/tabler/rocket';
import IconWorld from '~icons/tabler/world';
import IconTerminal from '~icons/tabler/terminal-2';
import IconNetwork from '~icons/tabler/network';
import IconMask from '~icons/tabler/mask';

import IconShieldLock from '~icons/tabler/shield-lock';
import IconGitBranch from '~icons/tabler/git-branch';
import IconHelpCircle from '~icons/tabler/help-circle';

type IconComponent = any;

// Group types. Keyed lowercase so a core that changes casing cannot break the
// lookup silently.
const GROUP_ICONS: Record<string, IconComponent> = {
    selector: IconPointer,
    urltest: IconClock,
    fallback: IconReload,
    loadbalance: IconScale,
    relay: IconRoute,
    smart: IconBrain,
};

const SERVICE_ICONS: Record<string, IconComponent> = {
    direct: IconArrowRight,
    reject: IconBan,
    rejectdrop: IconCircleX,
    pass: IconTrackNext,
    passrule: IconForwardUp,
    rematch: IconRefresh,
    dns: IconServer,
    compatible: IconSettings,
};

// Only protocols with a non-arbitrary mapping are named. Inventing a glyph for
// each of the remaining ones would read as noise, so they fall through to the
// protocol default below.
// Only protocols with a non-arbitrary mapping are named; the rest fall through
// to the shield-lock protocol default. Tabler has no brand marks, so the former
// Xray/Trojan/WireGuard/… logos are gone as well.
const PROTOCOL_ICONS: Record<string, IconComponent> = {
    vmess: IconLetterV,
    shadowsocks: IconActivity,
    shadowsocksr: IconActivity,
    hysteria: IconBolt,
    hysteria2: IconBolt,
    tuic: IconRocket,
    http: IconWorld,
    socks5: IconNetwork,
    ssh: IconTerminal,
    masque: IconMask,
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
            return GROUP_ICONS[key] ?? IconGitBranch;
        case 'service':
            return SERVICE_ICONS[key] ?? IconSettings;
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
