# Web UI 与交互约定

核对日期：2026-11-20（本轮主页与任务详情）；代码基线：`c792c67` 及本轮工作树；状态：当前源码契约。跨页面体验要求统一见[产品规范](../standards/product.md)，这里定位已有能力与数据边界。

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

顶栏保留品牌、四个常显主导航与常驻工具，不折叠菜单。任务主导航使用 rectangle-list，与进入多选的 list-check 区分。ResizeObserver 根据品牌、完整导航、工具和间距的实际宽度判断空间是否足够；所有宽度统一使用布局 CSS 像素，保留 2px 整数测量余量，避免缩放后的元素矩形与未缩放容器宽度混用；不足时主导航仅显示图标，保留可访问名称、原生 title 与 aria-current，变宽后恢复文字。隐藏的非交互测量区不占布局、不进入焦点与无障碍树，语言、字体和工具变化会重新测量，不使用固定屏幕宽度阈值。优先保持单行文字，其次切换单行图标；单行图标仍放不下时，工具进入下一行。换行后按品牌与导航所在行的可用宽度重新判断文字，放得下则恢复文字，继续缩窄才再次隐藏文字。使用显式 Grid 行布局，将是否换行与是否显示文字独立决定，避免反复切换。ThemeMenu 使用 menuitemradio，方向键/Home/End、Tab 离开、选择/Escape 返回焦点；不套用已废弃的通用 listbox Menu。

顶栏最右侧的 NSFW 图片打码开关使用固定的 `NSFW` 文字按钮，开启时以强调色与浅色背景表示选中，悬停提示提供开启/关闭动作，默认关闭；开启后以 12px 半径模糊影片封面、发现详情图片与演员照片，验证码不受影响；头像图片向外扩展三倍模糊半径（36px） 后由容器裁切，避免模糊透明边缘透出占位底色。偏好仅存浏览器 `moji:nsfw-mask`，刷新恢复、同源标签页同步，不提交后端；存储不可用时仍可在当前页面切换。图片通过 `data-nsfw` 显式标记，根元素属性统一覆盖抽屉 Portal，刷新恢复在绘制前应用。工具栏在空间不足时自然换行，保留所有入口。

共享 Drawer 标题栏始终横向排列，关闭按钮固定在右侧且不压缩、不换行；左侧标题允许收缩和长文本折行，不随视口断点强制纵向排列。任务抽屉顶部标题统一 28px，不因窄屏或粗指针设备降为 20px；宽度不足时自然折行。

