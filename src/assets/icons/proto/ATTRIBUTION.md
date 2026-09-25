# Protocol logo attribution

Brand marks used to identify proxy protocols in the Proxies list. Each file here
is a derivative of an upstream logo, normalised for use as an icon (see
"Normalisation" below). Sources and licences:

| File | Protocol | Source | Licence |
|---|---|---|---|
| `xray.svg` | `Vless` | [arpicme/Proxy-App-Icon-set](https://github.com/arpicme/Proxy-App-Icon-set) — `full_size/XRAY.svg`; the mark originates with [XTLS/Xray-core](https://github.com/XTLS/Xray-core) (MPL-2.0) | Upstream repository states no licence |
| `trojan.svg` | `Trojan` | [trojan-gfw/logo](https://github.com/trojan-gfw/logo) — "Trojan-GFW Logo Transparent White" | MIT |
| `trusttunnel.svg` | `TrustTunnel` | [TrustTunnel/trusttunnel.org](https://github.com/TrustTunnel/trusttunnel.org) — `assets/logo.svg` | Upstream repository states no licence |
| `sudoku.svg` | `Sudoku` | GitHub org avatar, `https://avatars.githubusercontent.com/u/246639101` (a paper-plane mark) | GitHub-hosted avatar image; no separate licence file located |
| `gost-relay.svg` | `GostRelay` | [go-gost/gost-ui](https://github.com/go-gost/gost-ui) — `public/icon.png` (the project's own app icon; `gost.run`'s 48px favicon is a lower-detail crop of the same mark and was not used) | Upstream repository states no licence |
| `openvpn.svg` | `OpenVPN` | svgrepo.com, `openvpn-svgrepo-com.svg` | GPL, per svgrepo.com (site itself could not be independently re-checked — see below) |
| `tailscale.svg` | `Tailscale` | svgrepo.com, `tailscale-svgrepo-com.svg` | GPL, per svgrepo.com (ditto) |
| `wireguard.svg` | `WireGuard` | svgrepo.com (second source; the first attempt is documented below as rejected) | GPL, per svgrepo.com (ditto) |

Two of the three upstream repositories carry no licence file, which by default
means all rights reserved. They are used here nominatively — to identify the
protocol a node speaks — not as branding for this application. If any rights
holder objects, the mapping falls back to a generic MDI glyph with no code
change beyond the icon map.

Logos are trademarks of their respective owners. A copyright licence on a logo
file is not a trademark licence.

## Normalisation

Every file was put through the same pipeline so the marks sit consistently
alongside the MDI glyphs used for the remaining protocols:

1. **Monochrome.** All fills become `currentColor`, so a single asset works in
   both the light and dark themes and needs no per-theme pair. Where a logo is
   genuinely two-tone, the secondary tone is carried as an `opacity` rather than
   a second colour.
2. **Wordmarks removed.** `trusttunnel.svg` keeps only the mark; the upstream
   file is a lockup 147 units wide, of which the mark occupies the first ~24.
   `trojan.svg` drops a hidden backing rectangle present in the source.
3. **24x24 canvas.** Content is scaled to a 20x20 optical box and centred inside
   MDI's 24x24 viewBox. Bounding boxes were measured with `getBBox()` in a real
   browser, not read off the source `viewBox` — three of the sources carry
   nested transforms or padding that make the two disagree.

Sized by the consuming element; the files declare no `width`/`height`.

## Raster sources (sudoku.svg, gost-relay.svg)

Both started as raster PNGs, not vector art, so the pipeline above ran on a
traced outline instead of an edited original:

1. **Isolate the mark from its background.** `sudoku.svg`'s source is a GitHub
   avatar — a paper-plane glyph on a flat rounded-square chip — so the chip
   colour (`rgb(245,169,184)`) was measured and every pixel far from the
   glyph's own colours (white / `rgb(91,206,250)`) was dropped, leaving just
   the plane. `gost-relay.svg`'s source PNG is alpha-transparent outside the
   mark, but the ghost's two eyes are opaque white pixels *inside* it — an
   alpha-only mask (the first attempt) treats black body and white eyes alike
   as "opaque", losing the eyes entirely. The mask instead thresholds by
   luminance among the opaque pixels (dark = body, light = eye), so the eyes
   come through as holes potrace traces as such via the path's own fill rule,
   not as a separate colour layer.
2. **Trace to vector.** [`potrace`](http://potrace.sourceforge.net/) (GPL-2.0,
   Debian package, used as a build tool only — its licence does not attach to
   its output) converts the bitmap mask to path outlines. It treats dark
   pixels as ink by default, so the mask was traced with `--invert` to trace
   the light (foreground) pixels instead — the first attempt without it
   produced the *background's* outline, caught by the same bounding-box check
   described above returning the full canvas instead of the glyph's real
   extent.
3. Same 24x24 fit and `currentColor` treatment as every other file here.

## Stroke-based sources (openvpn.svg, tailscale.svg)

svgrepo.com's pages sit behind a bot-protection interstitial ("Vercel Security
Checkpoint") that automated fetching could not pass — every scripted attempt
got the challenge page back, not the SVG, whether fetched directly or through
a real browser. Left alone at the time rather than scripting around it; the
user then supplied the three source files' content directly, sidestepping the
problem entirely.

Unlike every icon above, these three are drawn as strokes (`fill:none;
stroke:#000`), not filled silhouettes, so the pipeline differs from "Trace to
vector" up to the point both meet again at the 24x24 fit:

1. Each source's `<style>`/CSS-class indirection (`.cls-1{fill:none;
   stroke:#000000;...}`) is dropped in favour of `fill="none" stroke=
   "currentColor"` baked directly onto one wrapping `<g>` — consistent with
   every other file here avoiding embedded `<style>` blocks, which have no
   guaranteed scoping once several such icons are inlined as components on the
   same page. Geometry (`d`, `cx`/`cy`/`r`, and OpenVPN's own `transform=
   "matrix(...)"` on its second path) is untouched.
2. **Bounding box, same rule as everywhere else in this file.** `getBBox()`
   excludes stroke width by spec (it measures geometry, not paint), so it is
   safe to measure before stroke width is finalised — and doing so surfaces
   the same kind of discrepancy the raster traces had: OpenVPN's real content
   bbox (`~50x49`) is larger than its own declared `viewBox` (`46x46`) because
   the ring's `matrix(...)` transform bleeds slightly past the nominal canvas.
3. **One target stroke weight for both, not each source's own.** OpenVPN's
   source stroke width is already close to a sane 24-box line-icon proportion;
   Tailscale's relies on the SVG default of 1 unit on a 48-unit box — visibly
   thinner, and at Tailscale's icon size that default width made its two
   concentric-ring nodes merge into what looked like solid filled circles
   rather than rings, an il­legible blob at 14-18px in an early attempt. Both
   now target the same final stroke width (in the *already-scaled* 24-unit
   output space — 1.7 for OpenVPN, 1.0 for Tailscale's finer ring detail),
   solved backwards to a pre-scale `stroke-width` value, so they read as the
   same weight of line as each other rather than each carrying over its
   source's incidental proportions.

**WireGuard's first source was tried and rejected**, the same way Hysteria
was: it was an intricate abstract squiggle with fine nested detail that no
stroke-width adjustment rescued at badge size (tried both the standard weight
and a visibly thinner one; both stayed an unreadable knot below ~24px). A
second, different svgrepo.com source — a filled silhouette of WireGuard's
actual wolf-head mark, not a stroke drawing — went through the normal filled-
icon pipeline (no stroke step needed) and reads as a wolf head at 14px, if
rougher and busier than the other marks in this set; accepted on that basis.
`wireguard.svg` is this second source, not the rejected one.

## Not included, and why

- **Hysteria / Hysteria2** — the upstream mark ([Wikimedia](https://commons.wikimedia.org/wiki/File:Hysteria_2_Logo.svg),
  CC0, by Tobyxdd) is a stylised italic "H" whose monochrome silhouette is
  unreadable at badge sizes; it was rendered at 16-64px and rejected on the
  result. Uses an MDI glyph instead.
- **Shadowsocks, Vmess, Tuic** — no usable protocol mark was found. Papirus
  ships app-launcher icons for some of these (full-colour discs with shadow
  layers) which do not survive reduction to a monochrome glyph, and its set
  has no WireGuard or Tailscale entry at all either. These use MDI glyphs.
- **Masque, AnyTLS** — no logo search was attempted; `mdi-domino-mask` and
  `mdi-semantic-web` were chosen by explicit product decision instead of a
  brand mark.
