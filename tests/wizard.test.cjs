'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { main } = require('../scripts/wizard.cjs');
const { config } = require('../scripts/runtime/install.cjs');

test('旧 scope 冲突停止全局修改并提供精确版本恢复入口', async t => {
  const f = fixture(t);
  const legacy = '@zerozhang-giza/kdl-agent';
  const root = path.join(f.prefix, process.platform === 'win32' ? 'node_modules' : 'lib/node_modules', ...legacy.split('/'));
  fs.mkdirSync(root, { recursive: true });
  const original = JSON.stringify({ name: legacy, version: '0.1.0-beta.4' });
  fs.writeFileSync(path.join(root, 'package.json'), original);
  await assert.rejects(main(['--yes', '--agent', 'codex', '--no-login'], f.deps), error => {
    assert.match(error.message, /npm uninstall -g @zerozhang-giza\/kdl-agent/);
    assert.match(error.message, /npm install -g @zerozhang-giza\/kdl-agent@0.1.0-beta.4/);
    return true;
  });
  assert.deepEqual(f.calls.map(call => call.args[0]), ['prefix']);
  assert.equal(fs.readFileSync(path.join(root, 'package.json'), 'utf8'), original);
  fs.rmSync(root, { recursive: true });
  await main(['--yes', '--agent', 'codex', '--no-login'], f.deps);
  assert.ok(f.calls.some(call => call.args.includes(`${config().pkg.name}@${config().pkg.version}`)));
});

test('旧包元数据损坏仍拒绝覆盖，不生成未经校验的恢复命令', async t => {
  const f = fixture(t);
  const root = path.join(f.prefix, process.platform === 'win32' ? 'node_modules' : 'lib/node_modules', '@zerozhang-giza', 'kdl-agent');
  fs.mkdirSync(root, { recursive: true });
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ name: '@zerozhang-giza/kdl-agent', version: '1.0.0;echo unsafe' }));
  await assert.rejects(main(['--yes', '--no-skills', '--no-login'], f.deps), error => {
    assert.match(error.message, /先记录旧包的精确版本/);
    assert.ok(!error.message.includes('unsafe'));
    return true;
  });
  assert.equal(f.calls.length, 1);
});

test('更新帮助使用当前发行配置中的包名', async t => {
  const f = fixture(t); f.deps.mode = 'update';
  await main(['--help'], f.deps);
  assert.ok(f.output.join('\n').includes(`${config().pkg.name}@<目标版本>`));
  assert.equal(f.calls.length, 0);
});

test('取消升级不会下载或修改安装', async t => {
  const f = fixture(t);
  f.deps.mode = 'update'; f.deps.tty = true;
  f.deps.confirm = async () => false;
  f.deps.install = async () => assert.fail('不应下载');
  await main(['--agent', 'codex'], f.deps);
  assert.equal(f.calls.length, 0);
});
test('非交互升级必须确认，不能跳过 Skill', async t => {
  const f = fixture(t); f.deps.mode = 'update';
  await assert.rejects(main(['--agent', 'codex'], f.deps), /非交互/);
  await assert.rejects(main(['--yes', '--no-skills'], f.deps), /不能跳过/);
  await assert.rejects(main(['--yes'], f.deps), /指定目标/);
  assert.equal(f.calls.length, 0);
});
test('确认升级同时安装 Skill，不访问登录凭证', async t => {
  const f = fixture(t); f.deps.mode = 'update';
  await main(['--yes', '--agent', 'codex'], f.deps);
  assert.deepEqual(f.calls.map(c => c.name), ['npm', 'npm', 'npx']);
});
test('同版本修复 Skill 跳过全局 npm 安装', async t => {
  const f = fixture(t); f.deps.mode = 'update';
  const root = path.join(f.prefix, process.platform === 'win32' ? 'node_modules' : 'lib/node_modules', ...config().pkg.name.split('/'));
  fs.mkdirSync(root, { recursive: true });
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ version: config().pkg.version }));
  f.deps.checkBinary = () => {};
  await main(['--yes', '--agent', 'codex'], f.deps);
  assert.deepEqual(f.calls.map(c => c.name), ['npm', 'npx']);
  assert.equal(f.calls[0].args[0], 'prefix');
});

