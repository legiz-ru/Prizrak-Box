// Prefixes proxy/group names that start with an ISO country code ("US · Los
// Angeles", "JP-Tokyo 01", "DE_01") with the matching flag emoji, unless the
// name already carries a flag. The emoji is rendered by the bundled Twemoji
// country-flag font (styles/global.css), so it looks the same on Windows.

const REGIONAL_A = 0x1f1e6;
const HAS_FLAG = /[\u{1F1E6}-\u{1F1FF}]{2}/u;
// Two capital letters followed by a separator, a digit or the end of the name.
const LEADING_CODE = /^([A-Z]{2})(?=[\s·\-_|:.\d]|$)/;

// ISO 3166-1 alpha-2 codes (UK is accepted as an alias of GB).
const CODES = new Set((
    'AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ ' +
    'CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR ' +
    'GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO ' +
    'JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR ' +
    'MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO ' +
    'RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV ' +
    'TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW UK'
).split(' '));

function flagOf(code: string): string {
    const cc = code === 'UK' ? 'GB' : code;
    return String.fromCodePoint(...[...cc].map(ch => REGIONAL_A + ch.charCodeAt(0) - 65));
}

export function withFlag(name: string | undefined | null): string {
    if (typeof name !== 'string' || !name) return name ?? '';
    if (HAS_FLAG.test(name)) return name;
    const match = LEADING_CODE.exec(name);
    if (!match || !CODES.has(match[1])) return name;
    return `${flagOf(match[1])} ${name}`;
}
