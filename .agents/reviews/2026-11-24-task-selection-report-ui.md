# 任务选种说明布局复核

日期：2026-11-24；状态：实现与验证完成，运行实例已更新，未提交。

首轮内联展开已由同日后续的按钮/独立窗口替代；当前实现以末尾记录为准。

用户要求资源选择说明融入选种阶段，入口与任务详情协调，并解决信息堆叠。用户路径为任务详情 → 选种节点内图标/文字入口 → 先看选择结果与选中资源 → 展开候选查看组/层级、原因与证据 → 原位折叠继续处理任务。

## 实现与边界

移除 TaskDetail 独立报告卡。TaskSummary 在已有 TaskTimeline 的选种节点传入 TaskSelectionReport；复用原生 details，图标与中性字色沿用现有样式。报告存在时始终可读，不依赖当前任务阶段；无报告不显示入口。切换任务由既有任务 key 重置展开。候选标题/状态为紧凑摘要，组/层级、未知规则、原因和条件证据分区显示，长标题与证据可换行。保留不完整覆盖与省略证据提示。没有新的 API、后台检查或数据保存行为；搜索预览/推荐继续使用原 ResourceEvaluationDetails。

## 验证

- 相关 TaskExperience/TaskSummary：43 项通过，覆盖选种/下载/完成阶段归属、默认折叠、逐候选展开、无报告、空结果与切换任务。
- 完整前端：84 个文件 / 549 项通过，日志 `/tmp/moji-stage-report-web-all.log`。
- TypeScript/生产构建、本地化完整性通过，日志 `/tmp/moji-stage-report-build.log`、`/tmp/moji-stage-report-i18n.log`。
- 固定 Playwright 环境最终 18 项通过，无基线更新：`/tmp/moji-stage-report-visual-final.log`。包含报告与原任务详情布局，中英文/深浅色/桌面/移动，键盘 Enter 展开、长标题、信息不足和覆盖不完整。
- 更新范围限报告新布局截图；人工查看中文浅色桌面、英文深色移动和折叠入口截图。文档及 git diff 检查另在交付前执行。

## 工作区与实例

工作区已有下载器隔离逻辑、文档和文案未提交修改；本轮不改其后端。部署前端从当前已提交版本加入本轮界面代码及资源说明词条构建，排除其他未提交修改，沿用运行实例后端的版本号。实际激活及健康结果在完成后补记。


运行交付：`.storage/runtime-refresh/stage-selection-report-20261124T063738Z`。当前后端二进制摘要与更新前一致；仅激活独立前端产物，版本号沿用当前后端。停机备份验证通过；healthz、首页产物与中英文实际语言请求正常，无启动错误。任务 2316 → 2316，配置原字节不变，策略 7/core 10/runtime 9 保持。工作区其他未提交后端与文案改动未纳入本次部署；没有提交或推送。


## 同日后续：按钮打开独立报告窗口

用户确认采用独立窗口。选种阶段的轻量 ghost-button 图标/文字入口与该阶段动作同区；报告存在才显示，不受当前任务阶段限制。TaskSelectionReport 本地 open 状态经 Portal 渲染共享 Drawer（task-selection-report），标题带任务番号；选中结果和资源优先，候选和证据保留分层。移除根级内联 details 和相关样式，不改变搜索推荐和报告数据请求。

沿用 useModalFocus：初始焦点在窗口面板，Tab 圈定在当前层，父任务窗口 inert；Escape 只关闭报告，关闭按钮和 Escape 都恢复入口焦点。切换任务或报告消失时组件卸载，不遗留窗口。任务页路由和滚动保持原有机制。未新增 API、自动重新匹配或业务行为。

最新验证：TaskExperience/TaskSummary 43 项通过；完整前端 549 项/84 文件通过（`/tmp/moji-report-dialog-all-tests.log`）；构建和本地化 10 项通过（`/tmp/moji-report-dialog-build.log`、`/tmp/moji-report-dialog-i18n.log`）；固定 Playwright 最终 18 项通过，无基线更新（`/tmp/moji-report-dialog-visual-final.log`）。浏览器验证按钮键盘打开、嵌套窗口焦点/父层 inert、Escape 返回、长标题与中英文/深浅色/桌面移动布局；人工查看中文浅色桌面与英文深色移动窗口截图。相关截图按最终窗口布局更新，没有更新无关基线。

