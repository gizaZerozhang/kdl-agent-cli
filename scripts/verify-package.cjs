'use strict';
const fs = require('node:fs');
const path = require('node:path');
const { config, checksums, TARGETS, target } = require('./runtime/install.cjs');
function verify(root = path.resolve(__dirname, '..')) {
  const { pkg, repository } = config(root);
  if (pkg.name.includes('pending') || !/^@[a-z0-9][a-z0-9._-]*\/[a-z0-9][a-z0-9._-]*$/.test(pkg.name)) throw new Error('请先绑定实际 npm 包名');
  const release = JSON.parse(fs.readFileSync(path.join(root, 'release.json'), 'utf8'));
  if (release.version !== pkg.version || release.repository !== repository || !/^[a-f0-9]{40}$/.test(release.commit) || release.dirty !== false) throw new Error('发行来源、版本或仓库不一致');
  if (!pkg.version.includes('-beta.') && !release.macosSigned) throw new Error('稳定发行缺少 macOS 签名公证记录');
  const sums = checksums(fs.readFileSync(path.join(root, 'SHA256SUMS'), 'utf8'));
  if (!pkg.version.includes('-beta.') && !sums.has('stable-acceptance.json')) throw new Error('稳定发行缺少配套验收记录');
  for (const item of TARGETS) {
    const [platform, arch] = item.split('-');
    if (!sums.has(target(pkg.version, platform, arch).name)) throw new Error(`缺少平台校验：${item}`);
  }
  for (const file of ['kdl-agent-skill.zip', 'openapi.yaml', 'BUILD.json', 'install.md', 'cli-guide.md', 'api-guide.md', 'release-notes.md']) if (!sums.has(file)) throw new Error(`缺少配套资料校验：${file}`);
  if (release.sha256 !== require('node:crypto').createHash('sha256').update(fs.readFileSync(path.join(root, 'SHA256SUMS'))).digest('hex')) throw new Error('校验清单与发行记录不一致');
  return release;
}
if (require.main === module) {
  try { const release = verify(); console.log(`发行检查通过：${release.version} (${release.commit})`); }
  catch (error) { console.error(`发行检查失败：${error.message}`); process.exitCode = 1; }
}
module.exports = { verify };
