# 快代理 CLI 安装指南

适用版本：`0.1.0-beta.5`。显式指定版本或 `@beta`；`latest` 不作为当前推荐入口，发行结果以同版本 Release 为准。

## 环境

- Node.js 22.14+，使用用户可写的 npm 全局目录；无需 Go/Python。
- 原生构建目标为 macOS arm64/amd64、Linux arm64/amd64、Windows amd64。最低系统与真实业务验收仍待补证；macOS beta 包未签名/公证。
- 网络需要访问 npm registry、GitHub Release 附件及 Skill 源码。受控网络可使用 `HTTPS_PROXY`；下载仍校验包内 SHA256，不关闭 TLS。

## 推荐安装

安装器对暂时性下载故障最多尝试 3 次；`KDL_AGENT_DOWNLOAD_TIMEOUT_MS` 可设置单次连接空闲超时（默认 60000，范围 1000–600000 毫秒）。失败后可重新运行安装，旧程序保留；404、证书与 SHA256 错误不自动重试。

使用 npm 安装向导。人类交互模式依次完成 CLI、同版本 Skill、隐藏输入登录、远端状态与首次只读查询；Skill 安装时选择实际使用的 Agent。

```bash
npx @kuaidaili/kdl-agent@0.1.0-beta.5 install
```

Agent 协助安装时指定目标工具，并把凭证输入留给用户本地终端：

```bash
npx @kuaidaili/kdl-agent@0.1.0-beta.5 install --yes --agent codex --no-login
```

`--agent` 使用 skills 工具支持的标识，例如 `codex`、`cursor`、`claude-code`。未经实测的工具不承诺兼容。只安装 CLI 可显式 `--no-skills`。仅安装成功不代表业务接入完成。

## 登录与验证

用户在[会员中心](https://www.kuaidaili.com/uc/agent/settings/)创建并授权 Agent 凭证，然后在本地终端执行：

```bash
kdl-agent auth login
kdl-agent auth status
kdl-agent account summary --format json
```

凭证输入隐藏；不要把密码、Agent 凭证或订单密钥发给安装助手。默认网关为 `https://agent-gateway.kdlapi.com`。网络或服务不可用时安装结果保留，登录失败不能报告接入成功。

## 独立入口

只安装 npm CLI：`npm install -g @kuaidaili/kdl-agent@0.1.0-beta.5`。

Skill 可以单独安装；下面示例仅选 Codex，可换为实际工具：

```bash
npx --yes skills@1.5.25 add https://github.com/kuaidaili/kdl-agent-cli/tree/v0.1.0-beta.5/skills/kdl-agent --global --skill kdl-agent --agent codex --yes
```

无 Node.js 的环境可下载 [GitHub Release](https://github.com/kuaidaili/kdl-agent-cli/releases/tag/v0.1.0-beta.5) 的对应原生包，核对 SHA256SUMS 后解压并放入用户 PATH；执行文件为 `kdl-agent` 或 `kdl-agent.exe`。

## 升级、回退与卸载

- 使用目标精确版本重新运行向导，CLI 和 Skill 固定同版本；beta 使用显式版本或 npm beta 标签，稳定版发布后才提供 latest 入口。
- 向导先下载校验再修改全局包，失败尝试恢复原 npm 版本。直接运行 npm install 的全局包替换由 npm 管理，失败后须按错误提示重新安装原版本；不要假定 npm 能原子回滚整个全局包。
- CLI、Skill、登录分别报告状态，重新运行仅重试必要步骤。并发安装会拒绝；确认没有安装进程后才清理提示的锁目录。
- 卸载：`npm uninstall -g @kuaidaili/kdl-agent`。`.kdl` 与 Skill 保留；本地退出使用 `auth logout`，服务端撤销在会员中心执行。
- npm 生命周期脚本被禁用时，第一次运行启动器仍会校验并补装二进制；缺少校验文件表示包不完整，应重新安装。

## 从个人包迁移

当前 npm 包属于公司用户 `kuaidaili`，GitHub 属于公司组织 `kuaidaili`。本轮没有创建 npm Organization。个人包 `@zerozhang-giza/kdl-agent` 的旧版更新检查仍查询旧 scope，需要显式迁移。

先记录 `kdl-agent --version` 和 `npm ls -g @zerozhang-giza/kdl-agent --depth=0`，停止正在执行的 CLI 任务。运行公司安装向导可先验证下载；若检测到旧全局包，向导停止并显示恢复命令，按提示执行：

```bash
npm uninstall -g @zerozhang-giza/kdl-agent
npx --yes @kuaidaili/kdl-agent@0.1.0-beta.5 install --yes --agent codex --no-login
kdl-agent --version
kdl-agent auth status
```

按实际工具调整 `--agent`，刷新 Skill 后自行完成一次只读查询；未登录时由用户在本机 `auth login`。卸载不删除 `.kdl` 或 Skill，不需要因账号归属变化重建业务授权。不要使用 `--force` 覆盖共用命令。

迁移失败且需要回退时，卸载公司包并按之前记录的精确旧版本重新运行旧包安装向导，恢复 CLI 与对应 Skill。Skill 单独失败时可重试同一公司版本向导，不必回退 CLI。历史 npm 包、tag 和 Release 保留。
