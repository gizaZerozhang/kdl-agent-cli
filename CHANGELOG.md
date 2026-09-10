# 更新记录

## 未发布

## 0.1.0-beta.3 - 2026-09-10

- 增加匿名新版检查、确认后同步升级 CLI/Skill 的 npm update 入口，保留登录配置和失败恢复。
- tag 流程经五平台验活和环境审批后自动公开 Release 并派发 npm；重复发布校验固定候选，冲突停止。

## 0.1.0-beta.2 - 2026-09-10

- GitHub Release 与 npm 已公开，五平台安装验证通过；首次实际 Trusted Publishing/OIDC 发行成功，registry 包与固定候选一致并提供 provenance。
- 新增代理 Basic 鉴权与白名单只读命令，明确订单 API 密钥和代理连接凭据的区别。
- 统一 Gateway 错误信封、字段级校验和 Retry-After 提示，补齐 HTTPS 连接指南与配套 Skill。
- 安装下载增加有限重试与半包清理，支持 KDL_AGENT_DOWNLOAD_TIMEOUT_MS；永久错误不重试，失败提供网络排查入口。
- 更新已公开的 beta 安装入口、发行证据、Trusted Publisher 配置与 latest 标签限制说明。

## 0.1.0-beta.1 - 2026-09-10

- GitHub Release 与 npm 已公开；五平台安装烟测、匿名 npm 精确版本/beta 向导与 Codex Skill 实装通过。
- npm 按平台下载安装器、SHA256 与版本校验、同版本 Skill 安装向导。
- 个人 npm 包 `@zerozhang-giza/kdl-agent`，终端命令保持 `kdl-agent`；升级、回退及卸载保留 `.kdl`。
- GoReleaser 候选、来源绑定检查、公开安装/API/Skill 文档与安装器回归。
- npm 打包允许清单同时验证文件缺失和意外夹带，包含同版本更新记录。
- 向导回归覆盖下载预检、固定版本、Skill 部分失败、升级失败恢复和登录/查询输出隔离。
- 查询与授权命令、家目录凭证、隐藏输入验证登录、网关隔离、状态检查和退出。
- 独立源码构建、五平台候选归档及 Go 测试。

公开源码与候选构建不代表正式业务上线或目标平台最低版本已验收。
