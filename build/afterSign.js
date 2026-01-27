const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

/**
 * afterSign hook для подписи Go бинарников внутри .app bundle
 *
 * Вызывается после того как Electron Forge подписал основной app,
 * но до notarization. Подписывает px и px-service с теми же параметрами.
 */
module.exports = async function afterSign(context) {
  // Только для macOS
  if (context.electronPlatformName !== 'darwin') {
    console.log('[afterSign] Skipping - not macOS platform');
    return;
  }

  // Проверяем нужно ли подписывать (есть ли identity)
  const identity = process.env.CODESIGN_IDENTITY || process.env.MAC_CODESIGN_IDENTITY;
  if (!identity) {
    console.log('[afterSign] Skipping code signing - no identity provided');
    return;
  }

  console.log('[afterSign] Signing Go binaries inside .app bundle...');
  console.log(`[afterSign] Using identity: ${identity}`);

  // Находим .app bundle
  const appPath = context.appOutDir;
  const appName = context.packager.appInfo.productFilename;
  const appBundlePath = path.join(appPath, `${appName}.app`);

  if (!fs.existsSync(appBundlePath)) {
    console.error(`[afterSign] ERROR: .app bundle not found at ${appBundlePath}`);
    return;
  }

  console.log(`[afterSign] Found .app bundle: ${appBundlePath}`);

  // Пути к Go бинарникам внутри bundle
  const resourcesPath = path.join(appBundlePath, 'Contents', 'Resources');
  const pxBinary = path.join(resourcesPath, 'px');
  const pxServiceBinary = path.join(resourcesPath, 'px-service');

  const binaries = [
    { name: 'px', path: pxBinary },
    { name: 'px-service', path: pxServiceBinary }
  ];

  // Подписываем каждый бинарник
  for (const binary of binaries) {
    if (!fs.existsSync(binary.path)) {
      console.log(`[afterSign] WARNING: ${binary.name} not found at ${binary.path}, skipping`);
      continue;
    }

    console.log(`[afterSign] Signing ${binary.name}...`);

    try {
      // Подписываем с теми же параметрами что и в workflow
      // --force: перезаписать существующую подпись
      // --options runtime: hardened runtime для notarization
      // --timestamp: добавить timestamp для проверки
      const signCommand = `codesign --force --sign "${identity}" --options runtime --timestamp "${binary.path}"`;

      execSync(signCommand, { stdio: 'inherit' });
      console.log(`[afterSign] ✓ ${binary.name} signed successfully`);

      // Проверяем подпись
      const verifyCommand = `codesign --verify --verbose "${binary.path}"`;
      execSync(verifyCommand, { stdio: 'inherit' });
      console.log(`[afterSign] ✓ ${binary.name} signature verified`);
    } catch (error) {
      console.error(`[afterSign] ERROR signing ${binary.name}:`, error.message);
      throw error; // Прерываем процесс сборки при ошибке
    }
  }

  console.log('[afterSign] All Go binaries signed successfully');
};