共享页面标题、任务分组标题、提示区及按钮行按组件实际空间自然换行，不在 860px/640px 断点统一改成纵向排列。首页将搜索模式与关键词分组；控件组整体移动，空间不足时组内按最小可用宽度折行。任务卡状态标签不因粗指针设备强制独占一行，触控点击尺寸仍保留。演员身份/操作、分页跳转、元数据来源刷新和索引缓存操作不再由固定视口阈值强制占满整行。任务页与演员列表右上角统一复用 [FilterCountHeading](../../web/src/components/layout/FilterCountHeading.tsx)，将单选筛选与匹配数量合并，任务操作栏不再重复状态筛选。任务显示全部任务、进行中、等待资源、已完成、需处理；演员显示全部、已订阅、Stash 收藏、待复核、检查失败，各演员分类可重叠，待复核按演员数而非作品数统计。数字来自后端 filterCounts，受当前关键词（演员含别名）和额外时间条件限制，不受当前分类、排序及分页限制；任务工作台与全局统计仍使用独立的全局 stats。最近 24 小时条件在切换分类时保留，直到显式清除。零数值可点击，尚未获取或暂不可用的数量显示 —。按真实控件宽度依次尝试四级密度：图标与全部名称、右侧数字徽标；图标与全部名称、下方徽标；仅保留选中项名称、下方徽标；隐藏所有名称、下方徽标。按钮间距第一级为 4px，其余级别为 2px，图标与名称间距为 4px。下方徽标字号由 12px 降至 11px，图标和名称保持同行；左侧标题在所有级别固定为 24px，触屏按钮仍至少 44×44px。标题与四级控件都使用真实测量副本，不以固定视口宽度判定。四级都无法与标题同排时才整体换行，并重新从第一级判断能否恢复完整内容，最后才在组内折行；变宽时反向恢复。原生 title 表达“只看”与数量单位；aria-pressed 表达单选状态，方向键/Home/End 仅移动焦点，Enter/Space 切换，点击当前项不重复请求。选中高亮由单个装饰性滑动窗口承载，选项切换与按钮宽度使用同一浏览器文档时间线、160ms 时长和 ease 曲线；窗口逐帧读取实际几何，不再叠加 CSS 缓动，避免宽度变化时滞后。支持连续切换、途中目标重定向、折行、RTL 和缩放；首次显示直接定位，减少动态效果偏好下关闭过渡。按钮自身的宽度也使用 160ms 过渡，目标值取自当前级别的静态测量副本，支持名称显隐和数量位数变化；减少动态效果时立即更新。窗口不占布局、不接收点击，选中按钮仍保留 aria-pressed 与独立焦点框。隐藏测量区不参与交互。任务、演员列表与演员详情的操作栏统一复用 [WorkspaceToolbar](../../web/src/components/common/workspace-toolbar/WorkspaceToolbar.tsx)，沿用演员栏的圆角容器和控件样式，顺序为固定在最左侧的多选切换、搜索、无外置标签的筛选、功能按钮、更多按钮。多选操作与原操作栏在同一位置淡入淡出切换，左侧图标由 list-check 改为 xmark；非活动面板和测量副本使用 inert，不参与键盘交互。按容器及真实控件宽度先压缩搜索、筛选和按钮文字，再优先收纳功能按钮，必要时换行，最多三行；超出行数才收纳筛选。操作图标统一取自 [icons.ts](../../web/src/components/common/workspace-toolbar/icons.ts)：list-check/xmark 用于进入/退出多选，square-check 用于本页全选，eraser 清空选择，rotate 刷新，rotate-right 重试，clipboard-list 打开任务工作台，user-plus/user-minus 订阅/取消订阅，binoculars 检查，clipboard-question 发行复核，download 创建下载任务，trash-can 删除，ellipsis 更多。仅刷新图标在请求中旋转，检查保持 binoculars 并通过文案及 aria-busy 表达进行状态。更多菜单保留原操作的名称、提示、禁用条件和回调，支持 Portal 定位、视口边界翻转、Escape、Tab、外部点击及焦点恢复；缩宽后恢复内联控件，减少动态效果偏好下关闭过渡。任务、演员列表及演员详情统一复用 [Pagination](../../web/src/components/common/pagination/Pagination.tsx)，各在操作栏下方和列表底部放置一处，页码与每页条数由路由 URL 控制并上下联动。条数统一 12/24/48/96，默认 24，变更回到第一页；单页和空结果仍可调整。常态显示条数、筛选结果总量、首尾/前后页及可直接编辑的页码（回车跳转、Escape/失焦恢复，无效页码不发请求）。整个控件组居中，组内依次为翻页与页码、总量统计、条数或展开按钮。按完整布局及仅收起条数后布局的实际宽度分阶段收纳：先将条数收进更多浮层，继续缩窄才收起首尾页；浮层只展示当前已收起的控件，总量、前后页和页码常显；始终单行，极长数字/文案仅在分页器内部横向滚动。浮层通过 Portal 避免被滚动容器裁切，支持边界翻转、缩放、Escape、Tab、点击外部关闭及焦点恢复。演员详情使用 totalCountExact/totalPages 区分精确总数与已加载数量；未知时显示已加载项数及页数 —，尾页禁用，直接跳转限于已知页，下一页沿用 hasNextPage 按需读取，精确总数恢复后自动启用完整导航。

任务和演员页的排序采用字段下拉框与独立方向按钮，作为一个操作栏控件整体压缩、换行或进入更多菜单，保留原生 select 和方向说明。方向切换与字段切换均回到第一页，保留搜索、分类、方向和既有选择，状态写入 URL；切换字段不重置方向。任务字段为卡片标题、创建时间、影片发布时间、进度，默认创建时间降序；演员字段为名称、Stash 创建时间、Stash 库内作品数、最近检查时间，默认名称升序，不再隐含订阅优先。标题和名称支持大小写无关的数字自然顺序，未知日期/检查时间始终后置。任务分组是当前页的纯前端展示，默认关闭，以 moji:tasks-grouped 保存浏览器偏好，不传后端；开关使用 object-group、aria-pressed 和原生提示，组内保留后端排序，不改变分页；取消分组后恢复连续卡片网格。进度实时更新保持当前顺序，手动刷新后重新排序；其余排序在列表空闲时仍可响应成员变更刷新。

