# 快代理 CLI 使用指南

适用版本：`0.1.0-beta.1`。安装、升级和回退见[安装指南](./install.md)，请求字段与错误码见[API 指南](./api-guide.md)。本版本用于预发行验证，安装成功不代表服务端和最低系统已通过业务验收。

## 常用命令

```bash
kdl-agent --version
kdl-agent auth login
kdl-agent auth status --format json
kdl-agent account summary --format json
kdl-agent order list --format json
kdl-agent product list --format json
```

各子命令通过 `--help` 查询实际参数。普通查询信封使用 `success/data/request_id`；订单 Secret 和代理便利命令输出自身结果对象。错误写入 stderr，判断成功应检查退出码，不能把所有响应假设成同一种 JSON。

## 配置与凭证

普通配置为家目录 `.kdl/config.toml`，凭证为 `.kdl/credentials.toml`；macOS/Linux 目录 0700、文件 0600，Windows 设置当前用户访问权限。凭证明文保存，同用户运行的程序仍可读取；订单 Secret 不持久化。

`--config`、`KDL_AGENT_CONFIG`、家目录默认依次决定普通配置文件；相对路径相对当前目录，凭证位置不随其变化。网关使用 `KDL_AGENT_GATEWAY_URL` 覆写，凭证使用 `KDL_AGENT_TOKEN` 临时注入；环境值不会自动保存。`--print-paths` 仅显示路径与来源，不发业务请求。

凭证按网关地址绑定，切换网关不会复用其他地址的本地凭证。登录隐藏输入，只读验证后保存；登录失败保留原值。`auth status` 区分未配置、有效、无效、不可达与错误；`auth logout` 只清理当前网关的本地凭证，不撤销服务端，也不移除环境注入。

旧 CWD `kdl-agent.toml` 不自动发现，含内嵌 token 或 device 的旧格式会拒绝加载。取消旧配置覆盖后重新登录，旧文件由用户自行清理，不让 Agent 读取或迁移秘密值。

## 业务授权

敏感 grant 为 `product.purchase.create`、`support.ticket.create`、`order.secret.read`，默认关闭，在会员中心显式授权。购买只创建待付款订单，付款仍在官网。创建工单及购买同一操作重试必须复用原幂等键与参数。

`order secret get` 会显式输出已授权订单密钥。仅需要提取代理或设置白名单时使用 `proxy fetch`、`order whitelist`，避免额外回显密钥。关闭 grant 只阻止后续获取，已取得的订单密钥需在订单设置主动轮换。

## 反馈

问题反馈到[仓库 Issues](https://github.com/gizaZerozhang/kdl-agent-cli/issues)，提供版本、系统、脱敏错误码及复现步骤。不要附带凭证文件、环境变量全集、订单 Secret 或完整业务数据。
