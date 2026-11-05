# Web UI 与交互约定

索引器管理区分“添加”（保存实例）与“启用”（参与搜索）。目录保留已添加规则并显示启用状态，允许明确添加另一个账号/镜像配置；未添加、无声明式登录流程的公开规则可用“添加并启用”保存默认配置。站点和外部 Jackett 启用状态复用 SettingsSwitch 与现有 switch 样式，组件显式引入样式，不依赖先访问其他设置页。站点隐私类型使用红色 private、绿色 public、黄色 semi-private 背景标签，并提供文字和深色主题对照。CRUD 使用有 accessible name/title 的 FontAwesome 图标；移除仍须确认。规则 loginSupported 为 false 时不显示登录入口，不能以 public/private 推断登录能力。

站点目录和移除确认通过 body Portal 挂载，避免设置卡片的裁剪/定位上下文遮挡。useIndexerReorder 以可见手柄启动桌面原生拖拽；原生鼠标 dragstart 后的 pointercancel 不撤销拖拽，仅取消正在捕获的触摸/笔流程。保留移动触摸、键盘上下按钮和保存期间禁用；排序仍通过窄命令持久化配置版本。浏览器回归使用真实鼠标/触摸，并检查 switch 轨道/隐藏原生输入的实际样式，而不只断言控件 role。

Home 规则作业入口使用 homeIndexerSummary.definitionJob/definitionErrorCode 的持久快照，状态恢复来自后端；HomeIndexerDefinitionEvents 仅订阅 stream/revision 和 job id/revision 以触发合并重读，不拉完整已安装清单，也不直接拼接瞬时事件。刷新/重新可见同时重开规则订阅，连接代次变化补读；订阅错误保留概况并可重试。只显示当前失败/中断，后续作业取代旧失败；中断原因去重，长关联标识可换行，查看规则库不触发更新动作。局部作业读取失败保留其他站点信息，关注总数不重复展示该作业。

内置站点编辑器“高级设置”已接额外可信来源。原始多行草稿参与离开保护；格式/16 项限制错误阻止保存和测试，已保存来源回填，修改时显式替换，清空时提交 []，未编辑不提交。测试复用当前草稿但不保存。站点编辑器与放弃确认 Portal 挂载，标题和保存栏有视口断言；textarea 使用现有主题输入样式，来源在 RTL 下仍从左到右显示。中英文案和帮助说明凭据授权、Cookie 域规则、内网边界及已接受任务保留固定参数。

保存的内置站点通过 IndexerLoginDialog/useIndexerLogin 登录；外部 Jackett 仍由 Jackett 管理。挂载仅 network-only 读取待回答挑战，begin/answer/cancel 均由明确操作触发。窗口关闭后空闲挑战可重开/刷新恢复；答案提交即清空，不写本地存储或设置。当前实例/配置作用域变化中止旧请求并丢弃迟到结果，响应的实例/配置/挑战 ID 也校验归属。到期只改变界面，不轮询/自动重登；重新登录先清理旧流程，取消可中止在途回答，旧成功不能覆盖取消状态。取消已过期流程是幂等成功。验证码为文本或本站 API 投影的图片，图片加载失败提供重登，长站点名换行；新面板复用 Portal/模态焦点栈与固定操作栏。规则无独立登录流程只说明无需单独登录，不推断无需凭据。

站点列表和内置站点编辑器通过 IndexerTestDialog/useIndexerCheck 显式调用 testIndexer；编辑器传递与保存相同的当前稀疏草稿，测试不保存配置。默认只检查配置/认证，搜索需关键词，资源获取单独选择并提示配额；每阶段分别显示，Jackett 认证保持未单独验证，未执行搜索不显示零结果。输入、草稿或范围改变立即清除旧结果并中止在途请求；停止/关闭及迟到结果受请求归属保护，不自动测试。面板通过 body Portal 避开设置页层叠/裁剪，使用已有模态焦点栈，关闭后恢复编辑草稿与焦点。

规则库面板的 IndexerDefinitionPolicy 区域通过 useIndexerDefinitionPolicy 读取 network-only 配置并订阅配置事件，编辑器复用 useSettingsAutosave 的串行稀疏 patch、准确确认 revision 与草稿保护。检查周期为 1–168 小时，自动应用默认关闭；策略保存不会安装规则或立即发起检查。冲突后保留输入，只有确认载入当前设置才丢弃草稿。源/schema 是只读固定属性，活动版本继续来自 manifest/core 状态。

