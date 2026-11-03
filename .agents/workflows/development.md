# 开发与验证

状态：2026-11-01 按 `71ef6fa` 的 package scripts、CI 和现有构建/生成入口核对。命令均从仓库根目录运行。代码变更遵守 [工程规范](../standards/engineering.md)，界面和用户流程另遵守 [产品体验规范](../standards/product.md)。

## 从用户结果到验证

1. 先说明用户目标、当前事实与期望行为，核对已有产品决定；涉及默认策略、名称、主流程或精度/耗时取舍时先向用户复核，常规技术选择自主推进。
2. 找到受影响领域、状态所有者、契约和最接近的既有界面/组件；涉及新功能或流程变化时写一段内部用户路径及成功、失败、返回/刷新场景，局部修复核对已有路径，不自动把设计文字放进 UI。
3. 在已有架构和设计语言上实现；需要调整 API、配置、数据或恢复协议时同步闭合，不能只让页面先假定后端支持。
4. 按下表选择验证，并逐句检查新增用户文案的依据、必要性和表达；界面变化对照邻近区域及相关状态人工走查，不能只看测试数量或孤立截图。
5. 检查 diff，更新当前契约/未完成清单与必要交接；记录用户确认、验证结果、局限和剩余项。文档任务按文档范围验证，不无故重跑业务 suite。

## 环境与起步

Go 最低版本看 [go.mod](../../go.mod)（当前 1.25）；前端 CI 显式使用 Node 22，依赖按 `web/package-lock.json` 安装；真实多部件媒体测试需要 FFmpeg/ffprobe。Docker 构建镜像的 Go 版本与最低开发版本可以不同。

```sh
git status --short
git log -5 --oneline
npm --prefix web ci
```

已具备依赖时无需反复安装。原生开发使用 `make web-dev`，Vite 默认 5173、代理后端 10000，可通过 `VITE_MOJI_API_TARGET` 改代理。后端启动参数、示例配置和完整部署步骤见 [README](../../README.md)。

此工作区可能已有真实实例和本地数据；检查代码不需要启动第二个实例。集成演练用单独配置、端口、临时 data root 和假服务，不能只换数据库目录却仍连真实 qB/Stash 执行操作。

## 按范围选择检查

| 变更 | 必要检查 |
| --- | --- |
| `.agents` 文档和入口 | `python3 .agents/scripts/check_docs.py`、`git diff --check`、人工语义核对 |
| Go 业务局部 | `GOCACHE=/tmp/moji-go-cache go test ./internal/<package>`；有跨模块影响再全量 |
| 后端公共契约、持久化或生命周期 | `GOCACHE=/tmp/moji-go-cache go test ./...`；并发风险加受影响包 `-race` |
| React 数据或交互 | `npm --prefix web test -- <相关文件>`、`npm --prefix web run build` |
| 文案、错误映射、locale | `npm --prefix web run i18n:export`（资源改动时），再 `npm --prefix web run i18n:check` |
| 可见布局、焦点或移动交互 | 相关 Playwright spec；固定环境检查实际截图及浏览器错误 |
| 媒体合并、字幕、FFmpeg 调用 | `make test-media`，确认工具集成没有跳过 |
| 版本/发布脚本 | `python3 scripts/version_test.py`、`python3 scripts/docker_publish_test.py` |
| Docker/路径挂载/镜像运行 | `make docker-build`、`make docker-test`，按平台分别记录 |

测试应证明行为或关键不变量；不为文档或低影响格式调整补镜像实现的测试。每项检查说明通过、失败、skip 或环境受阻，不能扩大结论。

## GraphQL 与生成文件

Moji schema 源在 `graphql/moji/schema.graphql` 和 `graphql/moji/types/`，gqlgen 配置在 [gqlgen.yml](../../graphql/moji/gqlgen.yml)。Web 操作源在 `web/src/**/*.graphql`。

```sh
make generate-moji-graphql
```

