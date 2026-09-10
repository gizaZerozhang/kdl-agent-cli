#!/usr/bin/env node
'use strict';
// npx 安装向导先执行预检；真正调用业务命令时仍会校验并补装原生程序。
if (process.env.npm_command !== 'exec') {
  require('./runtime/install.cjs').ensureInstalled().catch(error => {
    console.error(`安装失败：${error.message}`);
    process.exitCode = 1;
  });
}
