#!/usr/bin/env node

const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const readline = require('readline');

// Detect system language
function getSystemLanguage() {
  try {
    const locale = process.env.LANG || process.env.LC_ALL || process.env.LC_MESSAGES || '';
    if (locale.toLowerCase().includes('ru')) {
      return 'ru-ru';
    }
  } catch (error) {
    // Ignore
  }
  return 'en-us';
}

// Ask user for language preference
async function askLanguage() {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  return new Promise((resolve) => {
    console.log('\n╔════════════════════════════════════════════════╗');
    console.log('║      Prizrak-Box Installer - Language         ║');
    console.log('║      Установщик Prizrak-Box - Язык            ║');
    console.log('╚════════════════════════════════════════════════╝\n');
    console.log('Please select your language / Пожалуйста, выберите язык:\n');
    console.log('  1. English');
    console.log('  2. Русский\n');

    rl.question('Enter your choice (1 or 2) / Введите ваш выбор (1 или 2): ', (answer) => {
      rl.close();
      resolve(answer.trim() === '2' ? 'ru-ru' : 'en-us');
    });
  });
}

// Find MSI file
function findMSI(language, arch) {
  const msiDir = path.join(__dirname, '..', 'out', 'msi', arch);
  const files = fs.readdirSync(msiDir);

  // Find MSI with matching language
  const msiFile = files.find(f => f.endsWith(`.msi`) && f.includes(language));

  if (msiFile) {
    return path.join(msiDir, msiFile);
  }

  // Fallback to any MSI
  const anyMsi = files.find(f => f.endsWith('.msi'));
  return anyMsi ? path.join(msiDir, anyMsi) : null;
}

// Launch MSI installer
function launchMSI(msiPath, language) {
  console.log(`\n🚀 Launching installer: ${path.basename(msiPath)}`);
  console.log(`   Language: ${language === 'ru-ru' ? 'Русский' : 'English'}\n`);

  try {
    // Launch MSI with elevated privileges
    execSync(`msiexec /i "${msiPath}" /qb`, { stdio: 'inherit' });
    console.log('\n✅ Installation completed!');
  } catch (error) {
    console.error('\n❌ Installation failed:', error.message);
    process.exit(1);
  }
}

async function main() {
  const arch = process.env.ARCH || process.arch;

  // Check if MSI files exist
  const msiDir = path.join(__dirname, '..', 'out', 'msi', arch);
  if (!fs.existsSync(msiDir)) {
    console.error('❌ MSI installers not found. Please run "npm run build:msi" first!');
    process.exit(1);
  }

  // Detect system language
  const systemLang = getSystemLanguage();
  console.log(`Detected system language: ${systemLang === 'ru-ru' ? 'Russian' : 'English'}`);

  // Ask user for language preference
  const selectedLang = await askLanguage();

  // Find and launch MSI
  const msiPath = findMSI(selectedLang, arch);

  if (!msiPath) {
    console.error(`❌ MSI installer not found for ${selectedLang} ${arch}`);
    process.exit(1);
  }

  launchMSI(msiPath, selectedLang);
}

if (require.main === module) {
  main().catch(error => {
    console.error('Error:', error);
    process.exit(1);
  });
}
