# Web UI 与交互约定

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前实现。

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

[手动候选链接](../TODO.md#u-003)、[任务种子分区](../TODO.md#u-004)、[日志滚动](../TODO.md#o-001) 仍未实施，不因设置重构完成而自动关闭。

2026-10-28 设置跟进：任务删除策略位于“任务与订阅”，独立保存。语言选择仅在“系统”，写入 `system.language`（auto/zh-CN/en），成功保存后生效；初次和窗口恢复焦点读取已保存配置，并防止迟到响应覆盖新保存值。

设置分区加载/错误标题来自当前服务或业务分区映射；禁止引用退休的 connections/automation/ingest tab 翻译。当前共享编辑器为 SettingsPolicySection/PolicySettingsEditor，连接和交付分别为 ServiceConnectionSettings/LibraryDeliverySettings；公共兼容字段在 GraphQL documents 边界使用业务别名。

2026-10-28 自动保存：表单统一使用 useSettingsAutosave 与 SettingsSaveStatus，不再显示保存按钮。普通配置通过 typed patchSettings 的 YAML v2 字段增量更新；下载器策略使用可选字段输入，速度限制仍为单字段命令。列表按整体替换、对象递归合并；清缓存、刷新来源等副作用仍需显式操作。

2026-10-28 顶栏（U-005）：沿用 Stash 的导航分组和简洁选中态，顶栏配色使用项目主题变量：浅色为白色表面、深色为深蓝灰表面，正文/次要文字/选中及焦点使用 text/muted/accent；选中仅改变图标和文字颜色，不使用胶囊背景、边框或下划线。桌面显示品牌、主导航和右侧工具；小于 1000px 主导航改为展开按钮，面板使用两列并允许长文案换行。右侧按性能监控、统计、设置、帮助直接排列，主题随后；性能仅在已保存开发者模式下显示。所有宽度均直接显示工具，没有“更多”和语言入口；语言仍在系统设置。600px 以下导航按钮隐藏可见页名以给常驻工具留空间，保留完整无障碍名称。路由链接保留 aria-current，Escape 返回导航触发器，外部点击和路由切换关闭；主题使用 menu 键盘模式。实施参考与验证见[导航复核](../reviews/2026-10-28-navigation.md)。
