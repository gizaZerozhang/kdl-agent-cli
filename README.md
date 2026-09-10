# 快代理 CLI

通过快代理 Agent Gateway 查询账户、订单、产品、消息与工单，并执行已授权的购买、工单创建、订单密钥读取及代理便利操作。

源码与问题反馈：https://github.com/gizaZerozhang/kdl-agent-cli

源码许可证见 [LICENSE](./LICENSE)。

## 从源码构建

维护者需要 Go 1.23.6+、Git。构建原生二进制后，使用者无需安装 Go、Python 或 Node.js。

```bash
go test ./...
go build -trimpath -o bin/kdl-agent .
./bin/kdl-agent --help
```

Windows 构建输出使用 `bin/kdl-agent.exe`。默认版本为开发版；带版本及来源信息的构建见下文。

## 登录与首次查询

```bash
kdl-agent auth login
kdl-agent auth status
kdl-agent account summary
kdl-agent order list --format json
```

登录在本地终端隐藏输入 Agent 凭证，向 `https://agent-gateway.kdlapi.com` 验证成功后保存；需要先在[快代理会员中心](https://www.kuaidaili.com/uc/agent/settings/)创建并授权凭证。服务可用性和账户权限以实际环境为准。

普通配置位于用户家目录 `.kdl/config.toml`，明文凭证位于 `.kdl/credentials.toml`，按规范化网关地址关联。macOS/Linux 目录 `0700`、凭证文件 `0600`；Windows 设置当前用户访问权限。同用户运行的程序仍可读取文件，不提供磁盘加密或同用户隔离。订单 Secret 不持久化。

`auth status --format json` 的状态为 `unconfigured`、`valid`、`invalid`、`unreachable` 或 `error`；只有 `valid` 代表远端验证有效。`auth logout` 只清除当前网关的本地凭证，服务端撤销仍在会员中心执行。

## 配置与自动化

| 配置 | 语义 |
| --- | --- |
| `--config`、`KDL_AGENT_CONFIG` | 普通配置路径，优先级为参数 > 环境变量 > 家目录默认；显式相对路径相对 CWD，不改变凭证文件位置 |
| `KDL_AGENT_GATEWAY_URL` | 当前运行的网关基址，覆盖文件；使用 HTTPS，仅本机回环测试允许 HTTP |
| `KDL_AGENT_TOKEN` | 当前运行的 Agent 凭证，覆盖本地对应网关的凭证，不自动保存 |
| `--print-paths` | 显示配置与凭证位置、来源后退出，不发送业务请求 |
| `--format json` | 供自动化处理的结果；错误和诊断输出到 stderr |
| `--color auto\|always\|never` | 终端颜色，`NO_COLOR` 优先 |

自动化应通过受控运行环境注入凭证，并核对目标地址；需要明确保存时可通过 `auth login --token-stdin` 从受控标准输入读入。不把凭证放在命令参数、日志、工单或模型对话中。切换网关不会自动发送其他网关的本地凭证。

旧版本的 CWD `kdl-agent.toml` 不自动发现；显式读取含 `token` 或 `[device]` 的旧格式会拒绝加载。移除旧 `--config`/`KDL_AGENT_CONFIG` 后重新登录并验证，再由用户清理旧文件和撤销旧凭证。

## 命令

| 命令 | 用途 |
| --- | --- |
| `account funds/summary` | 账户查询 |
| `order list/show/stats/guide` | 订单、使用量及接入指南 |
| `product list/spec/quote/trial-eligibility` | 产品、规格、报价及试用资格 |
| `msg list/show` | 站内信 |
| `ticket preview/create` | 工单预览与创建 |
| `product buy` | 创建待付款订单，不自动付款 |
| `order secret get` | 显式读取并输出已授权订单密钥 |
| `proxy fetch`、`order whitelist` | 代理提取和白名单便利命令 |

参数以各子命令 `--help` 为准。敏感授权仅 `product.purchase.create`、`support.ticket.create`、`order.secret.read` 三项，默认关闭。购买与工单的同一操作重试必须复用幂等键。撤销授权仅阻止后续读取，已获取的订单密钥须由用户在订单设置中主动轮换。

## 构建发行包

维护者安装 Python 3.10+ 后执行：

```bash
python3 scripts/build.py --version 0.1.0-rc.1 --output dist/candidate
```

输出 macOS arm64/amd64、Linux arm64/amd64、Windows amd64 的归档包及 `SHA256SUMS`，包含 `BUILD.json` 与第三方许可。输出目录必须不存在；版本与 commit 注入可执行文件。可用 `--local` 只构建当前目标进行开发验证。

编译目标不等于已支持的最低系统版本。最低系统目标为 macOS 12、Ubuntu 22.04、Windows 11，逐个平台实测后才能发布支持声明。当前公开源码及候选构建不表示生产业务或正式二进制发行已验收。

## 维护与反馈

提交前运行 `go test ./...` 和 `go vet ./...`。问题报告包含版本、系统、脱敏错误码与复现步骤；请勿包含 Agent 凭证、订单 Secret、客户数据或完整配置文件。第三方许可见 `THIRD_PARTY_NOTICES.txt`。
