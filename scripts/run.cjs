#!/usr/bin/env node
'use strict';
const { spawn } = require('node:child_process');
async function main() {
  const args = process.argv.slice(2);
  if (args[0] === 'install') return require('./wizard.cjs').main(args.slice(1));
  const binary = await require('./runtime/install.cjs').ensureInstalled();
  const child = spawn(binary, args, { stdio: 'inherit' });
  for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => child.kill(signal));
  child.on('error', error => { console.error(`运行失败：${error.message}`); process.exitCode = 1; });
  child.on('exit', (code, signal) => { process.exitCode = code ?? (signal === 'SIGINT' ? 130 : 1); });
}
main().catch(error => { console.error(`运行失败：${error.message}`); process.exitCode = 1; });
