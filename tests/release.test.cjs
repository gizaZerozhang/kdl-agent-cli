'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const crypto = require('node:crypto');
const { TARGETS, target } = require('../scripts/runtime/install.cjs');
const { verify } = require('../scripts/verify-package.cjs');

function candidate(t, version = '0.1.0-beta.1') {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kdl-package-check-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const pkg = { name: '@kuaidaili/kdl-agent', version, repository: { url: 'git+https://github.com/kuaidaili/kdl-agent-cli.git' } };
  const files = [...TARGETS].map(item => target(version, ...item.split('-')).name);
  files.push('kdl-agent-skill.zip', 'openapi.yaml', 'BUILD.json', 'install.md', 'cli-guide.md', 'api-guide.md', 'release-notes.md');
  const sums = files.map(file => `${'a'.repeat(64)}  ${file}\n`).join('');
  const record = { version, repository: 'kuaidaili/kdl-agent-cli', commit: 'b'.repeat(40), dirty: false, macosSigned: false, sha256: crypto.createHash('sha256').update(sums).digest('hex') };
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify(pkg));
  fs.writeFileSync(path.join(root, 'release.json'), JSON.stringify(record));
  fs.writeFileSync(path.join(root, 'SHA256SUMS'), sums);
  return { root, record, pkg, sums };
}
test('beta 候选完整性检查通过', t => {
  const c = candidate(t);
  assert.equal(verify(c.root).version, c.pkg.version);
});
test('稳定版缺少签名公证记录时拒绝打包', t => {
  const c = candidate(t, '0.1.0');
  assert.throws(() => verify(c.root), /签名公证/);
});
test('拒绝源码 dirty 或来源、版本不一致', t => {
  const c = candidate(t);
  for (const change of [{ dirty: true }, { commit: 'unknown' }, { version: '9.9.9' }, { repository: 'other/repo' }]) {
    fs.writeFileSync(path.join(c.root, 'release.json'), JSON.stringify({ ...c.record, ...change }));
    assert.throws(() => verify(c.root), /来源、版本或仓库/);
  }
});
test('拒绝缺平台或被替换的校验清单', t => {
  const c = candidate(t);
  fs.writeFileSync(path.join(c.root, 'SHA256SUMS'), c.sums.replace(/^.*\n/, ''));
  assert.throws(() => verify(c.root), /缺少平台/);
  fs.writeFileSync(path.join(c.root, 'SHA256SUMS'), c.sums.replace('a', 'c'));
  assert.throws(() => verify(c.root), /校验清单/);
});