旧 Jackett 搜索/预览/目录/阻塞候选操作及缓存类型已移除，设置策略编辑器只负责任务、资源规则与 Stash 元数据三个实际路由。索引排序使用当前站点管理面板及其 reorder hook，旧无路由分支和排序组件已删除。ServiceCard 仅保留当前 Stash 诊断内容，不再实现固定 Jackett/qB 状态卡；HomeIndexers 和 HomeDownloader 继续分别持有真实索引与下载观测。

任务选源由 useTaskIndexerSearch 管理 URL/操作，TaskIndexerSearch 表达交互，手动磁力和上传路径保留。sourcingTask/run/page/sort 保存恢复范围，参数更新保留原列表参数和 backgroundLocation；刷新不重搜。首次打开沿用自动搜索，服务端按任务与范围复用 run；外任务结果不可选择，过期不隐式重跑。选择只提交 ID 与当前 revision，错误保留候选并有界回读最新任务；已接收提示不宣称下载完成。桌面选源抽屉使用稳定视口高度，避免异步结果改变面板位置。Home 搜索发送统一 indexers 模式，旧 session 的 jackett 值在读取时转换。usePreparedTorrentDownload 管理显式准备、取消、失效和作用域保护；发现/任务候选共用 PreparedTorrentDownloadActions。种子由本站短期 attachment 下载；磁力通过本站准备附件读取后展示原生链接，界面不跳到私有定位 URL。过期只移除当前入口，不自动获取；准备失败保留候选和任务选择。

2026-11-02 发现页已迁移到持久资源搜索：索引器模式汇总内置/Jackett，本地统一目录选择范围，明确 start 才发起搜索；run ID/范围/排序/页码进 URL，刷新、返回、过期和取消均按服务器状态恢复。候选只传 searchID/candidateID 和用户确认的番号入队，不回传标题/URL 替换来源。新增 previewTorrentSelection mutation，从终态 run 一致读取最多 1000 条服务端候选后交现有 taskruntime 规则预览，再排序分页；跨页 LOCK、未知与零值、空预览及资源归属均保留。文件预览明确触发、沿用检查预算与原始数据缓存，刷新不隐式获取；旧结果为空不回退原始候选。旧 Jackett 搜索/预览/目录 hooks 和筛选组件已无生产消费者并删除。StashBox 创建流程保留，任务抽屉已迁移，服务器下载句柄及其显式界面入口已接线。

2026-11-02 搜索数据层已新增 torrent-search 操作源、useTorrentSearch 与 useTorrentSearchEvents；挂载/刷新/重连只读取已有 run，显式 start 才创建工作，取消回读真实状态，入队绑定当前 run。通知按 run/stream/sequence 过滤，200ms 合并定向回读；切换 run 清理旧计时器，旧操作结果不能替换新 run 的错误。Graphcache 将分页 run 作为独立快照嵌入，避免相同 ID 的不同页面互相覆盖。中英错误及导出 locale 已更新。发现页和任务抽屉均已切换到这些 hook；界面验证按最新阶段证据。

最近边界核对：2026-11-01，代码基线 `71ef6fa`；功能实现证据见文内日期专项。

