'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const { main, registryIntegrity } = require('../scripts/publish-candidate.cjs');
const file = require.resolve('../package.json');
const integrity = 'sha512-' + crypto.createHash('sha512').update(fs.readFileSync(file)).digest('base64');
test('重复发布一致时不执行 npm，不一致时失败', async () => {
  const run = () => assert.fail('不能发布');
  await main(file, { read: async () => integrity, run });
  await assert.rejects(main(file, { read: async () => 'sha512-other', run }), /冲突/);
});
test('新发布校验远端候选，命令保留 beta/provenance', async () => {
  let reads = 0;
  await main(file, { read: async () => ++reads === 1 ? null : integrity, run(name, args) {
    assert.equal(name, 'npm'); assert.ok(args.includes('--provenance')); assert.ok(args.includes('beta'));
    return { status: 0 };
  } });
  assert.equal(reads, 2);
});
test('发布失败和 registry 故障不会报告成功', async () => {
  await assert.rejects(main(file, { read: async () => null, run: () => ({ status: 1 }) }), /发布失败/);
  await assert.rejects(registryIntegrity(async () => ({ status: 503, ok: false })), /HTTP 503/);
  assert.equal(await registryIntegrity(async () => ({ status: 404 })), null);
});
