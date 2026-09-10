'use strict';
const fs = require('node:fs');
const path = require('node:path');
const https = require('node:https');
const crypto = require('node:crypto');
const { pipeline } = require('node:stream/promises');
const { Transform } = require('node:stream');
const { spawnSync } = require('node:child_process');
const { HttpsProxyAgent } = require('https-proxy-agent');
const tar = require('tar');
const yauzl = require('yauzl');

const ROOT = path.resolve(__dirname, '../..');
const MAX_BYTES = 256 * 1024 * 1024;
const HOSTS = new Set(['github.com', 'release-assets.githubusercontent.com', 'objects.githubusercontent.com']);
const TARGETS = new Set(['darwin-arm64', 'darwin-amd64', 'linux-arm64', 'linux-amd64', 'windows-amd64']);

function config(root = ROOT) {
  const pkg = JSON.parse(fs.readFileSync(path.join(root, 'package.json'), 'utf8'));
  if (!/^\d+\.\d+\.\d+(?:-beta\.\d+)?$/.test(pkg.version)) throw new Error('不支持的发行版本');
  const url = new URL(pkg.repository.url.replace(/^git\+/, '').replace(/\.git$/, ''));
  if (url.origin !== 'https://github.com' || !/^\/[\w-]+\/[\w.-]+$/.test(url.pathname)) throw new Error('无效源码仓库地址');
  return { pkg, repository: url.pathname.slice(1), root };
}

function target(version, platform = process.platform, arch = process.arch) {
  const system = platform === 'win32' ? 'windows' : platform;
  const cpu = arch === 'x64' ? 'amd64' : arch;
  if (!TARGETS.has(`${system}-${cpu}`)) throw new Error(`不支持的平台：${platform}/${arch}`);
  const ext = system === 'windows' ? 'zip' : 'tar.gz';
  return { name: `kdl-agent_${version}_${system}_${cpu}.${ext}`, binary: system === 'windows' ? 'kdl-agent.exe' : 'kdl-agent' };
}

function checksums(text) {
  const values = new Map();
  for (const line of text.trim().split(/\r?\n/)) {
    const match = /^([a-f0-9]{64})  ([A-Za-z0-9_.-]+)$/.exec(line);
    if (!match || values.has(match[2])) throw new Error('校验清单格式错误或文件重复');
    values.set(match[2], match[1]);
  }
  return values;
}

function allowedURL(raw) {
  const url = new URL(raw);
  if (url.protocol !== 'https:' || url.username || url.password || (url.port && url.port !== '443') || !HOSTS.has(url.hostname)) {
    throw new Error('下载地址不属于允许的 GitHub HTTPS 资源');
  }
  return url;
}

function downloadTimeout() {
  const raw = process.env.KDL_AGENT_DOWNLOAD_TIMEOUT_MS || '60000';
  if (!/^\d+$/.test(raw) || Number(raw) < 1000 || Number(raw) > 600000) throw new Error('KDL_AGENT_DOWNLOAD_TIMEOUT_MS 必须为 1000–600000 毫秒');
  return Number(raw);
}

async function downloadOnce(raw, destination, redirects = 0) {
  const url = allowedURL(raw);
  const timeout = downloadTimeout();
  if (redirects > 5) throw new Error('下载重定向次数过多');
  const proxy = process.env.HTTPS_PROXY || process.env.https_proxy;
  const response = await new Promise((resolve, reject) => {
    const request = https.get(url, { agent: proxy ? new HttpsProxyAgent(proxy) : undefined, headers: { 'User-Agent': 'kdl-agent-installer' } }, resolve);
    request.setTimeout(timeout, () => request.destroy(Object.assign(new Error('下载超时，请检查网络或 HTTPS_PROXY'), { code: 'ETIMEDOUT' })));
    request.on('error', reject);
  });
  if ([301, 302, 303, 307, 308].includes(response.statusCode)) {
    response.resume();
    if (!response.headers.location) throw new Error('下载重定向缺少地址');
    return downloadOnce(new URL(response.headers.location, url).href, destination, redirects + 1);
  }
  if (response.statusCode !== 200) {
    response.resume();
    throw Object.assign(new Error(`下载失败 HTTP ${response.statusCode}；请核对版本及 Release 是否公开`), { statusCode: response.statusCode });
  }
  let size = 0;
  const limit = new Transform({ transform(chunk, encoding, callback) {
    size += chunk.length;
    callback(size > MAX_BYTES ? new Error('发行包超过大小限制') : null, chunk);
  } });
  await pipeline(response, limit, fs.createWriteStream(destination, { flags: 'wx', mode: 0o600 }));
}

// 仅重试临时下载故障；校验、解压与替换仍由安装事务执行一次。
async function download(raw, destination, { attempt = downloadOnce, sleep = ms => new Promise(resolve => setTimeout(resolve, ms)) } = {}) {
  allowedURL(raw);
  downloadTimeout();
  if (fs.existsSync(destination)) throw new Error('下载目标已存在');
  for (let retry = 0; ; retry++) {
    try { return await attempt(raw, destination); } catch (error) {
      fs.rmSync(destination, { force: true });
      const transient = ['ETIMEDOUT', 'ECONNRESET', 'ECONNREFUSED', 'EAI_AGAIN', 'ERR_STREAM_PREMATURE_CLOSE'].includes(error.code) || [408, 500, 502, 503, 504].includes(error.statusCode);
      if (!transient || retry >= 2) throw error;
      await sleep(1000 * 2 ** retry);
    }
  }
}

async function hashFile(file) {
  const hash = crypto.createHash('sha256');
  for await (const chunk of fs.createReadStream(file)) hash.update(chunk);
  return hash.digest('hex');
}

