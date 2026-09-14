'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { releaseChannel, checkRelease, validateAcceptance, validateSigning, CHECKS, SIGNING_ENV } = require('../scripts/release-policy.cjs');

function acceptance() {
  return { schema_version: 1, version: '0.1.0', reviewer: 'test-reviewer',
    checks: CHECKS.map(id => ({ id, status: 'passed', evidence: `https://example.invalid/evidence/${id}` })) };
}

test('稳定和 beta 渠道严格区分，拒绝开发版和非法版本', () => {
  assert.equal(releaseChannel('0.1.0'), 'latest');
  assert.equal(releaseChannel('0.1.0-beta.6'), 'beta');
  for (const version of ['0.1.0-rc.1', '01.1.0', '0.1.0-beta.01', '0.1.0;exit 0', '0.1.0+local']) {
    assert.throws(() => releaseChannel(version), /发行版本/);
  }
});

test('稳定版必须具备同版本完整验收证据，不接受缺项、失败或重复', () => {
  assert.equal(validateAcceptance(acceptance(), '0.1.0').version, '0.1.0');
  for (const mutate of [
    value => { value.version = '0.2.0'; }, value => { value.reviewer = ''; },
    value => { value.checks.pop(); }, value => { value.checks[1] = value.checks[0]; },
    value => { value.checks[0].status = 'pending'; }, value => { value.checks[0].evidence = ''; },
  ]) {
    const value = acceptance(); mutate(value);
    assert.throws(() => validateAcceptance(value, '0.1.0'), /验收/);
  }
});

test('稳定 tag 缺验收或签名配置时停止，beta 不要求 Apple 账号', t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kdl-policy-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const writePackage = version => fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ version }));
  writePackage('0.1.0');
  assert.throws(() => checkRelease('v0.1.0', { root }), /缺少/);
  fs.mkdirSync(path.join(root, 'release'));
  fs.writeFileSync(path.join(root, 'release/stable-acceptance.json'), JSON.stringify(acceptance()));
  assert.throws(() => checkRelease('v0.1.0', { root, env: {}, signing: true }), /签名配置/);
  const env = Object.fromEntries(SIGNING_ENV.map(name => [name, 'sentinel']));
  assert.equal(checkRelease('v0.1.0', { root, env, signing: true }), 'latest');
  assert.throws(() => checkRelease('v0.2.0', { root, env }), /不一致/);
  writePackage('0.1.0-beta.6');
  assert.equal(checkRelease('v0.1.0-beta.6', { root, env: {}, signing: true }), 'beta');
});

test('签名记录必须同时绑定双架构、来源和完整候选', () => {
  const source = { version: '0.1.0', commit: 'a'.repeat(40) };
  const sums = new Map(['arm64', 'amd64'].map(arch => [`kdl-agent_0.1.0_darwin_${arch}.tar.gz`, 'b'.repeat(64)]));
  const record = { ...source, sha256: 'c'.repeat(64), team_id: 'ABCDEFGHIJ',
    archives: Object.fromEntries(['arm64', 'amd64'].map(arch => [arch, { verified: true, sha256: 'b'.repeat(64) }])) };
  validateSigning(record, source, record.sha256, sums);
  assert.throws(() => validateSigning({ ...record, commit: 'd'.repeat(40) }, source, record.sha256, sums), /当前候选/);
  assert.throws(() => validateSigning(record, source, 'e'.repeat(64), sums), /当前候选/);
  record.archives.amd64.verified = false;
  assert.throws(() => validateSigning(record, source, record.sha256, sums), /双架构/);
});
