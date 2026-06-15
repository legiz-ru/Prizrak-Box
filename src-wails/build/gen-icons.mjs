// Generates high-quality Windows .ico and Linux tray.png from the master
// appicon.png (512x512). Uses sharp (Lanczos downscale) for crisp results
// at every size, and writes BMP-in-ICO entries for maximum tool compatibility
// (go-winres, Electron Packager, etc.).
//
// Usage:
//   node src-wails/build/gen-icons.mjs
//
// Output:
//   src-wails/build/appicon.ico   — multi-size (16–256) Windows .exe icon
//   src-wails/build/tray.ico      — same as appicon.ico (historical alias)
//   src-wails/build/tray.png      — 256px PNG (Linux tray)
//   build/appicon.ico             — same ico for the Electron build
//   src-wails/build/darwin/appicon-macos.png — 1024px padded master for the
//                                   macOS .icns (Taskfile feeds it to sips +
//                                   iconutil). Artwork is inset to Apple's
//                                   icon-grid safe area so the Dock / Cmd+Tab
//                                   icon matches native apps instead of filling
//                                   the tile edge-to-edge.

import sharp from 'sharp';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SIZES = [256, 128, 64, 48, 40, 32, 24, 20, 16];

async function main() {
  const srcPng = path.join(__dirname, 'appicon.png');
  const srcStat = fs.statSync(srcPng);
  console.log(`Source: ${srcPng} (${srcStat.size} bytes)`);

  // Downscale to every size using sharp's Lanczos kernel
  const buffers = {};
  for (const s of SIZES) {
    buffers[s] = await sharp(srcPng)
      .resize(s, s, { fit: 'contain', kernel: 'lanczos3', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .ensureAlpha()
      .raw()
      .toBuffer();
    console.log(`  ${s}×${s}: ${buffers[s].length} bytes raw`);
  }

  // Build ICO with BMP entries (most compatible)
  const icoBin = buildIco(SIZES, buffers);

  // Write to Wails build dir
  const wailsOut = path.join(__dirname, 'appicon.ico');
  fs.writeFileSync(wailsOut, icoBin);
  console.log(`\nWrote ${wailsOut} (${icoBin.length} bytes, ${SIZES.length} images)`);

  // Same content as tray.ico (historical alias)
  const trayIco = path.join(__dirname, 'tray.ico');
  fs.writeFileSync(trayIco, icoBin);
  console.log(`Wrote ${trayIco}`);

  // tray.png (256px, for Linux tray)
  const trayPngBuf = await sharp(buffers[256], { raw: { width: 256, height: 256, channels: 4 } })
    .png()
    .toBuffer();
  const trayPng = path.join(__dirname, 'tray.png');
  fs.writeFileSync(trayPng, trayPngBuf);
  console.log(`Wrote ${trayPng} (${trayPngBuf.length} bytes)`);

  // Also update the root build/appicon.ico (Electron build)
  const rootOut = path.join(__dirname, '..', '..', 'build', 'appicon.ico');
  fs.writeFileSync(rootOut, icoBin);
  console.log(`Wrote ${rootOut}`);

  // macOS padded master: inset the artwork into Apple's icon-grid safe area
  // (~80.5% of the canvas, i.e. 824px content on a 1024px transparent tile) so
  // the Dock / Cmd+Tab icon renders at the same visual size as native apps.
  const CANVAS = 1024;
  const CONTENT = Math.round(CANVAS * 0.8047); // Apple grid: 824 / 1024
  const inset = Math.round((CANVAS - CONTENT) / 2);
  const artBuf = await sharp(srcPng)
    .resize(CONTENT, CONTENT, { kernel: 'lanczos3' })
    .png()
    .toBuffer();
  const macDir = path.join(__dirname, 'darwin');
  fs.mkdirSync(macDir, { recursive: true });
  const macOut = path.join(macDir, 'appicon-macos.png');
  const paddedBuf = await sharp({ create: { width: CANVAS, height: CANVAS, channels: 4, background: { r: 0, g: 0, b: 0, alpha: 0 } } })
    .composite([{ input: artBuf, left: inset, top: inset }])
    .png()
    .toBuffer();
  fs.writeFileSync(macOut, paddedBuf);
  console.log(`Wrote ${macOut} (${CONTENT}px content inset ${inset}px on ${CANVAS}px tile)`);

  // Build the macOS .icns directly from the padded master so it can be produced
  // on any OS (no sips / iconutil needed) and never drifts from the master.
  // Each entry is a PNG; the OSType + render size pairs mirror what `iconutil`
  // emits from a full .iconset (so Finder/Dock pick the right size everywhere).
  const ICNS_ENTRIES = [
    ['icp4', 16], ['ic11', 32],  // 16   + 16@2x
    ['icp5', 32], ['ic12', 64],  // 32   + 32@2x
    ['ic07', 128], ['ic13', 256], // 128  + 128@2x
    ['ic08', 256], ['ic14', 512], // 256  + 256@2x
    ['ic09', 512], ['ic10', 1024], // 512 + 512@2x
  ];
  const chunks = [];
  for (const [osType, size] of ICNS_ENTRIES) {
    const png = await sharp(paddedBuf).resize(size, size, { kernel: 'lanczos3' }).png().toBuffer();
    const header = Buffer.alloc(8);
    header.write(osType, 0, 'ascii');
    header.writeUInt32BE(8 + png.length, 4);
    chunks.push(header, png);
  }
  const body = Buffer.concat(chunks);
  const icnsHeader = Buffer.alloc(8);
  icnsHeader.write('icns', 0, 'ascii');
  icnsHeader.writeUInt32BE(8 + body.length, 4);
  const icnsOut = path.join(macDir, 'appicon.icns');
  fs.writeFileSync(icnsOut, Buffer.concat([icnsHeader, body]));
  console.log(`Wrote ${icnsOut} (${ICNS_ENTRIES.length} PNG entries)`);
}

// Convert raw RGBA pixels to 32bpp BMP entry bytes.
// BMP in ICO uses BGRA bottom-up rows + 1bpp AND mask (all zeros for 32bpp).
function rgbaToBmpEntry(rgba, size) {
  const rowBytes = size * 4;
  const xorSize = rowBytes * size; // no row padding needed for 32bpp

  // AND mask: 1bpp, each row padded to 4 bytes
  const andRowBytes = ((size + 31) >>> 5) * 4;
  const andSize = andRowBytes * size;

  const total = 40 + xorSize + andSize; // BITMAPINFOHEADER + XOR + AND
  const buf = Buffer.alloc(total);
  let off = 0;

  // BITMAPINFOHEADER (40 bytes)
  buf.writeUInt32LE(40, off); off += 4;  // biSize
  buf.writeInt32LE(size, off); off += 4;  // biWidth
  buf.writeInt32LE(size * 2, off); off += 4; // biHeight (double for ICO)
  buf.writeUInt16LE(1, off); off += 2;    // biPlanes
  buf.writeUInt16LE(32, off); off += 2;   // biBitCount
  buf.writeUInt32LE(0, off); off += 4;    // biCompression (BI_RGB)
  buf.writeUInt32LE(0, off); off += 4;    // biSizeImage (0 for BI_RGB)
  buf.writeInt32LE(0, off); off += 4;     // biXPelsPerMeter
  buf.writeInt32LE(0, off); off += 4;     // biYPelsPerMeter
  buf.writeUInt32LE(0, off); off += 4;    // biClrUsed
  buf.writeUInt32LE(0, off); off += 4;    // biClrImportant

  // XOR mask: BGRA pixels, bottom-up rows
  for (let y = size - 1; y >= 0; y--) {
    const srcOff = y * rowBytes;
    for (let x = 0; x < size; x++) {
      const si = srcOff + x * 4;
      // RGBA -> BGRA
      buf[off++] = rgba[si + 2]; // B
      buf[off++] = rgba[si + 1]; // G
      buf[off++] = rgba[si + 0]; // R
      buf[off++] = rgba[si + 3]; // A
    }
  }

  // AND mask: all zeros (alpha channel handles transparency)
  // Already zero-filled by Buffer.alloc
  return buf;
}

function buildIco(sizes, sizeBuffers) {
  const count = sizes.length;
  const headerSize = 6 + count * 16;

  // Compute entries
  const entries = sizes.map((s) => ({
    w: s,
    h: s,
    data: rgbaToBmpEntry(sizeBuffers[s], s),
  }));

  // Calculate offsets
  let offset = headerSize;
  for (const e of entries) {
    e.offset = offset;
    offset += e.data.length;
  }

  const total = offset;
  const buf = Buffer.alloc(total);
  let off = 0;

  // ICO header
  buf.writeUInt16LE(0, off); off += 2;  // reserved
  buf.writeUInt16LE(1, off); off += 2;  // type: 1 = icon
  buf.writeUInt16LE(count, off); off += 2; // count

  // Directory entries
  for (const e of entries) {
    buf[off++] = e.w === 256 ? 0 : e.w;  // width (0 = 256)
    buf[off++] = e.h === 256 ? 0 : e.h;  // height (0 = 256)
    buf[off++] = 0;                       // color count
    buf[off++] = 0;                       // reserved
    buf.writeUInt16LE(1, off); off += 2;  // planes
    buf.writeUInt16LE(32, off); off += 2; // bpp
    buf.writeUInt32LE(e.data.length, off); off += 4; // size
    buf.writeUInt32LE(e.offset, off); off += 4; // offset
  }

  // Image data
  for (const e of entries) {
    e.data.copy(buf, off);
    off += e.data.length;
  }

  return buf;
}

main().catch((e) => { console.error(e); process.exit(1); });
