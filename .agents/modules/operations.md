# 下载器、配置与运行交付

Home 规则作业提示：indexerdefs.CurrentJob 单次读取最新持久作业，不检查源/下载；admin 注入窄 DefinitionJobReader，只把当前 FAILED/INTERRUPTED 投影到概况并计入关注数，新的运行/完成/取消作业取代旧失败。CurrentJob 读取失败只给 definitionErrorCode，其他概况保留；调用取消仍返回 ctx 错误。GraphQL 复用 IndexerDefinitionJob，详情入口到规则库；初读/刷新不依赖曾收到订阅。实际临时归档失败→关闭重开 Manager→Home 首读恢复的测试验证 job/correlation 保留且源检查/下载计数不变。

健康场景已用临时 DB/真实应用装配及本地假 Jackett 验证：搜索成功与取种失败独立保留，取种显式恢复不改搜索观测；Home 只投影本地状态，无隐式联网。目录失联保留最后确认数量和时间，切换 API 账号返回未知且不沿用旧目录/观测。内置实例当前账号及活动规则版本的 Summary 查询读取相应命名空间，旧行保留而不投影到新账号/规则。缓存命中不刷新实际验证时间；这些离线/本地集成证据不等于真实站点可用性验收。

生产索引器使用 NewPersistentNetwork：共享网络 gate 按请求 origin＋代理协议/主机出口隔离，跨账号/规则共享同一出口限流，健康仍按账号配置/规则分别投影。每个新 gate 在首次联网前懒读 runtime v6，恢复未过期冷却与失败次数；存储不可读返回 INDEXER_STATE_UNAVAILABLE，HTTP 尚未发出。保存失败保留真实上游结果、内存退避和 dirty 标记，结束内存冷却后需先成功保存再准入，避免因持久化失败丢弃或重复取种。保存受 2 秒预算与 Network 生命周期等待约束，诊断按分钟限量且不带 SQL/URL。容量满不驱逐未过期记录；存储写失败或 runtime 被重建时不承诺重启恢复。普通成功零写入，成功/挑战解除清除既有记录；CF 解除仍遵守序号围栏，不覆盖新失败或 429。HTTPSession 返回的 429 等待时间使用 gate 实际指数退避截止；外部 Jackett 搜索/目录读取使用每次调用的响应 receipt，把首次 429 映射为实际 gate 截止；目录客户端原先只返回普通状态错误也不再丢失该限流信息。网关 HTTP 200 内某个远端站点的失败仍仅影响该站，不上升为整个网关的网络限流。诊断共享受控网络及其限流，和生产 Cookie/查询缓存隔离不是取消上游限流。

可信来源高级编辑复用站点 trustedOrigins 契约：每行完整 HTTP(S) origin（可带端口和结尾斜杠）、最多 16 个；拒绝路径、查询/片段、凭据和通配符，空列表明确撤销，遗漏保留。后端保存原子校验，草稿测试复用相同校验且不落盘。信任仅授予凭据/表单与 Cookie 会话使用，不授予私网或降级访问。HTTPSession 的 CookieJar 收发也按来源授权：不能仅删 Header 后让 net/http 自动向未信任同父域子站发送域 Cookie，也不能接受其覆盖账号 Cookie。显式信任仍遵守 Cookie 域和 Secure 规则；DNS/IP 校验对可信来源照常执行。

FlareSolverr 前置协议与网络层：新增 [pkg/flaresolverr](../../pkg/flaresolverr/client.go) v1 单次 request.get/form POST 客户端和 Network.newSolverClient 受控端点封装。solver URL 是明确配置的私有 origin 授权，不跟随服务重定向；站点 Cookie/代理凭据只在协议 JSON 中，HTTP 无环境 Cookie 或站点代理认证。JSON 封装最多 32 MiB、页面最多 16 MiB，整体求解截止及排队预算来自调用者，错误不携带上游正文/URL。以规范化端点的私有摘要复用本地客户端/连接，最多 16 个持有端点；同端点完整求解串行，最后引用释放或网络关闭后不可重开。不创建持久浏览器会话；当前上游单次请求和 sessions.create 使用相同代理初始化路径，认证代理同时发送 username/password 两键，包括其中一个为空的情况。HTTP 500 作为协议拒绝，和 429 网络冷却分开；控制调用的响应头等待使用求解截止，保留普通 HTTP 的 30 秒响应头限制。HTTPSession 已接明确 CF/DDoS-Guard 挑战检测：仅认证/搜索可单次求解，再用 Cookie 与浏览器 UA 配对验证原请求一次；浏览器只执行 GET 导航，不接收认证表单。取种与 before HTML 均不求解/重放。采纳前检查回复同 origin、Cookie 域/公共后缀/容量；正式会话持久恢复 UA 配对。自身 CF 503 冷却凭序号解除，不覆盖较新的失败或任何 429。父级截止约束求解和排队，配置冻结挑战超时（默认 90 秒，上限 300 秒）。外部浏览器属于显式信任服务：Moji 检查委托初始公共目标及最终来源，不能约束浏览器内部每次导航。挑战超时/solver 的配置 API/UI 已接：网络配置可仅 solver、仅代理或二者同时配置，读侧只返回配置状态；空输入保留、清除显式并检查配置版本。挑战超时 10–300 秒的稀疏更新与其他超时分离；create/update 使用相同 patch 输入。未做真实浏览器/代理/站点验收。

