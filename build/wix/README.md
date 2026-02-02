# Prizrak-Box MSI Installer Documentation

## Multilingual Support

The MSI installer supports English and Russian languages.

### Automatic Language Detection

By default, the installer will use English. Windows will automatically apply the appropriate language transform if the system locale matches Russian (1049).

### Manual Language Selection

To manually select the installer language, use the following command:

#### For English:
```cmd
msiexec /i Prizrak-Box-Setup.msi
```

#### For Russian:
```cmd
msiexec /i Prizrak-Box-Setup.msi TRANSFORMS=:1049
```

Or set the Product Language property:
```cmd
msiexec /i Prizrak-Box-Setup.msi PRODUCTLANGUAGE=1049
```

## Installation Features

### Available Features

1. **Main Application** (Required)
   - Core application files
   - Desktop and Start Menu shortcuts
   - Protocol handler registration

2. **TUN Service Mode** (Optional, Enabled by Default)
   - Installs px-service as a Windows service
   - Allows TUN mode to run without administrator privileges
   - Can be disabled during Custom Setup

### Feature Selection

The installer uses a feature tree interface where you can select which components to install:

- **Main Application** (required, cannot be disabled)
  - Core application files
  - Desktop and Start Menu shortcuts
  - Protocol handler

- **TUN Service Mode** (optional, enabled by default)
  - Windows service for TUN mode
  - Allows running TUN without admin rights

You can click on the icons next to each feature to:
- Install the feature (hard drive icon)
- Skip the feature (red X icon)

## Pre-Installation Actions

The installer automatically performs the following actions before installation:

1. Stops running processes:
   - Prizrak-Box.exe
   - px.exe
   - px-service.exe

2. Stops and removes the PrizrakBoxService Windows service

This ensures a clean installation without file conflicts.

## Uninstallation

During uninstallation, the installer will:

1. Stop all running Prizrak-Box processes
2. Stop and remove the PrizrakBoxService
3. Remove all installed files and shortcuts
4. Clean up registry entries

## Building the Installer

To build the MSI installer with multilingual support:

```bash
# Build the service binary
npm run build:service:windows

# Build and package the application
npm run make
```

The output will be in the `out/make/wix/` directory.

## Technical Details

- **Installer Type**: Windows Installer (MSI) using WiX Toolset
- **UI Flow**: WixUI_FeatureTree with custom navigation (Welcome → **License** → Features → Directory → Install)
- **Languages**: English (1033), Russian (1049)
- **License**: GNU GPL v3 (displayed in License Agreement dialog)
- **Upgrade Code**: c1d377b2-2c61-4c5e-8773-8e3c703b8b41
- **Service Name**: PrizrakBoxService
- **Service Binary**: resources\px-service.exe

### UI Navigation Flow

1. **Welcome Dialog** - Introduction
2. **License Agreement** - GPL3 license (must accept to continue)
3. **Custom Setup** - Feature selection tree:
   - Main Application (required)
   - TUN Service Mode (optional, enabled by default)
4. **Installation Directory** - Choose install location
5. **Ready to Install** - Confirm settings
6. **Installation Progress** - Installing files and configuring service
7. **Completion** - Finish

## Troubleshooting

### License not showing

If the license agreement dialog doesn't appear, ensure that:
- `build/wix/license.rtf` exists
- The file is accessible during the build process

### TUN Service not visible in Feature Selection

If the TUN Service option doesn't appear in the feature tree:
- The feature should be visible directly in the Custom Setup dialog
- Look for "TUN Service Mode" at the same level as "Main Application"
- Both features are children of the main "Prizrak-Box" feature
- Click on the feature icons to toggle installation (enable/disable)

### Language not changing

To force a specific language:
1. Use the command-line parameter shown above
2. Or rebuild the MSI with a different default language by changing `language: 1033` to `language: 1049` in `forge.config.ts`

## Files

- `wix.xml` - Main WiX installer template
- `license.rtf` - GPL3 license in RTF format
- `en-us.wxl` - English localization strings
- `ru-ru.wxl` - Russian localization strings
