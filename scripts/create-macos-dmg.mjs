import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(__dirname, '..')

const arch    = process.argv[2] || 'arm64'
const outName = process.argv[3] || `prizrak-box-macos-${arch}.dmg`

const appName    = 'Prizrak-Box.app'
const appPath    = path.join(repoRoot, appName)
const outPath    = path.join(repoRoot, outName)
const tempDmg    = path.join(repoRoot, `prizrak-box-macos-${arch}-tmp.dmg`)
const stageDir   = path.join(repoRoot, 'dmg-stage')
const volumeName = 'Prizrak-Box'
const dmgBg      = path.join(repoRoot, 'build', 'dmg-background.png')

if (!fs.existsSync(appPath)) {
  console.error(`[create-macos-dmg] .app not found: ${appPath}`)
  process.exit(1)
}

// Staging directory
fs.rmSync(stageDir, { recursive: true, force: true })
fs.mkdirSync(stageDir, { recursive: true })
execFileSync('cp', ['-R', appPath, path.join(stageDir, appName)])

// /Applications symlink
try { fs.symlinkSync('/Applications', path.join(stageDir, 'Applications')) } catch { /* ok */ }

// README files — three languages (no Fix Quarantine: app is signed & notarized)
fs.writeFileSync(path.join(stageDir, 'README.md'), `# Prizrak-Box — macOS Installation

1. Drag **Prizrak-Box.app** into **Applications**.
2. Open the app from Applications.

The app is digitally signed. If a security dialog appears, click **Open**.

Support & updates: https://github.com/legiz-ru/prizrak-box
`, 'utf8')

fs.writeFileSync(path.join(stageDir, 'README.ru.md'), `# Prizrak-Box — Установка на macOS

1. Перетащите **Prizrak-Box.app** в папку **Программы** (Applications).
2. Откройте приложение из папки «Программы».

Приложение подписано цифровой подписью. При появлении диалога безопасности нажмите **Открыть** (Open).

Поддержка и обновления: https://github.com/legiz-ru/prizrak-box
`, 'utf8')

fs.writeFileSync(path.join(stageDir, 'README.zh.md'), `# Prizrak-Box — macOS 安装说明

1. 将 **Prizrak-Box.app** 拖拽到 **应用程序** (Applications) 文件夹。
2. 从「应用程序」文件夹打开应用。

该应用已经过数字签名。如果出现安全对话框，请点击**打开** (Open)。

支持与更新：https://github.com/legiz-ru/prizrak-box
`, 'utf8')

// Optional background image (add build/dmg-background.png to enable)
const useBackground = fs.existsSync(dmgBg)
if (useBackground) {
  const bgDir = path.join(stageDir, '.background')
  fs.mkdirSync(bgDir, { recursive: true })
  fs.copyFileSync(dmgBg, path.join(bgDir, 'dmg-background.png'))
}

// Create writable temp DMG from staging
fs.rmSync(outPath, { force: true })
fs.rmSync(tempDmg, { force: true })
execFileSync('hdiutil', ['create', '-volname', volumeName, '-srcfolder', stageDir, '-ov', '-format', 'UDRW', tempDmg], { stdio: 'inherit' })

// Mount
const attachOut = execFileSync('hdiutil', ['attach', '-readwrite', '-noverify', '-noautoopen', tempDmg], { encoding: 'utf8' })
const deviceLine = attachOut.split('\n').map(x => x.trim()).find(x => x.startsWith('/dev/'))
const device = deviceLine ? deviceLine.split(/\s+/)[0] : ''
if (!device) throw new Error(`failed to parse mounted device:\n${attachOut}`)

// Style with AppleScript
// Window: 860×530 px content area — icons row at y=250, readme row at y=470
// Three READMEs spread evenly across the bottom: x = 215, 465, 715
const bgLine = useBackground
  ? `set background picture of opts to file ".background:dmg-background.png"`
  : 'set background color of opts to {65535, 65535, 65535}'
execFileSync('osascript', ['-e', `
tell application "Finder"
  tell disk "${volumeName}"
    open
    set current view of container window to icon view
    set toolbar visible of container window to false
    set statusbar visible of container window to false
    set the bounds of container window to {120, 120, 980, 650}
    set opts to the icon view options of container window
    set arrangement of opts to not arranged
    set icon size of opts to 100
    ${bgLine}
    delay 0.2
    set position of item "${appName}" of container window to {230, 240}
    set position of item "Applications" of container window to {700, 240}
    try
      set position of item "README.md" of container window to {215, 460}
    end try
    try
      set position of item "README.ru.md" of container window to {465, 460}
    end try
    try
      set position of item "README.zh.md" of container window to {715, 460}
    end try
    close
    open
    update without registering applications
    delay 0.2
    close
  end tell
end tell
`], { stdio: 'inherit' })

// Detach and compress
execFileSync('hdiutil', ['detach', device], { stdio: 'inherit' })
execFileSync('hdiutil', ['convert', tempDmg, '-format', 'UDZO', '-imagekey', 'zlib-level=9', '-o', outPath], { stdio: 'inherit' })

// Cleanup
fs.rmSync(tempDmg, { force: true })
fs.rmSync(stageDir, { recursive: true, force: true })

console.log('[create-macos-dmg] wrote', outPath)
