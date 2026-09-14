# 发行维护

当前 GitHub 为 `kuaidaili/kdl-agent-cli`，npm 使用公司用户 `kuaidaili` 的 `@kuaidaili/kdl-agent`，本轮不创建 npm Organization。公司首版 `0.1.0-beta.5` 已公开；个人包与历史附件保留，迁移步骤见[安装指南](./install.md#从个人包迁移)。

## 稳定发行准备验收（2026-09-14）

- 实现 `4a95c1d`、工作流 `da9cb1f` 已合入 main；[CI](https://github.com/kuaidaili/kdl-agent-cli/actions/runs/34822987091) 的 macOS、Linux、Windows 源码回归及候选构建全部通过。npm 40 项、签名验证 Python 4 项、Go race/vet、GoReleaser 配置与工作流语法检查通过。
- Windows Server runner 原生验证当前用户 DACL、禁止父目录权限继承、登录/更新/退出和配置替换失败后的凭证恢复；最低 Windows 11 与 macOS 12、Ubuntu 22.04 目标机器仍需独立验收。
- Apple 尚未开通，只有接入与模拟负例，没有实签/公证证据；公司 OIDC 首次实发、最低系统、生产业务和入口验收仍待完成。稳定发行缺任何必要记录或配置即停止。
- 本轮未创建新 tag/npm 版本，既有 beta.5 附件保留。官网及文档的 stg3 链接留到发布前最后切换。

## 公司 beta.5 发行记录

- 2026-09-14 已公开 [GitHub Release](https://github.com/kuaidaili/kdl-agent-cli/releases/tag/v0.1.0-beta.5) 和 [npm 包](https://www.npmjs.com/package/@kuaidaili/kdl-agent/v/0.1.0-beta.5)，源码 `aa10cdb3dc4c1bee6901d3ad28fd9f0eb1d83d4a`；[候选、五平台与 Release](https://github.com/kuaidaili/kdl-agent-cli/actions/runs/34815938364)全部通过。
- 35 项 npm 回归、Go race/vet 通过；15 个附件校验通过，npm 包 21,767 字节，SHA256 `914f721549ec97148217a7e848e0f19cad149c723249c8dbf0aebc03ce39c74b`，匿名 registry 下载与 Release 及 registry integrity 一致。
- 旧个人 beta.4 CLI/Skill 实装、旧包冲突保护、失败后恢复旧版和匿名 npx 迁移公司 beta.5 通过；同版本重试及公司包 `update check` 的 `up_to_date` 通过。公司 Skill 与同 tag 一致，保留配置且未创建凭证文件。
- 首发使用公司账号手工发布，无 OIDC provenance；Trusted Publisher 已绑定并回读确认，限定公司仓库 `publish-npm.yml` / `npm-production`。新绑定实际 OIDC 发行待下一需要发布的版本验证，不重发 beta.5。
- [npm 流水线](https://github.com/kuaidaili/kdl-agent-cli/actions/runs/34816218128)审批后校验既有精确版本内容一致并跳过重复发布，不移动 dist-tag；该成功结果不作为公司 OIDC 实发证据。
- 公司包 `beta` 与首次建包自动附加的 `latest` 均指向 beta.5；只推荐精确 beta 或 `@beta`。旧个人包保留，迁移弃用提示尚未设置；稳定验收与官网部署分别跟进。

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

工具链固定 Go 1.23.6、GoReleaser 2.18.1、Node 22.14+；可信发布使用 npm 11.5.2。`v*` tag 触发 `release.yml`：先经 `github-release` 审批，测试、构建、嵌入 BUILD.json、固定 npm tgz、五平台运行验活，再审批公开固定 Release，自动派发同 tag 的 npm 工作流。beta 在 Linux 构建；稳定候选同样在 Linux 构建，不依赖 Apple 账号；业务、生产运行和入口验收缺项时停止。beta.3 的个人包已完成 OIDC 验证；公司绑定的实发仍待下一版本。

审批前检查 npm 候选 pack-report.json 的允许文件范围、各平台运行结果。工作流自动运行 `verify-release.cjs`，重试时核验已有 Release 附件，不覆盖候选；npm 发布失败时重试 `publish-npm.yml` 并指定同一 tag。

npm Trusted Publisher 的公司绑定目标：GitHub owner `kuaidaili`、repository `kdl-agent-cli`、workflow `publish-npm.yml`、environment `npm-production`。首次建包后配置并回读确认；环境审批后通过 OIDC/provenance 发布固定候选并验证 registry integrity，同版本一致时跳过，不移动 dist-tag，冲突时停止。发布后仍需隔离目录安装验收和 provenance 验证。

日常操作：更新 package.json/锁文件与配套资料，测试通过后提交干净 commit，再创建并推送新的 `vX.Y.Z-beta.N` tag。已发行版本不可重用。工作流完成后同步官网精确版本和文档，官网部署独立执行。

## 稳定版门

2026-09-14 发布决定：取消客户端人工平台/最低系统专项验收、Windows ACL 人工验收，以及 macOS Developer ID 签名、公证的发行前置要求。无需为本次发布开通 Apple 账号或协调测试机器；既有五平台构建、自动运行烟测、权限和安装器回归继续执行。

本次稳定版 macOS 程序未签名/公证，系统可能阻止运行；不宣称签名完成或最低系统兼容已实测。不改变 macOS 系统安全设置。npm 的 SHA256、HTTPS、固定来源与版本校验继续启用，`release.json` 如实记录 `macosSigned: false`。

签名验证工具保留供后续接入使用，当前工作流不注入 Apple Secrets、不调用签名服务。若候选声明已签名，仍必须校验其双架构签名记录、来源 commit 与归档哈希，禁止伪造签名状态。

### 同版本稳定验收记录

完成验收后提交 `release/stable-acceptance.json`，包含 `schema_version: 1`、实际 `version`、`reviewer`，以及 `checks` 数组。每项为 `id`、`status: passed`、可访问的 HTTPS `evidence` 索引；不放凭证、业务响应和内部部署参数。必须逐项覆盖：

- `business-e2e`、`production-operations`、`public-entry`：同版本业务/授权回归、运行与回滚、正式安装入口及 Skill。

未完成时不创建虚假的 passed 文件；检查器只校验记录完整性，环境审批人必须核对证据与当前 tag，不能把手工状态当作实测。`public-entry` 与官网正式域名/文档切换放在发布前最后一步；其余工作先完成。验收记录随稳定 Release 附件及校验清单交付。

稳定版本为 `X.Y.Z`，发布 `latest` 并取消 GitHub prerelease；预发行 `X.Y.Z-beta.N` 发布 `beta`。版本不匹配、稳定门缺项、签名状态不实或重复版本内容冲突时拒绝；同版本一致时跳过且不移动标签。发行门实现不能替代公司 OIDC 的首次实际发布与 provenance 验证。

保护 `v*` tag 禁止更新和删除；配置 `github-release`、`npm-production` 环境审批及最小权限。流水线不会覆盖已存在 Release，失败时下载既有候选核对来源，不自动重建替换公开附件。GitHub 已公开但 npm 失败时，只补发原 tgz；npm 同版本已存在时先核对 registry 的 integrity，不重复发布。回退 npm dist-tag，提示问题版本弃用，不能把弃用等同于禁止精确安装。