function safeEntry(name) {
  if (!name || name.includes('\\') || name.startsWith('/') || /^[A-Za-z]:/.test(name) || name.split('/').includes('..')) throw new Error('归档包含非法路径');
}

async function extractBinary(archive, binary, destination) {
  let found = false;
  async function save(stream, name, size) {
    if (name !== binary) { stream.resume(); return; }
    if (found || size > MAX_BYTES) throw new Error('归档可执行文件重复或过大');
    found = true;
    await pipeline(stream, fs.createWriteStream(destination, { flags: 'wx', mode: 0o755 }));
  }
  if (archive.endsWith('.zip')) {
    await new Promise((resolve, reject) => yauzl.open(archive, { lazyEntries: true }, (error, zip) => {
      if (error) return reject(error);
      const fail = err => { zip.close(); reject(err); };
      zip.on('error', fail);
      zip.on('end', resolve);
      zip.on('entry', entry => {
        try {
          safeEntry(entry.fileName);
          const mode = entry.externalFileAttributes >>> 16;
          if ((mode & 0o170000) === 0o120000) throw new Error('归档不允许符号链接');
          if (entry.fileName !== binary) return zip.readEntry();
          zip.openReadStream(entry, (err, stream) => {
            if (err) return fail(err);
            save(stream, entry.fileName, entry.uncompressedSize).then(() => zip.readEntry(), fail);
          });
        } catch (err) { fail(err); }
      });
      zip.readEntry();
    }));
  } else {
    const pending = [];
    let invalid;
    await tar.t({ file: archive, strict: true, onReadEntry(entry) {
      try {
        safeEntry(entry.path);
        if (!['File', 'Directory', 'ExtendedHeader', 'GlobalExtendedHeader'].includes(entry.type)) throw new Error('归档不允许链接或特殊文件');
        const task = save(entry, entry.path, entry.size);
        task.catch(() => {});
        pending.push(task);
      } catch (err) { invalid = err; entry.resume(); }
    } });
    await Promise.all(pending);
    if (invalid) throw invalid;
  }
  if (!found) throw new Error('归档缺少可执行文件');
  fs.chmodSync(destination, 0o755);
}

function regular(file) {
  const stat = fs.lstatSync(file);
  if (!stat.isFile() || stat.isSymbolicLink()) throw new Error('程序路径不是普通文件');
}

function verifyBinary(file, version, commit) {
  regular(file);
  const result = spawnSync(file, ['--version'], { encoding: 'utf8', timeout: 15000, windowsHide: true });
  const output = result.stdout?.trim();
  if (result.status !== 0 || !output?.startsWith(`kdl-agent ${version} (commit `) || (commit && output !== `kdl-agent ${version} (commit ${commit})`)) throw new Error('下载程序无法运行或版本、来源不匹配');
}

async function ensureInstalled({ root = ROOT, fetchFile = download, platform = process.platform, arch = process.arch, verify = verifyBinary } = {}) {
  const { pkg, repository } = config(root);
  const release = JSON.parse(fs.readFileSync(path.join(root, 'release.json'), 'utf8'));
  if (release.version !== pkg.version || release.repository !== repository || !/^[a-f0-9]{40}$/.test(release.commit) || release.dirty !== false) throw new Error('发行元数据与 npm 包不一致');
  const artifact = target(pkg.version, platform, arch);
  const expected = checksums(fs.readFileSync(path.join(root, 'SHA256SUMS'), 'utf8')).get(artifact.name);
  if (!expected) throw new Error('npm 包缺少对应平台的校验值，请重新安装完整发行包');
  const directory = path.join(root, '.native');
  fs.mkdirSync(directory, { recursive: true });
  if (fs.lstatSync(directory).isSymbolicLink()) throw new Error('安装目录不能是符号链接');
  const lock = path.join(directory, '.lock');
  try { fs.mkdirSync(lock); } catch (error) {
    if (error.code === 'EEXIST') throw new Error('另一个安装正在进行；确认没有安装进程后可删除 .native/.lock 重试');
    throw error;
  }
  const destination = path.join(directory, artifact.binary);
  const old = destination + '.old';
  let temporary;
  try {
    if (fs.existsSync(old) && !fs.existsSync(destination)) { regular(old); fs.renameSync(old, destination); }
    if (fs.existsSync(destination)) {
      try { verify(destination, pkg.version, release.commit); return destination; } catch { regular(destination); }
    }
    temporary = fs.mkdtempSync(path.join(directory, '.download-'));
    const archive = path.join(temporary, artifact.name);
    const extracted = path.join(temporary, artifact.binary);
    await fetchFile(`https://github.com/${repository}/releases/download/v${pkg.version}/${artifact.name}`, archive);
    if (await hashFile(archive) !== expected) throw new Error('SHA256 校验失败，未替换现有程序');
    await extractBinary(archive, artifact.binary, extracted);
    verify(extracted, pkg.version, release.commit);
    if (fs.existsSync(old)) { regular(old); fs.unlinkSync(old); }
    const hadOld = fs.existsSync(destination);
    if (hadOld) fs.renameSync(destination, old);
    try { fs.renameSync(extracted, destination); } catch (error) {
      if (hadOld) fs.renameSync(old, destination);
      throw error;
    }
    if (hadOld) fs.unlinkSync(old);
    return destination;
  } finally {
    if (temporary) fs.rmSync(temporary, { recursive: true, force: true });
    fs.rmdirSync(lock);
  }
}

module.exports = { ROOT, TARGETS, config, target, checksums, allowedURL, download, hashFile, extractBinary, ensureInstalled, verifyBinary };