任务、演员列表和演员作品卡片复用 [CardSelectionControl](../../web/src/components/common/CardSelectionControl.tsx) 及共享选中样式。选择控件统一为左上角的 24px 圆角方形复选框、44px 点击区；任务卡标题仅在多选状态预留 32px，普通状态不占位，演员与作品卡的控件覆盖图片左上角，不挤压标题。多选时未选控件常显，不可选作品显示禁用复选框并提供现有业务状态作为原因；资格、跨页选择及批量结果清理规则不变。选中统一使用强调色勾选框、相同边框宽度和轻底色，不给封面叠色；键盘焦点单独强调控件，减少动态效果及强制颜色模式有对应处理。多选期间单条写操作隐藏、退出后恢复，隐藏区域保留占位并退出焦点及可访问树；只读详情、错误及来源入口保留。

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

TaskDetail 通过 TaskSummary 将影片封面、简要资料、任务状态和生命周期合并到一张摘要卡，不重复详情标题的番号。无 Stash 关联时使用任务影片快照，有关联时改用实际 Stash 资料及各影片跳转入口；封面放大并按原图比例完整显示，封面与标题、日期、片商、演员保持左右布局，跳转入口以图标与右侧影片标题同行；任务状态、原因和处理入口集中到流程节点，完成终点显示“已入库”，原生 title 提供动作提示；异常保留真实的加载、失败和删除提示。原始资料来源地址、媒体库位置、选源方式、分类及创建/更新时间归默认收起的详细信息。待资源任务的检查安排保持可见。TaskFiles 的下载文件/候选默认展开，跳过组以原生 details 默认收起、保留数量与大小；切换任务恢复收起，键盘可展开。TaskDetail 在 TaskFiles 内嵌一份 DownloadManager，文件选择抽屉不加载下载器。种子卡只显示一次 qB 实际名称；无当前观测时保留任务中的历史名称，不以索引标题补齐。按下载/做种/停止状态展示相关指标，不再重复大型总体进度条；任务详情全部操作平铺，复用 TorrentActions 的资格判断与确认保护；全局列表仍保留紧凑菜单。技术字段在原生 details 默认收起并保留复制，内容路径和保存目录仍区分。文件统计明确选中数量/全部数量及选中大小，人工选择同步更新；跳过组继续默认收起。查询失败保留最近确认观测并禁用传输操作；切换任务隔离保留数据，成功空结果清除旧观测。删除确认每次默认不删除 Stash；无关联禁用、关联丢失清选，提交重验。配置入口 `/settings/tasks?panel=deletion` 不提交删除。

发行复核沿用现有标题、筛选和动作，表面/间距/边框使用主题及 token；封面复用 SceneArtwork，在空值或加载失败时显示占位。跨页选择、下载/拒绝和重新考虑语义不变。