2026-11-02 索引器第四页签“网络与缓存”沿用 ?panel=cache。缓存开关、成功/空结果保留时间及容量上限已接窄更新与版本检查，支持聚合状态、全部/指定站点清理及确认。新增 IndexerNetworkPanel、网络配置创建/修改/删除 Drawer 及请求/单站搜索超时自动保存；代理 URL 视作凭据，读取只返回是否配置，留空保留、显式清除，名称创建后不更改，直连不可改删，引用中的配置不能删除。编辑固定打开时的版本，后台通知不授权覆盖草稿；成功刷新站点网络选项。共用 SettingsAutosaveProvider、直读客户端、Drawer 与语义样式，失败保留输入，超时冲突可确认载入当前值。CACHE 仅刷新缓存区，网络区响应 CONFIGURATION/重连，不新增轮询。规则策略、FlareSolverr 地址及 10–300 秒浏览器验证超时已接。网络配置可仅设置 FlareSolverr，读取只返回是否配置，地址留空保留，独立勾选清除；自动保存沿用相同版本保护。服务根地址提示说明 Cookie/代理凭据委托及取种不自动重试，帮助记录外部浏览器信任边界。代理/solver 地址在 RTL 下仍从左到右显示；网络编辑与移除面板 Portal 挂载到 body，避免父卡片限制定位，保存栏有视口断言；代表性真实服务验收与未测环境限制见 [完成记录](../reviews/2026-11-03-indexer-acceptance.md#功能验收完成2026-11-04)。

站点观测由 Home 的 components/indexers/IndexerObservationView 显示，搜索和取种分别保留上次结果、真实时间、稳定原因、等待时间及按需关联标识；设置站点项不再显示错误和运行记录。Home 成功计数只表示上次搜索成功，不代替取种/完整下载成功；无匹配配置/规则观测与观测存储不可读分别表达。观测组件独立加载样式，现有事件触发局部回读，不增加轮询。

2026-11-02 索引器事件：Home 和索引器设置页复用 useIndexerEvents，接收仅含 streamId/sequence/kind 的失效通知，200ms 合并重读局部快照；新流/序号缺口/重连不依赖历史重放，离开取消待处理定时器。规则事件复用同一 Home 合并入口；操作级订阅失败局部呈现，重试同时补读和重开失败订阅。配置/目录/搜索及取种完成通知已接线；健康观测见上段，Home 已显示当前失败/中断规则作业并链接规则库。

2026-11-02 Home 索引器基础接线：HomeIndexers 与 HomeDownloader 同级，系统详情只保留 Stash/交付；HomeServiceStatus 不再取固定 Jackett 状态及两者冗余统计，首次配置不要求 Jackett。新增独立有界本地概况查询，显示启用内置站点、最多五项配置/规则异常及配置且启用的外部网关；未知目录不当 0，过期保留最后确认数与时间。hook 保留最近成功内容，规则/配置/目录/运行状态事件合并刷新、重连及重新可见补读，没有新增定时轮询；规则失败作业入口已补齐。完整验收与限制见 [完成记录](../reviews/2026-11-03-indexer-acceptance.md#功能验收完成2026-11-04)。

2026-11-02 索引器管理：站点、规则库、外部 Jackett 入口已接真实管理 API；旧 queryOrder 参数仍可进入统一站点排序。IndexerManagementProvider 共享站点直读与规则订阅，安装/修复和连接保存后刷新站点；设置客户端无 graphcache，不能只配置全局缓存更新。动态凭据仅提示已配置，未提交保留、显式清除；编辑离开保护、目录分页、拖动/触摸/键盘移动、规则作业刷新恢复及跨进程 streamId 边界有测试。测试/登录挑战与 Home 健康的功能验收已完成，实际范围与限制详见 [完成记录](../reviews/2026-11-03-indexer-acceptance.md#功能验收完成2026-11-04)。

新增、重构、修复与文案修改必须遵守 [产品体验规范](../standards/product.md) 和 [工程规范](../standards/engineering.md)。本文件描述当前 Web 契约与功能证据，具体控件/视觉/文案的通用要求只在规范中维护。当前样式或旧截图不是保留无效文案、另造设计或跳过产品复核的理由。

主要源码：[路由](../../web/src/app/routeConfig.tsx)、[providers](../../web/src/app/AppProviders.tsx)、[GraphQL client](../../web/src/graphql/client.ts)、[graphcache](../../web/src/graphql/cache.ts)、[AppLayout](../../web/src/app/AppLayout.tsx)。

## 结构与入口

React 19、React Router 7、TypeScript、Vite、urql graphcache、graphql-ws、i18next；组件/styles 沿用当前体系，不引入并列页面框架或第二个全局实体缓存。

| URL / 入口 | 职责 |
| --- | --- |
| `/` | 注意事项、任务进度、完成与检查摘要、下载器和依赖状态 |
| `/tasks`、`/tasks/:taskId`、`/tasks/:taskId/resolve` | 有界任务列表、详情、寻源干预和处理台 |
| `/performers`、`/performers/:performerId` | 演员、订阅、检查和场景选择 |
| `/performers/review` | 发行决定，放在动态 performer 路由前 |
| `/discover` | StashBox 与 Jackett 发现和手动创建 |
| `/stats` | 统计入口 |
| `/settings/:section` | 分区设置；`/settings` 跳索引器；旧 connections/ingest/automation 路由分别重定向 indexers/stash/tasks |
| 帮助、种子列表、数据库性能 | 共享抽屉/弹窗；性能入口需已保存开发者模式 |

路由层持有 URL 和组合 hooks，页面/组件负责显示与交互。GraphQL documents 按页面/设置 tab 拆分；详情、候选、文件与历史按需加载，不恢复早期巨型 dashboard 查询。

## 操作界面的目标

首先回答“有没有事情需要处理、任务到哪了、下一步做什么”。Home 以注意任务和真实进度为中心，服务配置在辅助区域。详情打开后保留来源页面、筛选和滚动上下文；Home 可通过 `task/check/resolve/files` 参数恢复相应抽屉。

长任务展示后端持久状态，刷新或另一浏览器打开仍能识别活动执行。按钮 pending 防重复请求，不代替后台 run/reservation。下载进度只在真实观测之间过渡，其他阶段显示 phase，不做假百分比。

异步交互给出加载、成功、部分成功、失败和可继续的动作。刷新失败保留可用旧数据，标明需要更新；不要让一个摘要失败隐藏整个页面。

## 数据同步按领域选择

| 领域 | 当前策略 | 恢复 |
| --- | --- | --- |
| 任务 | summary/detail 快照 + taskEvents + 有界传输日志 | stream/sequence 检查、挂载消费者定向读取 |
| Home | 独立有界投影，事件合并刷新，前台 30 秒校准 | 挂载、恢复可见性及断线重试 |
| 订阅检查 | 共享可见性轮询：活动 2 秒、空闲 10 秒 | 重新读持久 run/reservation |
| 下载器 | 3 秒读取后台快照，详情按需 | 旧快照 + 连接/过期状态 |
| 数据库性能 | WebSocket 最新值推送；隐藏/关闭暂停 | 重连直接获最新缓存，15 秒或错误标过期 |
| 订阅实体、服务和日志 | 各自 GraphQL subscription | 按现有 hook 的边界恢复 |

不为每个组件另加 timer；扩展已有领域 hook。也不把“禁止任何轮询”作为全局规则。任务快照协议详见 [任务](tasks.md)，Home 细节见 [主页说明](../../docs/home-workspace.md)。

设置使用分区客户端与 SettingsAutosaveProvider：重新进入读取新设置，独立 loading/error 不污染待保存状态。字段编辑立即更新本地值，500ms 防抖发送 pending 增量；同一表单串行提交，确认只清除本次已提交的改动，失败保留并可重试。分区切换不取消队列，不将凭据或 pending 写入浏览器存储。成功后再以持久设置控制全局能力，例如语言和开发者模式按钮。

## 文案与无障碍

- 支持 zh-CN/en 和 auto；qps-ploc/ar-XB 用于开发验证。资源权威内容在 `web/src/i18n/resources.ts`，公共 JSON 通过 export 同步。
- 用户错误根据 `extensions.code/params/correlationId` 本地化；未知代码有安全 fallback，原始上游诊断放日志。
- 检查动作沿用 `faBinoculars`，qB 状态走统一官方映射。新 UI 需中文、英文、深浅色、移动端和必要的 pseudo/RTL 验证。
- `Drawer/useModalFocus` 已有 dialog、焦点圈定、Escape、嵌套 modal 与返回焦点。旧“抽屉完全缺无障碍”结论已失效；扩展时复用该能力。
- ThemeMenu 使用 menuitemradio，打开聚焦当前选项；支持方向键循环、Home/End、Escape 返回触发器，选择后返回焦点，Tab 离开关闭。
- icon-only 按钮必须有 accessible name；不嵌套交互按钮；状态不只依靠颜色。破坏性操作说明后果并确认，服务端仍重验证资格。

## 修改时验证

纯格式调整通常 build 即可；交互/数据契约变化按最低有效边界测 hooks、组件、路由或 graphcache。跨页选择测提交变量与部分结果，实时变化测迟到响应/断线/缺口，不能只做静态截图。

可见布局变化运行相关 Playwright spec，在固定 CI 环境复核截图；不为接受意外差异提高容忍值。命令与环境见 [开发流程](../workflows/development.md)。

## 已记录的后续需求

设置结构、独立分区保存、固定排版与实际配置迁移已完成，见 [设置复核](../reviews/2026-10-28-settings.md)。当前 tab 为 indexers/downloader/stash/tasks/system/logs/about；共享 shell 与字段组件统一状态、说明、加载和敏感值显隐。后台四种资源效果与逐站停止语义见 [任务](tasks.md)。

日志（O-001）按旧到新显示最近 200 条匹配记录并向下跟随。上滚超过底部 24px 或点击暂停时冻结当前显示列表，WebSocket 与 200 条有界缓存继续更新；恢复按钮回到最新列表底部。切换级别重置跟随，重连/缺口沿用快照恢复；复制采用当前可见顺序，下载是活动文件全部级别。日志区支持键盘滚动、暂停时关闭 live 宣告，移动端消息换到第二行。证据见[日志复核](../reviews/2026-10-31-logs.md)。手动候选链接和任务种子分区已完成，见[种子交互复核](../reviews/2026-10-29-task-torrents.md)。

2026-10-28 设置跟进：任务删除策略位于“任务与订阅”，独立保存。语言选择仅在“系统”，写入 `system.language`（auto/zh-CN/en），成功保存后生效；初次和窗口恢复焦点读取已保存配置，并防止迟到响应覆盖新保存值。

设置分区加载/错误标题来自当前服务或业务分区映射；禁止引用退休的 connections/automation/ingest tab 翻译。当前共享编辑器为 SettingsPolicySection/PolicySettingsEditor，连接和交付分别为 ServiceConnectionSettings/LibraryDeliverySettings；公共兼容字段在 GraphQL documents 边界使用业务别名。

2026-10-28 自动保存：表单统一使用 useSettingsAutosave 与 SettingsSaveStatus，不再显示保存按钮。普通配置通过 typed patchSettings 的 YAML v2 字段增量更新；下载器策略使用可选字段输入，速度限制仍为单字段命令。列表按整体替换、对象递归合并；清缓存、刷新来源等副作用仍需显式操作。

2026-10-28 顶栏（U-005）：沿用 Stash 的导航分组和简洁选中态，顶栏配色使用项目主题变量：浅色为白色表面、深色为深蓝灰表面，正文/次要文字/选中及焦点使用 text/muted/accent；选中仅改变图标和文字颜色，不使用胶囊背景、边框或下划线。桌面显示品牌、主导航和右侧工具；小于 1000px 主导航改为展开按钮，面板使用两列并允许长文案换行。右侧按性能监控、统计、设置、帮助直接排列，主题随后；性能仅在已保存开发者模式下显示。所有宽度均直接显示工具，没有“更多”和语言入口；语言仍在系统设置。600px 以下导航按钮隐藏可见页名以给常驻工具留空间，保留完整无障碍名称。路由链接保留 aria-current，Escape 返回导航触发器，外部点击和路由切换关闭；主题使用 menu 键盘模式。实施参考与验证见[导航复核](../reviews/2026-10-28-navigation.md)。

2026-10-29 设置布局：根据用户限定的 Handoff 原型范围，使用 218px 桌面贴边导航与右侧最大 960px 配置面板；设置工作区占满顶栏下方视口，侧栏到底，右侧独立滚动；700px 以下使用顶部横向导航。保留全局顶栏，不添加设置 Logo、面包屑、顶部服务状态或侧栏版本。二级页签通过 `?panel=` 直达并支持前进/后退；分区仍为原七项，未新增概览或仅 YAML 编辑能力。页签具有 tablist/tab/tabpanel 语义、方向键/Home/End 及 RTL 支持。SettingsSubpanel 使用 hidden 保持表单挂载，继续使用原分区队列、错误和自动保存；从侧栏切换业务分区时队列仍由 SettingsAutosaveProvider 持有。共享表单采用 26px 页面标题、14px 分组标题、12–13px 字段与说明；媒体交付按两个视角配对展示下载及媒体库路径。详见[布局复核](../reviews/2026-10-29-settings-layout.md)。

2026-10-29 设置跟进：说明只保留功能范围、非直观行为和操作后果，并以领域源码核实；不重复标题、不显示空闲或已保存提示，保留保存中与失败重试。服务连接移除启用胶囊与后台状态订阅，三个面板通过 testServiceConnection 测试当前填写内容；输入修订后丢弃过期结果。开关在设置作用域统一几何变量，正常 38×22、小号 32×18，输入、轨道和滑块共享尺寸。图片缓存关闭仅禁用新图片落盘，已有缓存仍被清理，因此容量/保留期继续可编辑。文案依据见[审核记录](../reviews/2026-10-29-settings-copy.md)。

Stash、qBittorrent 和 Jackett 服务连接表单的测试结果仍为无背景纯文字，返回后 5 秒自动清除；改动输入或新测试隔离旧结果。站点列表和配置窗口改用按钮内的直接连接检查，不再打开搜索/取种诊断对话框：testIndexer 的 connectionOnly 模式禁止同时请求搜索或取种，内置站点访问站点地址，Jackett 请求对应站点的 Torznab caps；均不登录，也不证明搜索或取种可用。沿用受保护的索引网络、超时和错误码，短期结果只存于当前界面内存，配置版本或草稿变化后失效，离开设置取消请求。测试全部包含已停用站点、最多两站并行，每站独立结果；不创建下载任务或保存草稿。

索引站点顺序支持桌面原生拖拽、移动触摸手柄及手柄上的上下方向键，不再提供重复的上下按钮；落定后自动保存 `indexers.query_order`。站点名称承载 HTTP(S) 链接，语言和公开/私有/半私有类型同行；来源标签固定在各屏宽的项右上角（RTL 为逻辑尾侧），窄屏操作移到名称下方。配置使用扳手图标，项内图标按钮统一 26px 正方形，文字测试按钮等高、宽度随结果自适应。目录分类取当前规则 caps.categories 与 categorymappings.cat 的去重并集，已添加状态位于操作下方；添加与启用仍是两个动作。类型标签使用绿/红/黄圆角矩形，同时适配显式及自动深色主题。

设置右侧由全宽 `.settings-main` 承担滚动，内部 `.settings-main__content` 保持最大 960px 和原响应式留白，滚动条位于工作区边缘。添加站点弹窗使用 Drawer 的局部 className，头部与滚动内容分别留白，让内容滚动条贴弹窗边缘；不改变其他对话框的布局。站点区头部提供添加、刷新 Jackett 站点、测试全部。Jackett 页复用连接表单及原增量自动保存，合并为一张连接卡；总开关文案为“启用”，作用说明复用 FieldLabel 的 hover/focus tooltip；链接回统一站点列表，产品文案统一称 Jackett。

2026-10-29 种子交互（U-003/U-004）：任务寻源处理（含处理台）与发现页共用 TorrentCandidateLinks/CandidateSeeders。分别提供磁力、种子下载、站点详情；只使用上游原始地址，拒绝非对应协议、相对地址及 userinfo，HTTP(S) 链接以 noreferrer/no-referrer 打开，不预取、不展开含授权参数的下载地址，也不将下载地址推断为详情地址。链接与选用按钮独立；无链接明确提示，缺失/null/负数做种数显示未知，真实 0 仍显示 0。Jackett 解码以 -1 保留未知数，GraphQL 搜索结果仍沿用整数契约。

任务详情由 TaskDetail 将单一 DownloadManager 作为 torrentControls 插槽放入 TaskFiles 的种子信息中，位于静态标识与文件组之间；TaskDrawer 不再额外渲染。内嵌模式去除第二层卡片与当前任务回链，保留 qB 名称、官方状态、真实进度、刷新、失联/缺失提示及危险操作确认；仍只使用原有一组查询与可见性轮询。独立文件选择抽屉不加载下载器观测。

2026-10-31 待发行专区位于发现页搜索与推荐之间，复用主题与 Drawer；领域 WebSocket 事件合并后定向重读当前分页，重连、焦点及恢复可见性重新读取，无新增定时 HTTP 列表轮询。section=upcoming 和 upcomingSearch/upcomingPerformer/upcomingDays/upcomingPage 保存展开状态与筛选。日期按服务端业务时区显示，核验失败保留内容，已移交记录移出列表。没有下载按钮，预告片尚未接入。实现：[UpcomingReleases](../../web/src/components/discovery/UpcomingReleases.tsx)。

2026-10-31 时区跟进：系统常规增加发行日期时区输入，auto/留空跟随运行环境，手动值通过 typed system.release_timezone 增量保存并验证。服务端返回 effectiveReleaseTimezone 和待发行 today；相对天数使用服务器日历日期，包括无法映射为浏览器 IANA 名称的系统 Local 时区。

2026-10-31 待发行 UI：卡片复用演员详情的 card-grid / performer-scene-card 主题、16:9 封面和字级，共享 SceneArtwork 处理空封面及加载失败；演员链接与详情按钮独立。只显示番号、日期、两行标题和演员，异常才显示待确认。详情使用 Drawer media 变体与播放器展示区，保留完整标题、预计日期、演员、来源及刷新图标，预告片尚未接入；沿用模态焦点约束、Escape 和焦点恢复。移交后仅 QUEUED 决策显示发行复核入口。

2026-10-31 演员详情新增 upcomingState：状态来自核心库记录，非浏览器日期推断。卡片原状态位置显示待发行、待确认或发行处理中；创建入口和本页全选排除这些影片。已选项在可见快照转为待发行后移除，其他分页选择保留。usePerformerScenes 复用领域 WebSocket，按演员过滤并合并 200ms 内的事件后重读当前分页，复用重连/丢事件校准，不增加列表轮询。
