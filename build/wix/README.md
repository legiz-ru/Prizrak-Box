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

### Setup Types

- **Typical**: Installs all features (recommended for most users)
- **Custom**: Choose which features to install
- **Complete**: Installs all features

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
- **UI Flow**: WixUI_Mondo (Welcome → License → Setup Type → Features → Directory → Install)
- **Languages**: English (1033), Russian (1049)
- **License**: GNU GPL v3
- **Upgrade Code**: c1d377b2-2c61-4c5e-8773-8e3c703b8b41
- **Service Name**: PrizrakBoxService
- **Service Binary**: resources\px-service.exe

## Troubleshooting

### License not showing

If the license agreement dialog doesn't appear, ensure that:
- `build/wix/license.rtf` exists
- The file is accessible during the build process

### TUN Service not visible in Custom Setup

If the TUN Service option doesn't appear in Custom Setup:
- Make sure you selected "Custom" setup type (not "Typical")
- The feature is located under the main application in the feature tree

### Language not changing

To force a specific language:
1. Use the command-line parameter shown above
2. Or rebuild the MSI with a different default language by changing `language: 1033` to `language: 1049` in `forge.config.ts`

## Files

- `wix.xml` - Main WiX installer template
- `license.rtf` - GPL3 license in RTF format
- `en-us.wxl` - English localization strings
- `ru-ru.wxl` - Russian localization strings
