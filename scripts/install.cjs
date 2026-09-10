#!/usr/bin/env node
'use strict';
// npx 安装向导先执行预检；真正调用业务命令时仍会校验并补装原生程序。
if (process.env.npm_command !== 'exec') {
  require('./runtime/install.cjs').ensureInstalled().catch(error => {
    console.error(`安装失败：${error.message}`);
    console.error('检查 HTTPS_PROXY 或设置 KDL_AGENT_DOWNLOAD_TIMEOUT_MS 后重新运行安装；GitHub 不可达时可从同版本 Release 手动安装并核对 SHA256。');
    process.exitCode = 1;
  });
}
