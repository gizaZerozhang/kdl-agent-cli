'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const tar = require('tar');
const yazl = require('yazl');
const { pipeline } = require('node:stream/promises');
const { ensureInstalled, hashFile, target, checksums, allowedURL, extractBinary } = require('../scripts/runtime/install.cjs');
const { parse } = require('../scripts/wizard.cjs');

function temporary(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kdl-npm-test-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  return root;
}
async function fixture(t, platform = 'linux', arch = 'x64') {
  const root = temporary(t);
  const source = path.join(root, 'source');
  const packageRoot = path.join(root, 'package');
  fs.mkdirSync(source); fs.mkdirSync(packageRoot);
  const version = '0.1.0-beta.1';
  const artifact = target(version, platform, arch);
  const archive = path.join(root, artifact.name);
  fs.writeFileSync(path.join(source, artifact.binary), 'candidate-binary');
  if (archive.endsWith('.zip')) {
    const zip = new yazl.ZipFile();
    zip.addBuffer(Buffer.from('candidate-binary'), artifact.binary);
    zip.end();
    await pipeline(zip.outputStream, fs.createWriteStream(archive));
  } else await tar.c({ file: archive, cwd: source, gzip: true }, [artifact.binary]);
  fs.writeFileSync(path.join(packageRoot, 'package.json'), JSON.stringify({ version, repository: { url: 'git+https://github.com/gizaZerozhang/kdl-agent-cli.git' } }));
  fs.writeFileSync(path.join(packageRoot, 'release.json'), JSON.stringify({ version, repository: 'gizaZerozhang/kdl-agent-cli', commit: 'a'.repeat(40), dirty: false }));
  fs.writeFileSync(path.join(packageRoot, 'SHA256SUMS'), `${await hashFile(archive)}  ${artifact.name}\n`);
  const options = { root: packageRoot, platform, arch, verify(file) { assert.equal(fs.readFileSync(file, 'utf8'), 'candidate-binary'); }, async fetchFile(url, destination) {
    assert.equal(url, `https://github.com/gizaZerozhang/kdl-agent-cli/releases/download/v${version}/${artifact.name}`);
    fs.copyFileSync(archive, destination);
  } };
  return { root, source, packageRoot, archive, artifact, options };
}

test('平台映射、版本固定和不支持的平台', () => {
  assert.equal(target('0.1.0-beta.1', 'win32', 'x64').name, 'kdl-agent_0.1.0-beta.1_windows_amd64.zip');
  assert.equal(target('0.1.0-beta.1', 'darwin', 'arm64').binary, 'kdl-agent');
  assert.throws(() => target('1.0.0', 'win32', 'arm64'), /不支持/);
});
test('校验清单拒绝缺失格式、重复和路径', () => {
  const entry = `${'a'.repeat(64)}  app.zip`;
  assert.throws(() => checksums(`${entry}\n${entry}`), /重复/);
  assert.throws(() => checksums(''), /格式/);
  assert.throws(() => checksums(`${'a'.repeat(64)}  ../app.zip`), /格式/);
});
test('下载及所有重定向目标必须是允许的 HTTPS 域名', () => {
  for (const url of ['http://github.com/a', 'https://github.com.evil.test/a', 'https://user:secret@github.com/a', 'https://github.com:8443/a']) assert.throws(() => allowedURL(url));
  assert.equal(allowedURL('https://release-assets.githubusercontent.com/a?token=opaque').hostname, 'release-assets.githubusercontent.com');
});
for (const platform of ['linux', 'win32']) {
  test(`${platform} 归档安装成功、同版本跳过、不残留安装锁`, async t => {
    const f = await fixture(t, platform);
    const binary = await ensureInstalled(f.options);
    assert.equal(fs.readFileSync(binary, 'utf8'), 'candidate-binary');
    assert.equal(await ensureInstalled({ ...f.options, fetchFile() { throw new Error('不应再次下载'); } }), binary);
    assert.equal(fs.existsSync(path.join(f.packageRoot, '.native/.lock')), false);
  });
}
test('损坏归档不替换旧程序并允许重试', async t => {
  const f = await fixture(t);
  const directory = path.join(f.packageRoot, '.native');
  fs.mkdirSync(directory);
  const binary = path.join(directory, f.artifact.binary);
  fs.writeFileSync(binary, 'old-binary');
  await assert.rejects(ensureInstalled({ ...f.options, fetchFile(url, file) { fs.writeFileSync(file, 'corrupt'); } }), /SHA256/);
  assert.equal(fs.readFileSync(binary, 'utf8'), 'old-binary');
  await ensureInstalled(f.options);
  assert.equal(fs.readFileSync(binary, 'utf8'), 'candidate-binary');
});
test('下载失败和版本不符保留旧程序', async t => {
  const f = await fixture(t);
  fs.mkdirSync(path.join(f.packageRoot, '.native'));
  const binary = path.join(f.packageRoot, '.native', f.artifact.binary);
  fs.writeFileSync(binary, 'old-binary');
  await assert.rejects(ensureInstalled({ ...f.options, fetchFile() { throw new Error('timeout'); } }), /timeout/);
  await assert.rejects(ensureInstalled({ ...f.options, verify() { throw new Error('version mismatch'); } }), /version mismatch/);
  assert.equal(fs.readFileSync(binary, 'utf8'), 'old-binary');
});
test('并发安装失败关闭且不删除他人的锁', async t => {
  const f = await fixture(t);
  const lock = path.join(f.packageRoot, '.native/.lock');
  fs.mkdirSync(lock, { recursive: true });
  await assert.rejects(ensureInstalled(f.options), /另一个安装/);
  assert.equal(fs.existsSync(lock), true);
});
test('安装目录拒绝符号链接', async t => {
  const f = await fixture(t);
  const directory = path.join(f.root, 'external'); fs.mkdirSync(directory);
  fs.symlinkSync(directory, path.join(f.packageRoot, '.native'), process.platform === 'win32' ? 'junction' : 'dir');
  await assert.rejects(ensureInstalled(f.options), /符号链接/);
  assert.deepEqual(fs.readdirSync(directory), []);
});
test('归档缺少二进制时失败', async t => {
  const f = await fixture(t);
  fs.writeFileSync(path.join(f.source, 'README.md'), 'not a binary');
  const archive = path.join(f.root, 'empty.tar.gz');
  await tar.c({ file: archive, cwd: f.source, gzip: true }, ['README.md']);
  await assert.rejects(extractBinary(archive, 'kdl-agent', path.join(f.root, 'out')), /缺少/);
});
test('向导参数明确区分安装、Skill 与登录', () => {
  assert.deepEqual(parse(['--yes', '--agent', 'codex', '--no-login']), { yes: true, noLogin: true, noSkills: false, agents: ['codex'] });
  assert.throws(() => parse(['--agent']), /需要/);
  assert.throws(() => parse(['--token', 'sentinel']), /未知/);
});
