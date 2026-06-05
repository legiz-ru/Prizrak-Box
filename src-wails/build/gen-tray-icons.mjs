// Generates the Windows tray icons from the monochrome-style app tiles in
// tray-src/ (sourced from github.com/arpicme/Proxy-App-Icon-set). Windows has no
// macOS-style template icon, so we ship a per-theme pair (Wails swaps them via
// SetIcon / SetDarkModeIcon following the taskbar theme):
//
//   light taskbar -> dark_background tile   -> tray-win-light.ico
//   dark  taskbar -> white_background tile  -> tray-win-dark.ico
//
// Plus an "active" variant of each (TUN or system-proxy enabled) with a green
// badge at the bottom-right. The badge is separated from the logo by a fully
// transparent ring so it never overlaps the artwork, on either theme.
//
// Run from src-wails/:  node build/gen-tray-icons.mjs
// (uses sharp, already a project dependency; output .ico files are committed,
//  so there is no build- or run-time dependency.)

import sharp from 'sharp';
import fs from 'node:fs';
import path from 'node:path';

const SIZES = [16, 20, 24, 32];
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
  const r = Math.round(size * 0.27);          // dot radius
  const margin = Math.round(size * 0.08);     // gap from the edges
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

async function pngFromRGBA(rgba, size) {
  return sharp(rgba, { raw: { width: size, height: size, channels: 4 } }).png().toBuffer();
}

// Multi-image .ico with each entry stored as PNG (Vista+).
function buildICO(pngs) {
  const count = pngs.length;
  const header = Buffer.alloc(6);
  header.writeUInt16LE(0, 0);     // reserved
  header.writeUInt16LE(1, 2);     // type: icon
  header.writeUInt16LE(count, 4); // image count

  const dir = Buffer.alloc(16 * count);
  let offset = 6 + 16 * count;
  pngs.forEach((p, idx) => {
    const b = dir.subarray(idx * 16);
    b[0] = p.size >= 256 ? 0 : p.size; // width
    b[1] = p.size >= 256 ? 0 : p.size; // height
    b[2] = 0;                          // palette
    b[3] = 0;                          // reserved
    b.writeUInt16LE(1, 4);             // planes
    b.writeUInt16LE(32, 6);            // bit depth
    b.writeUInt32LE(p.data.length, 8); // bytes
    b.writeUInt32LE(offset, 12);       // offset
    offset += p.data.length;
  });
  return Buffer.concat([header, dir, ...pngs.map((p) => p.data)]);
}

async function main() {
  for (const v of variants) {
    const svgPath = path.join(srcDir, v.svg);
    const inactive = [];
    const active = [];
    for (const size of SIZES) {
      const rgba = await rasterRGBA(svgPath, size);
      inactive.push({ size, data: await pngFromRGBA(rgba, size) });
      active.push({ size, data: await pngFromRGBA(withBadge(rgba, size), size) });
    }
    fs.writeFileSync(path.join(outDir, `${v.out}.ico`), buildICO(inactive));
    fs.writeFileSync(path.join(outDir, `${v.out}-active.ico`), buildICO(active));
    console.log(`wrote ${v.out}.ico and ${v.out}-active.ico`);
  }
}

main().catch((e) => { console.error(e); process.exit(1); });
