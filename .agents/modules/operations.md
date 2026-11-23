# 下载器、配置与运行交付

核对日期：2026-11-19；代码基线：`ec7b093`；状态：当前源码与仓库工作流配置，不表示服务已更新或已发布。

主要源码：[应用](../../cmd/moji/main.go)、[设置](../../cmd/moji/settings_store.go)、[config store](../../internal/config/store.go)、[下载器](../../internal/downloader/service.go)、[managed 存储](../../internal/downloader/storage.go)、[HTTP](../../internal/controller/api/api.go)。索引网络、规则、登录与执行历史归[索引器模块](indexers.md)。

## 配置与客户端

配置优先 `-config`，否则 MOJI_CONFIG 或 config.yaml。当前 YAML v4，按 indexers/downloader/stash/tasks/system/logging 分组，权威映射见[设置说明](../../docs/settings.md)。存量旧配置的读取边界以 config loader 为准；启动不重写，成功保存和离线工具输出 v4，未知版本/混用字段及已移除网络档案拒绝。新配置变更不新增自动迁移或数据库回退。

config.Store.ApplyPatch 在同一锁内读取当前值、合并、验证、临时文件 fsync 与原子 rename，成功才发布内存，失败核对有效文件。typed patch 遗漏/null 不变，空串/列表/false 保留；列表替换，对象递归合并。连接 provider 后续调用使用已保存值；已接收任务的冻结配置不被覆盖。

pkg/stash、stashbox、jackett、qbittorrent、flaresolverr 负责协议，领域负责资格/编排。未配置 Jackett 不阻止启动；qB/Stash 缺配置限制相关能力。qB 首次登录失败允许启动后续重连，明确 403 才重新认证重试；写超时不自动重放。qB 5 start/stop 收到 404 回退 resume/pause。

testServiceConnection 用当前输入和短生命周期客户端，十秒截止、不保存或更新依赖状态，只返回稳定结果。Stash 读版本，Jackett API key 读配置站点目录，qB 独立会话读版本；这不证明各站搜索/取种可用。种子元数据按主机串行且至少间隔一秒，Jackett `/dl/<indexer>/` 分组，429/5xx/网络超时及相关 Jackett 404 退避并尊重 Retry-After；见[metadata host](../../internal/taskruntime/torrent_metadata_host.go)。

## 下载器观测与操作

后台每三秒独立读取 qB，包含已完成 Moji 任务种子；任务数据库同步失败时仍更新 qB 快照，保留旧关联并暂停自动清理，只保留最新一批待同步观测。页面读取缓存不等 qB 请求；快照写盘失败不丢当前进程新观测。断连保留最近完整成功状态，不推断所有种子消失。

启停、校验、汇报、限速与移除以之后的真实观测确认。缺失/error/missingFiles 只阻塞相关活动下载，不回退已入库任务；已完成任务源丢失为中性信息。添加拒绝只按已验证精确 info hash 核对，不按标题猜测关联。

详情分片由 [pieces](../../internal/downloader/pieces.go) 按需读取真实 `pieceStates`，读取限时 3 秒、并发 2，响应体上限 8 MiB，最多处理 200 万片并返回 512 段；不加入全量下载器轮询。Tracker 总数来自 `num_complete/num_incomplete`，活动时间来自 `time_active`，最后完整可见来自 `seen_complete`；旧观测缺失这些字段时保持未知，不与 Moji `LastSeenAt` 混用。

全项目 qB 状态复用[qbittorrentStateLabel](../../web/src/utils/qbittorrentState.ts)及[官方 fixture](../../web/src/utils/__fixtures__/qbittorrent-statuses.json)，保留 `[F]`、大小写和空格；Moji missing/removed 不冒充 qB 官方状态。free_space_on_disk 是默认下载盘可用量，字节和不证明准确磁盘占用，缺失显示未知。

## 自动清理与 qB 原生传输设置

自动清理唯一配置是 YAML `downloader.cleanup`，默认关闭；trigger 单选 AFTER_INGEST 或 CAPACITY，切换保留容量参数，下一轮读取生效。核心只保存执行状态和外部意图，无策略数据库回退或自动转换。分享率/做种时长仅可作为容量排序，不再触发 Moji 删除。

清理要求种子精确 hash 关联且全部关联任务完成入库，不自动清理未关联种子。删文件需验证独立交付副本、本地身份及没有其他 qB 内容路径交叠。手动移除未入库关联种子也受保护。共用内容检查统一由[content_isolation](../../internal/downloader/content_isolation.go)负责：未知路径的元数据/停止下载种子仅在进度、已完成量和大小全零，且文件 API 明确非 null 空列表时可豁免；单次列表最多 16 个文件查询、共两秒，证据不跨刷新。删文件前重新完整核对列表及文件，失败保留。

