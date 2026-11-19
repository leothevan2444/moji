# 任务、寻源与实时工作台

核对日期：2026-11-19；代码基线：`ec7b093`；状态：当前源码契约，不表示运行实例已更新。

主要源码：[taskflow](../../internal/taskflow/service.go)、[影片确认](../../internal/taskflow/films.go)、[影片身份](../../internal/film/film.go)、[runtime](../../internal/taskruntime/service.go)、[寻源队列](../../internal/taskruntime/sourcing_queue.go)、[工作台](../../internal/taskruntime/task_workspace.go)、[事件](../../internal/taskruntime/task_events.go)。

## 创建与身份

所有生产任务先通过 taskflow 确认 StashBox 来源、影片 ID 和影片快照，以完整番号额外防重。`film.Code` 只统一大小写和首尾空白；不截断前缀、来源后缀或前导零。文件匹配使用已知目标的完整番号，不从名称猜测任务身份。只有 taskflow 的资源推荐可以从公开资源名提取番号；私有 URL 和 tracker 凭据不能成为外发搜索关键词。

| 入口 | 契约 |
| --- | --- |
| Web `createFilmTask` | `ResolveFilm` 读取当前影片，`ConfirmFilm` 重验来源、ID、摘要和发行日期；浏览器不提交影片真值 |
| 发现页影片、演员所选影片 | `QueueDiscoveredScene` / `QueueResolvedScene`，服务器解析并确认归属 |
| 订阅、发行复核、待发行移交 | `QueueSubscriptionRelease`，保留订阅领域的决定与授权边界 |

自动任务以 `AUTO` 进入 `SOURCING/PENDING`，先持久化意图，由两个寻源 worker 执行。指定资源以 `USER_SELECTED` 保留来源或原始附件，进入下载待处理；失败不自动换源。原 `addTorrent`、`qbittorrentAdd`、`downloadMedia`、`queueTorrentCandidate` API 不存在。磁力和种子链接仍可作为确认影片后的指定资源；已有任务也可人工补充资源。用户流程见[创建下载任务](../../docs/task-creation.md)。

新任务不能改变已有任务的影片身份。番号、影片关联和种子身份最终校验归 runtime 与核心库；调用方预检不能代替提交校验。GraphQL 可以直接读取和操作已有任务，但不能绕过 taskflow 创建。

## 执行与恢复

阶段为 `SOURCING → DOWNLOADING → PENDING_INGEST → TRANSFERRING → SCANNING → COMPLETED`，配合 `PENDING/RUNNING/BLOCKED/DONE`；SCANNING 包含生成、识别与验证。下载 100% 不表示入库完成。

任务推进随应用启动并持续运行，无全局自动处理开关；非正检查间隔使用默认一分钟。订阅检查有独立开关。重要选择、附件、路径和外部意图先提交核心库，再执行副作用；revision/CAS 和 generation fence 拒绝旧 worker 写入。应用关闭取消并等待 worker。文件计划及完成判据见[媒体](media.md)。

暂时没有资源、搜索不完整、站点请求/冷却和自动取种失败进入等待资源，默认六小时重查（配置 1–168 小时），可立即检查或人工提供资源。人工已选资源获取失败保留冻结来源与规则 pin，重试同一资源。身份不符、重复、库核验失败、无效请求和诊断存储不可用等阻塞不能盲目循环。删除任务停止其追踪。

寻源请求由[版本化 codec](../../internal/taskruntime/sourcing_request.go)统一读写，上限 64 KiB。version 1 使用明确实例 ID；无版本旧请求的 Trackers 都是 Jackett ID，包括 UUID 形状。未知版本或非法身份在搜索前拒绝，保留原请求。

## 资源选择

自动寻源按站点顺序搜索，只有资源通过身份、规则、元数据及去重才停止；手动搜索使用同一顺序但收集全部站点。新引用准入、缓存与交互搜索的生命周期归[索引器](indexers.md)。

