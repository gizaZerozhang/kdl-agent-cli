# 发行维护

当前 GitHub 为 `kuaidaili/kdl-agent-cli`，npm 使用公司用户 `kuaidaili` 的 `@kuaidaili/kdl-agent`，本轮不创建 npm Organization。公司首版为 `0.1.0-beta.5`，实际发布结果以同版本 Release 和 registry 为准；个人包与历史附件保留，迁移步骤见[安装指南](./install.md#从个人包迁移)。

## 公司发行身份

- tag 保护与 `github-release`、`npm-production` 环境审批随仓库转移保留；当前获公司仓库 Admin 的维护者继续审批。
- 首次创建公司包需公司 npm 账号完成本人认证，发布已有固定候选；随后把 Trusted Publisher 绑定到 `kuaidaili/kdl-agent-cli`、`publish-npm.yml`、`npm-production`，不能沿用旧 owner 的绑定。
- 新包首次手工发布不声明 OIDC provenance；后续通过新绑定实际发布后再记录验证结果。旧个人包只有在新包安装验收通过后才添加迁移提示。
- 新包仍使用 beta；npm 首次建包可能自动附加 latest，实际标签须回读，不能把 latest 当作稳定验收证据。
- 历史记录中的个人仓库地址保留用于追溯；旧地址重定向、旧二进制下载和旧 Skill tag 读取须实测。

## 首次 beta 发布记录

beta.3 已发行：源码 `75934e505a6355bbe71b3db998c3026f6f8ff318`，[候选、五平台与 Release](https://github.com/gizaZerozhang/kdl-agent-cli/actions/runs/34465026715)全部通过，自动派发 [npm Trusted Publishing](https://github.com/gizaZerozhang/kdl-agent-cli/actions/runs/34465592589) 成功。包 SHA256：`8618dc172bdf4877da6dbc3aaed7a140e7152747271fa8430964e382837bf63c`，registry integrity 匹配并提供 provenance；beta 指向 beta.3，latest 保留 beta.1。本机隔离环境按原样 npx 完成 beta.2→beta.3 CLI/Skill 同步升级，匿名版本检查通过，无凭证落盘。`npm exec --package=...` 替代写法会干扰嵌套 skills 调用，不属于已验证入口。

beta.2 发行记录：`v0.1.0-beta.2`，源码 `b96a7061450d8b4c6899c52442307ac0ba9fdb89`。[候选与五平台验证](https://github.com/gizaZerozhang/kdl-agent-cli/actions/runs/34460103213) 全部通过；[npm Trusted Publishing](https://github.com/gizaZerozhang/kdl-agent-cli/actions/runs/34460767167) 已实际完成 OIDC/provenance 发布。固定 npm 包 SHA256 为 `149e3bf2bcc98a414582baf18b0494a2786f11ecd2f541ffb645f5b999420822`，registry 下载包与 Release 一致。`beta` 指向 beta.2，`latest` 仍为 beta.1；继续使用精确 beta 或 beta 标签。

以下为 beta.1 首发时的历史记录；后续 OIDC 验证已由上述 beta.2 发行完成。

2026-09-10 已公开 [v0.1.0-beta.1](https://github.com/gizaZerozhang/kdl-agent-cli/releases/tag/v0.1.0-beta.1) 与 [npm 0.1.0-beta.1](https://www.npmjs.com/package/@zerozhang-giza/kdl-agent/v/0.1.0-beta.1)，源码 commit 为 `a50bc08164a9ab11a8f9d8b5435eb3553fcdeed8`。发布包 SHA256 为 `eac0576e2ef9c1cdc2c76d95f48152a82e8a8b28aea2364b992509c9ddafc9f6`，registry 下载包与 Release 固定候选一致。公开 tag 和附件不可覆盖。

[发行 CI](https://github.com/gizaZerozhang/kdl-agent-cli/actions/runs/34433571096) 的候选、五平台安装和 Draft 共 7 项通过；21 项 npm 回归、Go test/vet、匿名 npx 精确版本与 beta 向导、Codex Skill 实装、同版本跳过、跨目录执行、卸载保留配置/Skill、禁用生命周期脚本后首次运行补装通过。最低系统、Windows ACL 和真实业务仍待专项验收。

首次发布由账号启用 2FA 后手工完成，不含 OIDC provenance；beta.2 已验证 Trusted Publishing，不重复发布现有版本。

首次指定 `--tag beta` 后，registry 同时添加了 `latest`；完成认证后的官方 `npm dist-tag rm` 返回 HTTP 400，回读确认未删除。npm CLI [同类报告 #8490](https://github.com/npm/cli/issues/8490)记录了相同行为。当前文档只提供精确 beta 或 `@beta` 入口，latest 不作为稳定就绪证据；待稳定版本验收后再移动该标签，不虚构版本或撤销现有发布。

## 维护者检查

```bash
npm ci --ignore-scripts
npm test
go test -race ./...
go vet ./...
```

公开构建只在本独立仓库创建 tag，不向内部 Gateway 仓库推送版本。package.json、tag、CLI、Skill 与文档版本必须一致；beta 使用 beta 标签，稳定版使用 latest。

## 固定候选

发行要求干净源码。使用 GoReleaser 生成原生包、补齐配套资料后，再创建小型 npm 安装包；校验清单嵌入 npm 包，不能在 npm 发布后重新构建替换原生包。

```bash
python3 scripts/release-assets.py preflight
goreleaser release --clean --skip=publish
python3 scripts/release-assets.py collect dist dist/candidate
npm run release:prepare -- dist/candidate dist/npm
```

工具链固定 Go 1.23.6、GoReleaser 2.18.1、Node 22.14+；可信发布使用 npm 11.5.2。`v*` tag 触发 `release.yml`：测试、构建、嵌入 BUILD.json、固定 npm tgz、五平台运行验活，然后经 `github-release` 环境审批校验并公开 Release，自动派发同 tag 的 npm 工作流。当前仅开放 beta；稳定版签名及业务门尚未接入。beta.3 已完成自动串联与 OIDC 发行验证。

审批前检查 npm 候选 pack-report.json 的允许文件范围、各平台运行结果。工作流自动运行 `verify-release.cjs`，重试时核验已有 Release 附件，不覆盖候选；npm 发布失败时重试 `publish-npm.yml` 并指定同一 tag。

npm Trusted Publisher 的公司绑定目标：GitHub owner `kuaidaili`、repository `kdl-agent-cli`、workflow `publish-npm.yml`、environment `npm-production`。首次建包后配置并回读确认；环境审批后通过 OIDC/provenance 发布固定候选并验证 registry integrity，同版本一致时跳过，不移动 dist-tag，冲突时停止。发布后仍需隔离目录安装验收和 provenance 验证。

日常操作：更新 package.json/锁文件与配套资料，测试通过后提交干净 commit，再创建并推送新的 `vX.Y.Z-beta.N` tag。已发行版本不可重用。工作流完成后同步官网精确版本和文档，官网部署独立执行。

## 稳定版门

稳定发行要求 Apple Developer ID 签名、公证和验证记录，以及最低系统与真实业务验收；缺少材料时发行工作流停止。beta 可输出未签名候选，必须保留预发行与平台验收说明，不能标记稳定就绪。

保护 `v*` tag 禁止更新和删除；配置 `github-release`、`npm-production` 环境审批及最小权限。流水线不会覆盖已存在 Release，失败时下载既有候选核对来源，不自动重建替换公开附件。GitHub 已公开但 npm 失败时，只补发原 tgz；npm 同版本已存在时先核对 registry 的 integrity，不重复发布。回退 npm dist-tag，提示问题版本弃用，不能把弃用等同于禁止精确安装。
