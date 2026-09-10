'use strict';
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');
const { ensureInstalled, target } = require('./runtime/install.cjs');
const { verify } = require('./verify-package.cjs');

async function main(root, candidate) {
  const release = verify(root);
  const artifact = target(release.version);
  const binary = await ensureInstalled({ root, async fetchFile(url, destination) {
    assert.ok(url.endsWith(`/v${release.version}/${artifact.name}`));
    fs.copyFileSync(path.join(candidate, artifact.name), destination);
  } });
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'kdl-release-smoke-'));
  try {
    const env = { ...process.env, HOME: home, USERPROFILE: home };
    for (const name of Object.keys(env)) if (name.startsWith('KDL_AGENT_')) delete env[name];
    const run = args => spawnSync(binary, args, { cwd: home, env, encoding: 'utf8', timeout: 15000 });
    assert.equal(run(['--version']).stdout.trim(), `kdl-agent ${release.version} (commit ${release.commit})`);
    assert.equal(run(['--help']).status, 0);
    assert.equal(run(['--print-paths']).status, 0);
    const status = run(['auth', 'status', '--format', 'json']);
    assert.notEqual(status.status, 0);
    assert.equal(JSON.parse(status.stdout).status, 'unconfigured');
    assert.equal(await ensureInstalled({ root, fetchFile() { throw new Error('同版本不应重复下载'); } }), binary);
    console.log(`候选安装与跨目录烟测通过：${process.platform}/${process.arch} ${release.version}`);
  } finally { fs.rmSync(home, { recursive: true, force: true }); }
}
main(...process.argv.slice(2).map(item => path.resolve(item))).catch(error => { console.error(error.message); process.exitCode = 1; });
