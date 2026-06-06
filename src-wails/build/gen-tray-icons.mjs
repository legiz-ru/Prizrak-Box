// Generates the Windows tray icons from the app tiles in tray-src/ (sourced from
// github.com/arpicme/Proxy-App-Icon-set). Windows has no macOS-style template
// icon, so we ship a per-theme pair (Wails swaps them via SetIcon /
// SetDarkModeIcon following the taskbar theme):
//
//   light taskbar -> dark_background tile   -> tray-win-light.png
//   dark  taskbar -> white_background tile  -> tray-win-dark.png
//
// Plus an "active" variant of each (TUN or system-proxy enabled) with a green
// badge at the bottom-right, separated from the logo by a transparent ring.
//
// IMPORTANT: Wails' Windows systray decodes the icon bytes with
// CreateIconFromResourceEx, which accepts a single icon image (PNG) — NOT a
// multi-image .ico container. So we output 64×64 PNGs (matching Wails' own
// builtin systray-light.png/systray-dark.png); Windows scales them to the tray
// size.
//
// Run from src-wails/:  node build/gen-tray-icons.mjs
// (uses sharp, already a project dependency; output .png files are committed.)

import sharp from 'sharp';
import fs from 'node:fs';
import path from 'node:path';

const SIZE = 64;
const GREEN = [46, 204, 64, 255]; // #2ECC40

const here = path.dirname(new URL(import.meta.url).pathname);
const srcDir = path.join(here, 'tray-src');
const outDir = here;

// theme -> source tile
const variants = [
  { svg: 'dark_background.svg',  out: 'tray-win-light' },  // used on a LIGHT taskbar
  { svg: 'white_background.svg', out: 'tray-win-dark'  },  // used on a DARK  taskbar
];

async function rasterRGBA(svgPath, size) {
  const { data } = await sharp(svgPath)
    .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .ensureAlpha()
    .raw()
    .toBuffer({ resolveWithObject: true });
  return data; // Buffer, RGBA, size*size*4
}

// Returns a copy of the RGBA buffer with the green "active" badge composited at
// bottom-right, surrounded by a transparent ring so it never touches the logo.
function withBadge(rgba, size) {
  const out = Buffer.from(rgba);
  const r = Math.round(size * 0.27);                // dot radius
  const margin = Math.round(size * 0.08);           // gap from the edges
  const gap = Math.max(1, Math.round(size * 0.10)); // transparent separation
  const cx = size - margin - r;
  const cy = size - margin - r;
  const rDot = r * r;
  const rRing = (r + gap) * (r + gap);
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const dx = x - cx, dy = y - cy;
      const d2 = dx * dx + dy * dy;
      const i = (y * size + x) * 4;
      if (d2 <= rDot) {
        out[i] = GREEN[0]; out[i + 1] = GREEN[1]; out[i + 2] = GREEN[2]; out[i + 3] = GREEN[3];
      } else if (d2 <= rRing) {
        out[i + 3] = 0; // punch a transparent ring around the dot
      }
    }
  }
  return out;
}

function pngFromRGBA(rgba, size) {
  return sharp(rgba, { raw: { width: size, height: size, channels: 4 } }).png().toBuffer();
}

async function main() {
  for (const v of variants) {
    const rgba = await rasterRGBA(path.join(srcDir, v.svg), SIZE);
    fs.writeFileSync(path.join(outDir, `${v.out}.png`), await pngFromRGBA(rgba, SIZE));
    fs.writeFileSync(path.join(outDir, `${v.out}-active.png`), await pngFromRGBA(withBadge(rgba, SIZE), SIZE));
    console.log(`wrote ${v.out}.png and ${v.out}-active.png`);
  }
}

main().catch((e) => { console.error(e); process.exit(1); });
