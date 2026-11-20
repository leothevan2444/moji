# Web UI 与交互约定

核对日期：2026-11-20（本轮主页与任务详情）；代码基线：`226da06` 及本轮工作树；状态：当前源码契约。跨页面体验要求统一见[产品规范](../standards/product.md)，这里定位已有能力与数据边界。

主要源码：[路由](../../web/src/app/routeConfig.tsx)、[providers](../../web/src/app/AppProviders.tsx)、[客户端](../../web/src/graphql/client.ts)、[缓存](../../web/src/graphql/cache.ts)、[布局](../../web/src/app/AppLayout.tsx)。React/Router、urql、graphql-ws 与 i18next 沿现有体系使用，不引入第二套状态或界面框架。

## 路由与共享入口

| URL / 入口 | 职责 |
| --- | --- |
| `/` | 注意任务、进度、订阅与检查、索引器、下载器和 Stash |
| `/tasks`、`/tasks/:taskId`、`/tasks/:taskId/resolve` | 有界列表、详情、人工寻源与处理台 |
| `/performers`、`/performers/:performerId` | 演员、订阅、检查、跨页选片 |
| `/performers/review` | 发行决定，先于动态演员路由 |
| `/discover` | StashBox 影片、统一索引搜索与待发行 |
| `/stats` | 统计 |
| `/settings/:section` | indexers/downloader/stash/tasks/system/logs/about；默认 indexers；旧 connections/ingest/automation 重定向 |
| 帮助、种子列表、数据库性能 | 共享抽屉；性能入口受已保存开发者模式控制 |

路由组合 URL 和领域 hook，组件负责交互，纯转换就近复用。GraphQL 操作按页面/设置分区；详情、文件、候选和历史按需读。页面状态可返回，打开抽屉保留来源、筛选与滚动。任务创建必须经影片确认，见[任务模块](tasks.md)。

| 能力 | 首选实现与边界 |
| --- | --- |
| 主题/静态 token/共享表面 | [theme.css](../../web/src/styles/theme.css)、[_tokens.scss](../../web/src/styles/_tokens.scss)、[_mixins.scss](../../web/src/styles/_mixins.scss)；语义前景 success/danger/warning/info 统一由主题提供，shared mixin、首页、索引器和日志直接消费，不再各自覆写明暗色 |
| 缓存清理确认 | [ConfirmationDialog](../../web/src/components/common/ConfirmationDialog.tsx) 包装 Drawer/Portal，图片、StashBox 和索引搜索缓存共用；请求中禁用确认/取消/Escape/遮罩关闭，失败保留面板并允许重试，取消不执行操作 |
| 抽屉与模态焦点 | [Drawer](../../web/src/components/layout/Drawer.tsx)、[useModalFocus](../../web/src/hooks/useModalFocus.ts)；dialog、Escape、嵌套栈和焦点恢复 |
| 设置外壳/字段/状态 | [SettingsPanels](../../web/src/components/settings/SettingsPanels.tsx)、[SettingsTabCommon](../../web/src/components/settings/SettingsTabCommon.tsx)、SettingsSwitch；PolicySettingsEditor 已复用 FieldLabel |
| 说明浮层 | [Tooltip](../../web/src/components/common/Tooltip.tsx)、SettingsInfoTooltip；Portal、边界定位、Escape；任务删除缺少关联提示沿用原生 title |
| 图片与候选链接 | SceneArtwork、[TorrentCandidateLinks/CandidateSeeders](../../web/src/components/tasks/TorrentCandidateLinks.tsx)、PreparedTorrentDownloadActions |
| 日期/容量/qB 状态/用户错误 | [formatters](../../web/src/utils/formatters.ts)、[qbittorrentStateLabel](../../web/src/utils/qbittorrentState.ts)、[queryError](../../web/src/services/queryError.ts)；时间戳/仅时分秒分别使用 formatDateTime/formatTime，遵循应用语言及 system.timezone；时间消费者以 useAppTimezone 订阅变更，不依靠重挂载页面；发行日仍保留日历语义，不能从浏览器日期推断发行资格 |

顶栏保留品牌、主导航与常驻工具。小于 1000px 展开两列导航，600px 以下隐藏可见页名并保留可访问名；aria-current、外部关闭、Escape 与路由关闭保持。ThemeMenu 使用 menuitemradio，方向键/Home/End、Tab 离开、选择/Escape 返回焦点；不套用已废弃的通用 listbox Menu。

