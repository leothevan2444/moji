# 任务、寻源与实时工作台

内置会话在登录/搜索/获取前统一解析规则默认地址：未设置镜像时采用冻结规则的首个 links 地址，HTTP 来源检查和 Cardigann 执行使用同一地址，账号会话 key 也随实际来源隔离。不会把空配置值直接交给 HTTP 层。生产 Cookie 按原逻辑持久化；显式诊断使用 NewTransientSessions 独立认证、只保留内存且关闭释放，不覆盖生产会话。

公开索引操作统一采用持久 run/candidate 身份；旧 jackettSearch、previewJackettSelection、jackettIndexers、blockedTaskTorrentCandidates 查询及客户端可替换定位信息的预览类型已移除。GraphQL Resolver 不再注入旧 Tracker，预览服务只保留 PreviewTorrentSelectionContext。外部 Jackett 通过统一 indexer 适配器运行；明确手动磁力/上传仍是独立入口，未删除。

命名查询参数资格由 [indexerquery](../../internal/indexerquery/parameters.go) 统一供搜索服务与独立 Cardigann 使用：仅接受 Year/Season/Ep、七种 ID、音乐和图书结构化字段（共 18 个），名称大小写不敏感，重复大小写别名拒绝，值和显式空串保留。模式、关键词、类别、分页与派生标记不可由参数 map 覆盖；无效查询在来源读取或持久 run 接受前拒绝。外部 Jackett 当前 JSON 搜索适配器只表达关键词/类别，不支持结构化参数或非零 offset，明确返回 INDEXER_CAPABILITY_UNSUPPORTED，不静默忽略。GraphQL 类型说明及中英文错误文案同步；当前 Web 关键词入口不增加结构化表单。

新引用准入：新任务选种、现有任务重新选种及新下载经 indexer.ResolveReference/FetchCurrentReference 核对当前启用状态、账号/网络配置修订和内置规则 digest；同一规则未改而其他定义更新，不使其旧结果自动失效。Jackett 本地策略验证不刷新远端目录，连接未改时冷启动目录不阻断已保存结果。文件预览在命中检查缓存前同样验证来源，并直接使用借用记录，候选随后过期不撤销本次获取。已接收任务及自动寻源中的 FetchRecord/FetchReference 继续使用其冻结快照，不被之后的停用、账号改动或短期候选过期撤销；已有附件及下载句柄也不重新抓站。回归见[当前来源准入](../../internal/indexer/current_reference_test.go)与[预览缓存边界](../../internal/taskruntime/current_reference_test.go)，完整验证进度见 R-004。

显式资源下载基础：应用持有 [indexerdownloads.Service](../../internal/indexerdownloads/service.go)，通过当前有效引用获取服务取得验证后的原始数据，短期内存句柄只代表已获取附件，不代表任务决定。5 分钟绝对到期，64 MiB/256 项；获取前预留 8 MiB，最多 8 个并发获取、16 个读取。读取持有不可变 reader 和释放 lease，过期的活跃读取继续占预算。HTTP GET/HEAD 只返回已保存 torrent 或 magnet 附件，禁止缓存及来源泄漏，不重新抓站点、不创建任务。规则与任务 pin 不依赖这些句柄，重启后句柄失效。prepareTorrentDownload 只接受 searchID/candidateID，经持久 run 归属/过期检查后准备；发现和任务候选均显式获取下载入口，刷新不重复获取，失败不影响选源。磁力附件仅从本站句柄读取后展示原生 magnet 链接；不将私有定位信息加入候选查询。

现有任务选源：resolveTaskSourcingCandidate 只接受任务、run、candidate 和 expectedRevision，run 必须属于该任务；taskruntime 重新核对番号、阶段、重选清理状态和提交不确定性。来源快照、任务更新、审计及规则 pin 在同一 core v6 事务保存，已有不确定 TASK_ADD 不允许被新选择覆盖。失败保留旧来源与引用；成功由原 worker 获取、验证并提交，响应 PENDING 仅表示已接收。任务关联搜索默认复用尚未过期的同范围 run，显式 refresh 才新建。验证见 [现有任务选源测试](../../internal/taskruntime/resolve_indexed_test.go)及 [API 归属测试](../../internal/graphqlapi/torrent_resolve_test.go)。

