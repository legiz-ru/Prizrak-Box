# Prizrak-Box Custom MSI Installer

This directory contains a custom MSI installer built with WiX Toolset, independent of electron-forge.

## Features

✅ **GPL3 License Agreement** - Shows during installation
✅ **Multi-language Support** - English and Russian with language selection
✅ **Feature Selection** - Choose which components to install:
   - Main Application (required)
   - TUN Service Mode (optional, enabled by default)
✅ **Process Management** - Automatically stops running processes before installation
✅ **Service Management** - Stops and removes old PrizrakBoxService before installation
✅ **TUN Service Installation** - Installs service using `px-service.exe -install`

## Installation Dialog Sequence

1. **Language Selection** - Choose English or Russian (via launcher)
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

### 2. Build MSI Installers

```bash
npm run make:msi
```

This will:
- Compile WiX sources for both English and Russian
- Generate MSI files for both languages
- Output to `out/msi/{arch}/`

### 3. Launch Installer with Language Selection

```bash
npm run launch:msi
```

This will:
- Detect system language
- Show language selection dialog
- Launch the selected MSI installer

## Manual Building

If you want to build manually:

```bash
# 1. Package the app
npm run package

# 2. Build MSI
node installer/build-msi.js

# 3. Launch with language selection
node installer/launcher.js
```

## File Structure

```
installer/
├── wix/
│   ├── Product.wxs          # Main product definition
│   ├── UI.wxs               # UI dialogs and sequence
│   ├── Files.wxs            # File components
│   └── localization/
│       ├── License.rtf      # GPL3 license
│       ├── en-us.wxl        # English strings
│       └── ru-ru.wxl        # Russian strings
├── build-msi.js             # MSI build script
├── launcher.js              # Language selection launcher
└── README.md                # This file
```

## How It Works

### 1. Build Script (`build-msi.js`)

- Compiles WiX sources using `candle.exe`
- Links compiled objects using `light.exe`
- Generates separate MSI files for each language
- Uses variables for version, architecture, paths

### 2. Launcher (`launcher.js`)

- Detects system language
- Presents language selection menu
- Launches appropriate MSI with `msiexec`

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
- name: Build Custom MSI Installers
  if: matrix.os == 'windows-latest'
  run: |
    npm run make:msi
  env:
    ARCH: ${{ matrix.arch }}

- name: Upload MSI Installers
  if: matrix.os == 'windows-latest'
  uses: actions/upload-artifact@v3
  with:
    name: msi-installers-${{ matrix.arch }}
    path: out/msi/${{ matrix.arch }}/*.msi
```

## Language Selection

Unlike electron-forge's WiX maker, this custom installer supports true language selection:

1. **Automatic Detection** - Detects system language (Russian/English)
2. **Manual Selection** - User can choose preferred language via launcher
3. **Separate MSI Files** - Each language has its own MSI file
4. **Fully Localized** - All UI strings translated

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

## Advantages Over electron-forge Maker

✅ Full control over UI dialogs and flow
✅ True multi-language support with language selection
✅ Custom actions with PowerShell scripts
✅ Feature selection with FeatureTree dialog
✅ Process and service management
✅ Separate MSI files per language
✅ Better control over upgrade logic
✅ Easier to customize and extend
