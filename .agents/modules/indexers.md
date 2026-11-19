# 索引器、规则库与交互搜索

核对日期：2026-11-19；代码基线：`ec7b093`；状态：当前源码契约；站点实时可用性不由静态核对保证。

主要源码：[统一服务](../../internal/indexer/service.go)、[Cardigann](../../internal/cardigann)、[规则库](../../internal/indexerdefs/service.go)、[站点管理](../../internal/indexeradmin)、[交互搜索](../../internal/torrentsearch/service.go)、[资源附件](../../internal/indexerdownloads/service.go)、[登录](../../internal/indexerlogin/service.go)、[诊断](../../internal/indexercheck/service.go)。用户配置见[索引器说明](../../docs/indexers.md)。

## 领域所有权

| 区域 | 职责 |
| --- | --- |
| cardigann | 有预算的 v11 声明式解析、模板、过滤器、HTML/JSON、认证与资源定位；不拥有任务或全局配置 |
| indexer | 内置/Jackett 适配、冻结来源、逐站执行、网络、查询/附件缓存和执行记录 |
| indexerdefs / indexeradmin | 运行时规则包与持久引用；配置实例、目录、排序和脱敏投影 |
| indexerquery / indexercategory | 共享参数资格和 Torznab 分类映射，不持有网络或任务状态 |
| torrentsearch | 持久交互 run、逐站进度、候选归属/分页、取消及重启中断 |
| indexerdownloads | 显式获取后的短期附件句柄，不创建任务或代替持久选择 |
| indexercheck / indexerlogin | 草稿诊断/连接检查；已保存实例的有界认证挑战 |

内置与可选 Jackett 可同时使用；无索引器可启动，搜索返回未配置。自动寻源的停止条件、规则和任务接收归[任务模块](tasks.md)。不会加载 C# 专用索引器或任意脚本。规则可加载、合成差分通过、真实站点可访问、完整任务下载通过是不同证据；完整 v11 语料加载不代表所有站点实测。

## 规则与查询契约

规则运行时从 Prowlarr/Indexers 获取 v11，不随源码/镜像打包。首次安装是显式动作；source=prowlarr、schema_version=11 固定。检查周期 1–168 小时（默认 24），自动应用默认关闭；应用每分钟评估持久 lastCheckedAt。保存策略不立即安装或检查，关闭自动应用不取消已接收作业。

包按不可变修订下载、校验、fsync 后激活；manifest、active/previous、作业与 pin 归核心库，切换与审计同事务。失败不替换旧活动包。active、previous、持久引用、在途 lease 和更新目标受保护。资产备份/恢复见[存储](storage.md)。

模板字段顺序、重复字段及 append/noappend 有执行含义；受限求值器不暴露 Go 方法。正则/CSS/JSON/日期兼容层有执行预算，不能静默忽略不支持语义。范围及固定上游语料证据见[功能验收](../reviews/2026-11-03-indexer-acceptance.md)和[差分工作流](../../.github/workflows/indexer-compatibility.yml)；历史设计见[设计快照](../reviews/2026-11-01-builtin-indexers-design.md)。

命名参数由 indexerquery 统一校验：18 个结构化字段，大小写不敏感，重复大小写别名拒绝，显式空值保留；参数 map 不能覆盖模式、关键词、分类、分页或派生标记。无效查询在读取来源或接纳 run 前拒绝。Jackett JSON 适配器只支持关键词/分类，结构化参数或非零 offset 明确返回 INDEXER_CAPABILITY_UNSUPPORTED。Web 保持关键词入口。

内置适配层将未来发布时间限制为同次检索 UTC 时间，原规则字段/定位符保留，纯解释器不承担宿主策略。仅 infohash/title 的公开结果按已确认行为附公开 tracker；私有站点不自动生成公开 magnet fallback，完整 magnet 保留内容。ParseMagnet 校验 btih hex/base32 或 btmh SHA-256，保留 xt/tracker/name；v2 URI 校验不等于 v2 torrent 文件及端到端下载验收。

## 来源、网络与缓存

每次搜索冻结来源、账号、网络和规则快照。新选择/重新选源/显式附件准备经 ResolveReference/FetchCurrentReference 核对当前启用、配置与定义 digest；无关定义更新不使未变规则失效。预览先准入再读检查缓存，已借用引用不因中途候选过期失效。已接受任务继续使用核心冻结来源，不被之后停用或候选 TTL 撤销。

未配置镜像时使用冻结规则的首个 links 地址，会话和请求 origin 一致。所有内置站点共用一个 network 代理/FlareSolverr 配置，无逐站网络档案。请求/每站预算、并发、响应字节和重定向由现有受控网络限制，不能绕过它另建客户端。

gate 按 origin＋代理出口隔离并跨账号共享限流，懒读 runtime 冷却失败则不联网；写失败保留已收到资源和内存退避，dirty 补存成功后再准入。最多 8192 冷却记录，不驱逐有效冷却；普通成功零写入，实际成功/挑战解除清既有失败，序号 fence 不覆盖较新失败或 429。Retry-After 最长 24 小时。runtime 重建或写入失败不承诺重启恢复。外部 Jackett 网关 200 内单站失败不升级为整个网关限流。

可信来源最多 16 个完整 HTTP(S) origin，拒绝路径、参数、片段、凭据和通配符；空列表撤销，遗漏保留。凭据、表单与 CookieJar 收发都按来源授权，仍受 Cookie 域/Secure、公共地址和 HTTPS 降级限制。信任不授权私网访问。

