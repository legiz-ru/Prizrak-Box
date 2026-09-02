// Generates the Windows tray icons from the same monochrome ghost silhouette used
// on macOS (build/tray-macos.png). Windows has no template-icon support, so we
// ship a per-theme pair of crisp monochrome silhouettes (matching macOS quality):
//
//   light taskbar -> black ghost (tray-win-light.png)
//   dark  taskbar -> white ghost (tray-win-dark.png)
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
// Run from src-wails/:
//   node build/gen-tray-icons.mjs
//   go run build/reencode_icons.go
// (uses sharp, already a project dependency; output .png files are committed.)

import sharp from 'sharp';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const SIZE = 64;
const GREEN = [46, 204, 64, 255]; // #2ECC40

const here = path.dirname(fileURLToPath(import.meta.url));
const outDir = here;

// Load the macOS template icon (monochrome ghost silhouette) and resize to SIZE.
// Returns raw RGBA pixels at SIZE×SIZE.
async function loadSilhouette(size) {
  const macPath = path.join(outDir, 'tray-macos.png');
  const { data } = await sharp(macPath)
    .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .ensureAlpha()
    .raw()
    .toBuffer({ resolveWithObject: true });
  return data; // Buffer, RGBA, size*size*4
}

// Returns a copy of the RGBA buffer with black pixels converted to the given RGB.
// Only affects opaque/non-transparent pixels; transparency is preserved.
function recolor(rgba, size, r, g, b) {
  const out = Buffer.from(rgba);
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const i = (y * size + x) * 4;
      if (out[i + 3] > 0) {
        // Preserve alpha, replace color
        out[i] = r;
        out[i + 1] = g;
        out[i + 2] = b;
      }
    }
  }
  return out;
}

// Returns a copy of the RGBA buffer with the green "active" badge composited at
// bottom-right, surrounded by a transparent ring so it never touches the logo.
function withBadge(rgba, size) {
  const out = Buffer.from(rgba);
  const r = Math.round(size * 0.153);               // dot radius (×1.7)
  const margin = Math.round(size * 0.08);           // gap from the edges
  const gap = Math.max(1, Math.round(size * 0.04)); // transparent separation
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
  const silhouette = await loadSilhouette(SIZE);

  // Light taskbar: black ghost on transparent
  const light = silhouette; // already black
  fs.writeFileSync(path.join(outDir, 'tray-win-light.png'), await pngFromRGBA(light, SIZE));
  fs.writeFileSync(path.join(outDir, 'tray-win-light-active.png'), await pngFromRGBA(withBadge(light, SIZE), SIZE));
  console.log('wrote tray-win-light.png and tray-win-light-active.png');

  // Dark taskbar: white ghost on transparent
  const dark = recolor(silhouette, SIZE, 255, 255, 255);
  fs.writeFileSync(path.join(outDir, 'tray-win-dark.png'), await pngFromRGBA(dark, SIZE));
  fs.writeFileSync(path.join(outDir, 'tray-win-dark-active.png'), await pngFromRGBA(withBadge(dark, SIZE), SIZE));
  console.log('wrote tray-win-dark.png and tray-win-dark-active.png');
}

main().catch((e) => { console.error(e); process.exit(1); });
