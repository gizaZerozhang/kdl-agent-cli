# 发行维护

当前个人 npm 包为 `@zerozhang-giza/kdl-agent`。个人账号阶段结束后，统一更新 package.json 中的 npm 身份、repository 和 Skill 来源，保留历史版本并验证旧包迁移。

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

工具链固定 Go 1.23.6、GoReleaser 2.18.1、Node 22.14+；可信发布使用 npm 11.5.2。`v*` tag 触发 `release.yml`：测试、构建、嵌入 BUILD.json、固定 npm tgz、五平台运行验活，然后在 `github-release` 环境创建 Draft Release。当前仅开放 beta；稳定版签名及业务门尚未接入。

发布前检查 npm 候选 pack-report.json 的允许文件范围、各平台运行结果，并下载 Draft 附件运行 `node scripts/verify-release.cjs <下载目录>`。公开同一批附件后，重新匿名验证下载，再发布候选 tgz 到 npm。首次建包通过本人 `npm login`/验证完成：`npm publish <候选 tgz> --access public --tag beta`，首次手工发布不声明 OIDC provenance。

后续在 npm 配置 Trusted Publisher：GitHub owner `gizaZerozhang`、repository `kdl-agent-cli`、workflow `publish-npm.yml`、environment `npm-production`。配置并验证后，运行该工作流并指定已公开的 beta tag，通过 OIDC/provenance 发布同一候选。未完成 npm 端绑定前，工作流不能视为可用的发布身份。

## 稳定版门

稳定发行要求 Apple Developer ID 签名、公证和验证记录，以及最低系统与真实业务验收；缺少材料时发行工作流停止。beta 可输出未签名候选，必须保留预发行与平台验收说明，不能标记稳定就绪。

保护 `v*` tag 禁止更新和删除；配置 `github-release`、`npm-production` 环境审批及最小权限。流水线不会覆盖已存在 Release，失败时下载既有候选核对来源，不自动重建替换公开附件。GitHub 已公开但 npm 失败时，只补发原 tgz；npm 同版本已存在时先核对 registry 的 integrity，不重复发布。回退 npm dist-tag，提示问题版本弃用，不能把弃用等同于禁止精确安装。