FlareSolverr 是显式信任的外部浏览器：仅认证/搜索遇明确挑战允许求解一次并以 Cookie/UA 验证一次，取种与 before 页面不求解/重放，不发送认证表单。统一 max_timeout_ms 默认 90000、范围 1000–300000，限制排队及完整调用且不延长父截止；不创建远端持久会话。协议响应 32 MiB、页面 16 MiB，同端点串行、本地最多 16 端点；初始目标和最终来源校验不能约束外部浏览器内部全部导航。代理凭据与站点 Cookie 仅按明确配置委托。

查询缓存只返回深副本，精确查询合并不共享调用者取消；新接受流程各自注册引用，认证/挑战失败失效相关实例。附件缓存按冻结来源/定位合并，torrent 经 torrentmeta 验证并按 hash＋完整字节摘要区分私密 tracker，magnet 按完整内容隔离。成功数据绝对有效 10 分钟，64 MiB/512 定位、64 在途/等待预算；不缓存错误，不额外重试。最后等待者取消网络，旧请求未退出前不重叠抓取；关闭取消并等待，迟到回填受 fence 保护。清查询缓存不删除已获取附件。

## 交互搜索与附件下载

run 先写 runtime v5，应用两个 worker 执行，请求结束不取消。取消先落盘并保留已有结果；重启标记未完成项中断，不自动重搜。保留 30 分钟，最多 50 run、每 run 1000、全局 10000 候选，查询/站点/候选共用 64 MiB。候选只投影公开字段/opaque ID，冻结来源不对外公开；转核心后的任务不依赖 run TTL。订阅最多 128，慢客户端不阻塞关闭。

预览只接收 run/候选 ID 和规则开关，一致读取终态最多 1000 条，先执行完整集合 LOCK 再排序分页；非法分页在检查前拒绝，失败不回退未过滤原结果。文件检查复用当前任务预算及原始字节，缓存上限 256 条/32 MiB。

indexerdownloads 显式准备使用当前有效引用，句柄绝对有效 5 分钟、64 MiB/256 项；获取前预留 8 MiB，最多 8 个获取、16 个读取。活跃 reader 持 lease 并继续占预算。HTTP GET/HEAD 仅返回已存 torrent/magnet，不重新抓站、不创建任务、不泄漏上游定位，不缓存；重启句柄失效。磁力附件从本站读取后提供原生链接。

## 登录、诊断与目录

登录只接受已保存内置实例和 expectedRevision，最多 4 流程、同实例 1 个、含等候验证码总预算 5 分钟；持有固定规则 lease，共享正式站点会话。挑战绑定实例/账号/网络/规则，答案不落盘；图片最多 8 MiB，仅投影安全 MIME data URI。取消/过期/关闭取消并等待在途操作后释放，不能清掉新流程的挑战。loginSupported 仅表示规则声明 login，不表示账号有效，不能按隐私类型推断。

普通 testIndexer 使用稀疏草稿和独立临时会话，不保存配置、不污染生产 Cookie/缓存/任务；默认配置/认证，搜索和获取显式授权，最多 4 在途、每实例 1 个、3 分钟、20 条展示。Jackett 未独立观测的认证不标成功。

connectionOnly 检查不登录/搜索/取种：内置访问站点地址，Jackett 访问对应 Torznab caps。全局 4 执行名额、最多 1024 等候，三分钟含排队，排队不准备会话；普通诊断满额仍返回忙碌。clientId 活动订阅提供 QUEUED/RUNNING 快照，非持久后台作业。保存实例用 HMAC connectionRevision 校验真实连接参数，无关排序/配置变更不取消测试；草稿仍校验全局版本。

Jackett 目录由应用 worker 在启动、配置变更及每分钟检查，复用 5 分钟成功缓存/30 秒失败退避、合并与 30 秒截止。只在启用且连接完整时请求，不阻塞 HTTP 启动；失败保留同账号最近目录，发布 CATALOG 事件，Home 读取不联网。

## 执行历史与统计

每站搜索结束和资源获取返回前先保存统一 indexer_executions，失败返回 INDEXER_STATE_UNAVAILABLE。JSON 是真值，筛选/统计列由 SQLite 派生；不再写独立近期快照或聚合计数。报告只有稳定码、身份/版本、计数、独立耗时和关联 ID，不含上游原文、任意查询参数、凭据或定位 URL。

缓存命中保留调用上下文但不算实际执行；合并调用按实际关联 ID 去重；取消/未发请求不计统计。搜索均值只含可靠的 adapter 独立耗时，不估算旧数据，空样本或零均值为空；成功率/次数基于全部有效搜索。统计与记录数同读事务。

完整保留近 14 天，另保护仍配置站点最近 10 次有效执行，无全局条数上限。目录未知不当站点移除，保守保护已有 Jackett 最近执行。通用 diagnostics TTL 不触及本表。查询先按站点/结果/操作/时间/字面搜索筛选，再按 ID 游标分页，每页最多 100。

## 修改时验证

解释器验证固定语料/差分与预算；网络验证账号隔离、CookieJar、取消、429/冷却、迟到结果及 solver 不重放取种；引用验证配置变更、TTL、任务接受和重启；查询/附件缓存验证预算和私密字节不互换。真实站点与破坏性操作仍显式 opt-in，命令见[开发流程](../workflows/development.md)。
