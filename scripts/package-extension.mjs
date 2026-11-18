import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execSync } from 'node:child_process';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(__dirname, '..');
const extDir = path.join(rootDir, 'extension');
const iconsDir = path.join(extDir, 'icons');
const exportsDir = path.join(rootDir, 'storage', 'exports');
const logoSrc = path.join(rootDir, 'logo.png');

fs.mkdirSync(iconsDir, { recursive: true });
fs.mkdirSync(exportsDir, { recursive: true });

if (fs.existsSync(logoSrc)) {
  fs.copyFileSync(logoSrc, path.join(extDir, 'logo.png'));
  for (const sz of [16, 48, 128]) {
    const outPath = path.join(iconsDir, `icon${sz}.png`);
    execSync(`sips -z ${sz} ${sz} "${logoSrc}" --out "${outPath}"`, { stdio: 'ignore' });
  }
}

const zipPath = path.join(exportsDir, 'trendscout-chrome-extension.zip');
try {
  execSync(`cd "${extDir}" && zip -r "${zipPath}" .`, { stdio: 'inherit' });
  console.log(`Packaged Chrome Extension to ${zipPath}`);
} catch (err) {
  console.error('Zip warning:', err);
}
