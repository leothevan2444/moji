# 规则组信息布局

日期：2026-11-24。状态：实现、验证与运行更新完成。前一阶段契约、v7 本地配置转换与运行记录见[资源选择体验复核](2026-11-23-resource-selection-ux.md)。

## 变更

层级规则标签直接显示在卡片标题下，不再有重复的可见 legend，保留 role=group/list 与可访问名称；空列表返回 null，不增加占位空白。中文准入/排除/层级帮助统一用“规则”，独立规则编辑器仍区分其内部“匹配条件”。

兜底复用 FieldLabel/tooltip 和 SettingsSwitch，局部 fit-content、justify-content:start、gap 8px，取消该行的 switch auto margin，避免通用设置布局将开关推至远端；不影响其他设置页。

规则组卡片保持名称和手柄、整卡编辑/焦点、状态按钮及删除确认，正文改为三栏准入/排除/层级数量，下面最多展示三项有序层级名称，长名称省略并保留 title，更多项显示数量；底栏合并使用状态、兜底状态与删除，垂直居中。使用 dl/ol 的语义结构。浏览器排序定位改为 grid 的直接卡片子项，避免把新增预览列表的 li 当作可拖卡片。

## 验证与运行

相关前端 40 项、最终 i18n 10 项与生产构建通过。最终无更新基线模式 60 项浏览器首轮全通过，覆盖无重复 legend、switch 距 label 0–12px、中文组窗口无“条件”文案、卡片交互/删除/排序及共享设置布局；规则区基线共 54 项。人工检查中文桌面组卡、编辑页及移动/深浅场景。首次排序定位包含预览 li，已修正为直接卡片子项；另一次移动测试出现语言不一致并重试通过，最终完整回归未复现。文档 91 个 Markdown 与 git diff --check 通过。无后端、API、配置格式或选源语义变化，未重复运行 Go 检查。

运行目录 `.storage/runtime-refresh/group-layout-20261123T161034Z/`（UTC 名称，对应本地 11 月 24 日）。复用原后端二进制，正常停止、离线 core/runtime 备份与 quick_check 后加载新前端；healthz 成功，首页逐字匹配构建，启动无 error/fatal/panic。实际配置为 8 条规则/1 组，配置字节保持不变；更新前后 2316 个任务全部保留。策略 v7，core 10/runtime 9，无迁移。证据见 verification.json、database-before、config-before.yaml 与私有启动/回退记录。没有执行真实站点搜索或手动下载，未提交/推送。


## 同日后续：固定高度与底栏兜底

用户要求所有组卡片直接固定至预览截断高度，采用 248px：头部 66px、底栏 41px、正文保留 3+1 行预览空间。组名最多两行，数字标签和兜底状态超长省略并保留 title；保持正文与操作可见。空组与 6 层级（预览 3+更多）的组具有相同高度，长组名不会挤掉最后预览行。

兜底从 ResourceRuleGroupEditor 正文移至 ResourceSelectionEditor 的 Drawer footer 左侧，复用 FieldLabel tooltip 与 SettingsSwitch，保持 8px 紧凑间距；右侧删除不变。footer 的 onChange 直接更新同一 group.fallback 草稿，空层级提示仍随开关实时更新，沿用串行自动保存及失败反馈。

验证：相关前端 40 项与生产构建通过；资源基线 56 项通过，长组名+六层级/空组对照用例 2 项再通过，最终无更新模式 62 项浏览器首轮全通过。新增断言覆盖空/截断卡片均为 248px、内容无溢出、四行预览及长名不挤掉最后一行，footer 内有且仅有一个兜底 switch、正文无 switch，原有切换/保存/空层级说明与焦点/拖拽均保持。人工检查中文主卡片、移动编辑底栏及长名称截断卡片。文档 91 个 Markdown 和 git diff --check 通过；本轮无文案/后端变化，未重复 i18n/Go 检查。

运行目录 `.storage/runtime-refresh/fixed-group-cards-20261123T162506Z/`（UTC，本地 11 月 24 日）。复用相同后端，正常停止、离线备份与 quick_check 后启动新前端；healthz 成功、首页与构建逐字一致，无启动 error/fatal/panic。配置 8 规则/1 组字节未改，2316 个任务全部保留；v7/core 10/runtime 9 无迁移。证据在 verification.json、database-before 与私有启动/回退参数。未执行真实站点搜索或手动下载，未提交/推送。