运行交付目录：`.storage/runtime-refresh/selection-report-dialog-20261124T071157Z`。独立前端从已提交源码叠加本轮界面构建，排除工作区其他未提交变更；沿用原后端二进制与版本。备份及健康检查通过，前端产物匹配、中英文实际词条可读、启动无错误。任务 2316 → 2316，配置原字节、策略 7/core 10/runtime 9 和后端摘要保持；没有提交或推送。


## 同日后续：选种报告命名与纯文字按钮

用户确认入口去掉图标，中文统一为“选种报告”，英文为“Selection report”；窗口标题为报告名称与任务番号。入口直接使用共享 ghost-button，移除无图标后无用的局部布局规则。窗口标题图标改为文档语义 faFileLines，候选展开箭头不变。

相关单元测试 43 项通过，增加入口无 SVG 断言；TypeScript/生产构建、本地化 10 项通过。最终 Playwright 报告用例 10 项通过，无基线更新：`/tmp/moji-report-label-visual-final.log`。人工复核中文桌面入口，与“手动选种”动作同区且主次明确。其余日志为 `/tmp/moji-report-label-tests.log`、`/tmp/moji-report-label-build.log`、`/tmp/moji-report-label-i18n.log`。

已激活独立前端产物：`.storage/runtime-refresh/selection-report-label-20261124T074217Z`。健康及首页产物核对通过，中英文实际 report 词条分别为“选种报告”/“Selection report”；后端摘要和版本不变，配置原字节不变，2316 个任务保留，无启动错误。没有提交或推送。


## 同日后续：移除报告标题图标

用户确认窗口标题也去掉图标。TaskSelectionReport 停止传入 Drawer.titleIcon，并移除 faFileLines 导入，标题仅为“选种报告 · 番号”；候选展开箭头保持原行为。相关单测 43 项、独立生产构建和最终报告视觉 10 项通过（`/tmp/moji-report-title-tests.log`、`/tmp/moji-report-title-release-build.log`、`/tmp/moji-report-title-visual-final.log`）；人工查看中文浅色桌面截图，标题与关闭按钮沿用共享窗口样式。

已更新实例：`.storage/runtime-refresh/selection-report-title-20261124T080315Z`，健康和前端产物核对通过，无启动错误；2316 个任务及配置原字节保持，后端和版本沿用原实例。没有提交或推送。


## 同日后续：查看按钮与最左位置

用户要求按钮改为“查看选种报告”，且与其他选种动作同现时置于最左。新增独立 viewReport 词条（英文 View selection report），report 词条仍用于“选种报告 · 番号”窗口标题；TaskTimeline 在动作区先渲染报告入口，再渲染阶段主动作，以 DOM 顺序保证视觉和键盘顺序一致，沿用弹性换行。

相关单测 43 项、构建、本地化 10 项与最终视觉 10 项通过；增加动作区首个按钮的断言，人工复核“查看选种报告”“手动选种”并列的中文桌面截图。日志 `/tmp/moji-report-view-tests.log`、`/tmp/moji-report-view-build.log`、`/tmp/moji-report-view-i18n.log`、`/tmp/moji-report-view-visual-final.log`。

已更新实例：`.storage/runtime-refresh/view-selection-report-20261124T081034Z`。健康、首页产物及实际中英文按钮/标题词条通过验证；后端、版本、配置和 2316 个任务保持，无启动错误。没有提交或推送。


## 同日后续：统一选种动作按钮样式

用户确认复用现有任务按钮，不建立新样式。TaskSelectionReport 的入口增加 task-primary-action 和 task-primary-action--secondary，与“手动选种”共享中性次要样式，“立即检查资源”继续使用主要操作配色。没有修改 TaskPrimaryAction、共享 CSS 或主页/任务卡片规则。桌面高度 38px，移动/粗指针沿用 44px 触控规则。

相关单测 43 项与生产构建通过。新增真实三按钮浏览器场景，比较高度、字号、字重、圆角、padding、hover transform 和主次文字颜色；中文/英文、深浅色、桌面/移动均覆盖。初始断言误将移动端也设为 38px，核对既有 CSS 后改为按触控媒体条件验证 44px，没有降低产品触控规格。新增动作区 8 项通过，最终报告/动作区/原任务详情布局共 26 项通过，无基线更新（`/tmp/moji-report-style-visual-final.log`）。人工复核中文浅色桌面和英文深色移动截图。

已更新实例：`.storage/runtime-refresh/selection-action-styles-20261124T082023Z`，独立前端产物核对及健康正常；后端、配置、版本与 2316 个任务保持，无启动错误。没有提交或推送。