该 target 同时更新后端 executable/model 和 Web bindings。仅修改 Web 操作也可 `npm --prefix web run codegen`。Stash/StashBox 查询源和 schema 在各自 `graphql/` 目录，使用 `make generate-stash-client` 或 `make generate-stashbox-client`。不要为改 Moji API 无差别刷新全部外部 schema。

生成代码放在 `internal/graphqlapi/generated`、`internal/graphqlapi/model`、`pkg/*/graphql`、`web/src/graphql/generated`；修改生成源并运行工具，检查 resolver 实现是否被保留。schema 改完先生成再运行依赖生成类型的全量测试。

## 前端环境限制与视觉验证

视觉基线要求 `@playwright/test`、CI 和 wrapper 中的 `mcr.microsoft.com/playwright:v1.61.1-noble` 一致。

```sh
npm --prefix web run test:visual:docker -- task-workspace.spec.ts
# 只有确认是有意 UI 改动，才更新并逐张复核：
npm --prefix web run test:visual:docker -- --update-snapshots=all
```

本机 Vite 若报 `spawnSync sh EPERM`，先核对错误发生在 `scripts/version.sh` 调用及执行权限/沙箱，不修改业务代码掩盖环境错误。可在固定 Playwright 容器运行检查；同时核对 Node 版本，容器内置 Node 不一定等于 CI 的 setup-node 22。2026-10-27 复核中的容器实际为 Node 24.17，不能据那次成功声称当前或 Node 22 的完整 CI 复现。

若只需容器隔离检查，可复用下面的命令；它不会自动切换 Node 版本：

```sh
docker run --rm --ipc=host \
  --user "$(id -u):$(id -g)" --env HOME=/tmp --env CI=1 \
  --mount "type=bind,source=$PWD,target=/work" --workdir /work/web \
  mcr.microsoft.com/playwright:v1.61.1-noble /bin/sh -c \
  'git config --global --add safe.directory /work && node --version && npm test && npm run i18n:check && npm run build'
```

权限受限时按工具权限机制执行。`httptest` socket/IPv6 被拒绝、浏览器系统库缺失、镜像拉取 429 都要区分环境问题与产品回归；未完成的检查仍应明确记录。

## 实际 CI 范围

[ci.yml](../../.github/workflows/ci.yml) 当前配置：

- Web：固定 Playwright 容器，setup-node 22，npm ci、i18n、Vitest、build、Chromium 视觉。
- Go：版本、发布脚本及索引差分比较器的 Python 测试，以及 `go test ./...`，实际运行所有包的默认离线测试，包含 storage、downloader 和内置索引模块。真实服务测试仍显式 opt-in。
- Docker：原生 amd64/arm64 各构建与 smoke，适用事件导出已测镜像。
- Indexer compatibility：调用[兼容工作流](../../.github/workflows/indexer-compatibility.yml)，固定 SDK/Jackett 构建与合成 HTTP 场景真实差分、固定 v11 语料下载/编译及报告制品；仅开发 CI 需要 .NET。
- Publish：依赖 Web、Go、Docker 和 indexer-compatibility 四个 jobs，通过 [发布工作流](../../.github/workflows/docker-publish.yml) 合并制品发布。

2026-11-03 已落实全包 Go 配置并完成本地无测试结果缓存验证，见[CI 覆盖记录](../reviews/2026-11-03-cardigann-requests.md#ci-go-coverage)。兼容语料与上游首批差分 CI 已配置并验证本地同等命令，范围见[复核](../reviews/2026-11-03-cardigann-requests.md)。这不表示远端 Actions 已运行；更多差分场景及整体验收仍属 R-004 未完项。

## 交付

检查 diff、未跟踪文件和敏感数据；在改变行为的同一变更更新对应模块。用户要求提交时再创建提交并报告 hash。阶段复核按 [复核流程](review.md) 记录证据，活动 TODO 只留未完事项。
