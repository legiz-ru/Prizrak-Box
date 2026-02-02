const path = require('path');
const fs = require('fs-extra');
const { execSync } = require('child_process');

const ARCH = process.env.ARCH || 'x64';
const VERSION = require('../package.json').version;
const IS_ARM64 = ARCH === 'arm64';

const PATHS = {
  root: path.resolve(__dirname, '..'),
  installer: path.resolve(__dirname, 'wix'),
  out: path.resolve(__dirname, '..', 'out'),
  appFiles: path.resolve(__dirname, '..', 'out', 'Prizrak-Box-win32-' + ARCH),
  msiOut: path.resolve(__dirname, '..', 'out', 'msi', ARCH)
};

async function harvestFiles() {
  console.log('🔍 Harvesting application files with heat.exe...');

  const heatWxsFile = path.join(PATHS.msiOut, 'HarvestedFiles.wxs');

  // Use heat.exe to harvest all files from the app directory
  const heatCmd = `heat.exe dir "${PATHS.appFiles}" -nologo -cg ApplicationFiles -gg -sfrag -srd -sreg -dr INSTALLDIR -var var.SourceDir -out "${heatWxsFile}"`;

  try {
    execSync(heatCmd, { stdio: 'inherit' });
    console.log(`✅ Files harvested to: ${path.basename(heatWxsFile)}\n`);
    return heatWxsFile;
  } catch (error) {
    console.error('❌ Failed to harvest files:', error.message);
    throw error;
  }
}

async function buildMSI(language = 'en-us') {
  console.log(`\n🔧 Building MSI installer for ${ARCH} (${language})...`);

  // Ensure output directory exists
  await fs.ensureDir(PATHS.msiOut);

  // Prepare WiX variables
  const wixVars = {
    ProductVersion: VERSION.replace(/-.*$/, '').replace(/^(\d+\.\d+\.\d+).*/, '$1'), // Keep only major.minor.patch
    Platform: IS_ARM64 ? 'arm64' : 'x64',
    Win64: 'yes',
    ProgramFilesFolder: 'ProgramFiles64Folder',
    SourceDir: PATHS.appFiles,
    Language: language === 'ru-ru' ? '1049' : '1033',
    Culture: language
  };

  const wixDefines = Object.entries(wixVars)
    .map(([key, value]) => `-d${key}="${value}"`)
    .join(' ');

  try {
    // Step 1: Harvest files with heat.exe
    const harvestedWxs = await harvestFiles();

    // Step 2: Compile WiX sources (.wxs -> .wixobj)
    console.log('📦 Compiling WiX sources...');

    const wxsFiles = [
      path.join(PATHS.installer, 'Product.wxs'),
      path.join(PATHS.installer, 'UI.wxs'),
      path.join(PATHS.installer, 'Files.wxs'),
      harvestedWxs
    ];

    const wixobjFiles = [];

    for (const wxsFile of wxsFiles) {
      const wixobjFile = path.join(PATHS.msiOut, path.basename(wxsFile, '.wxs') + '.wixobj');
      wixobjFiles.push(wixobjFile);

      const candleCmd = `candle.exe -nologo -arch ${IS_ARM64 ? 'arm64' : 'x64'} ${wixDefines} -ext WixUIExtension -ext WixUtilExtension -out "${wixobjFile}" "${wxsFile}"`;

      console.log(`  Compiling ${path.basename(wxsFile)}...`);
      execSync(candleCmd, { stdio: 'inherit', cwd: PATHS.installer });
    }

    // Step 3: Link WiX objects (.wixobj -> .msi)
    console.log('🔗 Linking MSI package...');

    const msiFile = path.join(PATHS.msiOut, `Prizrak-Box-${VERSION}-${ARCH}-${language}.msi`);
    const locFile = path.join(PATHS.installer, 'localization', `${language}.wxl`);

    const lightCmd = `light.exe -nologo ${wixobjFiles.map(f => `"${f}"`).join(' ')} -ext WixUIExtension -ext WixUtilExtension -cultures:${language} -loc "${locFile}" -out "${msiFile}" -sval`;

    execSync(lightCmd, { stdio: 'inherit', cwd: PATHS.installer });

    console.log(`✅ MSI created: ${msiFile}\n`);
    return msiFile;

  } catch (error) {
    console.error('❌ Failed to build MSI:', error.message);
    throw error;
  }
}

async function main() {
  console.log('🚀 Prizrak-Box MSI Installer Builder\n');

  // Check if app is built
  if (!fs.existsSync(PATHS.appFiles)) {
    console.error(`❌ Application files not found at: ${PATHS.appFiles}`);
    console.error('   Please run "npm run package" first!');
    process.exit(1);
  }

  // Build MSI for both languages
  console.log('Building MSI installers for all languages...\n');

  try {
    await buildMSI('en-us');
    await buildMSI('ru-ru');

    console.log('🎉 All MSI installers built successfully!\n');
  } catch (error) {
    console.error('❌ Build failed');
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { buildMSI };
