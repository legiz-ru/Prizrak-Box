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

async function buildMultiLanguageMSI() {
  console.log(`\n🔧 Building multi-language MSI installer for ${ARCH}...`);

  // Ensure output directory exists
  await fs.ensureDir(PATHS.msiOut);

  // Prepare WiX variables (use English as primary language)
  const wixVars = {
    ProductVersion: VERSION.replace(/-.*$/, '').replace(/^(\d+\.\d+\.\d+).*/, '$1'),
    Platform: IS_ARM64 ? 'arm64' : 'x64',
    Win64: 'yes',
    ProgramFilesFolder: 'ProgramFiles64Folder',
    SourceDir: PATHS.appFiles,
    Language: '1033', // Primary language: English
    Culture: 'en-us'
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
      path.join(PATHS.installer, 'dialogs', 'LanguageSelectionDlg.wxs'),
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

    // Step 3: Link WiX objects to create multi-language MSI (.wixobj -> .msi)
    console.log('🔗 Linking multi-language MSI package...');
    console.log('   Building with language transform approach...');

    const msiFile = path.join(PATHS.msiOut, `Prizrak-Box-${VERSION}-${ARCH}.msi`);
    const wixobjArgs = wixobjFiles.map(f => `"${f}"`).join(' ');
    const enLocFile = path.join(PATHS.installer, 'localization', 'en-us.wxl');
    const ruLocFile = path.join(PATHS.installer, 'localization', 'ru-ru.wxl');

    // Build base English MSI
    const enMsiFile = path.join(PATHS.msiOut, `temp-en.msi`);
    console.log('  Building English MSI...');
    const lightEnCmd = `light.exe -nologo ${wixobjArgs} -ext WixUIExtension -ext WixUtilExtension -cultures:en-us -loc "${enLocFile}" -out "${enMsiFile}" -sval`;
    execSync(lightEnCmd, { stdio: 'inherit', cwd: PATHS.installer });

    // Build Russian MSI
    const ruMsiFile = path.join(PATHS.msiOut, `temp-ru.msi`);
    console.log('  Building Russian MSI...');
    const lightRuCmd = `light.exe -nologo ${wixobjArgs} -ext WixUIExtension -ext WixUtilExtension -cultures:ru-ru -loc "${ruLocFile}" -out "${ruMsiFile}" -sval`;
    execSync(lightRuCmd, { stdio: 'inherit', cwd: PATHS.installer });

    // Create language transform
    const mstFile = path.join(PATHS.msiOut, `1049.mst`); // 1049 is the LCID for Russian
    console.log('  Creating Russian language transform...');
    const torchCmd = `torch.exe -nologo -p -t language "${enMsiFile}" "${ruMsiFile}" -out "${mstFile}"`;
    execSync(torchCmd, { stdio: 'inherit', cwd: PATHS.installer });

    // Copy base English MSI to final location
    fs.copyFileSync(enMsiFile, msiFile);

    // Embed the transform in the MSI
    console.log('  Embedding Russian transform into MSI...');
    // Use WiX's EmbedTransform or Windows msidb tool
    // Try different methods in order of preference
    let transformEmbedded = false;

    // Method 1: Try using WiSubStg.vbs (Windows SDK)
    try {
      const wisubstgPath = path.join(process.env.SystemRoot || 'C:\\Windows', 'System32', 'WiSubStg.vbs');
      if (fs.existsSync(wisubstgPath)) {
        const embedCmd1 = `cscript.exe //Nologo "${wisubstgPath}" "${msiFile}" "${mstFile}" 1049`;
        execSync(embedCmd1, { stdio: 'inherit', cwd: PATHS.installer });
        transformEmbedded = true;
        console.log('  ✓ Transform embedded using WiSubStg.vbs');
      }
    } catch (e) {
      console.log('  ⚠️  WiSubStg.vbs method failed, trying alternative...');
    }

    // Method 2: Try using msidb.exe if available
    if (!transformEmbedded) {
      try {
        const embedCmd2 = `msidb.exe -d "${msiFile}" -r "${mstFile}"`;
        execSync(embedCmd2, { stdio: 'inherit', cwd: PATHS.installer });
        transformEmbedded = true;
        console.log('  ✓ Transform embedded using msidb.exe');
      } catch (e) {
        console.log('  ⚠️  msidb.exe not available or failed');
      }
    }

    // If transform couldn't be embedded, continue without it
    if (!transformEmbedded) {
      console.log('  ⚠️  Could not embed Russian transform automatically');
      console.log('  ℹ️  MSI will work with English only');
      console.log('  ℹ️  To add Russian support manually, use Orca or WiSubStg.vbs');
    }

    // Clean up temporary files
    try {
      if (fs.existsSync(enMsiFile)) fs.unlinkSync(enMsiFile);
      if (fs.existsSync(ruMsiFile)) fs.unlinkSync(ruMsiFile);
      if (fs.existsSync(mstFile)) fs.unlinkSync(mstFile);
    } catch (e) {
      // Ignore cleanup errors
    }

    console.log(`✅ MSI created: ${msiFile}`);
    if (transformEmbedded) {
      console.log(`   Supported languages: English (default), Russian (embedded transform)`);
    } else {
      console.log(`   Supported languages: English only`);
    }
    console.log();
    return msiFile;

  } catch (error) {
    console.error('❌ Failed to build MSI:', error.message);
    throw error;
  }
}

async function main() {
  console.log('🚀 Prizrak-Box Multi-Language MSI Installer Builder\n');
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
    // Build single multi-language MSI
    await buildMultiLanguageMSI();

    console.log('🎉 Multi-language MSI installer built successfully!\n');
    console.log('ℹ️  The installer includes:');
    console.log('   - Language selection dialog (English/Russian)');
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

module.exports = { buildMultiLanguageMSI };