function fixture(t) {
  const prefix = fs.mkdtempSync(path.join(os.tmpdir(), 'kdl-wizard-'));
  t.after(() => fs.rmSync(prefix, { recursive: true, force: true }));
  const calls = [];
  const output = [];
  t.mock.method(console, 'log', text => output.push(text));
  t.mock.method(console, 'error', text => output.push(text));
  const deps = { tty: false, readConfig: config, async install() {}, run(name, args, options) {
    calls.push({ name, args, options });
    return { status: 0, stdout: args[0] === 'prefix' ? prefix : 'PRIVATE_RESPONSE_SENTINEL' };
  } };
  return { deps, prefix, calls, output };
}
test('固定版本安装 CLI 和 Skill，no-login 不发认证请求', async t => {
  const f = fixture(t);
  await main(['--yes', '--agent', 'codex', '--no-login'], f.deps);
  assert.equal(f.calls.length, 3);
  assert.ok(f.calls[1].args.includes(`${config().pkg.name}@${config().pkg.version}`));
  assert.ok(f.calls[2].args.includes(`https://github.com/${config().repository}/tree/v${config().pkg.version}/skills/kdl-agent`));
  assert.ok(f.calls[2].args.includes('--agent'));
});
test('预下载失败不修改全局包', async t => {
  const f = fixture(t);
  f.deps.install = async () => { throw new Error('checksum failure'); };
  await assert.rejects(main(['--yes', '--no-skills', '--no-login'], f.deps), /checksum/);
  assert.equal(f.calls.length, 0);
});
test('Skill 失败保留 CLI，并阻止后续登录', async t => {
  const f = fixture(t);
  const run = f.deps.run;
  f.deps.run = (name, args, options) => name === 'npx' ? { status: 1 } : run(name, args, options);
  await assert.rejects(main(['--yes', '--agent', 'codex'], f.deps), /CLI 已安装，Skill 安装未完成/);
  assert.equal(f.calls.length, 2);
});
test('全局升级失败尝试恢复原精确版本', async t => {
  const f = fixture(t);
  const root = path.join(f.prefix, process.platform === 'win32' ? 'node_modules' : 'lib/node_modules', ...config().pkg.name.split('/'));
  fs.mkdirSync(root, { recursive: true });
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ version: '0.0.9' }));
  const run = f.deps.run;
  f.deps.run = (name, args, options) => {
    const result = run(name, args, options);
    return args.includes(`${config().pkg.name}@${config().pkg.version}`) ? { status: 1 } : result;
  };
  await assert.rejects(main(['--yes', '--no-skills', '--no-login'], f.deps), /全局安装失败/);
  assert.ok(f.calls[2].args.includes(`${config().pkg.name}@0.0.9`));
});
test('远端状态与首次查询通过，不记录捕获的响应', async t => {
  const f = fixture(t);
  await main(['--yes', '--no-skills'], f.deps);
  assert.deepEqual(f.calls.slice(-2).map(call => call.args.slice(0, 2)), [['auth', 'status'], ['account', 'summary']]);
  assert.ok(!f.output.join('\n').includes('PRIVATE_RESPONSE_SENTINEL'));
});
test('非交互登录未完成时保留安装结果并提示本机登录', async t => {
  const f = fixture(t);
  const run = f.deps.run;
  f.deps.run = (name, args, options) => args[0] === 'auth' ? { status: 2 } : run(name, args, options);
  await assert.rejects(main(['--yes', '--no-skills'], f.deps), /登录待用户在本地终端完成/);
});
