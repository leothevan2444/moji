# 内置索引与 Cardigann v11 实施方案

日期：2026-11-01；Moji 基线：502cfa9。

状态：实施中，当前进展与未完项见 [R-004](../TODO.md#r-004)。用户已授权按本文完成内置索引功能，确认完整 v11 声明式能力、完整寻源下载闭环、对照 Jackett 修正差异及高可靠性目标；2026-11-01 进一步确认规则库运行时下载和独立更新，以及 Home 新增与下载器平级的索引器区域、移除固定 qBittorrent/Jackett 状态卡。基础引擎、规则管理与站点管理已接应用生命周期、GraphQL 及站点/规则库/外部 Jackett 设置界面；网络/缓存与 Home 进展见下段，搜索 API、发现/任务候选和显式下载入口已迁移；规则更新策略、人工挑战及 solver 配置已接 API、界面和运行生命周期。完整语义对照与真实站点/代理/solver 验收仍待完成，不表示已上线；未单独确认的产品默认值仍按文末列出的取舍处理。

约束：[工程规范](../standards/engineering.md)、[产品体验规范](../standards/product.md)、[架构地图](../architecture.md)、[任务](../modules/tasks.md)、[存储](../modules/storage.md)、[Web](../modules/web.md)、[开发验证](../workflows/development.md)。

2026-11-03 用户确认：规则仅给出 infohash/title、需自动生成 magnet 时，沿用 Jackett 附加公开 tracker 的默认行为。站点直接提供的完整 magnet 保留原文；私有站点仅 hash 的结果不自动创建公开 magnet fallback。该选择已授权实施，不再列为待确认。

2026-11-04 用户确认：未来发布时间沿用 Jackett，限制为本次检索时间。该宿主策略在内置适配层实施，保留规则字段供取种模板使用；回归和冻结双架构验收见[证据](../reviews/2026-11-03-indexer-acceptance.md#e77e65f-冻结双架构验收2026-11-04)。

2026-11-02：Home 的独立概况查询、索引器/下载器同级布局与固定卡片移除、配置/目录/搜索完成通知及局部订阅恢复已接线。运行观测按配置/规则隔离持久保存，管理及 Home 显示上次搜索/取种结果；第四面板“网络与缓存”已接缓存策略、状态、清理、代理网络配置管理及请求/搜索超时编辑，保存具版本冲突保护。本地跨场景健康验收已通过，本地规则失败作业入口验收已完成，真实站点验收仍未完成。

站点分阶段测试已接 testIndexer GraphQL、实际应用装配以及站点列表/当前编辑草稿界面；认证默认执行，搜索与获取资源显式选择，取消/作用域变化隔离旧结果。配置、认证、搜索和获取分别报告，Jackett 认证不作独立成功推断，未执行搜索不显示零结果；测试不保存配置或创建任务。保存站点的有界登录生命周期和 begin/answer/cancel GraphQL 已接线，登录挑战界面已接开始、恢复、回答、取消和到期交互，solver 认证/搜索执行已接，配置 API/UI 已接，真实服务对照仍待完成，完整验收以 R-004 为准。

2026-11-03 solver 接入前复核：[FlareSolverr 官方协议](https://github.com/FlareSolverr/FlareSolverr/blob/master/README.md) 当次读取的 README blob 为 ed5431ad3e303f9ee100f60b2b1f7468dc265956；调用 `/v1`，返回 Cookie 与 User-Agent，后续请求需要匹配 UA。文档区分 sessions.create 的代理凭据支持与 request.get 单次代理的认证限制，接入时需按实际版本验证，不假定凭据可直接透传。上方固定 Prowlarr 实现检测挑战后调用 solver、注入 Cookie/UA 并重发原请求；Moji 的取种请求可能消耗配额，不能直接照搬重放策略。后续实现须保留剩余时间预算、匹配代理出口、账号会话隔离与可选依赖边界，完整验收仍见 R-004。

2026-11-03 协议落地补充：[当前 service 请求路径](https://github.com/FlareSolverr/FlareSolverr/blob/master/src/flaresolverr_service.py)（blob e189178636668e2f01f026bfa270ded648ea3673）和[代理初始化](https://github.com/FlareSolverr/FlareSolverr/blob/master/src/utils.py)（blob 37640ac2615da178cbe806d4aca3a41459547ef2）显示单次请求与显式会话共用 get_webdriver(proxy)。因此客户端按 Prowlarr 的单次请求方式传递认证代理，两个认证键同时保留，不引入远端持久浏览器生命周期；浏览器清理由 solver 的临时请求 finally 处理，真实服务/版本仍需后续验证。当前已实现纯 Go 协议、端点预算与受控网络封装，站点挑战自动处理及设置 UI 尚未接线。

2026-11-03 solver 执行接线：明确挑战仅允许认证/搜索求解一次；发送 GET 导航和同账号 Cookie/代理，不发送认证 POST。原请求以采纳的 Cookie/UA 再验证一次；取种及 before 页面不委托/重放。同 origin 与 Cookie 批量验证后采纳，持久会话恢复保留 UA 配对；自身 CF 503 的序号围栏不能清除更新的失败或 429。求解继承父级剩余截止而非普通请求短超时。外部浏览器为显式信任服务，初始 DNS/最终来源校验不能约束其内部全部跳转；配置 API/UI 已接，真实 solver/站点验收待完成。

2026-11-03 solver 配置接线：网络 create/update 统一稀疏输入，支持 solver-only/代理/二者组合；FlareSolverr URL 读侧不回显，空输入保留、清除显式，配置冲突不覆盖草稿。挑战超时默认 90 秒、范围 10–300 秒，沿用自动保存及确认重读；父级剩余预算仍优先。网络列表显示已配置浏览器验证，编辑器及中英帮助说明服务根地址、委托凭据、适用场景与保存不测试连接。未调用真实服务、未改实际配置。

2026-11-03 可信来源接线：复用现有站点字段与管理/草稿测试 API，补高级编辑、完整 origin 校验、16 项限制与空列表撤销；信任不授予内网。回归先复现并修复 net/http CookieJar 在 Header 去除后仍向未信任同父域子站发送域 Cookie、接收其覆盖账号 Cookie 的问题；Jar 的发送/接收都按明确来源授权，显式信任仍按 Cookie 域规则。实际管理 API、配置原子拒绝、浏览器保存/刷新/撤销和未保存草稿测试已覆盖；真实站点验收仍待完成。

2026-11-03 持久冷却接线：新增 runtime v6 和生产 NewPersistentNetwork，按来源/代理出口的私有摘要恢复 gate 冷却与指数失败次数，保留跨账号共享限流。懒读失败禁止联网；保存失败保留已收到资源/内存退避并报告，dirty 状态在后续请求前补存。8192 条上限不驱逐有效冷却，回收过期项；CF 自身解除同步删除持久记录，内置 HTTPSession 的 429 报告实际 gate 截止，外部 Jackett 首报截止对齐已完成。runtime 可重建或写入失败时不作强持久恢复承诺。实际外部站点验收仍未完成；本地跨场景健康证据见下段。

2026-11-03 跨场景健康：Jackett 搜索与目录 API 的首次 429 使用实际 gate 截止，和冷却准入/持久 health/逐站报告一致；网关 HTTP 200 内远端单站失败不误升为网关限流。真实应用装配/本地假服务验证搜索和取种互不覆盖、Home 无隐式联网、目录失联保留确认数及账号隔离；内置账号/活动规则版本切换读对应命名空间，旧观测保留。结合已有缓存命中/取消/迟到结果/重开库测试完成本地健康矩阵；真实站点可用性仍需显式验收。

2026-11-03 Home 规则作业：新增当前作业窄只读接口并接现有概况，只展示最新失败/中断，局部读取错误不丢其他状态；来源检查/下载不由 Home 读取触发。精简规则订阅只用于失效通知，首次读取/刷新/重连从持久快照恢复，链接规则库处理。临时库重开、API 投影、局部重试/恢复、无轮询/无更新动作、真实 WS 断开重连和中英/RTL/移动布局已验证。

## 1. 目标、范围与兼容承诺

实现单进程 Go Cardigann 引擎，运行时从 Prowlarr/Indexers 下载 definitions/v11、按不可变快照安装；规则文件不打包进 Moji 源码/镜像。完整完成搜索、按站点会话获取 torrent/magnet、Moji 筛选验证、持久选择、qB 提交和恢复。内置与外部 Jackett 能同时配置，Jackett 不再是启动前提。

不迁移 C# 专用站点，不增加通用脚本执行器，不建设独立爬虫服务、全网资源数据库或完整 Prowlarr 管理产品。暂不移除 Jackett 适配器。

引擎目标是整个 v11 声明式语义。按代表性规则渐进验收，不设置永久站点白名单。区分四项证据：规则能加载、执行语义通过样本验证、站点实时可访问、种子能通过 Moji 校验并提交。不能用全部 YAML 加载成功宣称全部站点实测通过。

引擎内部建模 v11 caps 支持的 basic/movie/tv/music/book 查询参数；Moji 首版用户流程只暴露现有关键词/番号与分类，不新增无实际产品需求的音乐、图书界面。只有特定查询模式的定义应显示实际能力，不伪装支持关键词搜索。

## 2. 上游调查与采用的边界

本轮阅读以下源码；HEAD 记录供复核，实施前下载固定快照并重新核对。本文未批量执行全部定义，也未操作真实站点、Jackett 或 qB。

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

采用职责拆分、搜索/下载共用站点会话、明确查询能力、精确缓存与失败退避；沿用 Moji 的逐站验证后停止，不照搬上游面向聚合工具的并发全站搜索策略。规则更新采用固定包与原子切换，不直接复制覆盖活动规则目录。

需要特别验证的兼容点：

1. fields 的顺序、重复键及 append/noappend 修饰有执行含义。保留 yaml.Node 到有序 Field 列表，不能直接解到 Go map；允许重复键的范围按兼容契约限定，其他配置重复键报错。
2. 模板不是原生 Go text/template 的直接同义物；例如 .Config.filter-id、.Result 中间字段、真假值和 URL 转义。建立受限词法/语法与求值器，不以字符串替换拼凑完整兼容，也不暴露 Go 方法调用。
3. 上游 .NET 正则、AngleSharp CSS、JSON 选择器和日期语义需要兼容层。Go regexp 不支持的构造不能静默跳过；库选择由全库特征扫描与兼容样例确定，必要时选带执行超时的 .NET 风格正则库。日期解析同时验证格式、时区、相对时间，不能直接把规则日期格式传给 Go time.Parse。
4. schema 与执行器可能不完全一致。保留差异表；按固定规则快照和可验证执行行为裁决，不能为了通过 schema 擅自改写社区 YAML。

## 3. 命名与包组织

领域统一叫 indexer，规则解释器叫 cardigann，pkg/jackett 保留为外部协议客户端。2026-11-04 按实际消费者收敛：生产任务/发现/搜索使用中性 indexer；internal/tracker 仅保留第 4 节要求的既有 REST 调试兼容桥，以及无 Manager 的测试/兼容构造器，不再作为生产索引引擎。不能为删除包名破坏仍明确保留的接口；后续删除这些实际消费者时再移除桥。

    internal/indexer/
      types.go                 站点实例、查询、候选、来源引用、报告
      service.go               已配置来源注册、实际逐站执行、批次回调
      catalog.go               可用定义、能力与兼容诊断
      builtin.go               Cardigann 适配器
      jackett.go               pkg/jackett 适配器与类型转换
      transport.go             站点请求、共享节流、退避、代理、重定向
      sessions.go              按实例隔离的会话、登录同步
      cache.go                 短期查询/资源缓存、请求合并
      storage.go               Manager 注入的 runtime repository
      errors.go                稳定代码与脱敏诊断
    internal/cardigann/
      definition.go            有序规则模型和编译后不可变计划
      loader.go                YAML/schema/能力检查与诊断位置
      template.go              兼容模板解释
      filters.go               注册式过滤器
      selectors.go             HTML/JSON 选择语义
      search.go                搜索请求与结果字段执行
      login.go                 声明式登录与挑战流程
      download.go              前置请求、下载选择器、infohash
      testdata/                合成响应、脱敏样本、预期结果
    internal/indexercategory/ 标准 Torznab 类别身份；适配器与解释器共享
    internal/indexerdefs/
      service.go               安装/检查/更新/激活/回滚编排
      source.go                上游版本发现与固定 commit 下载
      bundle.go                暂存、受限解包、manifest 与摘要
      validation.go            与 cardigann 编译器衔接、实例设置差异
      activation.go            活动修订、重启核对、引用固定与回滚
      storage.go               Manager 注入的核心记录与更新作业
    internal/torrentsearch/
      service.go               专属交互搜索生命周期、应用 worker 与通知
      store.go                 Manager.Runtime 持久接收、批次和逐站报告
      read.go                  一致快照、分页/排序、候选归属与取消终态
      types.go                 有界领域输入与脱敏输出，不依赖 GraphQL 模型
    pkg/jackett/               仅 Jackett HTTP/Torznab 协议
    pkg/flaresolverr/          可选 FlareSolverr HTTP 协议
    scripts/indexer-definitions/ CI 固定语料下载、特征扫描与差异报告；不打包生产规则
    cmd/moji-indexer-check/    固定语料静态编译与能力报告
    cmd/moji-indexer-replay/   无真实网络的脚本场景解释/取种重放
    scripts/indexer-diff/      固定 Jackett 构建、真实解释器差分和合成用例

这些是责任边界和建议文件名，不要求空建所有文件。cardigann 不依赖 Moji GraphQL、配置 store、SQLite、taskruntime；通过窄 Session/Transport 接口获取外部能力。indexer 不依赖 taskruntime；接受方回调控制是否停止。协议适配放 pkg，当前应用专用规则引擎先放 internal，不提前包装成插件 SDK。

cmd/moji 组装 indexer、indexerdefs 与适配器、注入 Manager 与应用 context、启动/等待搜索和规则更新 worker；resolver 只做输入和类型转换。cardigann 返回编译与兼容诊断，indexerdefs 不重复维护一套解释器；indexer 固定使用已激活的规则修订。taskflow 继续拥有全部新任务创建，taskruntime 继续拥有身份、筛选、选择、qB 副作用与恢复。

## 4. 抽象与原接口修改

需要一层统一索引领域抽象，已有两个真实消费者：内置 Cardigann 与外部 Jackett。分成单实例适配能力与应用编排，避免一个泛化 Provider 接口包含全部管理功能。

    type Adapter interface {
        Search(ctx context.Context, instance InstanceSnapshot, q Query) (Batch, error)
        Fetch(ctx context.Context, instance InstanceSnapshot, ref ResourceRef) (Artifact, error)
    }

    type Searcher interface {
        SearchSequential(ctx context.Context, q Query,
            accept func(context.Context, Batch) (bool, error)) (SearchReport, error)
    }

    type ResourceFetcher interface {
        Fetch(ctx context.Context, ref ResourceRef) (Artifact, error)
    }

Searcher/ResourceFetcher 在 taskruntime 消费边界定义；管理/目录/搜索 run 接口在 GraphQL 的对应消费边界定义。Adapter 是 indexer 内部注册边界，Test、Catalog、CRUD 不塞进它。所有调用显式传 ctx，移除依靠 SearchOption 中可选 ctx 的用法。

核心类型：

| 类型 | 契约 |
| --- | --- |
| Definition | 上游 definition ID、schema、规则摘要、设置描述、语言、隐私、查询能力、编译诊断 |
| Instance | Moji 稳定 ID、来源类型、definition/remote ID、用户名称、启用状态；名称和域名变化不改 ID |
| Query | Mode、Keywords、分类、结构化模式参数、实例范围、limit/offset、缓存策略；不根据影片规则擅改实际查询词。命名参数仅为已支持的结构化字段，大小写别名规范化、重复别名拒绝，不覆盖 typed/派生上下文；外部适配不支持的条件明确报告 |
| Release | 独立 Moji 类型；标题、分类、日期、大小、做种/连接数、原始 GUID、来源、详情与 ResourceRef；未知值用可空类型 |
| ResourceRef | version、instanceID、definitionDigest、实例配置修订、资源标识及服务器端解析数据；禁止包含可供 API 任意改写的请求头 |
| Artifact | TorrentBytes 或 Magnet 二选一、来源与获取时刻；原始字节保持不变，不信任上游声明的 hash |
| Batch/Report | 每站状态、候选、耗时、cacheHit、截断/部分响应标记、错误代码、correlation ID、下次允许时间 |

v11 的额外字段保留在引擎的类型化结果中；Moji Release 只投影真实消费者需要的字段。种子分辨身份仍归 runtime。定义写死的 seeders=1 等保留来源事实，不把它宣传为 Moji 实时探测值。

实例标识建议内置使用 UUID，外部使用 jackett/<远端稳定ID>；保留单一可选 Jackett 连接，首版不增加无需求的多服务连接管理。definition ID 不能当实例 ID，同一规则允许不同账号/镜像实例。远端站点只在 Jackett 管理；Moji 管理使用开关与搜索顺序，不通过管理 API 修改远端配置。

| 原实现 | 变更 |
| --- | --- |
| tracker.Tracker、SequentialSearcher | 迁移到窄 Searcher；内部手动收集使用同一逐站逻辑，不再维护另一套次序 |
| jackett.SearchResult/Indexer | 在 Jackett 适配器转换为 Release/Instance；runtime、发现与 GraphQL 不再依赖 Jackett 类型 |
| runtime inspectTorrent(string URL) | 改为按 Release/ResourceRef 获取 Artifact 的回调，文件检查与 prepare 共享该次获取 |
| runtime fetchTorrentMetadata(URL) | 手动 URL 路径保留；索引来源走 ResourceFetcher；共用基础请求节流能力，不执行两套叠加限流 |
| PreviewJackettSelectionContext | 中性命名 PreviewTorrentSelection，输入服务器保存的候选引用 |
| cmd/moji jackettClientOf、stats 全局 hook | 改为按实例接收领域状态；未配置 Jackett 不生成故障 |
| /api/tracker/search | 作为既有调试兼容入口转调新服务；正常产品不依赖它；不扩展 Torznab 对外服务 |

不同来源按统一 query_order 真正逐站请求。启用但未排序项按稳定实例顺序追加；显式范围仅查询所选项。单站空、拒绝、限流、故障继续下一站；接受完整合格候选后停止。Report 保留部分失败，禁止把“一站成功无结果、另一站失败”折叠成已确认全局无资源。

内置和 Jackett 同一站点可以并存，默认不双重启用，也不偷偷自动补一个外部回退。用户配置到顺序里的外部站点才会被执行。跨源相同 hash 在 runtime 最终去重，保留来源替代下载入口，不在验证前粗暴按标题合并。

## 5. 引擎功能清单

| 领域 | 必须覆盖的能力 |
| --- | --- |
| 规则加载 | v11 schema、元数据、links/legacylinks、replaces、encoding、证书语义、有序字段、默认值与诊断位置 |
| 模板 | .Config/.Query/.Result/.Categories/.Today、条件、逻辑、range、join、re_replace、空值/真假值、字段修饰和执行次序 |
| 过滤器 | schema FilterBlock 与 RowFilterBlock 的完整清单，参数类型、负索引、捕获组替换、URL/HTML 编解码、日期/相对时间、JSON 数组等 |
| HTML/JSON | CSS 与 JSON 扩展选择语义、optional/default/case/remove、预处理、行合并/日期头/count/after、无结果判定 |
| 搜索 | 多 paths、每 path 类别/输入继承、GET/POST、queryseparator、请求头、重定向、分类映射、模式能力、结果限制与去重 |
| 登录 | form/post/get/cookie/oneurl、隐藏字段与 CSRF、selectorinputs/getselectorinputs、错误选择器、登录验证及失效重登 |
| 挑战 | image/text 声明式验证码与上游实际行为差异；通用 simpleCaptcha 流程；可选 FlareSolverr；自动任务遇人工挑战报告需处理 |
| 资源获取 | 直接 torrent/magnet、HTTP 到 magnet 重定向、before/pathselector、下载 selectors、method/headers、infohash/title、Referer 与 Cookie 连续性 |
| 执行边界 | ctx 取消、请求/字节/页面/重定向/模板/正则预算、速率与冷却、每站报告、无秘密诊断 |

验证码是实际功能，不以“暂不支持私有站”绕开。若 Prowlarr 对 schema 中某种挑战本身不实现，记录与 Jackett 的差异，明确采用的兼容行为并提供样本；不宣称可以自动破解任意挑战。人工挑战答案仅在短期 challenge 上下文中使用，不作为永久密码保存。

## 6. 搜索到下载的完整链路

1. taskflow 接受意图，经现有 taskruntime 持久化 SOURCING 请求，worker 执行；不在 resolver 中创建旁路任务。
2. indexer 固定本次实例配置与规则快照，按顺序产生逐站 Batch。配置修订在下次操作生效，进行中的请求不混用新旧凭据或规则。
3. runtime 验证番号与标题证据，按现有快速规则排序；文件检查按原 inspection_candidate_limit 执行，不为兼容引擎擅自扩大预算。
4. 需要文件检查的候选通过 ResourceFetcher 取得 Artifact。torrent 最多 8 MiB，解析、hash、名称与视频路径验证继续使用现有 runtime 实现。
5. 文件检查与最终 prepare 使用同一原始字节和解析结果；不能在 inspect 后再次抓取不同文件。Artifact 没通过验证不能写入已验证缓存。
6. 原有 LOCK/EXCLUDE/PREFER/DEMOTE、无命中回退、文件失败语义、magnet 边界、身份去重保持。资源获取失败不自动改为 magnet 绕过文件排除。
7. 选择、原始 torrent/规范 magnet、来源和规则摘要、请求上下文先写核心库，再发 qB。重启凭核心已选资源恢复，不依赖查询缓存或重跑搜索。
8. qB 提交超时按既有身份与 qB 观测核对，不因结果不明立即选择另一个候选。单站候选均不合格时继续下一站。

需要在当前任务持久模型增补稳定来源 ID、版本化 ResourceRef/来源审计；实际是否需要独立列由查询消费者决定，不能把长期关键引用只放 runtime。已保存的原始 torrent、hash、提交意图仍是恢复真值。

## 7. 配置与管理

建议升级 config_version 到 3，因为查询顺序 ID、启用策略、实例配置与保存契约变化，旧二进制不能安全理解新语义。保持现有业务分区；只重构受影响的 indexers 配置，不借机重写所有 legacy Go 字段映射。

以下是逻辑结构示例，值为建议默认，不是已经确认的运行配置：

    config_version: 3
    indexers:
      jackett:
        enabled: false
        url: ""
        api_key: ""
        password: ""  # 仅既有管理会话确有需求时使用，常规搜索不应要求它
        excluded_indexers: []
      builtin:
        definitions:
          source: prowlarr
          schema_version: 11
          check_interval_hours: 24
          auto_update: false
        instances:
          "7aef-example":
            definition: sukebeinyaasi
            name: Sukebei
            enabled: true
            base_url: https://sukebei.nyaa.si/
            settings:
              filter-id: 0
              cat-id: "2_2"
            network_profile: direct
        network_profiles:
          direct:
            proxy_url: ""
            flaresolverr_url: ""
        request_timeout_seconds: 30
        site_timeout_seconds: 120
        challenge_timeout_seconds: 90
      query_order:
        - "7aef-example"
        - jackett/onejav
      search_cache:
        enabled: true
        ttl_seconds: 300
        empty_ttl_seconds: 30
        max_size_mb: 32

示例仅展示用户启用后的状态。新安装默认没有已启用站点；允许正常启动，索引相关动作返回“尚未配置可用索引器”。已有配置升级保持原 Jackett、站点次序与范围，不自动切换为内置。

settings 的动态值是 bool/string/number 等受约束值，由 definition 的设置描述验证；API 使用类型化 SettingValue 输入，不开放任意 JSON 执行。secret 字段读侧仅返回是否已配置；更新不传保持，明确 clear 才清除。不得通过整份实例表单把掩码字符串写成密码。

保存分两种：全局缓存/连接/超时沿用 typed patchSettings；实例 add/update/remove、远端站点使用开关和 reorder 提供按 ID 的窄操作。所有操作共用 config.Store 同一把锁，锁内读取最新值、检查期望配置 revision、应用字段变化、验证、原子持久化，再发布配置。

原因：现有数组 patch 是整体替换，不能让一个站点编辑覆盖另一窗口的新站点；null 仍是“不改”，不能偷偷改成 map 删除。实例删除是明确命令，清空秘密是明确字段；未知规则设置被保留还是拒绝由对应规则修订验证，不任意丢弃。

规则源/schema/检查周期/自动更新策略归 indexers.builtin.definitions，沿用 typed patchSettings；schema_version 只能选择引擎声明支持的版本，首版仅 11。实际活动 commit、digest、前版和检查结果是安装状态，由 manifest/core 返回，不把它们作为用户任意改写的 YAML 版本号。首版固定官方 GitHub 源，不开放任意下载 URL/任意本地文件路径；下载网络允许继承明确的代理配置，不复用某个私有站的 Cookie 或凭据。

建议已有规则安装后每 24 小时检查更新，默认手动应用；用户可开启自动更新，仍必须完成全部校验。尚未安装时不因目录 Query 产生隐式下载，UI 明确提供“下载规则库”；已有内置实例但缺少规则时报告不可用并提供恢复安装入口，不悄悄查询 Jackett。

## 8. 状态、生命周期与资源预算

第一版优先使用已有三个 Manager owner，不增加数据库文件。

| 数据 | 所有者/位置 | 生命周期 |
| --- | --- | --- |
| 站点、顺序、网络配置、显式凭据 | 原子 YAML config.Store | 用户真值、纳入既有配置备份 |
| 已选资源、原始 torrent、hash、提交与来源意图 | core moji.db | 必须恢复，新增有序迁移 |
| 登录派生 Cookie、UA 关联、会话有效期 | runtime.db，新表由 Manager 注入 | 可重建；修复后重登；失效不冒充配置丢失 |
| 人工验证码临时上下文 | 有界内存 challenge store | 5 分钟建议值；重启失效，UI 要求重新发起 |
| 站点冷却、近次健康、搜索 run/候选快照 | runtime.db | 可重建/中断；不作任务真值 |
| 搜索缓存、解析计划、短期 Artifact | 有界内存 | 重启丢弃；不另开 cache SQLite |
| 规则包原始文件与 manifest | 当前存储 generation 下 indexers/bundles/<revision>/，不可变目录 | 运行时独立更新；保留活动、前版和被引用版本，纳入备份附属资产 |
| 规则安装/活动修订/前版/更新作业与重要审计 | core moji.db，Manager 注入 | 安装/激活先持久记录，重启核对；不放可丢弃搜索缓存 |

派生 Cookie 按 instance/config revision、网络路径隔离，匹配过期与 domain/path/secure，认证修改后清除，保持挑战 UA/出口 IP 一致。敏感 runtime 字段不进入普通查询/日志，目录与文件权限沿用受保护存储要求，备份中如包含这些字段同样视为秘密；不假设现有系统已经有静态加密。

建议初始预算：所有索引 HTTP 在途 4；同实例在途 1；同 origin/代理出口共享节流，遵守 requestDelay，默认最小 1 秒；登录同实例合并，其他请求等待。全局等待队列 64，满时返回繁忙。搜索单站 120 秒、交互 run 总预算 10 分钟，外部请求可取消；自动任务整个寻源预算 10 分钟，到期记录部分结果/原因并按现有策略稍后重查。挑战预算包含在站点预算内，不可另叠加无限超时。

页面 HTML/JSON 解压后建议上限 16 MiB、torrent 8 MiB、重定向 5 次、每站搜索请求预算建议 50 次。执行器预先计数，超预算显式标记截断/失败，不伪装完整结果。对固定语料统计超限需求后调整；这些是资源保护预算，不改变 YAML 各 path 的执行含义，也不按 UI 页大小擅自跳过后续 path。

瞬时只读搜索故障建议最多重试 1 次；尊重 Retry-After，失败指数冷却初始 30 秒、上限 30 分钟。认证失败/规则错误/人工挑战不盲目重试。下载请求可能消耗私有站 grab 配额，即使 GET 也不假定无副作用：禁止叠加多层重试，下载结果不明先报告；同候选请求合并并复用已取得原始数据。冷却状态与空结果缓存是两套语义。

所有 goroutine、定时清理和订阅由应用生命周期拥有。取消一个共享请求的等待者只取消自己的等待；所有等待者退出后才取消尚未完成的共享外部请求。不能用无取消的 Background 保活。

## 9. 缓存设计

建议做短期精确缓存；不做长期站点索引库，不缓存最终选种决定，不将上游失败写成空结果。所有 TTL/容量是待确认默认值。

| 缓存 | 初始建议 | 正确性约束 |
| --- | --- | --- |
| 编译规则 | 活跃规则包整个生命周期，未配置定义按目录需要加载 | digest/engine version 为键；不可变、可并发共享，实例状态不能写到计划里 |
| 内置成功搜索 | 5 分钟，32 MiB 全局、每实例最多 100 查询、每查询最多 1000 结果 | 只保存完整成功批次；超限不缓存或明确标记，不把截断结果当完整缓存 |
| 内置正常空结果 | 30 秒 | 必须确认正常响应且解析成功；挑战/错误页不算空 |
| 同键在途请求 | 所有同时等待者合并 | ctx 和配置快照一致，不合并不同账号/强制刷新策略 |
| 外部站点目录 | 建议 60 秒内存快照 | 刷新失败保留最近成功目录并标过期，不清空已配站点 |
| 外部 Jackett 搜索 | 首版不叠加 Moji 结果缓存 | 上游本身可能缓存，报告 freshness=UNKNOWN，不能把返回时刻当抓取时刻 |
| torrent 获取/文件检查 | 每次选择过程必复用；跨操作短期原始数据建议 10 分钟、64 MiB | 按来源与配置隔离；解析后按真实 hash 去重；解析结果和原始数据同生共死；任务选中后转 core |

搜索键包含 instance ID、规则 digest、配置/凭据修订、网络 profile 修订、模式、完整实际 query/分类/模式参数、页/offset/limit，以及会影响结果的语言/日期模板上下文。不能只用番号；不把不同分隔符/关键词大小写无依据合并。键中保存秘密修订标识，不保存可输出的明文密钥或密码摘要。

缓存原始规范候选，使用时重新执行当前 Moji 筛选、身份和去重；改变选种规则不需要重新抓站点。配置/域名/账号/代理改变让对应实例搜索、资源与会话失效；规则替换按 digest 失效。已被用户选中的持久任务不受清缓存影响。

不做 stale-while-revalidate 驱动自动下载，过期必须重新请求；刷新失败不复活旧缓存参与选种。历史 UI 可以保留上次搜索 run 内容并标明本次失败，但选用时要核验来源修订与资源是否仍有效。公开缓存命中不延长原 fetchedAt/TTL。

用户“重新搜索”、站点测试和对照工具绕过 Moji 查询缓存；可配置禁用缓存。对于 Jackett，当前 JSON ApiSearch 没有 Cache 字段，不能承诺加 cache=false 就强制上游刷新。若后续采用支持缓存控制的 Torznab 入口，需验证协议与认证后单独实现；目前明确标注上游缓存不可保证绕过。

签名下载 URL 不长期复用；资源获取遇明确 URL 过期时可按原站点资源身份重新解析一次，必须重新验证相同候选/hash，不能借此重搜并悄悄换资源。不能把缓存 torrent 通过忽略鉴权的全局 hash 查询接口暴露。

## 10. GraphQL 与辅助 HTTP 接口

新增 types/indexers.graphql；模型由 schema 生成。以下是语义与建议名称，实施时以 typed schema 落实，不提供任意 URL 代理接口。

| 接口 | 输入/输出与作用 |
| --- | --- |
| Query indexerDefinitions | filter/capability/privacy/language + 有界分页；返回目录、兼容状态和设置描述，不触网搜索 |
| Query indexerInstances | 已配内置、最近远端目录、启用状态、顺序、能力与过期状态；secret 只返 configured 标记 |
| Query indexerSettings / indexerCacheStatus | 各自领域投影；不要求拉取全站设置或敏感 runtime |
| Query indexerDefinitionStatus / indexerDefinitionUpdate | 活动/可用/前版、已安装修订、检查结果、更新作业阶段、实例影响与兼容错误；不在 Query 隐式下载 |
| Mutation checkIndexerDefinitionUpdate | 有界检查上游 v11 修订，返回结果；受控频率、条件请求，不能因点击刷新无限请求 GitHub |
| Mutation installIndexerDefinitions / updateIndexerDefinitions | 首装/更新接受后持久化作业，返回 ID；固定候选修订、后台下载校验并激活，支持 expectedActiveRevision 防止过期操作 |
| Mutation rollbackIndexerDefinitions / cancelIndexerDefinitionUpdate | 回到已安装兼容前版；取消仅在提交激活前有效，原子提交开始后返回当前实际结果，不能假称撤销 |
| Subscription indexerDefinitionEvents | 安装/更新阶段、完成/失败/中断与活动修订变化；刷新或重连定向读持久快照 |
| Query homeIndexerSummary | 有界本地索引状态投影、内置站点摘要、已配置且启用的 Jackett、规则可用状态与操作入口；不触发抓站或更新 |
| Mutation addBuiltinIndexer / updateBuiltinIndexer / removeBuiltinIndexer | 按稳定 ID、typed 设置值与 expectedRevision 原子修改；update 只改明确字段 |
| Mutation setIndexerEnabled / reorderIndexers | 远端可用状态与本地启用分别表达；顺序 revision 冲突明确报错 |
| Mutation refreshJackettIndexers | 只刷新远端目录；失败保留最近数据，不变更远端配置 |
| Mutation testIndexer | 当前草稿或已配实例；返回配置/认证/搜索/下载各阶段结果，不写正式配置、查询缓存或创建任务 |
| Query indexerLogin | 只恢复已有待回答挑战，不启动登录或上游请求；当前配置可清理已过期配置的流程 |
| Mutation beginIndexerLogin / answerIndexerChallenge / cancelIndexerLogin | 返回 challengeID、类型、展示资源/说明和到期；答案不落永久设置，临时身份不可跨实例用 |
| Mutation startTorrentSearch / cancelTorrentSearch | 返回 run ID；支持实例范围、关键词、分类、limit、refresh、可选 taskID；取消不撤销已创建下载任务 |
| Query torrentSearch | run 状态、逐站状态、分页候选、截断与部分失败、cache/freshness；不泄露原始下载凭据 |
| Subscription torrentSearchEvents / indexerEvents | 进度与状态变更，事件仅做通知；断线定向重读 run/实例，不全量轮询 |
| Mutation previewTorrentSelection | searchID + candidateIDs + flags；用服务端候选预览当前规则，返回检查预算及实际次数 |
| Mutation queueTorrentCandidate | 有效 code + searchID/candidateID；经 taskflow 创建意图，worker 继续取种、验证与提交 |
| Mutation resolveTaskSourcingCandidate | taskID + searchID/candidateID；runtime 重验证现有任务可操作状态和身份后处理 |
| Mutation clearIndexerCache | 可限定实例；只清查询/可重建资源缓存，不删任务、凭据、规则或已选文件 |
| Mutation prepareTorrentDownload | 为需要服务器会话的资源获取短期、不含秘密的下载句柄，不创建任务；失败独立呈现 |
| GET /api/indexers/downloads/<opaque-handle> | 返回已获取 torrent attachment 或安全 magnet 结果；短期有效、有界复用；适用 Moji 同一访问边界，no-store，不接受任意上游 URL |

测试下载会消耗站点配额，因此连接/登录测试和搜索测试不默认抓种；下载验证为显式选项，且不提交 qB。外部 Jackett 能力测试不要把“服务连接通过”冒充站点全流程通过。

交互搜索 run 是专属小模块，不引入通用 job 框架：QUEUED/RUNNING/COMPLETED/PARTIAL/FAILED/CANCELLED/INTERRUPTED。接受后先写 runtime，浏览器关闭继续执行；刷新凭 URL 中 run ID 恢复。重启把在途标中断，保留已有候选；新搜索显式发起。结果建议 30 分钟有效、每 run 1000 候选、最多 50 run、最多 10000 活跃候选且总快照字节上限 64 MiB，终态后按 TTL/LRU 清理；旧 ID 返回过期，不重跑。候选外部引用服务端保存，客户端不得回传 URL/headers 替换。

start 返回前写 run 意图，应用 worker 执行；HTTP 请求 context 仅控制接收，不错误取消已接受作业。客户端事件流包含 stream/sequence 或可判定失效标记，快照是恢复依据。cancel 显式记录后终止尚未完成的工作，保留已产生结果。选择已存在候选建立新任务时先复制必要引用到 core，不能让 30 分钟清理使已接受任务失效。

现有 jackettSearch、previewJackettSelection、jackettIndexers、blockedTaskTorrentCandidates 和相关 Web 操作迁移至中性命名。兼容面按真实消费者清点：仓库 UI 同步迁移；若有外部使用，旧 Jackett 命名字段仅委托外部适配器并短期 deprecate，不偷偷改成内置聚合语义。无真实消费者时同一发布移除，不保留永久双接口。旧 URL 直接添加和上传仍是明确手动入口，不能拿它们代替带来源会话的候选选择。

## 11. Web UI 修改与用户路径

保留现有七个业务设置分区、/settings/indexers、共享 shell 和 ?panel= 子页签导航。建议索引器页为“站点 / 规则库 / 外部 Jackett / 网络与缓存”四个二级页签；旧 connection、queryOrder 参数映射到对应入口，前进/后退与深链接有效。

站点页合并已配置内置和可用外部站点，用同一列表显示名称、来源、启用状态和必要异常；沿用拖动手柄、触摸与上下按钮，不显示序号。添加入口打开定义目录，支持名称、语言、隐私、能力筛选；未兼容项可查明原因，不能保存为可运行实例。所有规则可发现，不建立隐形少量站点白名单。

内置编辑 Drawer 按 schema settings 生成 text/password/checkbox/select 和必要 info 字段，通用连接/网络放可展开区域；展示当前实际 URL 与镜像选项，保存后下次操作生效。字段说明只保留影响决策的上游必要信息，上游 HTML 必须安全处理。动态站点标签保留原文，Moji 外壳和错误走 i18n。

编辑沿用 SettingsAutosaveProvider 的串行增量与失败保留，但实例 CRUD/重排发窄命令而不是整体替换列表。添加先在抽屉中形成完整草稿，点击“添加”原子创建，不能每输入一个字符保存半个实例；创建后编辑自动保存。测试明确使用当前草稿，不悄悄把草稿变成运行配置。

外部页保留 Jackett 地址/API key 与连接测试；未配置正常显示可选，不在 Home 报警；远端站点详情提示在 Jackett 管理。常规索引列表与状态不依赖 Jackett 管理密码。

发现页把“Jackett 搜索”改成“资源搜索”，已有 StashBox 模式保持；来源选择改用统一 indexerInstances 投影，搜索 run ID 进入 URL。展示逐站真实进度、部分失败、截断和继续操作；完成结果复用现有候选链接与做种组件，必要时服务器代取种链接。无需会话的详情链接保持严格协议校验，不把下载 URL 当详情。

任务寻源处理、任务处理台、规则预览全部走相同来源引用；任务仍显示来源站点和为何不合格。旧直接 torrent 链接不能用于跳过新来源解析链。

Home 按用户已确认方向增加“索引器”区域，与现有“下载器”区域平级，具体契约见下节。不开独立索引仪表盘。

### 11.1 规则库管理组件与路径

新增 IndexerDefinitionsPanel，位于 ?panel=definitions。拆分 DefinitionUpdateStatus（真实作业进度与错误）、DefinitionChangesDrawer（规则增删改与已配实例影响）、DefinitionRevisionList（可回退修订）及 useIndexerDefinitionUpdates（快照、命令、WebSocket 恢复）；复用 Drawer、SettingsPanels、SettingsSaveStatus 和现有语义样式，不做第二套配置状态容器。

| 状态 | 用户看到与可做的事 |
| --- | --- |
| 尚未安装 | “下载规则库”；不显示为整个 Moji 故障；Jackett 与现有下载可继续使用 |
| 首次下载/校验 | 真实阶段、可确定时的已下载字节、进度与取消；无 Content-Length 不造百分比 |
| 已安装 | v11、当前修订短标识、安装时间、站点数、检查更新入口；正常状态不常驻大段说明 |
| 有更新 | 新增/修改/移除规则数量、已配实例影响、“更新规则库”；技术摘要和 diff 按需展开 |
| 更新进行中 | 下载/校验/检查实例/应用阶段；旧规则仍可使用；刷新或另一窗口可找到同一作业 |
| 更新失败/不兼容 | 明确“当前规则仍在使用”、失败阶段、需更新 Moji 或修改设置等下一步与重试 |
| 可回退 | 上一修订与影响预览；确认后切换，不能无提示放开不兼容设置 |

首次添加站点路径：添加站点→未安装提示→显式下载→完成后回到同一抽屉的规则目录→选择并配置站点。搜索引用规则目录时只读已安装快照；若尚未安装返回稳定代码 DEFINITION_LIBRARY_NOT_INSTALLED。检查更新失败不把现有目录替换成空列表。

自动检查/自动应用策略在同页自动保存；更新、回滚、取消均为显式动作。可选自动应用时仍显示后台作业，兼容校验失败保留旧版；是否自动检查、周期与默认不自动应用是建议，未视为用户已确认。

### 11.2 Home 索引器与下载器

用户已确认：新增 HomeIndexers 信息区；HomeDownloader 移出当前系统面板成为同级区域；删除系统列表中的固定 Jackett/qBittorrent 状态行及详情中的两个固定 ServiceCard。下载器现有连接、速度、容量、异常和种子列表入口仍在 HomeDownloader；不因移除重复卡片丢失故障或管理动作。

两区复用同级 home-panel、标题 h2 和间距：宽屏由现有 Home 网格安排同层，窄屏依次显示，不能把索引器嵌到下载器或系统详情。现有任务/注意事项优先级保持；Stash、交付及其他系统信息按当前职责保留，不借此次改动移除。

HomeIndexers 显示已启用内置站点的紧凑概况与异常；Jackett 仅在已配置且启用时以外部来源行出现，故障时仍保留该行与排查入口，未启用时不占固定位置。Jackett 是网关，统计站点数量时不把网关自身再计一个站点；远端目录未知用未知，目录过期保留最近确认数，不伪造 0。显示入口为管理索引器、需处理站点、首次规则安装或失败更新；正常时不把规则版本和所有站点详情堆满首页。

“可用”依据配置完整、规则已安装/兼容、非已确认认证失败/冷却等状态分别表达；尚未搜索的站点标记未验证，不能用有配置就推断实时联网成功，也不为 Home 高频主动测试所有站点。建议 homeIndexerSummary 返回总计和最多 5 个异常/活动来源摘要，完整列表在设置；保留各状态的更新时间、过期与局部失败。

HomeIndexers/useHomeIndexers 读取独立有界快照并复用 indexerEvents/规则更新事件，事件合并后只刷新该区，断线重连补读；不把整个定义目录加入 homeOverview，不新建高频 HTTP 轮询。Home 顶部 needsSetup/problems、首页 GraphQL documents、状态文案与 tests/visual 同步移除 Jackett 必需和两个固定状态的假设；对索引能力与下载器的真实故障保留可执行入口。

主路径验收：新用户添加公开站→测试→资源搜索→选用并下载；私有站添加→登录/验证码→恢复搜索；旧用户升级后沿原 Jackett 工作→添加内置→调整顺序；站点故障→查看原因/重试/手动补资源。覆盖刷新、断线、多窗口配置冲突、移动、键盘、长名称及部分成功。

## 12. 运行时规则库、独立更新与安全边界

### 12.1 来源与版本

生产规则 YAML 不进入源码、Go embed 或 Docker 镜像；项目只维护引擎能力契约、受支持的 v11 校验 schema、测试样本及语料锁文件。CI 从固定 commit 获取完整兼容语料，可使用 CI 缓存并校验摘要；运行测试不拉浮动最新，维护语料锁不等于把整个生产规则库复制进仓库。

首版下载源使用官方 Prowlarr/Indexers GitHub 仓库，版本发现限定 definitions/v11。通过该路径最新提交发现候选 revision，再按完整 commit 获取归档，只提取 v11 内容；避免逐个站点发几百个 HTTP 请求。复用 ETag/条件请求和合理退避；检查不可用时保留本地活动修订。源码归档入口、代理与具体下载格式在 A 阶段做受控验证。不能把检查之后再次从浮动 master 下载的文件当成同一快照。

不优先依赖 Prowlarr 的动态分发服务作为唯一来源：其客户端带版本选择和专用实现 blocklist，Moji 的声明式兼容目录应从上游 v11 源目录独立建立。分发服务可作为未来经过验证的来源适配，不默默混用两个源。规则更新范围只含 v11，发现 v12 等新格式不能自动升级引擎。

manifest 由本地安装过程生成，记录 repository、commit、schema、各文件摘要、下载与安装时间、引擎编译版本、兼容诊断和规则数量。固定 HTTPS 来源、commit 与摘要用于来源追踪及完整性核对，不宣称存在上游未提供的发布签名。下载规则会减少手工同步和用户升级成本，仍需维护引擎语义、处理上游破坏性变化及服务可达性；不承诺零维护。

### 12.2 安装、激活与回滚

    创建作业并固定候选 commit
      → 下载到暂存目录
      → 受限解包并生成 manifest
      → v11 schema 与全库编译/能力验证
      → 对比当前库、检查所有已配置实例设置
      → 发布不可变规则目录并 fsync
      → 核心事务切换活动修订、记录审计和作业完成
      → 发布事件，新操作使用新版

同一时刻只允许一个安装/更新/回滚操作，重复请求返回当前作业；start 接受时先写 core，浏览器生命周期不决定作业是否继续。建议状态为 QUEUED/DOWNLOADING/VALIDATING/CHECKING_INSTANCES/ACTIVATING/COMPLETED/FAILED/CANCELLED/INTERRUPTED，字节进度只在真实可确定时返回。版本检查是短操作，安装和更新是可恢复查看的后台工作，不另建通用工作流引擎。

下载暂存与正式目录在当前存储 generation 的 indexers/ 下，由 cmd 注入受保护根路径。压缩/解压大小、文件数、单文件和 YAML 深度有预算；拒绝目录穿越、符号/硬链接、非预期文件及解压炸弹，不将归档直接覆盖到活动目录。建议初始总目录预算 256 MiB，压缩包 32 MiB、解压 128 MiB，实施用全库测量核定。只接受候选包的 v11 定义/schema 元数据，远端 schema 不能自行放宽引擎内置能力校验。

激活前检查规则移除、设置改名/类型变更、新增必填、模板/过滤器/选择器不支持和规则摘要异常。活动实例受影响或全库兼容校验失败则本次不激活，返回具体定义和字段，保留旧版；不自动禁用站点、猜填凭据或使用 replaces 换站。检查期间可继续编辑实例，最终激活用配置 revision 和短时配置/激活协调门核对，修订变化重新检查；网络和编译不得占 SQL 事务。

活动修订以 core 中一条记录为唯一真值，不另写可能分叉的 CURRENT 指针。先落完整不可变目录，再在核心事务更新 active/previous、作业结果及重要审计。崩溃在事务前只留下可清理孤儿，事务后重启按 core 加载已发布目录；必须区分“未激活”和“已激活但事件未发出”。活动目录缺失/摘要错误时该内置索引能力明确不可用，不删除记录、不悄悄重置；提供恢复下载同一 commit 或校验后显式回滚入口，外部 Jackett 和其他模块继续运行。

规则激活不重启 Moji。搜索 run/自动寻源与资源获取固定 bundle revision、definition digest 与配置快照；正在运行的工作继续旧计划，新工作使用新计划。未完成任务的必要规则引用必须持久化并纳入清理引用检查，已持久化原始 torrent 的后续下载不再依赖规则包。解析缓存按 digest 选择性失效，未变规则不清全部会话，真正影响会话的规则变更按实例失效；不能让 schema 版本相同就复用已变下载规则。

保留当前、至少一个可回退前版及仍有持久/在途引用的版本；无引用旧版才按容量和保留策略清理，预算不足且无法安全清理时拒绝新下载，不能删除活跃引用。回滚只选择本地已安装并经当前引擎/配置重新检查的修订，同样用原子切换。下载/校验阶段取消可生效；激活事务开始后返回最终状态，不许显示“已取消”却已切换。

重启将未提交激活的作业标记中断，活动修订不变，用户可重试固定 commit；已发布但未引用的目录重新核对后复用。暂不承诺断点续传；HTTP 下载是否支持 Range 与“刷新后能看到进度”是两件事。进程重启不丢已完成安装，也不会因上游不可达阻止其他功能启动。

### 12.3 首装、备份与程序升级

没有规则包时 Moji 可以启动，内置站点目录显示尚未安装；首次安装需要联网，已安装后启动和搜索不需要连接 GitHub。GitHub 仅用于规则获取，不参与每次站点搜索。更新检查失败不使本地规则失效；只有兼容能力确需升级时才要求升级 Moji。

安装记录、活动修订、更新重要审计纳入 core 备份；精确对应的规则目录与 manifest 纳入 Manager Supplemental 资产。备份按备份 core 快照中的引用固定目录，不能仅取备份结束时的最新活动目录。restore/离线 generation 工具验证并恢复这些资产，使离线恢复不依赖仍可下载的远端 commit。规则库不是可随查询缓存一起清理的数据。

升级 Moji 时检查新引擎能否读取当前安装库并重建编译计划，不能自动拉最新或切到新 schema。二进制降级也要校验规则修订兼容性；回退程序与回退规则是两个独立操作，遇不兼容应协调选择，不能假设旧程序自带旧规则。

### 12.4 依赖与访问边界

复用 yaml.v3、既有 HTTP/context 和限流基础；按语料需要增加 DOM/CSS、JSONPath、.NET 风格正则、schema validator 等最少依赖。逐项记录使用理由、许可证、版本和执行预算，避免引入常驻浏览器/Node/.NET 运行时。FlareSolverr 是显式可选网络能力，配置后按挑战检测触发，不保证解决所有挑战；solver、Cookie/UA 与实际代理出口必须一致。

使用上游 C# 作为行为参照不等于可以忽略许可。Jackett 根许可为 GPL v2 文本，Prowlarr 为 GPL v3 文本；定义仓库根未发现独立 LICENSE。第一阶段确认引擎可能移植代码、测试样本、规则运行时获取与使用的授权/来源及发布方式，补齐 attribution/许可材料；动态下载不作为规避许可要求的理由。这里不对最终授权兼容性作法律结论。

抓站 URL 限 HTTP(S)，magnet 仅走专门解析；禁止 file/exec/任意模板函数。规则请求与重定向做统一 URL/地址检查，禁止 metadata/link-local 目标、用户信息 URL 和跨来源泄露 Authorization/Cookie；用户显式配置的内网 Jackett/代理/solver 端点走独立允许策略，不被公共站点规则继承。域名变化、镜像/下载 CDN 按经过验证的站点能力与明确用户配置支持，不能靠放开全部内网访问解决兼容。

不假设现有 Moji 已有完整账户权限系统：新 API 沿用部署访问边界，引用不可枚举/不可任意替换，敏感凭据不进入客户端，下载句柄不是任意代理。进一步多用户隔离或公网认证属于独立产品工作，不能假称本方案自动提供。

## 13. 迁移与回退

1. 离线解析 v2/既有旧配置，保留受保护原件；v3 转换保留所有无关字段和凭据的有效值，不默认启用内置来源。
2. 旧 query_order、已存 sourcing 请求范围中的裸 Jackett ID 映射到 jackett/<id>。旧任务请求读取按 payloadVersion 解释；已完成任务保留历史来源文案。新写入统一新版本，不永久猜测字符串前缀。
3. 任务来源、规则安装/活动修订/更新作业需要的新核心字段与表由新的有序 migration 增加；不改 core_v1/v2/v3 和已应用 checksum。runtime 的 search run/session/health 表同样新增迁移。实际版本号以实施时注册表为准；备份/恢复与 generation 资产协议同步扩展。
4. config 与 DB 不做伪原子跨文件转换。新二进制支持已明确的旧 config 读取与版本化旧任务 payload；启动/保存迁移可重复验证。离线准备并检查候选 config/DB，激活失败可恢复原件，不留下必须猜测修复的半状态。
5. 删除来源不删除已选任务：持久 torrent/magnet 仍能完成；未取得资源的引用明确报告来源不可用，允许用户重新配置或重新寻源。配置改变/规则 digest 改变时重新验证引用；不能混用旧配置任意抓取。
6. 二进制回退需协调恢复受支持的配置与 DB 备份；不承诺旧程序能读取新 schema。实际生产配置替换、服务重启、数据库迁移按实施阶段授权执行，本设计不操作现场。

## 14. 实施阶段与验收

| 阶段 | 交付 | 完成条件 |
| --- | --- | --- |
| A 基线与语料 | 固定测试用 v11 修订、运行时下载来源验证、许可结论、全库特征/差异矩阵、golden harness、初始预算 | 每个 schema 能力有对应实现项与验收；所有定义可分类，未支持不被忽略；生产不内嵌规则 |
| B 中性边界 | indexer 类型/接口、Jackett adapter、runtime 消费迁移、状态投影 | 现有 Jackett 逐站、规则、取种、恢复行为回归通过，任务不再依赖 Jackett 结果类型 |
| C 引擎执行 | 模板、选择器、过滤器、搜索、登录/挑战、下载解析 | 固定全库编译无未解释特征；合成/上游样本覆盖完整语义，兼容差异明确 |
| D 闭环与配置/UI | 内置 adapter、Artifact 复用、v3 config、迁移、GraphQL、搜索 run、规则首装/更新/回滚、设置与发现/任务 UI、Home 同级索引器/下载器 | 独立假服务覆盖从动态安装规则到搜索/qB，激活与备份恢复契约成立；浏览器完整路径通过，移除固定状态卡且不丢故障入口 |
| E 对照与可靠性 | 本地 Jackett 差异工具、代表性真实站点、取消/限流/并发/重启/磁盘压力验证 | 差异逐项解释或修复；公开、会话、JSON、下载中间页、挑战场景有证据 |
| F 默认交付 | 文档、部署示例、无 Jackett smoke、多架构镜像、升级回退证据 | 新安装不依赖 Jackett、旧安装行为保留；已启用来源正常运行且失败可处理 |

阶段按能力推进，不把“只支持几个站点”作为最终范围；每个可合入增量后端/GraphQL/Web/持久化要闭合，不提交假定尚不存在 API 的界面。未完成项统一链接到唯一 TODO，本文保留设计，不维护第二份活动进度表。

测试分层：

- 全库静态：固定语料 schema/编译、特征扫描、诊断定位、字段次序、模板与正则模式编译；不能仅测能反序列化。
- 语义 golden：同一 YAML/HTTP 响应、固定时钟与时区、请求序列/headers/encoding、规范结果；覆盖 HTML、JSON、登录、人工挑战、download.before、magnet 跳转和错误页。
- 对照：固定 Jackett 镜像版本、来源 YAML 和设置；必要时明确 Prowlarr/Jackett 定义差异；同一 fixture 对照避免网站实时变化。真实对照记录时间/网络/版本/缓存影响，不自动重复抓付费配额资源。
- 领域：标题/文件规则、检查预算、选择与原始字节一致、infohash/重复、部分失败、qB 超时不明、persist-before-submit、重启复核。
- 并发：同实例登录合并、缓存键隔离、请求取消、配置热更新、run 快照晚到/缺口、队列满、受限容量、清理与 active refs。
- 迁移：真实结构的脱敏旧配置与持久 payload、无关字段精确保留、未知版本拒绝、重复激活、失败恢复与回退演练。
- 规则更新：首装断网、限流、归档损坏/穿越/超限、schema 和实例设置不兼容、并发编辑/更新、激活前后崩溃、缓存按 digest 失效、在途与持久旧引用保护、取消边界、规则独立回滚、备份时并发激活与离线恢复。
- Home：索引器/下载器同级与窄屏顺序、Jackett 未配置/禁用不占位且不报警、已启用失联保留行、未知站点数、首次规则未安装、更新失败保留旧状态、固定卡片/提示/GraphQL/i18n 假设清理、Stash/交付与现有下载管理入口保持。
- Web：中英文、深浅色、宽窄屏、键盘、必要 pseudo/RTL；Playwright 固定环境人工复核最终截图和控制台。
- CI：新增包必须进入 Go 默认离线测试，由当前全包默认离线测试覆盖，继续扩展兼容语料与差分 CI；按规范运行全量 Go、必要 race、GraphQL 生成、Web tests/build/i18n、文档/diff 检查和 Docker smoke。

不预先承诺整个规则库真实可达率或一个虚构成功率；全库静态兼容与特定实时站点可靠性分别报告。性能验收记录语料规模、请求数、p50/p95、峰值内存、缓存字节、取消耗时及正常/限流负载，按测量调整预算。

2026-11-03 请求语义阶段：已按固定语料增加请求特征和编码报告，修正继承参数覆盖、空白输入、POST 去重、raw 两阶段编码、普通参数的声明编码及 download.before 空输入；Go CI 已接全包默认离线测试；完整 Go、受影响完整 race/vet 与 579 个固定定义静态检查已通过。源码依据与剩余编码/通用请求/查询语义差异见[请求复核](../reviews/2026-11-03-cardigann-requests.md)，随后补齐当前语料八个旧代码页的 .NET best-fit/replacement，用固定官方 SDK 无网络容器生成并复核 BMP/单字节摘要；小型生成表仅为语言兼容数据，Moji 无 .NET 运行时依赖。随后响应统一声明编码，修复 XML 重复解码并补齐安装期未知编码拒绝；展开预算和坏更新保留旧库有回归，完整 Go/race/vet 与固定语料通过。仍须完成其他执行语义、完整上游程序差分和整体验收。

## 15. 待用户复核的产品选择

已确认：规则库运行时下载并可独立更新；Home 索引器与下载器同级，已配置且启用的 Jackett 归入索引器，移除固定 qBittorrent/Jackett 状态与卡片。以下建议尚未确认，不影响先完成离线能力矩阵和测试准备：

1. 来源组织：统一站点列表、内置与显式配置的 Jackett 站点可混排；旧用户不自动切换，新用户不自动启用站点。
2. 配置演进：v3 与受保护升级；实例/排序窄操作，现有全局字段继续自动保存。
3. 缓存：默认开启内置精确查询 5 分钟/空结果 30 秒，无过期自动选种；Jackett 首版不叠加搜索缓存。
4. 规则更新默认策略：首次显式下载，安装后默认每 24 小时检查、手动应用，允许选择自动更新；固定官方源与 v11，暂不开放任意自定义规则上传。
5. 搜索体验：交互搜索使用可刷新恢复的 run 与 WebSocket 事件；退出页面继续，只有显式取消才停止，结果短期保留。

完全移除 Jackett 和新增专用解析器均留待上述闭环及可靠性验收后讨论。
