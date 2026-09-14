'use strict';
const fs = require('node:fs');
const crypto = require('node:crypto');
const { spawnSync } = require('node:child_process');
const pkg = require('../package.json');
const { releaseChannel } = require('./release-policy.cjs');

async function registryIntegrity(fetcher = fetch) {
  const response = await fetcher(`https://registry.npmjs.org/${encodeURIComponent(pkg.name)}/${pkg.version}`, { signal: AbortSignal.timeout(15000), redirect: 'error' });
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`npm registry 检查失败：HTTP ${response.status}`);
  const metadata = await response.json();
  if (metadata.name !== pkg.name || metadata.version !== pkg.version || !metadata.dist?.integrity) throw new Error('npm metadata 不完整');
  return metadata.dist.integrity;
}

async function main(file, { read = registryIntegrity, run = spawnSync, version = pkg.version, sleep = ms => new Promise(resolve => setTimeout(resolve, ms)) } = {}) {
  if (!file) throw new Error('需要已核验的 npm-package.tgz 路径');
  const channel = releaseChannel(version);
  const expected = 'sha512-' + crypto.createHash('sha512').update(fs.readFileSync(file)).digest('base64');
  const existing = await read();
  if (existing) {
    if (existing !== expected) throw new Error('已发布版本与候选内容冲突，禁止覆盖');
    console.log('精确版本已发布且内容一致，跳过重复发布；不移动 dist-tag。');
    return;
  }
  const result = run('npm', ['publish', file, '--access', 'public', '--tag', channel, '--provenance'], { stdio: 'inherit', timeout: 180000 });
  if (result.status !== 0) throw new Error('npm 发布失败；核对 registry 后重试同一候选');
  for (let attempt = 0; attempt < 6; attempt++) {
    const actual = await read();
    if (actual === expected) { console.log('npm 发布完成，registry integrity 与固定候选一致。'); return; }
    if (actual) throw new Error('npm 发布后内容不一致');
    await sleep(5000);
  }
  throw new Error('npm 发布后的 registry 可见性待确认；请重试同一工作流');
}
if (require.main === module) main(process.argv[2]).catch(error => { console.error(error.message); process.exitCode = 1; });
module.exports = { main, registryIntegrity };
