# 下载器、配置与运行交付

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前实现与仓库工作流配置。

主要源码：[应用接线](../../cmd/moji/main.go)、[设置编辑](../../cmd/moji/settings_store.go)、[config store](../../internal/config/store.go)、[下载器](../../internal/downloader/service.go)、[managed 下载器存储](../../internal/downloader/storage.go)、[HTTP API](../../internal/controller/api/api.go)。

## 外部客户端与配置

`pkg/stash`、`pkg/stashbox`、`pkg/jackett`、`pkg/qbittorrent` 处理协议；领域服务处理资格和编排。请求传递 context，有界并发和超时；凭据通过配置 provider 使用，不在源码或前端文案里硬编码。

配置路径优先 `-config`，否则 `MOJI_CONFIG` 或 `config.yaml`。当前必须配置 Jackett URL/API key；qB/Stash 未配置时相关能力受限。qB 首次登录失败允许启动并由后续观测重连。YAML v2 按 indexers/downloader/stash/tasks/system/logging 分组；旧格式可读取但启动不重写，成功保存写 v2，未知版本或混用字段拒绝。字段映射与离线工具见 [设置说明](../../docs/settings.md)。配置 store 通过临时文件、fsync 和原子 rename 保存，失败恢复内存中的最近有效值；连接 provider 在后续调用读取已保存设置。

qB 明确 403 时重新认证再试；写操作超时可能已被执行，不能一概自动重放。qB 5 start/stop 的 404 回退 resume/pause 是明确兼容行为。

种子元数据下载按主机串行并至少间隔一秒，Jackett `/dl/<indexer>/` 分组；429、5xx、网络超时及可能掩盖上游限流的 Jackett 404 触发退避，尊重 Retry-After。实现见 [metadata host](../../internal/taskruntime/torrent_metadata_host.go)，不要在日志输出带签名或 API key 的 URL。

## 独立下载器管理

`internal/downloader` 每三秒观测 qB，包括已完成任务的种子；快照读取不等待新 qB 请求，任务同步只保留最新一批观测。任务数据库失败时仍能观测 qB，保留最近关联信息并暂停自动清理。

当前入口是 Home 的下载器统计和“查看种子列表”弹窗，设置位于 `/settings/downloader`；没有独立 `/downloader` 页面。任务详情复用种子控制。列表以最后成功观测中仍存在的 qB 种子为范围，包含未关联种子；缺失任务不补成虚构的 qB 行。

支持启停、校验、重新汇报、限速和移除。命令接受后以实际后续观测确认，不乐观伪造 qB 状态；断连不代表全体消失。缺失/错误只阻塞相关活动下载，不回退已完成入库。

策略及删除命令在核心库，快照在 runtime；意图保存失败时不发送删除。结果不明保持 UNCERTAIN，完整成功 listing 确认消失后记来源。不能只读兼容构造器的 `downloader_state` 推断生产策略位置。

自动清理默认关闭，仅对下载完成、全部关联 Moji 任务完成的种子运行；不自动清理未关联种子。删除文件必须验证本地独立库副本和没有其他 qB 种子引用重叠内容路径；保护不满足则保留。手动移除未完成入库的关联种子同样受限。完整规则见 [下载器说明](../../docs/downloader.md)，其中旧的策略/runtime 存放描述以本模块和 managed storage 为准。

全项目 qB 状态使用 [qbittorrentStateLabel](../../web/src/utils/qbittorrentState.ts)，逐字保持官方中英文，包括 `[F]`、空格、大小写；来源 fixture 在 [官方状态依据](../../web/src/utils/__fixtures__/qbittorrent-statuses.json)。Moji 的 missing/removed 与自己的生命周期不冒充 qB 状态。

qB `free_space_on_disk` 是默认下载盘可用空间；已完成/待下载字节之和不是准确磁盘占用。缺失值显示未知。

## HTTP、可观测性和发布

后端默认 `:10000`；`/graphql` 接收 POST、multipart 与 WebSocket upgrade，普通 GET 被拒绝；另有 `/playground`、`/healthz` 和辅助 `/api`。`MOJI_WEB_DIST` 默认 `web/dist`，SPA 由 webui 提供。

stats 提供依赖状态，logging 提供可控文件滚动及实时日志事件。开发者模式的数据库性能按钮订阅现有 Manager，不新建存储 owner。日志、事件和性能计数都不能等同于持久审计历史。

版本唯一来源是 Git 标签/提交，由 [version.sh](../../scripts/version.sh) 解析，Make/Docker ldflags 与 Web 保持一致；普通 `go build` 未注入时为 dev。没有手工维护 VERSION 文件。

Docker 镜像含 Web、Moji、FFmpeg、ffprobe，非 root 运行；Dockerfile 当前使用 Go 1.26 构建镜像，源码最低版本由 go.mod 的 1.25 定义。运行数据挂载与四根映射按 [Docker 文档](../../docs/docker.md)，不要为了测试替换真实配置或重启现有服务。

CI 在原生 amd64/arm64 runner 构建及 smoke test，通过后发布复用同一镜像制品；发布依赖 Web、Go 和 Docker jobs。稳定标签、预发布、edge 的含义见根 README 和 [发布脚本](../../scripts/docker_publish.py)。未运行的架构/发布不能从本机测试推断成功。

## 修改时验证

客户端用 fake/httptest 覆盖认证、明确拒绝、超时不明与重连；真实 qB destructive 测试需要独立显式开关。

清理变更覆盖断连、无关联、交叉做种、独立副本、未完成入库、意图落盘失败与未知结果恢复。发布改动检查 version/Python 测试及相应镜像 smoke；具体命令见 [验证流程](../workflows/development.md)。

## 已记录的后续需求

设置、实际配置迁移及 Jackett 逐站搜索已交付，见 [专项复核](../reviews/2026-10-28-settings.md)。自动寻源的停止由 runtime 候选资格决定；手动搜索逐站收集，错误回退及取消有 fake HTTP 覆盖。逐站查询不等同于元数据下载主机队列。

[日志布局与跟随](../TODO.md#o-001) 仍待设计。

2026-10-28 增量设置：config.Store.ApplyPatch 在同一配置锁内读取最新已保存 YAML、合并、验证并原子持久化，成功后发布内存值；失败不提前更新内存。patchSettings 使用与 YAML v2 相同的 typed 可选字段，省略/null 保持原值，空串/空列表/false 明确保留。下载器 PatchPolicy 的 read/merge/write 同样与 SetPolicy 共用策略锁。

2026-10-29 连接测试：testServiceConnection 使用请求传入的当前字段创建短生命周期客户端，不写配置或更新运行依赖状态。Stash 读取版本，Jackett 使用 API key 调用 `/api/v2.0/indexers/all/results/torznab/api?t=indexers&configured=true` 读取站点定义，不查询各站资源；不能用需要后台登录会话的 `/api/v2.0/indexers` 来验证搜索密钥，qB 使用独立登录会话读取版本。请求受 10 秒 context 截止约束，结束释放闲置连接，结果只包含稳定枚举，不输出上游响应原文或凭据。协议及认证/错误/超时由[httptest](../../internal/graphqlapi/connection_test_service_test.go)验证。
