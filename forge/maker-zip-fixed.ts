import path from 'node:path';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import type {ForgeConfigMakerOptions} from '@electron-forge/shared-types';
import {MakerBase} from '@electron-forge/maker-base';
import fsExtra from 'fs-extra';
import got from 'got';

const execFileAsync = promisify(execFile);

const quoteForPowershell = (value: string): string => `"${value.replace(/"/g, '""')}"`;

async function createZipArchive(inputPath: string, outputPath: string, platform: NodeJS.Platform): Promise<void> {
    if (platform === 'win32') {
        const args = [
            '-nologo',
            '-noprofile',
            '-command',
            '& { param([String]$src, [String]$dest); Add-Type -A "System.IO.Compression.FileSystem"; ' +
                'if (Test-Path -LiteralPath $dest) { Remove-Item -LiteralPath $dest -Force; } ' +
                '[IO.Compression.ZipFile]::CreateFromDirectory($src, $dest); if ($?) { exit 0 } else { exit 1 } }',
            '-src',
            quoteForPowershell(inputPath),
            '-dest',
            quoteForPowershell(outputPath),
        ];

        await execFileAsync('powershell.exe', args, {
            cwd: path.dirname(inputPath),
            maxBuffer: Infinity,
        });
        return;
    }

    const parentDirectory = path.dirname(inputPath);
    const entryName = path.basename(inputPath);
    await execFileAsync('zip', ['-r', '-y', outputPath, entryName], {
        cwd: parentDirectory,
        maxBuffer: Infinity,
    });
}

type MakerZipConfig = ForgeConfigMakerOptions['zip'];

type MakeParams = Parameters<MakerBase<MakerZipConfig>['make']>[0];

type ReleaseManifest = {
    currentRelease: string;
    releases: Array<{
        version: string;
        updateTo: {
            name: string;
            version: string;
            pub_date: string;
            url: string;
            notes: string;
        };
    }>;
};

export default class MakerZipFixed extends MakerBase<MakerZipConfig> {
    name = 'zip';

    defaultPlatforms = ['darwin', 'mas', 'win32', 'linux'] as const;

    isSupportedOnCurrentPlatform(): boolean {
        return true;
    }

    async make({dir, makeDir, appName, packageJSON, targetArch, targetPlatform}: MakeParams): Promise<string[]> {
        const zipDir = ['darwin', 'mas'].includes(targetPlatform) ? path.resolve(dir, `${appName}.app`) : dir;
        const zipName = `${path.basename(dir)}-${packageJSON.version}.zip`;
        const zipPath = path.resolve(makeDir, 'zip', targetPlatform, targetArch, zipName);

        await this.ensureFile(zipPath);
        await createZipArchive(zipDir, zipPath, process.platform);

        if (targetPlatform === 'darwin' && this.config?.macUpdateManifestBaseUrl) {
            const manifestUrl = new URL(this.config.macUpdateManifestBaseUrl);
            manifestUrl.pathname += '/RELEASES.json';

            const response = await got.get(manifestUrl.toString(), {
                throwHttpErrors: false,
            });

            let manifest: ReleaseManifest = {
                currentRelease: '',
                releases: [],
            };

            if (response.statusCode === 200) {
                manifest = JSON.parse(response.body) as ReleaseManifest;
            }

            const updateUrl = new URL(this.config.macUpdateManifestBaseUrl);
            updateUrl.pathname += `/${zipName}`;

            manifest.releases = (manifest.releases || []).filter((release) => release.version !== packageJSON.version);
            manifest.currentRelease = packageJSON.version;
            manifest.releases.push({
                version: packageJSON.version,
                updateTo: {
                    name: `${appName} v${packageJSON.version}`,
                    version: packageJSON.version,
                    pub_date: new Date().toISOString(),
                    url: updateUrl.toString(),
                    notes: this.config.macUpdateReleaseNotes || '',
                },
            });

            const releasesPath = path.resolve(makeDir, 'zip', targetPlatform, targetArch, 'RELEASES.json');
            await this.ensureFile(releasesPath);
            await fsExtra.writeJson(releasesPath, manifest);
            return [zipPath, releasesPath];
        }

        return [zipPath];
    }
}
