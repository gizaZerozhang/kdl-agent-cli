# 快代理 CLI 安装指南

适用版本：`0.1.0-beta.1`，已发布的预发行版本，匿名安装与 Codex Skill 实装通过。当前显式指定版本或 `@beta`；npm 自动附加的 `latest` 同样指向该 beta，不代表稳定版已就绪。

## 环境

- Node.js 22.14+，使用用户可写的 npm 全局目录；无需 Go/Python。
- 原生构建目标为 macOS arm64/amd64、Linux arm64/amd64、Windows amd64。最低系统与真实业务验收仍待补证；macOS beta 包未签名/公证。
- 网络需要访问 npm registry、GitHub Release 附件及 Skill 源码。受控网络可使用 `HTTPS_PROXY`；下载仍校验包内 SHA256，不关闭 TLS。

## 推荐安装

使用 npm 安装向导。人类交互模式依次完成 CLI、同版本 Skill、隐藏输入登录、远端状态与首次只读查询；Skill 安装时选择实际使用的 Agent。

```bash
npx @zerozhang-giza/kdl-agent@0.1.0-beta.1 install
```

Agent 协助安装时指定目标工具，并把凭证输入留给用户本地终端：

```bash
npx @zerozhang-giza/kdl-agent@0.1.0-beta.1 install --yes --agent codex --no-login
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

只安装 npm CLI：`npm install -g @zerozhang-giza/kdl-agent@0.1.0-beta.1`。

Skill 可以单独安装；下面示例仅选 Codex，可换为实际工具：

```bash
npx --yes skills@1.5.25 add https://github.com/gizaZerozhang/kdl-agent-cli/tree/v0.1.0-beta.1/skills/kdl-agent --global --skill kdl-agent --agent codex --yes
```

无 Node.js 的环境可下载 [GitHub Release](https://github.com/gizaZerozhang/kdl-agent-cli/releases/tag/v0.1.0-beta.1) 的对应原生包，核对 SHA256SUMS 后解压并放入用户 PATH；执行文件为 `kdl-agent` 或 `kdl-agent.exe`。

## 升级、回退与卸载

- 使用目标精确版本重新运行向导，CLI 和 Skill 固定同版本；beta 使用显式版本或 npm beta 标签，稳定版发布后才提供 latest 入口。
- 向导先下载校验再修改全局包，失败尝试恢复原 npm 版本。直接运行 npm install 的全局包替换由 npm 管理，失败后须按错误提示重新安装原版本；不要假定 npm 能原子回滚整个全局包。
- CLI、Skill、登录分别报告状态，重新运行仅重试必要步骤。并发安装会拒绝；确认没有安装进程后才清理提示的锁目录。
- 卸载：`npm uninstall -g @zerozhang-giza/kdl-agent`。`.kdl` 与 Skill 保留；本地退出使用 `auth logout`，服务端撤销在会员中心执行。
- npm 生命周期脚本被禁用时，第一次运行启动器仍会校验并补装二进制；缺少校验文件表示包不完整，应重新安装。

## 账号迁移

当前使用个人账号发行。后续团队 scope 会使用新的 npm 包名；旧包提供迁移提示，用户显式卸载旧包并安装团队包，避免两个包争用 `kdl-agent` 命令。配置与凭证目录不变，迁移后核对版本及 auth status。
