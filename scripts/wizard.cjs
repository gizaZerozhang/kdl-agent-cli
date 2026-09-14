'use strict';
const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const readline = require('node:readline/promises');
const { config, ensureInstalled, verifyBinary } = require('./runtime/install.cjs');

function command(name, args, options = {}) {
  // npm/npx 在 Windows 是 .cmd；只传入配置中校验过的标识和固定参数。
  if (process.platform === 'win32' && ['npm', 'npx'].includes(name)) {
    if (args.some(arg => /[&|<>^%!"\r\n]/.test(arg))) throw new Error('命令参数包含 Windows shell 特殊字符');
    return spawnSync(process.env.ComSpec || 'cmd.exe', ['/d', '/s', '/c', `${name}.cmd ${args.map(arg => `"${arg}"`).join(' ')}`], { encoding: 'utf8', ...options });
  }
  return spawnSync(name, args, { encoding: 'utf8', ...options });
}

function parse(args) {
  const options = { yes: false, noLogin: false, noSkills: false, agents: [] };
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--yes' || arg === '-y') options.yes = true;
    else if (arg === '--no-login') options.noLogin = true;
    else if (arg === '--no-skills') options.noSkills = true;
    else if (arg === '--agent') {
      const name = args[++i];
      if (!name || !/^[a-z0-9-]+$/.test(name)) throw new Error('--agent 需要目标工具标识');
      options.agents.push(name);
    } else if (arg === '--help' || arg === '-h') options.help = true;
    else throw new Error(`未知安装参数：${arg}`);
  }
  return options;
}

async function confirmUpdate() {
  const prompt = readline.createInterface({ input: process.stdin, output: process.stdout });
  try { return /^y(es)?$/i.test((await prompt.question('确认升级 CLI 与同版本 Skill？[y/N] ')).trim()); }
  finally { prompt.close(); }
}

async function main(args, { run = command, install = ensureInstalled, readConfig = config, checkBinary = verifyBinary, tty = process.stdin.isTTY, mode = 'install', confirm = confirmUpdate } = {}) {
  const options = parse(args);
  if (mode === 'update') {
    options.noLogin = true;
    if (options.help) {
      console.log(`同步升级 CLI 与同版本 Skill\n检查新版：kdl-agent update check --format json\n用法：npx --yes ${readConfig().pkg.name}@<目标版本> update --agent codex [--yes]\n默认询问确认；非交互必须显式 --yes。保留登录配置，Skill 失败时重复同一命令修复。`);
      return;
    }
    if (options.noSkills) throw new Error('同步升级不能跳过 Skill；请移除 --no-skills');
  }
  if (options.help) {
    console.log('安装快代理 CLI 与同版本 Skill\n\n用法：npx <包名>@<版本> install [--yes] [--agent codex] [--no-login] [--no-skills]\n\n--yes 非交互安装；安装 Skill 时须指定 --agent\n--no-login 仅安装，稍后在本地终端执行 kdl-agent auth login\n--no-skills 跳过 Skill，适用于直接使用 CLI\n升级或回退：使用目标版本再次运行；卸载 npm 包保留 .kdl 与 Skill。');
    return;
  }
  if (!tty && !options.yes) throw new Error('非交互环境需 --yes；Agent 请同时使用 --no-login，敏感凭证由用户在本地终端输入');
  if (options.yes && !options.noSkills && !options.agents.length) throw new Error('非交互 Skill 安装须通过 --agent 指定目标工具，或显式 --no-skills');
  const { pkg, repository } = readConfig();
  if (!/^@[a-z0-9][a-z0-9._-]*\/[a-z0-9][a-z0-9._-]*$/.test(pkg.name)) throw new Error('npm 包名尚未绑定');
  console.log(`目标版本：${pkg.name}@${pkg.version}`);
  if (mode === 'update' && !options.yes && !await confirm()) { console.log('已取消升级，保留现有 CLI 与 Skill。'); return; }
  console.log('正在下载并校验原生程序...');
  await install();
  const prefixResult = run('npm', ['prefix', '-g'], { timeout: 15000 });
  if (prefixResult.status !== 0) throw new Error('无法读取 npm 全局目录，请检查 Node.js/npm');
  const prefix = prefixResult.stdout.trim();
  const modules = process.platform === 'win32' ? path.join(prefix, 'node_modules') : path.join(prefix, 'lib/node_modules');
  for (const legacy of pkg.kdl.legacyPackages || []) {
    if (!/^@[a-z0-9][a-z0-9._-]*\/[a-z0-9][a-z0-9._-]*$/.test(legacy) || legacy === pkg.name) throw new Error('旧 npm 包迁移配置无效');
    const legacyRoot = path.join(modules, ...legacy.split('/'));
    let entry;
    try { entry = fs.lstatSync(legacyRoot); } catch (error) { if (error.code !== 'ENOENT') throw error; }
    if (!entry) continue;
    let restore = '请先记录旧包的精确版本，以便必要时恢复。';
    try {
      const old = JSON.parse(fs.readFileSync(path.join(legacyRoot, 'package.json'), 'utf8'));
      if (old.name === legacy && /^\d+\.\d+\.\d+(?:-beta\.\d+)?$/.test(old.version)) restore = `恢复旧版：npm install -g ${legacy}@${old.version}`;
    } catch {}
    throw new Error(`发现旧包 ${legacy}，与公司包共用 kdl-agent 命令；未修改全局安装。\n先执行 npm uninstall -g ${legacy}，再重新运行本次公司包向导。\n.kdl 配置与 Skill 保留。${restore}`);
  }
  const installedRoot = path.join(modules, ...pkg.name.split('/'));
  let previous;
  try { previous = JSON.parse(fs.readFileSync(path.join(installedRoot, 'package.json'), 'utf8')).version; } catch {}
  const binary = path.join(installedRoot, '.native', process.platform === 'win32' ? 'kdl-agent.exe' : 'kdl-agent');
  let current = false;
  if (previous === pkg.version) { try { checkBinary(binary, pkg.version); current = true; } catch {} }
  if (!current) {
    const result = run('npm', ['install', '-g', `${pkg.name}@${pkg.version}`, '--registry=https://registry.npmjs.org/'], { stdio: 'inherit', timeout: 180000 });
    if (result.status !== 0) {
      if (previous && /^\d+\.\d+\.\d+(?:-beta\.\d+)?$/.test(previous)) {
        console.error(`正在尝试恢复原版本 ${previous}...`);
        const restore = run('npm', ['install', '-g', `${pkg.name}@${previous}`, '--registry=https://registry.npmjs.org/'], { stdio: 'inherit', timeout: 180000 });
        console.error(restore.status === 0 ? '已恢复原 npm 版本' : `恢复未完成，请执行 npm install -g ${pkg.name}@${previous}`);
      }
      throw new Error('全局安装失败；请检查网络和 npm 全局目录权限，不要使用 sudo 自动提权');
    }
  }
  await install({ root: installedRoot });
  console.log(`CLI：已安装 ${pkg.version}\n程序：${binary}\n命令目录：${process.platform === 'win32' ? prefix : path.join(prefix, 'bin')}（须位于 PATH）`);
  if (!options.noSkills) {
    const source = `https://github.com/${repository}/tree/v${pkg.version}/skills/${pkg.kdl.skill}`;
    const skillArgs = ['--yes', pkg.kdl.skillsPackage, 'add', source, '--global', '--skill', pkg.kdl.skill];
    if (options.yes) skillArgs.push('--yes');
    for (const agent of options.agents) skillArgs.push('--agent', agent);
    const result = run('npx', skillArgs, { stdio: 'inherit', timeout: 180000 });
    if (result.status !== 0) throw new Error('CLI 已安装，Skill 安装未完成；重新运行此向导重试，已安装 CLI 将跳过');
    console.log('Skill：已安装；目标 Agent 可能需要刷新会话');
  } else console.log('Skill：按要求跳过');
  if (options.noLogin) { console.log(mode === 'update' ? '升级完成，登录配置保持不变；请核对 kdl-agent --version，并刷新 Agent 会话。' : '登录：按要求跳过；用户稍后在本地终端执行 kdl-agent auth login 和 kdl-agent auth status'); return; }
  let status = run(binary, ['auth', 'status', '--format', 'json'], { timeout: 25000 });
  if (status.status !== 0) {
    if (!tty) throw new Error('CLI/Skill 已安装，登录待用户在本地终端完成：kdl-agent auth login');
    console.log('凭证管理：https://www.kuaidaili.com/uc/agent/settings/\n请仅在本地终端输入凭证，不发送给 Agent。');
    const prompt = readline.createInterface({ input: process.stdin, output: process.stdout });
    let answer;
    try { answer = await prompt.question('现在隐藏输入并验证登录？[y/N] '); } finally { prompt.close(); }
    if (!/^y(es)?$/i.test(answer.trim())) throw new Error('安装已完成，登录待完成；稍后执行 kdl-agent auth login');
    const login = run(binary, ['auth', 'login'], { stdio: 'inherit' });
    if (login.status !== 0) throw new Error('登录未完成；安装结果保留，请检查凭证或 Gateway 后重试');
    status = run(binary, ['auth', 'status', '--format', 'json'], { timeout: 25000 });
  }
  if (status.status !== 0) throw new Error('远端状态未通过验证；请执行 kdl-agent auth status 排查');
  const query = run(binary, ['account', 'summary', '--format', 'json'], { timeout: 25000 });
  if (query.status !== 0) throw new Error('状态有效，首次查询失败；请执行 kdl-agent account summary 排查');
  console.log('登录：远端验证有效\n首次只读查询：通过（账户数据未输出到安装日志）');
}
module.exports = { main, parse, command };