`resolveTaskSourcingCandidate` 只接受任务/run/candidate ID 和 expectedRevision，要求 run 属于该任务。来源快照、任务更新、审计和规则 pin 在同一核心事务保存；不确定的 TASK_ADD 不能被新选择覆盖。失败保留旧来源，成功仅表示已接收，原 worker 继续获取和提交。默认复用未过期的同范围 run，明确 refresh 才新建。

资源规则采用 LOCK / EXCLUDE / PREFER / DEMOTE。任意锁定条件命中后只在命中的合格候选中排序，无命中正常回退；排除优先，锁定不绕过身份和去重。文件锁定只认成功检查的命中，未检查/失败不阻止回退；文件排除拒绝未检查/失败的候选。软效果调整排序，单视频是软偏好；关闭规则仍保留身份和去重。人工明确选资源不强制套用自动偏好。[手动排除记忆](../TODO.md#r-003)尚未设计。

## 分页、Home 与实时一致性

TaskSummary 默认每页 24、可选 12/24/48，API 上限 100；完整文件、候选和检查点按需读，摘要与详情是不同缓存实体。默认创建时间走 SQL 分页；名称/搜索、进度和更新时间排序仍可能扫描轻量 headers，这不是常量成本。后台按 128 项 keyset 工作。

Home 使用聚合及有限样本；[HomeStats](../../internal/taskruntime/home_stats.go)由首页与 Widget 共用，Widget 只执行聚合，不加载详情或联网，不支持统计的 store 不回退全量 List。四项口径与界面见[主页](../../docs/home-workspace.md)。

跨页选择独立保存最多 100 个 ID，不按当前页裁剪。处理台开始时取得注意任务会话，按最早等待顺序逐项处理；新增任务留给下轮，会话 ID 集合随待处理数增长。

快照携带 streamId/sequence，READY 与注册边界绑定，进程重启更换 stream；总线不提供持久 replay。前端按任务合并 100ms 更新，以有界 512 任务传输日志保护迟到 HTTP 响应；revision 变化刷新当前详情，纯进度不重载冷数据。缺口/溢出由挂载消费者各自恢复有界快照，全局 bridge 不拉整个历史。普通进度不让卡片跳位，成员变化或实时排序在用户操作期间提供刷新入口。完整协议见[工作台](../../docs/task-workspace.md)。

## 删除与重新选种

删除确认的一次性 `deleteStashScene` 默认 false，不保存为偏好。无持久 scene ID 时禁用并用原生 title 说明；资格不依赖任务阶段或识别成功。Stash 删除先记录 external_commands，再按精确 ID 和任务交付目录核对文件归属，经原生 sceneDestroy 删除影片、funscript 和生成资源。逐个确认消失，超时/部分失败保留任务，重试先核对已消失 ID。Stash 步骤先于 qB 策略与任务删除，不承诺跨服务回滚。保护细节见[媒体交付](../../docs/media-delivery.md#optional-stash-deletion-when-deleting-a-task)。

重新选种保留任务与影片身份，先清理旧种子、交付与 Stash 产物；全部成功后清空执行检查点并回到 `SOURCING/PENDING`，唤醒自动寻源。清理失败保持受阻；不排除旧资源，仍可能再次选中。没有旧状态自动迁移。

## 修改时验证

创建与去重覆盖 taskflow/runtime/GraphQL 和持久化；选源覆盖 run 归属、来源准入、提交不确定及重启。媒体与外部删除覆盖持久意图、部分成功和响应丢失。工作台覆盖[后端](../../internal/taskruntime/task_workspace_test.go)、[序列恢复](../../web/src/graphql/taskEventTransport.test.ts)和[浏览器](../../web/tests/visual/task-workspace.spec.ts)。影片重构的历史人工转换证据见[复核](../reviews/2026-11-15-film-task-creation.md)，不作为再次迁移的授权。