## 设置、草稿与连接测试

设置桌面 218px 侧栏，右侧全宽滚动、内容最大 960px；700px 以下横向导航。`?panel=` 支持深链/前进后退，tablist/tab/tabpanel 有方向键、Home/End 和 RTL。SettingsSubpanel 用 hidden 保持当前分区表单挂载，跨业务分区队列由 SettingsAutosaveProvider 持有。

重新进入使用新鲜读取。useSettingsAutosave 在 500ms 后提交 typed 稀疏 patch，串行发送，只确认本次字段，晚响应不覆盖新输入；失败保留草稿并提供 SettingsSaveStatus 重试，空闲/成功不常驻提示。对象合并、列表替换、遗漏/null 不改，false/空值不能丢。配置冲突只有确认载入当前设置才丢弃草稿。副作用仍需显式操作，凭据/待保存项不写浏览器存储。

系统时区通过 IANA 下拉框写 system.timezone；useInterfaceSettings 初次/焦点读取并隔离旧响应，成功保存才更新全局时间显示，失败保留草稿及当前时区。任务推进频率只存 HIGH/MEDIUM/LOW（30 秒/1 分钟/5 分钟），默认 MEDIUM；订阅检查无独立启用开关。语言只在系统设置并写 system.language，保存成功后生效；初次/恢复焦点读取，不让旧响应覆盖新设置。任务删除策略归任务与订阅；qB 原生传输设置直读写 qB，Moji 自动清理在 YAML。PolicySettingsEditor 当前只负责任务、资源、元数据三个分区；进一步职责收窄见[A-001](../TODO.md#a-001)。

服务连接测试使用当前输入，不保存，结果纯文字并五秒消失，输入变化清旧结果。站点连接测试不等于认证或完整诊断，后端预算见[索引器](indexers.md)。useIndexerConnections 独立管理各站请求/结果，不阻塞其他配置动作；配置事件只取消 connectionRevision 改变或已移除站点，离页取消全部所属请求。

站点“添加”与“启用”不同；目录可明确添加另一账号/镜像，未添加且无登录流程的公开规则可添加并启用。排序由 useIndexerReorder 处理桌面原生拖拽、移动手柄和上下方向键，落定保存 query_order。图标有名称、移除有确认，隐私标签兼顾文字及明暗主题。目录/编辑器/确认 Portal 避免裁剪。trustedOrigins 原始草稿参与离开保护和测试，不保存测试草稿。

代理/FlareSolverr 共用全局网络；密码只写、空值保留、显式清除，关闭保留参数。规则库策略也用同一串行草稿保护。登录打开仅读待答挑战；显式开始/回答/取消，答案提交即清空，关闭后可恢复空闲有效挑战；配置/实例变化取消旧请求，过期不自动重登。完整能力归索引器模块。

## 数据同步

| 领域 | 现有策略与恢复 |
| --- | --- |
| 任务 | summary/detail＋taskEvents；stream/sequence、100ms 合并与传输日志；缺口由各挂载消费者恢复 |
| Home | 有界投影，事件合并与前台 30 秒校准；挂载/恢复可见性补读 |
| 订阅检查 | 共享可见性轮询，活动 2 秒、空闲 10 秒；读取持久 run/reservation |
| 下载器 | 3 秒读后台快照，详情按需；保留最近成功状态并标过期 |
| 索引器/规则 | useIndexerEvents 等失效通知，200ms 合并局部回读；新流/缺口/重连补读 |
| 演员场景/待发行 | 领域事件按演员过滤，200ms 合并当前分页；重连与可见性校准，无列表轮询 |
| 性能/订阅实体/服务/日志 | 各自订阅；性能隐藏暂停、15 秒或错误标过期，失败保留数据 |

不为每张卡另开 timer，不把事件当持久历史。删除成功解除列表/详情关联但保留 mutation 回执；不能在更新器中使同一 Task 实体失效而将成功变成 null。DELETED 事件仍清实体，失败项保留。

## Home 与任务交互

Home 的“我的订阅”保留大号订阅数、较小的最近检查异常数和管理入口；异常链接 `/performers?filter=error`，0 中性，未检查按需显示。人数读取当前订阅 ID/custom_fields 与本地最近检查投影，不拉全演员资料/目录；运行中不覆盖上次结果，历史任务重试失败不计检查异常。

History 中性入口放在检查状态行右侧，左侧空闲只显示下次检查安排，不常驻“自动检查已启用”；运行显示演员/真实进度，调度不可用时保留异常提示；窄屏只换行左侧。最近定时结果、保留统计和历史在抽屉按需读。订阅概况与检查状态独立加载/失败。Stash、索引器、下载器各有独立卡片，主页不再显示旧运行状态或规则作业失败区；详情及统计口径见[主页说明](../../docs/home-workspace.md)。

任务选源由 useTaskIndexerSearch 管理 URL/请求，TaskIndexerSearch 呈现；sourcingTask/run/page/sort/筛选保留列表参数和 backgroundLocation。首次打开搜索，刷新恢复，过期不自动重搜；选资源只提交 ID/revision。站点/关键词筛选先作用于保留集合再分页，变更回第一页且不重新请求站点；明确重搜清筛选。结果摘要的已结束含失败/取消/中断/未执行，部分结果不能冒充全部成功。

候选获取链接是独立动作，usePreparedTorrentDownload 管理准备/取消/失效和作用域。已就绪附件来自本站句柄，过期只移除入口，不自动重抓。TorrentCandidateLinks 校验协议和 userinfo，HTTP(S) 使用 noreferrer/no-referrer；不把私有定位 URL 放查询或外发。

TaskDetail 通过 TaskSummary 将影片封面、简要资料、任务状态和生命周期合并到一张摘要卡，不重复详情标题的番号。无 Stash 关联时使用任务影片快照，有关联时改用实际 Stash 资料及各影片跳转入口；封面放大并按原图比例完整显示，封面与标题、日期、片商、演员保持左右布局，跳转入口以图标与右侧影片标题同行；任务状态、原因和处理入口集中到流程节点，完成终点显示“已入库”，原生 title 提供动作提示；异常保留真实的加载、失败和删除提示。原始资料来源地址、媒体库位置、选源方式、分类及创建/更新时间归默认收起的详细信息。待资源任务的检查安排保持可见。TaskFiles 的下载文件/候选默认展开，跳过组以原生 details 默认收起、保留数量与大小；切换任务恢复收起，键盘可展开。TaskDetail 在 TaskFiles 内嵌一份 DownloadManager，文件选择抽屉不加载下载器。删除确认每次默认不删除 Stash；无关联禁用、关联丢失清选，提交重验。配置入口 `/settings/tasks?panel=deletion` 不提交删除。

发行复核沿用现有标题、筛选和动作，表面/间距/边框使用主题及 token；封面复用 SceneArtwork，在空值或加载失败时显示占位。跨页选择、下载/拒绝和重新考虑语义不变。

## 待发行、日志与文案

待发行 `section=upcoming` 与 upcomingSearch/Performer/Days/Page 恢复筛选。卡片/详情复用 SceneArtwork、现有影片卡与 Drawer media，16:9 封面；无下载按钮、预告片未接入。状态来自服务端持久 upcomingState，已入库/已有任务优先，未来影片不进入可选集合；只移除当前页已确认不合资格项，其他跨页选择保留。相对天数基于服务端 today/effectiveTimezone。

日志最近 200 条按旧到新；上滚离底超过 24px 或暂停时冻结显示，后台缓存继续更新，恢复回底。切换级别重置跟随，复制当前顺序，下载是活动文件全部级别；不是历史文件查询。路径/轮转见[运维](operations.md)。

中文/英文同步，qps-ploc/ar-XB 供开发验证。资源真值在 resources.ts，JSON 由 export 同步；错误按稳定 code/params/correlationId 本地化，未知有安全 fallback，真实上游原文不直接展示。已有可用内容在刷新失败时保留，真实长任务状态从后端恢复。

## 修改时验证

数据/交互选择最小有效 hook、组件、路由或 graphcache 测试；覆盖迟到结果、缺口、跨页及部分成功。可见布局/焦点改动按[开发流程](../workflows/development.md)运行相关 Playwright 并人工看图。本次设置/共享时区及验证范围见[设置复核](../reviews/2026-11-20-settings.md)；前次视觉收敛、历史基线校准及验证范围见[Stage2 复核](../reviews/2026-11-19-ui-consistency.md)。
