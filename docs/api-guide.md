# 快代理 Agent Gateway API 接入指南

Agent Gateway API 用于账户查询与业务操作；代理提取、白名单操作使用订单 OpenAPI。

| 接口体系 | 调用地址 | 身份凭证 | 主要用途 |
| --- | --- | --- | --- |
| Agent Gateway API（本文） | `https://agent-gateway.kdlapi.com/v1/` | 会员中心创建的 Agent 凭证，Bearer 认证 | 账户与订单查询、报价、创建待付款订单和工单、授权获取订单密钥 |
| 订单 OpenAPI | 对应订单返回的 `api_domain` | 订单 `secret_id`、`secret_key`，按对应接口规则认证 | 代理提取、白名单等订单操作 |

[订单 OpenAPI 文档](https://stg3.kuaidaili.com/doc/api/) · [授权说明](./cli-guide.md#授权凭证与订单密钥)

## 首次调用

在[Agent 设置](https://stg3.kuaidaili.com/uc/agent/settings/)创建凭证，由运行环境注入 `KDL_AGENT_TOKEN`，不要写入代码或对话。

```bash
curl --fail-with-body 'https://agent-gateway.kdlapi.com/v1/account/funds' \
  -H "Authorization: Bearer ${KDL_AGENT_TOKEN}" \
  -H 'Accept: application/json'
```

成功响应示意：

```json
{
  "success": true,
  "data": {
    "cash_balance": "100.00",
    "invoiceable_amount": "50.00",
    "currency": "CNY"
  },
  "request_id": "req_example"
}
```

## 公共约定

### 认证、请求头与数据范围

| 项目 | 规则 |
| --- | --- |
| 基址 | `https://agent-gateway.kdlapi.com`；以下接口路径均以 `/v1/` 开头 |
| 认证 | `Authorization: Bearer <AGENT_CREDENTIAL>`，所有业务接口均需有效凭证 |
| JSON 请求体 | 有正文时使用 `Content-Type: application/json`；不在 URL 中传递凭证 |
| 数据归属 | 账户由凭证确定；不通过自传用户 ID 切换账户；跨账户订单与不存在订单均返回 404 |
| 幂等键 | 创建待付款订单、创建工单必须带 UUID v4 `Idempotency-Key` |
| 无幂等键操作 | 基础查询、报价、工单预览、订单密钥获取不使用该键 |
| HTTPS | 校验证书与主机名；不关闭证书验证、不自动跟随重定向发送凭证 |
| 追踪 | 响应 `request_id` 用于调用监控与支持排查，不替代幂等键 |

三项敏感 grant 为 `product.purchase.create`、`support.ticket.create`、`order.secret.read`，默认关闭。使用者在会员中心管理；客户端遇到未授权时给出处理入口，不自行扩大权限。其余业务和额度约束仍然有效。

### 响应信封

成功：HTTP 2xx，`success=true`，结果位于 `data`，附 `request_id`。

失败：对应 HTTP 错误状态，`success=false`，`error.code` 为可编程判断依据，`error.message` 为说明，可能有 `error.details` 与 `error.next_action`。

```json
{
  "success": false,
  "error": {
    "code": "AUTH_REQUIRED",
    "message": "需要有效的 Agent 凭证",
    "next_action": {
      "type": "url",
      "label": "前往会员中心创建 Agent 凭证",
      "url": "https://stg3.kuaidaili.com/uc/agent/settings/"
    }
  },
  "request_id": "req_example"
}
```

客户端应同时处理 HTTP 状态和 JSON，不能只看 HTTP 200，也不能假定网络错误或入口代理异常一定返回此结构。不要依赖中文错误消息做分支。

金额使用十进制字符串并配套 `currency`；用十进制定点方式计算，不转换成二进制浮点后当作付款金额。订单、工单、消息 ID 按字符串传递。时间按字段中的时区和偏移解释，不使用本机时区替换业务时区。

### 分页与频率

订单和站内信列表支持 `limit`、`cursor`。`limit` 默认 20，范围 1—100；第一页省略 `cursor`，后续把 `next_cursor` 原样交给 URL 查询参数编码器，不解析或自行构造游标。

```json
{"success": true, "data": {"items": [], "next_cursor": null}, "request_id": "req_example"}
```

`next_cursor=null` 表示结束；存在游标时不能仅因当前页为空而停止。筛选条件改变后从第一页重新查询。

请求受全局、账户、凭证、Capability 和下游服务限制，取实际生效限制。不要把多个接口理解为各自拥有无限独立配额。429 响应带 `Retry-After`，按其秒数等待并降低并发；本版不承诺高频轮询吞吐。具体配额随正式运行说明核定。

### 创建操作的幂等重试

1. 为一次逻辑创建生成 UUID v4，和请求参数一起保留在调用方任务状态中。
2. 同一操作超时或响应丢失时使用相同键和相同业务参数重试，成功复播返回原结果。
3. 相同键提交不同业务参数返回 `IDEMPOTENCY_CONFLICT`；不能以自动换键方式消除错误。
4. 用户变更需求、重新报价后决定下单等新操作，应生成新键，并先核对原操作是否已完成。
5. 凭证被撤销或授权变更后仍需重新校验，不能把幂等重试当作绕过权限的方式。不跨凭证复用并假定全局去重，也不承诺无限期保留重试结果。

## 接口总览

| 方法与路径 | 用途 | 敏感 grant | 幂等键 |
| --- | --- | --- | --- |
| `GET /v1/account/funds` | 账户资金 | 无 | 不使用 |
| `GET /v1/account/summary` | 账户摘要 | 无 | 不使用 |
| `GET /v1/messages` | 站内信元数据列表 | 无 | 不使用 |
| `GET /v1/messages/{message_id}` | 站内信元数据详情 | 无 | 不使用 |
| `GET /v1/orders` | 代理订单列表 | 无 | 不使用 |
| `GET /v1/orders/{order_id}` | 代理订单详情 | 无 | 不使用 |
| `GET /v1/orders/{order_id}/stats` | 最近 24 小时错误与风控摘要 | 无 | 不使用 |
| `GET /v1/orders/{order_id}/guide` | 接入指引与配置模板 | 无 | 不使用 |
| `GET /v1/products` | Agent 对外开放产品目录 | 无 | 不使用 |
| `GET /v1/products/trial-eligibility` | 当前账户试用资格 | 无 | 不使用 |
| `GET /v1/products/{product_type}/spec` | 动态产品规格 | 无 | 不使用 |
| `POST /v1/products/quote` | 获取报价 | 无 | 不使用 |
| `POST /v1/products/purchases` | 创建新购待付款订单 | `product.purchase.create` | 必填 |
| `POST /v1/tickets/preview` | 预览工单 | 无 | 不使用 |
| `POST /v1/tickets` | 创建真实工单 | `support.ticket.create` | 必填 |
| `POST /v1/orders/{order_id}/secret` | 获取订单密钥 | `order.secret.read` | 不使用 |

“无”仅表示无需额外敏感 grant，不表示匿名可调用。下载[Agent Gateway API 的 OpenAPI 规范文件](https://github.com/gizaZerozhang/kdl-agent-cli/releases/download/v0.1.0-beta.4/openapi.yaml)。这里的 OpenAPI 指接口描述标准，文件描述的是本文的 Gateway API，不是原有订单 OpenAPI 产品的完整接口列表。通过[版本记录](https://github.com/gizaZerozhang/kdl-agent-cli/releases/tag/v0.1.0-beta.4)获取配套版本及兼容要求。

本指南的稳定原文地址为 `https://stg3.kuaidaili.com/kdl-agent/docs/api-guide.md`，同版本快照为 webhp `/kdl-agent/releases/{version}/docs/api-guide.md`。所有指南提供 Markdown 原文，稳定原文跟随推荐版本；接入旧版本时使用其配套快照和 OpenAPI。webhp 文档地址不是 Gateway API 基址。

## 账户与站内信

### 账户资金与摘要

两个账户接口无查询参数和请求体。

| 接口 | `data` 主要字段 | 解释 |
| --- | --- | --- |
| `GET /v1/account/funds` | `cash_balance`、`invoiceable_amount`、`currency`；可选 `as_of` | 现金余额、可开票金额、币种和更新时间 |
| `GET /v1/account/summary` | `realname_status`、`restrictions`、`order_counts`；可选 `risk_notice`、`order_risk_summary` | 实名状态、限制原因及下一步、订单和试用聚合计数 |

`realname_status` 为 `unverified`、`pending`、`verified`、`rejected` 或 `additional_verification_required`。`restrictions` 每项包括 `reason_code`、`reason`、`scope`、`affected_product_types` 和可为空的 `next_action`。

`order_counts` 包括 `total_orders`、`renewal_due_orders`、`trial_eligible_products`、`trial_eligible_specs`。风险摘要仅提供分类和数量，不是原始风控记录。

### 站内信

列表支持公共分页参数；详情的 `message_id` 是正整数形式的字符串。

- 列表：`data.items[]` 包含 `id`、`title`、`created_at`、`member_center_url`，另有 `next_cursor`。
- 详情：返回同样字段并增加 `is_read`。
- 两者均不返回正文、正文摘要或 HTML；客户通过返回的官网链接阅读正文，客户端不要虚构摘要。

## 代理订单与接入信息

### 列表与详情

`GET /v1/orders` 在公共分页参数之外支持可选 `status`：`active`、`closed`、`expired`、`cancelled`。该查询不包含待付款订单。

`data.items[]` 包含 `order_id`、`product_type`、`product_name`、`status`、`api_domain`，可选 `expires_at`、`auto_renew`；`next_cursor` 表示下一页。`product_type` 是程序使用的产品编码，`product_name` 是面向客户的产品中文名称。

`GET /v1/orders/{order_id}` 返回对应字段及：

| 字段 | 结构与含义 |
| --- | --- |
| `specification[]` | `name`、`label`、`value`、`unit`；具体字段随产品类型变化 |
| `quota_items[]` | `metric`、`label`、`unit`、`total`、`used`、`remaining`、`reset_period` |

额度必须连同单位和重置周期解释，例如 IP 数量和字节流量不能相加成同一个余额。查询不会返回真实订单密钥。

### 最近 24 小时摘要

`GET /v1/orders/{order_id}/stats` 返回 `order_id`、`product_type`、`product_name`、`period_start`、`period_end`、`generated_at`、`error_breakdown`、`risk_breakdown`、`member_center_stats_url`。`product_name` 是面向客户的产品中文名称。

分类项包含 `category`、`count`、`ratio`、`hint`。周期为最近 24 小时，按返回时间及 Asia/Shanghai 业务语义解释；不提供任意历史区间、请求总量时间序列、客户端 IP 或访问域名。产品不支持时返回 `UNSUPPORTED`，按返回的官网入口继续查看，不将失败当成“没有错误”。

### 接入指引

`GET /v1/orders/{order_id}/guide` 返回：

- `order_id`：当前订单。
- `documentation_url`：该订单相关官方接入文档。
- `auth_method`：接入认证方式。
- `config_template`：配置模板，仅含占位信息。
- 可选 `next_actions`：后续官网操作入口。

配置模板不会自动携带真实密钥。需程序化使用订单 OpenAPI 时按[订单密钥章节](#订单密钥与订单-openapi)授权获取。

## 产品规格报价与下单

### 目录与试用资格

`GET /v1/products` 无请求体，返回 `data.items[]`：`product_type`、`name`、`summary`、`suitable_scenarios`、`unsuitable_scenarios`、`documentation_url`、`purchase_url`、`value_dimensions`。

价值说明仅展示接口已公开的内容；未公开的 `content` 或 `evidence_url` 为空时，不由模型补写 SLA、资源独享或赔付承诺。

`GET /v1/products/trial-eligibility` 返回每个产品的 `product_type`、`eligible`、`trial_specs`，可有 `reason_code` 和 `trial_url`。规格包括 `trial_spec_id`、`name`、`configuration`、`limits`、`constraints`、`valid_days`。只有实际返回可用资格时才能告诉客户可申请试用，申请仍通过官网入口完成。

### 动态规格与报价

`GET /v1/products/{product_type}/spec` 的产品标识取自目录，返回 `product_type`、`spec_version`、`configuration_fields`、`configuration_rules`、`limits`、`constraints`。

按规格中字段类型、可选值、依赖及限制构造配置。规格和报价随业务变化，不在客户端硬编码全部产品参数、价格或库存。

`POST /v1/products/quote` 请求体：

| 字段 | 类型 | 必填 | 来源 |
| --- | --- | --- | --- |
| `product_type` | string | 是 | 产品目录 |
| `spec_version` | string | 是 | 当前规格，保留原始值 |
| `configuration` | object | 是 | 按动态规格构造，至少一项配置 |

以下是请求形态示例，执行前必须替换产品、规格版本和配置；不能将示例配置应用于任意产品：

```json
{
  "product_type": "PRODUCT_TYPE",
  "spec_version": "SPEC_VERSION_FROM_API",
  "configuration": {"pay_mode": "monthly", "duration_months": 1}
}
```

返回 `quote_id`、`product_type`、`spec_version`、`configuration`、`currency`、`total_amount`、`line_items`、`expires_at`、`purchase_url`。报价是后续下单依据，不是订单，也不是支付成功通知。

### 创建待付款订单

`POST /v1/products/purchases` 需要 `product.purchase.create` 和 UUID v4 `Idempotency-Key`。请求体仅包含以下业务字段：

| 字段 | 类型 | 必填 | 来源 |
| --- | --- | --- | --- |
| `quote_id` | string | 是 | 有效报价返回值 |
| `product_type` | string | 是 | 报价返回值 |
| `spec_version` | string | 是 | 报价返回值，不能编造或替换 |
| `configuration` | object | 是 | 报价返回的完整配置 |

不接受通过客户端传入金额来覆盖平台计算。保留规格版本的原始语义；遇到接口数据与校验不一致时提供 `request_id` 反馈，不能伪造版本号解决。

成功结果包含 `order_id`、`status=pending_payment`、`product_type`、`spec_version`、`configuration`、`currency`、`total_amount`、`expires_at`、`payment_url`、`member_center_url`。

将付款链接交给客户在官网完成。接口不自动付款，不支持续费、升级或批量续费。未付款订单不在代理订单列表/详情投影内，不能用订单详情 404 推断本次创建失败并自动再下单。

## 工单预览与创建

### 预览

`POST /v1/tickets/preview` 不落库，不需要敏感 grant 或幂等键。

| 字段 | 类型 | 必填 | 约束 |
| --- | --- | --- | --- |
| `subject` | string | 是 | 非空，最长 200 字符 |
| `body` | string | 是 | 非空，最长 10000 字符 |
| `order_id` | string | 否 | 本账户相关订单 |

响应为 `preview_text` 和 `attachments_hint`。附件提示不代表本接口支持上传。预览请求不接受创建工单专用的类型、时间和时区字段。

### 创建

`POST /v1/tickets` 需要 `support.ticket.create` 和 UUID v4 `Idempotency-Key`。

| 字段 | 类型 | 必填 | 约束 |
| --- | --- | --- | --- |
| `ticket_type` | string | 是 | `technical`、`billing`、`account`、`product`、`other` |
| `subject` | string | 是 | 1—200 字符，不含 CR/LF |
| `body` | string | 是 | 1—1500 字符，不含 CR/LF |
| `order_id` | string | 否 | 正整数形式的订单号，须属于当前账户 |
| `occurred_at_start` | string | 是 | RFC3339 时间，明确时区偏移 |
| `occurred_at_end` | string | 是 | RFC3339 时间，不早于起始时间 |
| `timezone` | string | 是 | IANA 时区，如 `Asia/Shanghai` |

请求体示例，实际提交前替换为客户真实问题及发生时间，去除凭证和密钥：

```json
{
  "ticket_type": "technical",
  "subject": "代理连接异常",
  "body": "指定时段内多次连接超时，请协助排查。",
  "occurred_at_start": "2026-09-08T09:00:00+08:00",
  "occurred_at_end": "2026-09-08T09:30:00+08:00",
  "timezone": "Asia/Shanghai"
}
```

预览允许的正文长度大于创建限制，创建前必须重新整理校验。请求不带附件、凭证或客户端自定状态。

成功返回 `ticket_id`、`status=pending_reply`、`ticket_type`、`member_center_url` 及可选 `order_id`。业务可能因待回复数量限制或提交互斥返回 `BUSINESS_RULE_VIOLATION`；提示客户查看已有工单，不循环创建绕过限制。

## 订单密钥与订单 OpenAPI

### 获取订单密钥

`POST /v1/orders/{order_id}/secret` 需要 `order.secret.read`，无请求体，不使用 `Idempotency-Key`，不要求设备身份或配对。

成功响应带 `Cache-Control: no-store`，`data` 包含：

| 字段 | 类型 | 用途 |
| --- | --- | --- |
| `order_id` | string | 当前订单 |
| `secret_id` | string | 订单 API 身份标识 |
| `secret_key` | string | 订单 API 密钥明文 |
| `api_domain` | string | 订单公开 API 的 HTTPS 根地址 |

不在本文给出可误用的密钥值。客户端不得将响应加入通用响应缓存、访问日志、错误诊断或 APM。`no-store` 是缓存控制要求，不能保证用户选择的外层 Agent 服务不会记录输出。

### 两段调用使用不同凭据

1. Agent 凭证仅用于访问 Gateway，获得该订单的密钥与 `api_domain`。
2. 按 `order guide` 返回的官方文档及订单 OpenAPI 签名规则使用 `secret_id`、`secret_key` 调用该订单服务。
3. `api_domain` 必须为 HTTPS、443 或默认端口、根路径，无用户信息、查询参数或片段；校验证书链与主机名，不自动跳转或降级 HTTP。
4. 不向订单 OpenAPI 发送 Agent Bearer，也不将订单密钥当作 Gateway Bearer。

Gateway 没有通用代理转发接口。CLI 的 `proxy fetch`、`proxy auth`（需支持该命令的版本）和白名单便利命令在本地取得订单密钥后直连订单 OpenAPI；下游能力、额度、频率和请求语义由订单服务决定，不能直接套用 Gateway 写接口的幂等规则。

订单 SecretId/SecretKey 用于签名调用订单 API，不是代理用户名和密码。`GET /api/getproxyauthorization/` 返回 `data.type=Basic`、`data.credentials`，用于代理连接认证；也可按订单设置使用 IP 白名单。Basic 值仅交给代理，不能作为目标网站请求头。完整操作见[从提取到连接代理](./cli-guide.md#从提取到连接代理)。HTTP 407 后统计为空不能证明没有鉴权失败，需核对代理侧上报与统计归属。

开启读取授权后，密钥可能进入客户选择的 Agent/模型。关闭 grant 或撤销 Agent 凭证仅阻止后续获取，已取得的密钥继续有效；需要失效时在订单 API 设置中重置，并同步更新其他客户端。详见[共同授权说明](./cli-guide.md#授权凭证与订单密钥)。

## 错误与恢复

| HTTP | `error.code` | 含义与处理 |
| --- | --- | --- |
| 401 | `AUTH_REQUIRED` | 缺少有效凭证或已过期；核对环境，在会员中心处理凭证 |
| 401 | `CREDENTIAL_REVOKED` | 凭证已撤销；换用有效凭证，不能继续重试旧凭证 |
| 403 | `FORBIDDEN` | 未授权当前操作；提示对应 grant 与处理入口 |
| 404 | `NOT_FOUND` | 对象不存在或不属于当前账户；核对 ID 与对象类型，不探测其他账户 |
| 409 | `IDEMPOTENCY_CONFLICT` | 同一键用于不同业务参数；核对原操作，不自动换键重建 |
| 409 | `CREDENTIAL_GENERATION_CONFLICT` | 凭证状态在执行期间变化；重新核对凭证与授权，先查原操作结果 |
| 409 | `BUSINESS_RULE_VIOLATION` | 业务数量、状态或互斥限制；按 `details.reason_code` 和官网状态处理 |
| 422 | `VALIDATION_ERROR` | 参数校验失败；修正类型、规格、时间或文本约束后再请求 |
| 422 | `QUOTE_EXPIRED` | 报价过期；重新报价并核对后再发起新的创建操作 |
| 429 | `RATE_LIMITED` | 按 `Retry-After` 等待并降低并发 |
| 501 | `UNSUPPORTED` | 产品或订单不支持该能力；使用返回的替代入口，不无限重试 |
| 502 | `UPSTREAM_ERROR` | 业务服务响应异常；有限退避并保留请求标识 |
| 503 | `SERVICE_UNAVAILABLE` | 服务依赖暂不可用；有限退避，不绕过认证或审计继续请求 |

收到未知错误码时保留 HTTP 状态、错误码和请求标识，采取保守失败处理，不因错误消息“看起来可忽略”继续写入。网络超时或 5xx 不能证明订单/工单没有创建；再次执行时遵守同键同参数规则。

支持排查只需提交发生时间、接口路径、错误码、`request_id` 及脱敏的必要参数。不要提交完整认证请求头、Secret 响应或未脱敏的工单正文。使用[调用监控](https://stg3.kuaidaili.com/uc/agent/monitor/)与[安全记录](https://stg3.kuaidaili.com/uc/agent/security/)核对 Gateway 调用和凭证变更。
