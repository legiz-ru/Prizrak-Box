import type {ForgeConfig} from '@electron-forge/shared-types';
import MakerZipFixed from './forge/maker-zip-fixed';
import {MakerWix} from '@electron-forge/maker-wix';
import {MakerSquirrel} from '@electron-forge/maker-squirrel';
import {MakerDMG} from '@electron-forge/maker-dmg';
import {MakerDeb} from '@electron-forge/maker-deb';
import {MakerRpm} from '@electron-forge/maker-rpm';
import {AutoUnpackNativesPlugin} from '@electron-forge/plugin-auto-unpack-natives';
import {VitePlugin} from '@electron-forge/plugin-vite';
import {FusesPlugin} from '@electron-forge/plugin-fuses';
import {FuseV1Options, FuseVersion} from '@electron/fuses';
import fs from 'node:fs/promises';
import path from 'node:path';

const isWindows = process.platform === 'win32';
const extraResource = isWindows
    ? ['src-go/px.exe', 'src-service/px-service.exe']
    : ['src-go/px', 'src-service/px-service'];
const arch = process.env.ARCH || process.arch;
const envProvidedIdentity = process.env.MAC_CODESIGN_IDENTITY
    || process.env.CODESIGN_IDENTITY
    || process.env.APPLE_IDENTITY;
const forceCodeSign = process.env.FORCE_CODE_SIGN === 'true';
const macSignInputsPresent = Boolean(envProvidedIdentity || forceCodeSign);
const macIdentity = envProvidedIdentity || 'Developer ID Application: Yaroslav Podieiapolskii (4Q268756HJ)';
const macNotarizeInputsPresent = macSignInputsPresent
    && Boolean(process.env.APPLE_ID && process.env.APP_SPECIFIC_PASSWORD && process.env.TEAM_ID);

const packagerConfig: ForgeConfig['packagerConfig'] = {
    asar: true,
    name: 'Prizrak-Box',
    executableName: 'Prizrak-Box',
    icon: 'build/appicon',
    extraResource,
    extendInfo: {
        LSMinimumSystemVersion: "10.13.0"
    },
    appBundleId: 'com.legiz-ru.prizrak-box',
    protocols: [
        {
            name: 'Prizrak-Box Protocol',
            schemes: ['prizrak-box']
        }
    ],
};

if (process.platform === 'darwin' && macSignInputsPresent) {
    packagerConfig.osxSign = {
        identity: macIdentity,
        hardenedRuntime: true,
        'gatekeeper-assess': false,
        entitlements: 'build/entitlements.mac.plist',
        'entitlements-inherit': 'build/entitlements.mac.plist',
        'signature-flags': 'library',
    };
}

if (process.platform === 'darwin' && macNotarizeInputsPresent) {
    packagerConfig.osxNotarize = {
        tool: 'notarytool',
        appleId: process.env.APPLE_ID,
        appleIdPassword: process.env.APP_SPECIFIC_PASSWORD,
        teamId: process.env.TEAM_ID,
    };
}

const config: ForgeConfig = {
    packagerConfig,
    rebuildConfig: {},
    makers: [
        new MakerZipFixed({}),
        new MakerSquirrel({
            name: 'Prizrak-Box',
            authors: ['legiz-ru'],
            setupIcon: 'build/appicon.ico',
            iconUrl: 'https://raw.githubusercontent.com/legiz-ru/Prizrak-Box/prizrak_dev/build/appicon.ico',
        }),
        new MakerWix({
            manufacturer: 'legiz-ru',
            description: 'A Simple Mihomo GUI',
            icon: 'build/appicon.ico',
            arch: 'x64',
            ui: {
                chooseDirectory: true,
            },
            cultures: 'en-US;ru-RU',
            language: 1049,
            beforeCreate: async (creator) => {
                creator.wixTemplate = await fs.readFile(
                    path.resolve(__dirname, 'build', 'wix', 'wix.xml'),
                    'utf8'
                );
            },
            upgradeCode: 'c1d377b2-2c61-4c5e-8773-8e3c703b8b41',
            registry: [
                {
                    key: 'HKEY_CLASSES_ROOT\\prizrak-box',
                    values: [
                        {
                            name: '',
                            type: 'REG_SZ',
                            value: 'URL:Prizrak-Box Protocol'
                        },
                        {
                            name: 'URL Protocol',
                            type: 'REG_SZ',
                            value: ''
                        }
                    ]
                },
                {
                    key: 'HKEY_CLASSES_ROOT\\prizrak-box\\DefaultIcon',
                    values: [
                        {
                            name: '',
                            type: 'REG_SZ',
                            value: '[APPLICATIONROOTDIRECTORY]Prizrak-Box.exe,0'
                        }
                    ]
                },
                {
                    key: 'HKEY_CLASSES_ROOT\\prizrak-box\\shell\\open\\command',
                    values: [
                        {
                            name: '',
                            type: 'REG_SZ',
                            value: '"[APPLICATIONROOTDIRECTORY]Prizrak-Box.exe" "%1"'
                        }
                    ]
                }
            ]
        }),
        new MakerDMG({
            icon: 'build/appicon.icns',
            title: `Prizrak-Box-${arch}`,  // dmg 挂载卷名称
        }),
        new MakerRpm({
            options: {
                icon: 'build/appicon.png',
                homepage: 'https://github.com/legiz-ru/Prizrak-Box',
            }
        }),
        new MakerDeb({
            options: {
                icon: 'build/appicon.png',
                maintainer: 'legiz-ru',
                homepage: 'https://github.com/legiz-ru/Prizrak-Box',
                mimeType: ['x-scheme-handler/prizrak-box']
            }
        })
    ],
    plugins: [
        new AutoUnpackNativesPlugin({}),
        new VitePlugin({
            // `build` can specify multiple entry builds, which can be Main process, Preload scripts, Worker process, etc.
            // If you are familiar with Vite configuration, it will look really familiar.
            build: [
                {
                    // `entry` is just an alias for `build.lib.entry` in the corresponding file of `config`.
                    entry: 'src-electron/main.ts',
                    config: 'vite.main.config.ts',
                    target: 'main',
                },
                {
                    entry: 'src-electron/preload.ts',
                    config: 'vite.preload.config.ts',
                    target: 'preload',
                },
            ],
            renderer: [
                {
                    name: 'px_window',
                    config: 'vite.config.ts',
                },
            ],
        }),
        // Fuses are used to enable/disable various Electron functionality
        // at package time, before code signing the application
        new FusesPlugin({
            version: FuseVersion.V1,
            [FuseV1Options.RunAsNode]: false,
            [FuseV1Options.EnableCookieEncryption]: true,
            [FuseV1Options.EnableNodeOptionsEnvironmentVariable]: false,
            [FuseV1Options.EnableNodeCliInspectArguments]: false,
            [FuseV1Options.EnableEmbeddedAsarIntegrityValidation]: true,
            [FuseV1Options.OnlyLoadAppFromAsar]: true,
        }),
    ],
};

export default config;
