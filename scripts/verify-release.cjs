'use strict';
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const tar = require('tar');
const { execFileSync } = require('node:child_process');
const { hashFile, checksums, config } = require('./runtime/install.cjs');
const { verify } = require('./verify-package.cjs');
const { checkRelease, validateSigning } = require('./release-policy.cjs');

async function main(directory) {
  const sumFile = path.join(directory, 'SHA256SUMS');
  for (const [name, expected] of checksums(fs.readFileSync(sumFile, 'utf8'))) {
    if (await hashFile(path.join(directory, name)) !== expected) throw new Error(`Release 附件校验失败：${name}`);
  }
  const npmSums = checksums(fs.readFileSync(path.join(directory, 'NPM-SHA256'), 'utf8'));
  if (npmSums.size !== 1 || npmSums.get('npm-package.tgz') !== await hashFile(path.join(directory, 'npm-package.tgz'))) throw new Error('npm 候选校验失败');
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'kdl-release-verify-'));
  try {
    const archive = path.join(directory, 'npm-package.tgz');
    let unsafe = false;
    await tar.t({ file: archive, strict: true, onReadEntry(entry) {
      if (!entry.path.startsWith('package/') || entry.path.includes('..') || entry.path.includes('\\') || !['File', 'Directory'].includes(entry.type)) unsafe = true;
    } });
    if (unsafe) throw new Error('npm 候选包含非法路径或链接');
    await tar.x({ file: archive, cwd: temporary, strict: true });
    const root = path.join(temporary, 'package');
    const release = verify(root);
    const { pkg, repository } = config();
    const packed = config(root).pkg;
    const commit = execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
    if (release.commit !== commit || release.version !== pkg.version || release.repository !== repository || packed.name !== pkg.name) throw new Error('npm 候选与当前 tag 源码不一致');
    if (!fs.readFileSync(path.join(root, 'SHA256SUMS')).equals(fs.readFileSync(sumFile))) throw new Error('npm 和 Release 校验清单不一致');
    if (release.macosSigned) {
      const proof = JSON.parse(fs.readFileSync(path.join(directory, 'macos-signing-verified.json')));
      validateSigning(proof, release, await hashFile(sumFile), checksums(fs.readFileSync(sumFile, 'utf8')));
    }
    if (checkRelease(`v${pkg.version}`) === 'latest') {
      if (!fs.readFileSync(path.join(directory, 'stable-acceptance.json')).equals(fs.readFileSync(path.join(__dirname, '../release/stable-acceptance.json')))) throw new Error('稳定验收记录与 tag 不一致');
    }
    console.log(`Release 与 npm 候选一致：${pkg.name}@${release.version} ${commit}`);
  } finally { fs.rmSync(temporary, { recursive: true, force: true }); }
}
if (require.main === module) main(path.resolve(process.argv[2])).catch(error => { console.error(error.message); process.exitCode = 1; });
module.exports = { main };
