# 任务、寻源与实时工作台

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

用户要求 [重新审核并规划重构自动选种](../TODO.md#r-002)，与 [Jackett 逐站查询](../TODO.md#r-001) 联合确定停止/回退边界；[手动排除记忆](../TODO.md#r-003) 尚待深入讨论。需求状态以 TODO 为准，不能把这些目标当作当前已生效规则。
