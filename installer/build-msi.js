const path = require('path');
const fs = require('fs-extra');
const { execSync } = require('child_process');

const ARCH = process.env.ARCH || 'x64';
const VERSION = require('../package.json').version;
const IS_ARM64 = ARCH === 'arm64';
const IS_WAILS = (process.env.BUILD_TYPE || 'electron') === 'wails';

// Electron and Wails MSIs share the same install directory but must NOT
// share an UpgradeCode — otherwise Windows Installer treats them as the same
// product chain and silently rolls back whichever was installed first.
const UPGRADE_CODE = IS_WAILS
  ? 'd5f799e1-4b83-4a7f-b662-1a3c924d5e0f'   // Wails build — separate upgrade chain
  : 'c1d377b2-2c61-4c5e-8773-8e3c703b8b41';  // Electron build (original)

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

async function buildMSI() {
  console.log(`\n🔧 Building MSI installer for ${ARCH}...`);

  // Ensure output directory exists
  await fs.ensureDir(PATHS.msiOut);

  // Prepare WiX variables (use English as primary language)
  const wixVars = {
    ProductVersion: VERSION.replace(/-.*$/, '').replace(/^(\d+\.\d+\.\d+).*/, '$1'), // MSI requires numeric version only
    FullVersion: VERSION, // Keep full version string with alpha/beta suffix for display
    Platform: IS_ARM64 ? 'arm64' : 'x64',
    Win64: 'yes',
    ProgramFilesFolder: 'ProgramFiles64Folder',
    SourceDir: PATHS.appFiles,
    Language: '1033', // Primary language: English
    Culture: 'en-us',
    UpgradeCode: UPGRADE_CODE,
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

    // Step 3: Link WiX objects to create single MSI (.wixobj -> .msi)
    console.log('🔗 Linking MSI package...');
    console.log('   Language: English');

    const msiFile = path.join(PATHS.msiOut, `windows-${ARCH}.msi`);
    const enLocFile = path.join(PATHS.installer, 'localization', 'en-us.wxl');

    // Build single English MSI (no language transforms)
    const lightCmd = `light.exe -nologo ${wixobjFiles.map(f => `"${f}"`).join(' ')} -ext WixUIExtension -ext WixUtilExtension -cultures:en-us -loc "${enLocFile}" -out "${msiFile}" -sval`;
    execSync(lightCmd, { stdio: 'inherit', cwd: PATHS.installer });

    console.log(`✅ MSI created: ${msiFile}`);
    console.log(`   Language: English`);
    console.log(`   Size: Optimized single-language build\n`);
    return msiFile;

  } catch (error) {
    console.error('❌ Failed to build MSI:', error.message);
    throw error;
  }
}

async function main() {
  console.log('🚀 Prizrak-Box MSI Installer Builder\n');
  console.log(`Architecture: ${ARCH}`);
  console.log(`Expected app path: ${PATHS.appFiles}`);

  // Check if app is built
  if (!fs.existsSync(PATHS.appFiles)) {
    console.error(`❌ Application files not found at: ${PATHS.appFiles}`);
    console.error('\n📁 Checking what exists in out/ directory:');

    try {
      const outDirs = fs.readdirSync(PATHS.out);
      outDirs.forEach(dir => {
        const fullPath = path.join(PATHS.out, dir);
        if (fs.statSync(fullPath).isDirectory()) {
          console.error(`   - ${dir}`);
        }
      });
    } catch (err) {
      console.error(`   Could not read out/ directory: ${err.message}`);
    }

    console.error('\n   Please run "npm run package" first!');
    process.exit(1);
  }

  console.log(`✓ Found application files\n`);

  try {
    // Build single English MSI
    await buildMSI();

    console.log('🎉 MSI installer built successfully!\n');
    console.log('ℹ️  The installer includes:');
    console.log('   - English UI');
    console.log('   - GPL3 license agreement');
    console.log('   - Feature selection (Main App + TUN Service)');
    console.log('   - Automatic process/service cleanup');
    console.log('   - TUN service installation\n');
  } catch (error) {
    console.error('❌ Build failed');
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { buildMSI };
