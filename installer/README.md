# Prizrak-Box Custom MSI Installer

This directory contains a custom MSI installer built with WiX Toolset, independent of electron-forge.

## Features

✅ **Built-in Language Selection** - First dialog allows choosing English or Russian
✅ **Single MSI File** - One installer with both languages embedded
✅ **GPL3 License Agreement** - Shows during installation
✅ **Feature Selection** - Choose which components to install:
   - Main Application (required)
   - TUN Service Mode (optional, enabled by default)
✅ **Process Management** - Automatically stops running processes before installation
✅ **Service Management** - Stops and removes old PrizrakBoxService before installation
✅ **TUN Service Installation** - Installs service using `px-service.exe -install`

## Installation Dialog Sequence

1. **Language Selection** - Choose English or Russian (inside MSI)
2. **Welcome** - Welcome to Prizrak-Box Setup Wizard
3. **License Agreement** - GPL3 with "I accept" checkbox
4. **Custom Setup** - Feature Tree with selectable components
5. **Installation Directory** - Choose install location
6. **Ready to Install** - Confirmation dialog
7. **Installing** - Progress bar with automatic process/service cleanup
8. **Install Service** - TUN service installation (if selected)
9. **Installation Complete** - Success message

## Prerequisites

- **WiX Toolset 3.x** - Must be installed and in PATH (candle.exe and light.exe)
- **Node.js** - For build scripts
- **PowerShell** - For process/service management during installation

## Building

### 1. Build the Application

```bash
npm run package
```

This will create packaged Electron app in `out/Prizrak-Box-win32-{arch}/`

### 2. Build MSI Installer

```bash
npm run make:msi
```

This will:
- Compile WiX sources for both English and Russian
- Generate **ONE** multi-language MSI file
- Output to `out/msi/{arch}/Prizrak-Box-{version}-{arch}.msi`

### 3. Run the Installer

Simply double-click the generated MSI file. The first dialog will ask you to choose the language.

## Manual Building

If you want to build manually:

```bash
# 1. Package the app
npm run package

# 2. Build multi-language MSI
node installer/build-msi.js
```

## File Structure

```
installer/
├── wix/
│   ├── Product.wxs                    # Main product definition
│   ├── UI.wxs                         # UI dialogs and sequence
│   ├── Files.wxs                      # File components
│   ├── dialogs/
│   │   └── LanguageSelectionDlg.wxs   # Custom language selection dialog
│   └── localization/
│       ├── License.rtf                # GPL3 license
│       ├── en-us.wxl                  # English strings
│       └── ru-ru.wxl                  # Russian strings
├── build-msi.js                       # MSI build script
└── README.md                          # This file
```

## How It Works

### 1. Build Script (`build-msi.js`)

- Harvests all application files using `heat.exe`
- Compiles WiX sources using `candle.exe`
- Links compiled objects using `light.exe` with multiple cultures
- Generates **ONE** MSI file with embedded English and Russian languages
- Uses variables for version, architecture, paths

### 2. Language Selection

The installer starts with a custom dialog (`LanguageSelectionDlg`) that presents:
- Radio button for English
- Radio button for Русский (Russian)

After selection, all subsequent dialogs use the chosen language.

### 3. Custom Actions

**Before Installation:**
- `StopProcessesBeforeInstall` - Stops Prizrak-Box.exe, px.exe, px-service.exe
- Stops and deletes PrizrakBoxService

**After Installation:**
- `InstallTunService` - Runs `px-service.exe -install` (if feature selected)

**Before Uninstallation:**
- `StopProcessesBeforeUninstall` - Cleanup before removing files

## Integration with GitHub Actions

Add to `.github/workflows/release.yml`:

```yaml
- name: Build Custom MSI Installer
  if: matrix.os == 'windows-latest'
  run: |
    npm run make:msi
  env:
    ARCH: ${{ matrix.arch }}

- name: Upload MSI Installer
  if: matrix.os == 'windows-latest'
  uses: actions/upload-artifact@v3
  with:
    name: msi-installer-${{ matrix.arch }}
    path: out/msi/${{ matrix.arch }}/*.msi
```

## Language Selection

**IMPORTANT:** Unlike the previous version, this installer has:
- ✅ **One MSI file** instead of two separate files
- ✅ **Built-in language selection** dialog at the start
- ✅ **No external launcher** needed
- ✅ **Automatic language switching** based on user choice

The language selection happens inside the MSI installer itself, not via an external script.

## Troubleshooting

### "candle.exe not found"

Install WiX Toolset 3.x and ensure it's in your PATH:
```
https://github.com/wixtoolset/wix3/releases
```

### "Application files not found"

Run `npm run package` first to build the Electron app.

### Permission Errors

The installer requires administrator privileges to:
- Stop system services
- Install to Program Files
- Create registry entries
- Install Windows service

## Advantages Over Previous Approach

✅ **Single MSI file** - Easier distribution and management
✅ **Built-in language selection** - No external launcher needed
✅ **Multi-language support** - Both languages embedded in one file
✅ **Cleaner user experience** - Choose language at the start
✅ **Full control over UI** - Custom dialogs and flow
✅ **Feature selection** with FeatureTree dialog
✅ **Process and service management**
✅ **Better control over upgrade logic**
✅ **Easier to customize and extend**

## Output

After building, you will have:
- `out/msi/x64/Prizrak-Box-{version}-x64.msi` (for 64-bit systems)
- `out/msi/arm64/Prizrak-Box-{version}-arm64.msi` (for ARM64 systems)

Each MSI file contains both English and Russian languages with a selection dialog at the start.
