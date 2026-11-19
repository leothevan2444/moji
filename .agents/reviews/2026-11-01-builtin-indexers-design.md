# 内置索引设计依据（2026-11-01）

历史快照：设计基线 `502cfa9`；实施与本地验收于 2026-11-04 完成。2026-11-19 治理时移除已失效的实施步骤和中途状态，保留下列固定上游来源与已采用边界；完整原方案由 Git 保存。当前职责、配置和预算见[索引器模块](../modules/indexers.md)，不从本文推断实时站点可用性、部署状态或新的运行授权。

## 采用边界

- 单进程 Go Cardigann v11 声明式解释器，运行时下载固定规则包，不把定义打包到源码或镜像；保留可选 Jackett，不迁移 C# 专用索引器或任意脚本。
- 搜索与获取共享站点会话，实例/账号/规则快照冻结，任务重要选择写核心；查询缓存和临时候选不作任务恢复真值。
- 自动寻源沿用 Moji 的逐站验证后停止，人工检索收集全站结果；解释器兼容、站点访问、资源验证与完整下载分别验收。
- 规则安装显式、更新可检查后人工应用，持久 pin 与 lease 保护已接收任务；原子激活与回滚不复制覆盖活动目录。
- 仅 infohash 的公开结果沿用公开 tracker，私有站点不做公开 magnet fallback；内置未来发布时间按同次检索时间限制，模板保留原字段。
- 用户要求所有内置站点共用代理和 FlareSolverr；旧网络档案、逐站网络选择和重复挑战超时已退出当前契约。

## 当次上游来源

| 来源 | 调查快照 | 对设计的影响 |
| --- | --- | --- |
| Prowlarr/Indexers | 5f5db4e58d06be18619c3a858b9a91ade6f341ca | VERSIONS 声明 CURRENT_VERSION=11、MAX_VERSION=12；运行时检查 v11 目录更新，再按解析到的 commit 固定下载，不以浮动 master 直接覆盖活动规则 |
| Prowlarr/Prowlarr develop | 3326e2d4aa978f35fa7e44f66758ebfc8f553d67 | 请求生成、响应解析、下载、会话、规则更新分工；当前更新器常量已是 v12，不能把 develop 全部行为当 v11 定义 |
| Jackett/Jackett master | cfdc645711c487d05cb55932e394b2370154d81e | Cardigann 单独实现；索引器持有搜索和下载能力；精确查询缓存、配置变更失效、测试不读写缓存、失败退避 |

主要一手来源：

- [Indexers VERSIONS](https://github.com/Prowlarr/Indexers/blob/5f5db4e58d06be18619c3a858b9a91ade6f341ca/VERSIONS)、[v11 schema](https://github.com/Prowlarr/Indexers/blob/5f5db4e58d06be18619c3a858b9a91ade6f341ca/definitions/v11/schema.json)。
- [Prowlarr Cardigann 模型](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannDefinition.cs)、[模板与过滤器](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannBase.cs)、[响应解析](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannParser.cs)。
- [Prowlarr 请求、登录与下载解析](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannRequestGenerator.cs)、[HTTP 执行与下载](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/HttpIndexerBase.cs)、[规则更新](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/IndexerVersions/IndexerDefinitionUpdateService.cs)。
- [Prowlarr FlareSolverr](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/IndexerProxies/FlareSolverr/FlareSolverr.cs)、[会话持久化](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/IndexerStatusService.cs)。
- [Jackett Cardigann](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/Definitions/CardigannIndexer.cs)、[基础索引器](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/BaseIndexer.cs)、[查询缓存](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Services/CacheService.cs)。


## 实施证据

- [阶段/schema 矩阵与最终完成记录](2026-11-03-indexer-acceptance.md)
- [请求、语料差分和 Go CI 覆盖](2026-11-03-cardigann-requests.md)
- [后续统一执行记录](2026-11-08-indexer-executions.md)

这些记录保留各自日期和验证局限，已完成方案不再作为当前待执行计划。
