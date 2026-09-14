'use strict';
const fs = require('node:fs');
const path = require('node:path');
const { spawnSync, execFileSync } = require('node:child_process');
const { config, checksums, hashFile } = require('./runtime/install.cjs');
const { verify } = require('./verify-package.cjs');
const { checkRelease, validateSigning } = require('./release-policy.cjs');

async function prepare(input, output) {
  const root = path.resolve(__dirname, '..');
  const { pkg, repository } = config(root);
  checkRelease(`v${pkg.version}`, { root });
  if (fs.existsSync(output)) throw new Error('npm 候选目录已存在，拒绝覆盖');
  const commit = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim();
  if (execFileSync('git', ['status', '--porcelain'], { cwd: root, encoding: 'utf8' }).trim()) throw new Error('发行打包要求干净源码；先提交改动');
  const sums = fs.readFileSync(path.join(input, 'SHA256SUMS'));
  const source = JSON.parse(fs.readFileSync(path.join(input, 'BUILD.json'), 'utf8'));
  if (source.commit !== commit || source.version !== pkg.version || source.dirty !== false || source.repository !== repository) throw new Error('原生候选不属于当前干净源码与版本');
  for (const [name, expected] of checksums(sums.toString())) {
    if (await hashFile(path.join(input, name)) !== expected) throw new Error(`候选文件校验失败：${name}`);
  }
  const signed = fs.existsSync(path.join(input, 'macos-signing-verified.json'));
  if (signed) {
    const record = JSON.parse(fs.readFileSync(path.join(input, 'macos-signing-verified.json'), 'utf8'));
    validateSigning(record, source, await hashFile(path.join(input, 'SHA256SUMS')), checksums(sums.toString()));
  }
  fs.mkdirSync(output, { recursive: true });
  const stage = path.join(output, 'package');
  fs.mkdirSync(stage);
  const runtimeFiles = ['package.json', 'LICENSE', 'README.md', 'CHANGELOG.md', 'THIRD_PARTY_NOTICES.txt', 'scripts/run.cjs', 'scripts/install.cjs', 'scripts/wizard.cjs', 'scripts/verify-package.cjs', 'scripts/runtime/install.cjs'];
  for (const name of runtimeFiles) {
    fs.mkdirSync(path.dirname(path.join(stage, name)), { recursive: true });
    fs.copyFileSync(path.join(root, name), path.join(stage, name));
  }
  fs.writeFileSync(path.join(stage, 'SHA256SUMS'), sums);
  const release = { version: pkg.version, repository, commit, dirty: false, macosSigned: signed, sha256: await hashFile(path.join(input, 'SHA256SUMS')) };
  fs.writeFileSync(path.join(stage, 'release.json'), JSON.stringify(release, null, 2) + '\n');
  verify(stage);
  const npmCLI = process.env.npm_execpath;
  const args = ['pack', '--ignore-scripts', '--json', '--pack-destination', path.resolve(output)];
  const result = npmCLI ? spawnSync(process.execPath, [npmCLI, ...args], { cwd: stage, encoding: 'utf8' }) : spawnSync('npm', args, { cwd: stage, encoding: 'utf8' });
  if (result.status !== 0) throw new Error('npm pack 失败：' + result.stderr);
  const packed = JSON.parse(result.stdout);
  if (packed.length !== 1 || packed[0].name !== pkg.name || packed[0].version !== pkg.version) throw new Error('npm pack 输出与候选身份不一致');
  const allowed = new Set([...runtimeFiles, 'SHA256SUMS', 'release.json']);
  for (const file of packed[0].files) if (!allowed.has(file.path)) throw new Error(`npm 包含非允许文件：${file.path}`);
  for (const name of allowed) if (!packed[0].files.some(file => file.path === name)) throw new Error(`npm 包缺少必要文件：${name}`);
  fs.writeFileSync(path.join(output, 'pack-report.json'), JSON.stringify(packed, null, 2) + '\n');
  console.log(`npm 候选：${path.join(output, packed[0].filename)}`);
}
if (require.main === module) {
  const [input, output] = process.argv.slice(2);
  if (!input || !output) { console.error('用法：npm run release:prepare -- <原生候选目录> <新 npm 候选目录>'); process.exitCode = 2; }
  else prepare(path.resolve(input), path.resolve(output)).catch(error => { console.error(error.message); process.exitCode = 1; });
}
module.exports = { prepare };