登录挑战由 [indexerlogin](../../internal/indexerlogin/service.go) 管理，并已接 cmd 生命周期和 beginIndexerLogin/answerIndexerChallenge/cancelIndexerLogin GraphQL；indexerLogin 只读查询恢复同一待回答挑战，不重新认证或取种。只接受已保存内置实例及 expectedRevision；编辑草稿仍使用站点测试。准备逻辑复用 admin 的只读校验，持有固定规则 lease；通过 indexer.LoginSession 共享正式寻源会话，Cookie 可用于后续搜索/取种。最多 4 个登录流程、同实例 1 个、等待验证码也计入容量、总预算 5 分钟；配置过期先拒绝回答；使用服务器确认的当前配置可清理旧版本流程并重新登录，过期版本请求不能借此销毁有效流程。取消/过期/服务关闭先取消并等待在途操作，再释放本流程的挑战和规则。清理完成前保留准入占位，旧清理不能删除新流程。ChallengeFor/AnswerFor/DiscardChallenge 绑定账号、网络、来源和登录规则的会话键；Close 只丢弃该登录实际观察的 challenge，不关闭共享生产会话。公开挑战 ID 使用流程 capability，不发内部 session challenge ID，答案只作为提交参数；图片最多 8 MiB，仅投影检测到的普通图片 MIME 的 data URI，不提供上游定位 URL，无法投影会取消流程。登录面板已接开始/恢复/回答/取消/到期交互；窗口关闭后空闲挑战可恢复，在途请求会中止。取消已过期流程返回幂等成功。solver 认证/搜索执行已接，配置 API/UI 已接，真实服务验收仍未完成。

临时会话的认证和验证码回答均不写会话数据库；回答后仅在该内存会话中保留 Cookie。正式会话仍保存派生 Cookie，答案不落盘；同一 challenge 仅可消费一次，重建生产会话不能恢复临时登录。

站点诊断由 [indexercheck](../../internal/indexercheck/service.go) 与 cmd 应用生命周期管理，testIndexer GraphQL 将当前草稿映射为稀疏补丁，校验 expectedRevision；新草稿才允许传 definitionId，已有实例不可借测试更换规则。准备阶段仅合并/校验草稿并持有规则 lease，不保存配置或创建实例。实例快照转换复用 indexeradmin.BuiltinSnapshot，实际适配器在共享受控网络上执行；测试使用独立临时会话，认证、搜索、下载分阶段，未观察到的 Jackett 认证不标成功。默认无搜索/下载，搜索和下载需明确范围；最多 4 个在途、每实例一个、3 分钟、20 条搜索展示。查询缓存、生产 Cookie、候选注册表和任务不承担测试状态。公开报告只有阶段、稳定错误码、版本与关联 ID，不返回资源定位符或凭据；登录挑战交互/solver 仍见 R-004 未完项。

规则更新策略归 indexers.builtin.definitions，用户在规则库页修改 check_interval_hours/auto_update，分别经窄配置命令与同锁 revision 确认。source=prowlarr、schema_version=11 固定；策略不修改安装状态。应用每分钟评估一次已安装规则的持久 lastCheckedAt，按配置周期（1–168 小时，默认 24）检查；默认手动应用。开启自动应用只影响后续到期检查，所有更新仍经过原有验证/激活流程；关闭不会取消已接收作业。首次安装仍是显式动作。

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前实现与仓库工作流配置。

主要源码：[应用接线](../../cmd/moji/main.go)、[设置编辑](../../cmd/moji/settings_store.go)、[config store](../../internal/config/store.go)、[下载器](../../internal/downloader/service.go)、[managed 下载器存储](../../internal/downloader/storage.go)、[HTTP API](../../internal/controller/api/api.go)。

## 外部客户端与配置

`pkg/stash`、`pkg/stashbox`、`pkg/jackett`、`pkg/qbittorrent` 处理协议；领域服务处理资格和编排。请求传递 context，有界并发和超时；凭据通过配置 provider 使用，不在源码或前端文案里硬编码。

配置路径优先 `-config`，否则 `MOJI_CONFIG` 或 `config.yaml`。2026-11-02 工作区已将 Jackett 改为可选依赖，无索引器时允许启动，索引动作返回未配置；qB/Stash 未配置时相关能力受限。qB 首次登录失败允许启动并由后续观测重连。YAML 按 indexers/downloader/stash/tasks/system/logging 分组；旧格式可读取但启动不重写。内置索引实施中的工作区现已支持 v3，成功保存与 moji-config 输出写 v3；v3 新增内置实例、规则库与网络配置、独立缓存设置及 Jackett 启用状态，v0/v2 读取保留既有启用状态和查询顺序。应用 provider、双向配置/规则协调门、规则库及站点管理 GraphQL 已接线；站点、规则库、外部 Jackett 以及代理/请求和搜索超时/缓存管理界面已接线；Home 已有同级索引器概况，规则更新策略和搜索迁移已接线；挑战/solver 配置已接，完整健康及真实服务验收仍未完成。未知版本或混用字段拒绝。字段映射与离线工具见 [设置说明](../../docs/settings.md)。配置 store 通过临时文件、fsync 和原子 rename 保存，失败核对实际有效文件；连接 provider 在后续调用读取已保存设置。