2026-11-02 中性预览与发现页：previewTorrentSelection 仅接受 run/候选 ID 和规则开关，不接收浏览器候选元数据；一致读取终态候选，交 PreviewTorrentSelectionContext 执行现有规则，然后投影原始公开字段、排序分页，完整集先执行 LOCK。检查预算来自现行任务规则，非法分页在任何检查前拒绝；返回真实成功检查数及可检查数，失效 run 不复活。发现页已接持久搜索、逐站状态、显式预览和 taskflow 候选入队；原始下载/磁力入口通过服务器短期句柄提供，不公开私有来源定位 URL。实际规则回归见 [选择器测试](../../internal/taskruntime/candidate_selector_test.go)、[预览归属测试](../../internal/torrentsearch/selection_test.go)和 [GraphQL 测试](../../internal/graphqlapi/torrent_preview_test.go)。

2026-11-02 持久交互搜索：新增独立领域 [torrentsearch](../../internal/torrentsearch/service.go)，复用 indexer 的单份冻结来源和逐站执行，不创建通用作业框架。先写 runtime v5 run 意图，应用两 worker 执行；HTTP 结束不取消已接收搜索。持久候选只含脱敏展示字段，查询事务一致地读取 revision、逐站状态与有界排序分页；取消先落盘并保留已产生结果，重启标记中断且不重跑。逐站取消/中断/未执行具有终态。保留 30 分钟，最多 50 run、每 run 1000、全局 10000 候选、查询/站点/候选合计 64 MiB，订阅最多 128 且慢客户端不阻塞关闭。startTorrentSearch、torrentSearch、cancelTorrentSearch、torrentSearchEvents、queueTorrentCandidate 已接 GraphQL 和应用；入队仅通过服务端 searchID/candidateID 归属解析到 taskflow，来源冻结转 core 后不依赖 run TTL。发现页、规则预览和阻塞任务页面和显式下载入口已迁移。

