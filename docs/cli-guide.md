# 快代理 CLI 使用指南

[安装指南](./install.md) · [源码仓库](https://github.com/gizaZerozhang/kdl-agent-cli) · [版本记录](https://github.com/gizaZerozhang/kdl-agent-cli/releases)

直接调用 HTTP 接口见 [Agent Gateway API 接入指南](./api-guide.md)。

## 安装与首次接入

按[安装指南](./install.md)安装 CLI、Skill 并登录。

可用平台：macOS arm64/amd64、Linux arm64/amd64、Windows amd64。

## 让 Agent 协助接入

将以下提示词交给 AI 助手：

> 帮我安装快代理 CLI：https://stg3.kuaidaili.com/kdl-agent/docs/install.md

凭证由用户在本地终端输入。安装后可以提问：“查看我的账户状态，列出可用代理订单。”

## 日常命令与任务

命令中的 `ORDER_ID`、`MESSAGE_ID`、`PRODUCT_TYPE`、`CURSOR` 替换为实际值。

### 查询命令

| 命令 | 内容与限制 |
| --- | --- |
| `kdl-agent account summary` | 实名状态、服务限制及订单聚合计数；不含资金明细 |
| `kdl-agent account funds` | 现金余额、可开票金额和币种；不是订单 IP/流量余量 |
| `kdl-agent order list --limit 10` | 本账户代理订单，支持 `--cursor`；不包含待付款订单 |
| `kdl-agent order show ORDER_ID` | 规格、状态、到期时间、额度与用量，按返回单位解释 |
| `kdl-agent order stats ORDER_ID` | 最近 24 小时错误与风控摘要；不是完整请求日志或全量流量报表 |
| `kdl-agent order guide ORDER_ID` | 接入文档与占位配置模板；不含真实密钥 |
| `kdl-agent product list` | 产品目录、适用场景与官网入口 |
| `kdl-agent product trial-eligibility` | 当前账户真实试用资格与试用入口；不自动开通试用 |
| `kdl-agent product spec --product-type PRODUCT_TYPE` | 当前规格字段、默认值、依赖、可售范围 |
| `kdl-agent msg list --limit 10` | 站内信标题、时间及官网入口，不含正文 |
| `kdl-agent msg show MESSAGE_ID` | 站内信元数据及是否已读，不含正文 |

列表命令的 `--limit` 有效范围为 1—100。下一页使用返回的 `data.next_cursor`，保持其原值并加引号：

```bash
kdl-agent order list --limit 10 --cursor 'CURSOR' --format json
```

`next_cursor=null` 表示结束。CLI 不会自动汇总全部分页，也没有订单 `--status` 筛选参数；直接 API 可以按状态筛选。不要用“当前页为空”推断整个账户没有订单。

### 产品报价与待付款订单

报价不会创建订单；`product buy` 会创建真实待付款订单，需要 `product.purchase.create` 授权。

1. 查询产品目录和指定产品的当前规格，按 `configuration_fields`、`configuration_rules` 构造完整配置。
2. 准备 `quote.json`，字段为 `product_type`、`spec_version`、`configuration`。详细请求结构见[API 产品接口](./api-guide.md#产品规格报价与下单)。文件不放 Agent 凭证或订单密钥。
3. 请求报价，检查金额、币种、配置和有效期。不能自行计算一个价格并当作平台报价。
4. 用户已明确要求下单时，将报价返回的 `quote_id`、`product_type`、`spec_version`、`configuration` 原样放入 `purchase.json`，生成本次操作独有的 UUID v4 幂等键。
5. 创建成功后返回订单号、`pending_payment` 状态、金额和 `payment_url`，用户在官网付款。不能报告成已付款或已交付。

```bash
kdl-agent product quote --body-file quote.json --format json
kdl-agent product buy --body-file purchase.json --idempotency-key '本次操作生成的UUID-v4' --format json
```

`--body-file` 是推荐方式，可完整保留动态规格。CLI 还提供 `--product-type`、`--spec-version`、`--pay-mode`、`--duration-months`，下单另有 `--quote-id`；这些简写不能覆盖所有产品配置。尤其不能为了满足简写参数而编造 `spec_version`，应原样使用报价结果。

同一次创建发生超时或响应丢失时，保留相同请求体和幂等键再核对或重试，不重新生成键。报价失效后重新报价；修改配置并重新创建属于新的业务操作，不能悄悄当成旧请求重试。

待付款订单不在 `order list/show` 范围内。刚下单后查询详情返回 404 不能据此判定创建失败，应使用下单响应和官网待付款入口核对。

### 工单预览与创建

预览只整理问题，不创建工单；真实创建需要 `support.ticket.create` 授权。

```bash
kdl-agent ticket preview --subject '代理连接异常' --body '请根据提供的发生时间和错误情况整理问题说明' --format json
```

可加 `--order-id ORDER_ID` 关联订单。预览只接受标题、正文、可选订单号，不接受工单类型、时区或发生时间字段。

创建时优先使用不含凭证、密钥的 `ticket.json`，包含类型、标题、正文、发生时间段和时区；字段限制见[API 工单接口](./api-guide.md#工单预览与创建)。不要编造问题发生时间，不把预览结果直接当作已创建记录。

```bash
kdl-agent ticket create --body-file ticket.json --idempotency-key '本次操作生成的UUID-v4' --format json
```

也可使用 `--type`、`--subject`、`--body`、`--occurred-at-start`、`--occurred-at-end`、`--timezone` 及可选 `--order-id`。命令行参数可能保存在 shell 历史中，涉及客户业务描述时优先使用受控文件；文件权限和保留期限由使用者管理，不放进代码仓库。

成功结果包含 `ticket_id`、`pending_reply` 和 `member_center_url`。同一次创建重试复用请求体和幂等键。初版不提交附件。

### 订单密钥与代理便利命令

下列命令都需要 `order.secret.read`。权限涵盖按订单获取账户内当前及未来订单的密钥，不是仅对某次命令授权。先阅读[密钥边界](#授权凭证与订单密钥)。

| 命令 | 行为 |
| --- | --- |
| `kdl-agent order secret get --order ORDER_ID --format json` | 直接输出 `order_id`、`secret_id`、`secret_key`、`api_domain`；密钥可能进入 Agent/模型 |
| `kdl-agent proxy fetch --order ORDER_ID --num 1 --format json` | 在内存中取得密钥后调用订单的 `getdps` 接口，输出代理列表；不是所有产品的通用提取器 |
| `kdl-agent order whitelist set --order ORDER_ID --ip 192.0.2.10 --format json` | 将白名单设置为指定 IP 列表，可能替换已有内容；示例 IP 需换成业务所需出口 IP |
| `kdl-agent order whitelist clear --order ORDER_ID --format json` | 清空订单白名单，影响现有接入 |

设置多项 IP 可重复使用 `--ip`。白名单设置不是增量添加；清空和替换必须符合用户明确要求，不作为排障默认动作。获取代理可能消耗订单额度，不能承诺是零副作用的验证。

变更前用 `kdl-agent order whitelist get --order ORDER_ID --format json` 读取当前列表，返回 `ipwhitelist` 与 `count`；变更后再次读取核对。清空不是恢复原配置，原值未知时不要继续写验收。

`order secret get` 默认表格输出遮蔽 `secret_key`；需要完整订单密钥时显式使用 JSON。普通查询、代理提取和白名单结果不额外回显密钥。订单密钥不写入 `.kdl` 或 CLI 缓存，但显式 JSON 输出可被终端、Agent 或外层工具记录，不能把“不落盘”理解为外层系统绝不会保存。

便利命令使用订单 OpenAPI 的规则，不享有 Gateway 创建订单/工单的幂等保证。下游请求结果不确定时先核对业务状态，避免重复提取或误改白名单。Agent 凭证仅发给 Gateway，不发给订单 OpenAPI。

## 从提取到连接代理

订单 SecretId/SecretKey 用于订单 API；连接代理使用订单配置的 Basic 认证或 IP 白名单。两者不能混用。

支持 `proxy auth` 的 CLI 版本可在内存中获取 Basic 认证（需要 `order.secret.read`）：

```bash
kdl-agent proxy auth --help
kdl-agent proxy auth --order ORDER_ID --format json
```

返回 `type` 和 `credentials`，后者为敏感的 Base64 编码值。默认表格模式隐藏该值。

客户端应把 `Basic <credentials>` 配置为代理认证。curl 使用 `--proxy-header`，不要使用发送给目标站的 `-H`；HTTPS 认证用于 CONNECT。以下本地 Python 示例将认证通过 stdin 交给 curl，不写入文件或命令参数：

```python
import json
import os
import subprocess

order_id = input("订单号：").strip()
proxy = input("已提取的代理 IP:端口：").strip()
if not proxy or any(c not in "0123456789.:[]" for c in proxy):
    raise SystemExit("代理地址格式无效")
auth = json.loads(subprocess.check_output([
    "kdl-agent", "proxy", "auth", "--order", order_id, "--format", "json"
]))
config = 'proxy = "http://' + proxy + '"\n'
config += 'proxy-header = ' + json.dumps("Proxy-Authorization: " + auth["type"] + " " + auth["credentials"]) + '\n'
config += 'url = "https://www.kuaidaili.com/"\n'
subprocess.run(["curl", "--config", "-", "--noproxy", "", "--silent", "--show-error",
                "--fail", "--max-time", "30", "--output", os.devnull,
                "--write-out", "%{http_code}\\n"], input=config, text=True, check=True)
```

示例需要支持 `proxy auth` 的 CLI 和本地 curl；目标地址按体验任务调整。不要开启 verbose/trace 或记录认证内容。HTTP 407 先检查代理认证和白名单，重复提取不能补齐认证。

`order stats` 是已入库的错误/风控摘要；空结果不能证明没有代理鉴权失败。代理侧 407 的上报、订单归属与延迟需结合实际统计记录核对。

## 输出与错误判断

### 格式

默认 `--format table` 供人阅读；Agent 使用 `--format json`，不要解析表格对齐或颜色。可用 `--color never` 关闭颜色，`--help` 查看当前版本支持的选项。

普通 Gateway 业务命令成功时输出响应信封。以下是空订单列表的示意，不是实际账户结果：

```json
{
  "success": true,
  "data": {"items": [], "next_cursor": null},
  "request_id": "req_example",
  "error": null
}
```

以下命令是直接结果对象，不套同一信封：

| 命令 | JSON 结构 |
| --- | --- |
| `order secret get` | `order_id`、`secret_id`、`secret_key`、`api_domain` |
| `proxy fetch` | `proxy_count`、`proxies` |
| `proxy auth`（需支持该命令的版本） | `type`、`credentials`（敏感的 Basic 编码值） |
| `order whitelist set/clear` | `whitelist_ip_count` |
| `order whitelist get`（需支持该命令的版本） | `ipwhitelist`、`count` |

### stdout、stderr 与退出码

结果进入 stdout，错误和风险提示进入 stderr。即使选择 JSON，错误诊断当前仍是文本；不能假定 stderr 总能解析成 JSON，也不能看到 Secret 命令的风险提示就认定请求失败。

| 退出码 | 含义 | 处理 |
| --- | --- | --- |
| `0` | 命令成功 | 再按该命令的数据结构处理结果 |
| `1` | 请求、服务或其他执行失败 | 查看错误码、`request_id` 和诊断；某些参数解析错误也可能返回此码 |
| `2` | 配置或命令内显式参数校验失败 | 检查登录状态、参数及文件格式 |

自动化先检查非零退出码，再处理 stdout。错误中有 `request_id` 时保留它用于排查；网络请求未到达服务端时可能没有此字段。

新版 CLI 遇到 429 时会在 stderr 提示服务端 `Retry-After` 等待秒数；按提示退避，不立即循环重试。CLI 不自动重放业务请求。

## 授权凭证与订单密钥

### 三类凭据

| 对象 | 用途 | 保存与披露 |
| --- | --- | --- |
| Agent 凭证 | 访问快代理 Gateway，关联账户与三项授权 | 由客户侧工具保存；不进入 Agent 对话、命令参数、诊断、日志或版本库 |
| 订单密钥 | 访问某个订单的 OpenAPI | 开启授权后可按订单获取；可以交给客户选择的 Agent/模型使用，需理解披露影响 |
| 代理 Basic 凭据 | 连接代理服务 | 通过订单 API 获取，仅交给代理客户端；Base64 编码不等于脱敏 |

Agent 凭证由会员中心管理。创建时核对有效期；默认有效期为 90 天，每账户最多 10 张 active 凭证，不支持永久凭证。凭证丢失且无法恢复时重新创建合适的凭证，并按需撤销旧凭证。

### 三项敏感授权

| 授权名称 | grant | 开启后允许 |
| --- | --- | --- |
| 创建待付款订单 | `product.purchase.create` | 根据有效报价创建新购待付款订单；不自动付款 |
| 创建工单 | `support.ticket.create` | 创建真实工单；不包含附件 |
| 获取订单密钥 | `order.secret.read` | 按订单读取账户当前及未来订单密钥，并用于该订单 OpenAPI |

三项默认关闭，在[Agent 设置](https://stg3.kuaidaili.com/uc/agent/settings/)按需开启。授权有效且用户已明确要求执行的业务动作，不再由 CLI 每次增加确认步骤；任务目标或关键参数不明确时仍需先澄清。基础查询仍要求有效凭证并受账户、订单归属和服务规则约束。

### 本地保存

发行版使用操作系统用户家目录：

```text
~/.kdl/
  config.toml        # Gateway 等普通配置
  credentials.toml   # 按 Gateway 绑定的明文 Agent 凭证
```

Windows 对应 `%USERPROFILE%\.kdl\`。macOS/Linux 目录权限为 `0700`、凭证文件为 `0600`；Windows 使用对应用户访问权限。默认不从当前项目目录发现或写入凭证，切换工作目录不切换账户。

凭证明文保存，同一系统用户运行的程序可读取。不要共享 `.kdl` 或让 Agent 读取凭证文件。

自动化场景可以通过 `KDL_AGENT_GATEWAY_URL`、`KDL_AGENT_TOKEN` 注入地址和凭证，当前运行的注入覆盖本地配置且不自动落盘。由运行环境管理秘密值，不通过聊天生成含真实凭证的脚本。切换 Gateway 时应使用该地址对应的凭证，不自动复用旧地址凭证。

普通客户无需设置自定义配置文件。需要时，普通配置路径优先级为 `--config` > `KDL_AGENT_CONFIG` > 家目录 `.kdl/config.toml`；显式相对路径相对当前工作目录，凭证仍固定在家目录。`auth status --format json` 区分 `unconfigured`、`valid`、`invalid`、`unreachable` 与 `error`，只有 `valid` 表示远端验证有效。

### 退出、撤销与密钥重置

| 操作 | 生效范围 | 不会发生什么 |
| --- | --- | --- |
| `kdl-agent auth logout` | 清除本地保存的 Agent 凭证 | 不撤销服务端凭证，不清理外部环境变量注入 |
| 关闭某项 grant | 阻止该凭证后续执行对应能力 | 不撤销已创建订单/工单，不使已取得的订单密钥失效 |
| 在会员中心撤销 Agent 凭证 | 阻止该凭证后续访问 Gateway | 不使已经取得的订单密钥失效 |
| 在订单 API 设置中重置订单密钥 | 按官网重置设置使旧密钥失效 | 不局限于某个 Agent，会影响所有使用旧密钥的客户端 |

订单密钥读取授权意味着密钥可能进入你选择的 Agent、模型或相关服务。怀疑泄露时，仅退出 CLI 或关闭 grant 不够；还需在订单 API 设置中完成重置，并同步更新其他客户端，核对旧密钥实际失效时间。

可在[安全记录](https://stg3.kuaidaili.com/uc/agent/security/)查看凭证及授权变更，在[调用监控](https://stg3.kuaidaili.com/uc/agent/monitor/)查看 Gateway 调用。便利命令直接调用的订单 OpenAPI 不应被误认为全部记录在 Gateway 监控中。

## 版本与升级回退

查看当前版本：

```bash
kdl-agent --version
```

升级或回退时，将下方版本号替换为[版本记录](https://github.com/gizaZerozhang/kdl-agent-cli/releases)中的目标版本，同步安装 CLI 和 Skill（示例目标为 Codex）：

```bash
npx --yes @zerozhang-giza/kdl-agent@0.1.0-beta.3 install --yes --agent codex --no-login
```

升级保留登录配置；Skill 安装失败时重试同一命令。完成后核对版本、刷新 Agent 会话，再执行一次账户查询。发布新版不会静默替换本机安装。

后续支持 `update check` 的版本会由 Skill 在任务开始时检查新版，提示并经你确认后同步升级。旧版需要先按上述流程升级一次。

卸载：

```bash
kdl-agent auth logout
npm uninstall -g @zerozhang-giza/kdl-agent
```

卸载 npm 包不删除 Skill；按 AI 助手的管理方式移除。

## 常见问题与排障

| 现象 | 优先检查 | 下一步 |
| --- | --- | --- |
| 命令不存在 | 是否解压、执行文件路径是否正确、用户 PATH 是否生效 | 使用完整路径运行 `--version`，重开终端 |
| 无法执行或架构不符 | 系统、芯片、最低版本及文件校验 | 下载对应官方包；不绕过系统全局安全设置 |
| SHA256 不一致 | 包与清单是否同版本、文件是否下载完整 | 停止安装并重新下载，仍失败时反馈版本和文件名 |
| `AUTH_REQUIRED`、`CREDENTIAL_REVOKED` | 当前 Gateway、凭证有效期、是否已撤销 | 在会员中心处理并重新隐藏输入登录 |
| `FORBIDDEN` | 所需 grant 是否开启，目标账户是否有对应业务权限 | 仅按实际任务开启权限，不重复尝试绕过 |
| 订单列表为空 | 是否登录正确账户；当前是否只有待付款订单 | 查看官网代理订单和待付款页面，不能据此判定服务故障 |
| 订单详情 404 | 订单是否属于本账户，是否尚待付款或不存在 | 核对原始结果及官网链接，不尝试其他账户订单号 |
| `VALIDATION_ERROR` | 参数、类型、动态规格、工单时间和长度 | 修正参数后重试，不能编造字段通过校验 |
| `QUOTE_EXPIRED` | 报价有效期 | 重新报价并核对金额，再作为新业务操作处理 |
| `IDEMPOTENCY_CONFLICT` | 同一键是否提交了不同参数 | 核对原操作结果；新需求才生成新键 |
| `RATE_LIMITED` / 429 | 并发、轮询与请求速度 | 按 API `Retry-After` 等待；CLI 未显示该值时降低频率并有限退避 |
| 网络错误、502、503 | 网络、目标环境、服务可用性 | 有限退避；创建请求保留相同键和参数；勿将网络失败判为凭证失效 |
| `proxy fetch` 失败 | 产品是否支持 `getdps`、订单额度、密钥授权及下游返回 | 使用订单接入指引排查，不对所有产品盲目执行相同提取命令 |
| 退出后自动化仍可调用 | 环境或任务系统是否仍注入凭证 | 在注入来源移除；需全局失效时在会员中心撤销 |
| 关闭授权后旧订单密钥仍可用 | 授权控制后续获取，不控制已取得的密钥 | 按需在订单 API 设置中重置，并更新所有相关客户端 |

反馈问题时提供：发生时间与时区、操作系统和架构、CLI 版本、去除敏感参数的命令、错误码及 `request_id`。不要附带 `.kdl`、旧配置文件、完整环境变量、订单 Secret 输出、未经脱敏的业务正文或终端录像。