内置索引在规则解析后的适配层沿用用户确认的 Jackett 发布时间策略：未来日期限制为本次搜索完成后的统一 UTC 检索时间，历史日期与未知值保持原状；规则字段和资源定位符中的原始日期保留，供后续取种模板使用。纯 Cardigann 解析器不承担该宿主策略。

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

Docker 镜像含 Web、Moji、FFmpeg、ffprobe，非 root 运行；Dockerfile 当前使用 Go 1.26 构建镜像，源码最低版本由 go.mod 的 1.25 定义。运行数据挂载与四根映射按 [Docker 文档](../../docs/docker.md)，不要为了测试替换真实配置或重启现有服务。打包配置示例为 v3，内置实例为空、Jackett 关闭且无地址/密钥；首装规则、可选外部来源和故障操作见[索引器使用说明](../../docs/indexers.md)。smoke 使用无 Jackett 的最小 v3 配置，断网、只读根和临时卷；启动及容器替换后校验本地概况/未安装规则/无隐式检查，存储按 CURRENT 解析 generation，保留核心/runtime/缓存库、配置保存和优雅退出验证。

CI 在原生 amd64/arm64 runner 构建及 smoke test，通过后发布复用同一镜像制品；发布依赖 Web、Go、Docker 和 indexer-compatibility jobs。稳定标签、预发布、edge 的含义见根 README 和 [发布脚本](../../scripts/docker_publish.py)。未运行的架构/发布不能从本机测试推断成功。

Docker 的 Web/Go 构建阶段使用 BUILDPLATFORM；Go 保持 CGO_ENABLED=0，并按 TARGETOS/TARGETARCH 输出目标二进制。最终 Alpine 仍为目标平台，实际执行 moji --version 后才生成交付镜像，不能以交叉编译成功替代目标执行/完整 smoke。跨架构本地构建仍需目标运行阶段的执行能力；这一组织不在交付镜像中增加模拟器，CI 的原生架构验收不变。

## 修改时验证

客户端用 fake/httptest 覆盖认证、明确拒绝、超时不明与重连；真实 qB destructive 测试需要独立显式开关。

清理变更覆盖断连、无关联、交叉做种、独立副本、未完成入库、意图落盘失败与未知结果恢复。发布改动检查 version/Python 测试及相应镜像 smoke；具体命令见 [验证流程](../workflows/development.md)。

## 已记录的后续需求

设置、实际配置迁移及 Jackett 逐站搜索已交付，见 [专项复核](../reviews/2026-10-28-settings.md)。自动寻源的停止由 runtime 候选资格决定；手动搜索逐站收集，错误回退及取消有 fake HTTP 覆盖。逐站查询不等同于元数据下载主机队列。

2026-10-31 日志（O-001）：路径固定为配置目录下 `logs/moji.log`，启动早期和配置加载后共用路径解析，不提供用户路径配置；旧 logging.file_path 读取时忽略，成功保存时移除。按大小轮转沿用 `moji.log.1`（最新）到 `.5` 的默认保留方式，默认单文件 10 MiB。旧目录文件不自动搬迁或删除；保留范围只覆盖当前路径对应的文件组。`/api/logs/current` 读取当前配置的活动文件，不合并备份；日志页不是磁盘历史查询。升级与读取说明见[设置文档](../../docs/settings.md#log-files-and-live-viewer)，验证见[日志复核](../reviews/2026-10-31-logs.md)。

2026-10-28 增量设置：config.Store.ApplyPatch 在同一配置锁内读取最新已保存 YAML、合并、验证并原子持久化，成功后发布内存值；失败不提前更新内存。patchSettings 使用与 YAML v2 相同的 typed 可选字段，省略/null 保持原值，空串/空列表/false 明确保留。下载器 PatchPolicy 的 read/merge/write 同样与 SetPolicy 共用策略锁。

2026-10-29 连接测试：testServiceConnection 使用请求传入的当前字段创建短生命周期客户端，不写配置或更新运行依赖状态。Stash 读取版本，Jackett 使用 API key 调用 `/api/v2.0/indexers/all/results/torznab/api?t=indexers&configured=true` 读取站点定义，不查询各站资源；不能用需要后台登录会话的 `/api/v2.0/indexers` 来验证搜索密钥，qB 使用独立登录会话读取版本。请求受 10 秒 context 截止约束，结束释放闲置连接，结果只包含稳定枚举，不输出上游响应原文或凭据。协议及认证/错误/超时由[httptest](../../internal/graphqlapi/connection_test_service_test.go)验证。