种子详情的连接数、可用副本数、观测时间及资源链接口径见[任务说明](tasks.md#详情信息与数据来源)。数值指标每行 2–3 项，刷新按钮保持正方形；任务详情操作平铺，共用禁用原因与删除资格复核。真实分片图由 [TorrentPieces](../../web/src/components/downloader/TorrentPieces.tsx) 随当前下载器观测更新，不另设全量轮询器。

资源选择设置分规则库、规则组两张 SettingsCard；组卡片显示使用状态并以 FontAwesome 按钮切换当前组，不再有独立选择字段。规则与组卡片均可整卡打开（独立动作除外），用共享 useListReorder 拖动/触摸/键盘排序；组手柄位于左上名称旁，状态与圆形选择按钮合并置于左下，编辑焦点绘制整卡外框；活跃组不常驻强调边框，名称与手柄精确垂直居中。规则/组从卡片或编辑窗删除统一经过 ConfirmationDialog；被引用规则列出影响组并清除所有引用，保留空层级。RuleEditor 单行条件复用字段/tooltip，TierCard 在组窗口内完整编辑。前置准入/排除、优先级/排除互斥由 UI picker 与 Go 校验共同保证。层级卡片桌面两列、窄屏单列，左上手柄代替上下箭头，复用同一排序 hook；新层级不预填规则，可保存空层级且永不命中。所有层级引用均为正向 scalar ID，不再有匹配/排除关系或独立动作菜单，标签直接 X 移除；不显示重复的列表标题，空层级不留占位。组编辑统一使用“规则”用语，兜底开关为局部 fit-content 行、gap 8px，保留 tooltip；组卡片为数量分栏、排序预览及合并状态底栏，固定 248px 高度，为 3 条预览+1 条更多提示预留空间；名称最多两行并保留 tooltip。兜底开关置于编辑窗口左下 footer，编排层直接修改 group.fallback，正文不重复控件。前置过滤桌面并排、窄屏堆叠，标题/hint 组成紧贴的左侧内容，右上加号复用原生 picker；前置标签无拖柄、按当前规则库顺序实时展示；层级右上加号在删除左侧。前置过滤标题 16px、title/hint 间距 2px，添加优先层级与说明同行；引用规则组信息置于窗口底栏。规则/组窗口及层级均在标题内重命名，正文不再重复名称字段；共享 Drawer 增可选 titleContent，窗口仍以实际名称提供可访问标签；资源编辑仅使用 initialFocus=panel，打开不聚焦标题，Tab 仍进入标题，其他窗口默认策略不变。窗口标题约占 header 可用宽度 40%，条件右上加号，资源 icon-only 控件固定 32px 方形，状态/手柄靠左与内容对齐。空表达式与双空数值范围在 encode 时剔除，草稿保留，无条件规则可保存且永不匹配。拆分主视图卡片、单行条件、层级卡片和编辑/删除编排；沿用自动保存、Drawer/Portal 与焦点栈，关闭按钮保留共享 Drawer 字体，不受编辑正文按钮字号覆盖。当前验收见[资源配置体验复核](../reviews/2026-11-24-resource-group-layout.md)。

## 待发行、日志与文案

待发行 `section=upcoming` 与 upcomingSearch/Performer/Days/Page 恢复筛选。卡片/详情复用 SceneArtwork、现有影片卡与 Drawer media，16:9 封面；无下载按钮、预告片未接入。状态来自服务端持久 upcomingState，已入库/已有任务优先，未来影片不进入可选集合；只移除当前页已确认不合资格项，其他跨页选择保留。相对天数基于服务端 today/effectiveTimezone。

日志最近 200 条按旧到新；上滚离底超过 24px 或暂停时冻结显示，后台缓存继续更新，恢复回底。切换级别重置跟随，复制当前顺序，下载是活动文件全部级别；不是历史文件查询。路径/轮转见[运维](operations.md)。

中文/英文同步，qps-ploc/ar-XB 供开发验证。资源真值在 resources.ts，JSON 由 export 同步；错误按稳定 code/params/correlationId 本地化，未知有安全 fallback，真实上游原文不直接展示。已有可用内容在刷新失败时保留，真实长任务状态从后端恢复。

## 修改时验证

数据/交互选择最小有效 hook、组件、路由或 graphcache 测试；覆盖迟到结果、缺口、跨页及部分成功。可见布局/焦点改动按[开发流程](../workflows/development.md)运行相关 Playwright 并人工看图。本次设置/共享时区及验证范围见[设置复核](../reviews/2026-11-20-settings.md)；前次视觉收敛、历史基线校准及验证范围见[Stage2 复核](../reviews/2026-11-19-ui-consistency.md)。

资源规则设置采用 v7 正向 ID 索引器匹配，仅标题、大小、做种，无检查范围、文件字段、后台检查开关和作业轮询。预览/任务推荐同步返回、本地暂存，刷新后重新匹配；组前置 ANY、层级 ALL 和独立兜底开关沿用。当前证据见[仅索引器选源复核](../reviews/2026-11-23-indexer-only-selection.md)。


任务详情的“选种报告”归选种节点，报告存在才显示“查看选种报告”纯文字按钮，排列在选种阶段动作区最左侧，选种结束后仍可读；点击经 Portal 打开独立共享 Drawer（task-selection-report），不内联展开或独占 drawer-card。标题使用无图标的纯文字“选种报告 · 番号”，先显示结果、选中资源与覆盖提示，候选标题/状态逐条折叠，组/层级、原因及证据分区展示。沿用 useModalFocus 圈定焦点、父窗口 inert、Escape 只关最上层和返回入口，不增加请求/改变页面路由。组件为 TaskSelectionReport，TaskSummary 将其传给 TaskTimeline 的 sourcingDetails，不改变其他阶段操作或预览的 ResourceEvaluationDetails。当前复核见[选种说明布局](../reviews/2026-11-24-task-selection-report-ui.md)。


选种阶段“查看选种报告”复用既有 task-primary-action 与 --secondary，与“手动选种”保持中性样式，“立即检查资源”保留主要操作配色；按钮尺寸、字重和悬停均由已有规则管理，桌面 38px、移动/粗指针 44px，不新增独立按钮样式或更改共享规则。