容量条件为默认下载盘剩余空间低于或等于十进制 GB 阈值，恢复至其上停止；仅处理 save path 与默认目录一致的候选，不猜多盘/自定义目录，不自动暂停下载。容量必须删源文件，按磁盘占用/分享率/做种时长降序、hash 破平。每轮最多一项，确认后观察六秒再读真实空间；连续三项无净增长暂停，不用种子大小假定释放量。

两种策略共用[cleanup](../../internal/downloader/cleanup.go)与 IDLE/RUNNING/WARNING/ERROR。准备状态先落核心，CONFIRMING 与删除意图同事务后才调用 qB；不明只核对同 hash，十五秒后静止警告。明确 4xx 拒绝（不含 408）失败并冷却三十秒，超时/5xx 不冒充明确失败。预算、截止和在途身份持久保存，runtime 丢失不重置；空间恢复或关闭策略重置保护，不变状态不重复核心写入。

Home 剩余空间旁 CleanupIndicator 只在执行/有限确认时动画，异常有颜色和原因，点击直接去设置，无详情窗口；失联/reduced-motion 停动画。配置、保护与 YAML 示例见[下载器说明](../../docs/downloader.md)。

“传输设置”保持 panel=speed，包含全局限速、qB 做种限制与排队；直接读写 qB，不存 Moji 配置。typed 稀疏 patch 经认证提交并回读，不一致返回未确认，缺失能力不伪造默认。qB 自身结束后删种不受 Moji 入库保护。

## Homepage Widget

GET /api/v1/widget/homepage 返回 attention、active、waiting、completedLast24Hours 四个非负任务计数；[用户配置](../../docs/homepage-widget.md)。使用 Homepage customapi，推荐三十秒刷新；沿现有访问边界，无专用 Moji key，不使用上游凭据。

[Handler](../../internal/controller/api/homepage.go)经窄接口读[HomeStats](../../internal/taskruntime/home_stats.go)，复用首页聚合，不读详情/联网；无下载器时 main 注入本地任务存储。成功/应用错误均 no-store，空库才四个零；查询失败 500、依赖缺失 503、五秒截止 504，错误只有稳定码/通用文案/关联 ID。GET 路由支持标准 HEAD，拒绝写方法。24h 使用持久入库时间闭区间，删除/重选随当前投影变化。首版隔离验收见[历史记录](../reviews/2026-11-11-homepage-widget.md)，不代表当前部署状态。

## HTTP、日志与发布

后端默认 :10000；/graphql 支持 POST、multipart、WebSocket upgrade，普通 GET 拒绝；另有 /playground、/healthz 和辅助 /api。MOJI_WEB_DIST 默认 web/dist，SPA 由 webui 提供。用户手动运行开发实例时，验证构建输出到临时目录，不能覆盖其静态资源或重启服务。

日志固定配置目录下 logs/moji.log，早期/加载后同路径解析；旧 logging.file_path 读时忽略、保存移除，旧文件不自动移动/删除。默认每份 10 MiB、保留 .1 至 .5；/api/logs/current 只读活动文件。实时日志/性能计数不是持久审计。详情见[设置](../../docs/settings.md#log-files-and-live-viewer)。

版本只来自[Git 标签/提交](../../scripts/version.sh)，Make/Docker ldflags 与 Web 一致，普通未注入 go build 为 dev。镜像含 Web、Moji、FFmpeg/ffprobe，非 root；Docker 构建 Go 1.26，源码最低 Go 1.25。示例配置当前 v4、内置实例为空、Jackett 关闭且无凭据；首装操作见用户文档。

CI 的原生 amd64/arm64 构建和 smoke 使用临时卷/最小配置，验证启动、持久化及优雅退出；Web/Go 构建使用 BUILDPLATFORM，Go 按 TARGETOS/TARGETARCH 且 CGO_ENABLED=0 输出，最终目标镜像实际执行 --version。交叉编译不能代替目标运行或完整 smoke。发布依赖 Web、Go、Docker、indexer-compatibility，通过后复用已测镜像制品，具体标签与命令见[Docker](../../docs/docker.md)及[开发流程](../workflows/development.md)。

## 修改时验证

客户端使用 fake/httptest 覆盖认证、明确拒绝、超时不明和重连。清理覆盖断连/未关联/未入库/共用文件/独立副本/意图失败/重启和未知结果；真实破坏性测试显式 opt-in。发布变更验证 version/Python 与相应镜像 smoke，不把本机或历史验收宣称为当前远端 CI/部署成功。清理设计与当时人工配置操作见[历史记录](../reviews/2026-11-13-downloader-cleanup.md)。