2026-11-02 统一寻源已接短期精确查询缓存与相同在途查询合并；缓存只返回候选深副本，当前任务接受/身份/资源规则/种子与文件验证仍由原执行路径决定。每次接受流程重新注册独立资源引用，已选原始数据继续先写核心再提交下载；查询缓存不是已接收任务的恢复来源。强制刷新不复活旧结果，认证/挑战失败使对应实例失效；等待者取消独立于共享网络工作，应用关闭取消并等待。持久搜索和手动候选选择已接线，剩余验收见 [R-004](../TODO.md#r-004)。

2026-11-02 ArtifactCache 已接 indexer.Service 的 Fetch/FetchReference/FetchRecord 及应用关闭；同一冻结来源/定位仅合并一次获取，忽略每次重新注册的 opaque ID，回传各调用者自己的引用和字节副本。成功 torrent 经共享 torrentmeta 解析，按来源、实际 info hash 及完整原始字节摘要去重，不能交换同 hash 中不同 announce/passkey；magnet 按完整内容隔离。2026-11-03 内置与外部资源统一用 [ParseMagnet](../../internal/torrentmeta/magnet.go) 校验有界 URI 的真实 btih（hex/base32）或 btmh SHA-256 身份，保留全部名称/tracker/xt 参数；命中缓存保留已解析哈希，非法 topic 不作成功缓存。v2/混合链接仅证明 URI 校验与缓存，未据此宣称 v2 种子文件解析及完整下载验收。成功数据 10 分钟绝对过期，64 MiB 载荷、512 定位项、64 在途/等待预算；不缓存错误或无效 body、不增加重试。最后等待者取消即取消网络工作，但旧请求未退出前拒绝重叠抓取；已知成功迟到数据可复用，关闭/清理 fence 不回填。共享工作归应用 context，关闭取消并等待。健康只在实际获取时记录一次，命中不改 AcquiredAt；清搜索缓存不删已获取原始数据。原 taskruntime 解析器抽到纯包 [torrentmeta](../../internal/torrentmeta/metainfo.go)，校验语义及原文件检查/身份流程保留。验证见 [获取协调器测试](../../internal/indexer/artifact_cache_test.go)。

2026-11-02 手动候选持久接收底座：taskflow.QueueTorrentCandidate→QueueIndexedResourceContext 只接受服务器资源 ID，经租约取得完整配置/定位快照，与任务、来源审计和规则 pin 同一 core 事务接收，再由既有寻源 worker 获取。EventingTaskStore 转发持久接收并只在提交后发布事件；旧 payload store 不承诺此能力。core v6 保存未获取前的私密来源，获取成功先以原始 torrent/精确 magnet 替换并解除规则 pin，再验证身份和提交；失败保留原选择并阻塞，不转为自动改搜，显式重试复用已获取字节。任务删除/人工明确替换清理来源与 pin。公开任务投影/事件不携带来源配置、原始附件或索引器私有下载位置，仍显示非敏感来源审计和 opaque 引用；显式手动 URL 行为保留。已验证真实应用组装的 taskflow/事件包装/worker 路径及重启、候选过期、账号修改、失败重试和 v5→v6 保留。公共 searchID/candidateID 入队 API 和发现页已接，任务页面迁移仍未完成。

内置索引实施状态（2026-11-02，工作区已在 cmd 组装，未部署）：WithIndexers 接统一逐站搜索和服务器资源 ID 取种，SelectionCandidate 投影保留原排序语义。文件检查与 prepare 共享解析过的原始种子，检查缓存最多 256 条/32 MiB；提交 qB 前保存现有核心附件与来源审计，恢复原始字节不依赖临时候选或再联网。旧 Trackers 请求范围在迁移边界转换为明确的 Jackett 实例 ID；中性 Query 保持显式命名。网络/缓存设置已接线，搜索 API 已接，界面迁移仍见 [R-004](../TODO.md#r-004)，不能据此宣称完整功能上线。回归见 [索引任务测试](../../internal/taskruntime/indexer_resources_test.go)、[选择持久化测试](../../internal/taskruntime/selected_resource_test.go)与[应用组装测试](../../cmd/moji/indexers_runtime_test.go)。

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前实现。

主要源码：[taskflow](../../internal/taskflow/service.go)、[runtime](../../internal/taskruntime/service.go)、[持久寻源队列](../../internal/taskruntime/sourcing_queue.go)、[阶段定义](../../internal/taskruntime/task_stage.go)、[工作台读取](../../internal/taskruntime/task_workspace.go)、[事件](../../internal/taskruntime/task_events.go)。

## 创建与执行

| 用户意图 | taskflow 入口 | 来源 |
| --- | --- | --- |
| 手动 magnet 或 torrent URL | `CreateFromManualTorrent` | `MANUAL` |
| 手动番号 | `CreateFromSearchCode` | `MANUAL` |
| 发现页场景 | `QueueDiscoveredScene`，服务器解析引用 | `SEARCH` |
| 演员所选场景 | `QueueResolvedScene`，领域层已验证归属 | `SEARCH` |
| 自动发行或人工确认下载 | `QueueSubscriptionRelease` | `SUBSCRIPTION` |

自动番号任务先写入 `SOURCING/PENDING` 和请求参数，再由两个寻源 worker 执行；HTTP 返回不等待搜索、元数据获取或 qB 提交。手动 torrent 仍有自己的创建路径，不能声称所有入口都是同一种异步队列。

番号、info hash、magnet 身份去重归 runtime 和核心库约束。调用者可以预检，但不能复制或绕过最终校验。新增生产入口走 taskflow；GraphQL 直接读 runtime 或执行重试、选文件等生命周期动作是允许的。

阶段为 `SOURCING → DOWNLOADING → PENDING_INGEST → TRANSFERRING → SCANNING → COMPLETED`，配合 `PENDING/RUNNING/BLOCKED/DONE`。`SCANNING` 下还包含 generate、identify、verify 子阶段。页面要展示用户下一步，不能仅根据 qB 100% 判定整个任务完成。

## 寻源契约

- 匹配支持大小写、分隔符和数值零填充等别名；冲突番号不能因为排名高而被选中。
- HTTP torrent 由 Moji 下载，最多 8 MiB；检查名称、视频路径和身份后，把验证过的原始字节交给 qB，保存精确 hash。纯 magnet 先验证标题/display name，随后通过元数据文件计划检查内容。
- 已选候选、原始 torrent 和请求选项持久化；重启继续核对，不在每次恢复时重新选择不同资源。
- 寻源请求 JSON 由 [版本化 codec](../../internal/taskruntime/sourcing_request.go) 统一读写，64 KiB 上限。新写入 version 1 使用明确的实例 ID；无版本旧格式中的所有 Trackers 都是 Jackett ID，包括 UUID 形状的字符串。未知版本或非法新身份在搜索前阻塞并保留原始请求；新入队、候选接收/替换及资源跟踪共用该边界。
- 暂无资源属于正常等待。默认每六小时重查，设置范围 1–168 小时；可立即检查或手动补资源。临时上游故障也可安排重查；无效请求、身份重复等需要用户处理的阻塞不盲目循环。
- 元数据请求按主机排队，Jackett 下载按 indexer 分组；限流和退避见 [运维](operations.md)。不要绕过队列另发高频请求。

文件计划、下载选择、交付和 Stash 完成判据见 [媒体](media.md)。重要选择和外部意图先提交，进度观测不引起核心任务反复写入；CAS revision 与 generation 防止旧 worker 覆盖新状态。

## 分页和批量

当前 UI 使用 `TaskSummary` 分页，默认 24、可选 12/24/48，API 上限 100；完整文件、候选和检查点只在打开详情时读取。摘要与详情是不同缓存实体，不能用摘要覆盖完整诊断或文件选择。

默认创建时间排序走 SQL 分页。名称/搜索、进度或更新时间排序仍可能扫描轻量 headers 并叠加实时观测；这与全量反序列化任务不同，也不是常量成本。后台工作用 128 项 keyset 批次；Home 使用聚合和有限样本。

跨页选择独立保存最多 100 个 ID；已选项另行读取，不能按当前页裁剪。处理台在开始时获取注意任务会话，按最早等待顺序每次打开一个详情；新增任务留给下个会话。会话 ID 集合随待处理数增长。

## 实时一致性

快照携带 `streamId/sequence`；`READY` 与注册边界绑定，进程重启更换 stream。提交、快照边界和发布需要保持一致；总线没有持久 replay。

前端在批处理前检查序列缺口，按任务合并 100ms 内更新；有界的 512 任务传输日志保护晚到 HTTP 响应。revision 变化只刷新当前详情，纯进度更新不重载冷数据。溢出时发送恢复边界，挂载中的页面/选择/处理台/详情各自恢复其有界快照；全局 bridge 不拉整个任务历史。

普通进度更新不让卡片跳位。成员变化在合适时机刷新；用户正在选择、打开详情或使用实时排序时提供明确刷新动作。重试、删除和选资源的确认结果也要在 socket 不可用时可见。

完整协议及验证范围见 [任务工作台](../../docs/task-workspace.md)。[早期任务创建说明](../../docs/task-creation.md) 中的 schema-13、永不重试阻塞等描述是旧阶段信息，以当前队列与 [媒体说明](../../docs/media-delivery.md) 为准，清理项见 TODO。

## 修改时验证

- 创建或去重：taskflow/runtime 领域与持久化测试，以及对应 GraphQL 契约。
- 队列/重启：应用 context、恢复参数、先持久化后副作用、重复提交和 generation fence。
- 工作台/事件：[后端工作台测试](../../internal/taskruntime/task_workspace_test.go)、[序列恢复测试](../../web/src/graphql/taskEventTransport.test.ts)、[浏览器工作台](../../web/tests/visual/task-workspace.spec.ts)。
- 不把已知的 legacy 全量查询兼容接口误认为当前 UI 的主读取路径；规模优化应针对真实调用方测量。

## 已记录的后续需求

资源规则及逐站寻源已按用户确认重构，见 [设置复核](../reviews/2026-10-28-settings.md)。LOCK 为命中锁定：标题或文件条件任意命中（OR）后只在命中的合格候选中按其他规则选择，无命中正常回退；EXCLUDE 命中即淘汰且优先于锁定。PREFER/DEMOTE 是软排序；关闭规则保留身份/去重。文件锁定只认成功检查的命中，未检查/失败不阻止正常回退；文件排除仍拒绝失败或未检查候选。代码、持久化和 API 统一使用 LOCK。自动寻源逐站搜索到通过身份、规则、元数据及去重的资源才停止；手动检索顺序一致但收集所有站结果。

[手动排除记忆](../TODO.md#r-003) 尚待深入讨论，不因本轮硬排除设置而自动产生记忆。
