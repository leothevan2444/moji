# 资源选择配置 UI 与交互整理

日期：2026-11-23。状态：实现与本地验证完成，运行实例已更新。用户要求卡片切换活跃组、单行条件、规则库排序、卡片直接删除和统一确认、前置跨区互斥、层级完整卡片编辑，并复用项目控件/样式。早期界面验收见[历史 UI 复核](2026-11-22-resource-selection-ui.md)。前一轮业务边界见[仅索引器选择复核](2026-11-23-indexer-only-selection.md)。

## 实现

主页面为规则库和规则组两张 SettingsCard；删除活跃组专用字段。组卡片主体为编辑按钮，底部 FontAwesome 激活和删除按钮独立，避免嵌套 button/事件冒泡误操作；显示 In use/Not in use，当前组不能被关闭为非法空选择。

库卡片使用既有 useListReorder，原生桌面 drag、触摸捕获、方向键及 RTL 沿用；顺序写入 rules 数组，仅改变展示组织，ID 引用和排序决定不变。卡片提供独立删除图标，共享规则仍需先解除引用。长名称最多两行，完整名称有 native title 与编辑入口。

ResourceSelectionEditor 编排对象 ID、编辑和删除确认；主视图卡片在 ResourceSelectionCards，单行条件/精确 GiB 输入在 ResourceConditionRow，规则和组各自编辑，优先级使用 ResourceTierCard 内联全部功能。复用 SettingsCard、FieldLabel/SettingsInfoTooltip、SettingsSwitch、Drawer/Portal、ConfirmationDialog 与模态焦点栈；关闭保留草稿，分区退出关闭全部窗口。

所有规则/组删除入口进入同一个确认流程，取消不改变数据，确认后再更新草稿。引用保护确认时重新检查。删除活跃组明确说明接续组或取消额外过滤，删除组不删除规则；确认/取消保持键盘焦点与已有自动保存失败反馈。条件、引用或层级的移除是各自编辑动作，不冒充删除规则定义。

准入/排除 picker 同时排除本区和另一区的已用 ID；移除后即可在另区选择。Go Validate 同步禁止交集，配置 patch 原子拒绝且不改变内存或文件。未改 schema、策略版本 6、选源匹配含义、元数据边界或数据库 schema。

## 验证与本机运行

- 全量前端：83 个文件、534 项通过；最终仅补充测试类型声明后，规则编辑相关 19 项再次通过。
- i18n：10 项通过；生产构建通过。初次最终构建暴露测试夹具的空数组推断为 never[]，已显式声明 Draft 类型，未修改产品行为。
- Go 全量通过；资源选择、配置与 GraphQL API race 通过。覆盖跨区互斥、无效 patch 不改内存/文件、规则库顺序不改选源结果。
- 固定 Docker 浏览器环境：最终无更新基线模式 44 项通过，包含 38 项资源选择与 6 项共享设置/索引器回归。桌面鼠标拖动、移动触摸、键盘排序，四种规则/组删除入口与取消/焦点/保存请求、跨区互斥、内联层级和无横向溢出均通过。人工检查中文移动主页面、英文移动单行条件及确认框、中文深色层级卡片。
- 文档路由检查：90 个 Markdown 通过；git diff --check 通过。仅更新资源选择本次有意布局基线，未刷新其他设置基线。

运行目录：`.storage/runtime-refresh/resource-ux-20261123T031711Z/`。保存前实例参数于权限受限私有文件，正常停止后为 core/runtime 创建离线 SQLite 备份并 quick_check，再启动新前后端；失败处理保留原版启动路径。`verification.json` 记录 `/healthz` 成功、实际首页与新构建逐字一致、启动无 error/fatal/panic、8 条规则/1 个规则组和无跨区冲突。

本次配置字节保持不变，无配置转换或数据库迁移：core schema 10、runtime schema 9。更新前后 2301 个任务全部保留。没有执行真实站点搜索、种子获取或手动下载提交；常规后台工作沿既有配置运行。未提交或推送仓库改动。


## 同日后续：卡片操作与规则编辑简化

此节取代上文“被引用规则须先解除引用”的早期约束。用户确认允许删除被引用规则；确认框列出引用组并说明同步清除准入/排除/层级引用，删去因此变空的层级，不制造空层级的隐式匹配。规则组定义与其他规则保留。

规则/组卡片空白面可打开编辑，主按钮仍提供键盘入口，空白点击先聚焦主按钮供关闭恢复；删除/选择/拖拽控件独立。两类手柄均无边框，组手柄左下，共享桌面/触摸/方向键排序与持久化；组排序不改 activeGroup。规则卡片最小宽度从 150 调至 220px，名称最多两行省略并保留完整 title。组的状态文字并入空心圆/勾选圆选择按钮，去掉 shutdown 图标与重复状态。

层级卡片桌面三列，900px 以下两列、640px 以下一列。排除与所有层级引用（包括 negate）双向互斥，UI picker 与 Go Validate 都保证，不改准入复用规则。

条件空文本（含全空白）或数值双空范围在 encode 时剔除，编辑草稿仍保留可继续填写；数字单空界不限制，0 是有效边界。无有效条件的规则可保存为 conditions: []，规则 OR 空集不命中，既有层级否定仍按规则结果求反；非空的非法范围/正则仍报错。策略版本 6 与数据库结构不变；没有自动配置迁移。只读 moji-config 验证本地配置已符合新增互斥。

最终验证：前端 83 个文件、538 项通过，最后补充整卡焦点恢复后相关 22 项再通过；i18n 10 项、生产构建、Go 全量及领域/配置/API race 通过。浏览器更新本轮有意基线后，最终无更新模式 52 项全通过；另两项确认未选中按钮 path fill 为 none、两端组排序正确。初轮组排序测试立即反向还原导致自动保存正确省略无净变化，已改为先等待拖动持久化再测试键盘还原。人工检查桌面三列层级、移动主视图/组操作、引用删除确认、深色组窗口。文档 90 个 Markdown 与 git diff --check 通过。

运行版本：`.storage/runtime-refresh/resource-cards-20261123T073846Z/`。正常停止、离线备份与 quick_check 后启动新实例，healthz 成功，首页逐字匹配新构建，启动无 error/fatal/panic。本次捕获的配置为 8 条规则、2 个规则组（用户期间新增），原样保留且满足两类互斥；更新前后 2301 个任务全部保留，core 10/runtime 9 无迁移。证据为目录中的 verification.json、database-before 与私有回退启动记录。未执行真实站点搜索或手动下载操作，未提交/推送。


## 同日后续：整卡焦点、空层级与紧凑过滤

本节替代上节层级清理与手柄位置：组手柄左上名称旁、状态按钮左下，规则/组编辑焦点用 :has(:focus-visible) 在外卡绘制；独立动作保留自己的焦点。TierCard 左上手柄复用 useListReorder 的 mouse/touch/keyboard，去掉上下箭头。SettingsCard 的标题扩为 ReactNode，保留共享卡片结构容纳带手柄标题；正文标签、字段、switch、Drawer/ConfirmationDialog 与焦点栈继续复用。

新优先级不添加任何默认引用，空层级可持久保存。Go Validate 去除非空引用要求，Evaluate 已有 ALL 的 empty=No 契约不变；新增用例确认空第一层级不会抢占后续命中、不会绕过关闭兜底，开启兜底按原逻辑选用。删除引用规则仅清除所有组内引用，保留因此变空的层级 ID/名称/顺序。策略仍为 v6，无配置/数据库迁移。

前置过滤桌面双列、窄屏单列，标签与添加 picker 同行。列表语义保持 role=list 的子项仅包含标签，外层 flex flow 承载 picker。窗口确实使用共享 Drawer；关闭按钮不一致源自 resource-selection-dialog 的 button font-size:12px 覆盖，已限至 drawer-body，头部恢复共享样式，浏览器对比共享确认窗口的 close font-size/font-family。

最终验证：前端 83 个文件、539 项通过；标签 DOM 整理后相关 38 项再通过；i18n 10 项与最终生产构建通过。Go 全量通过，新增空层级保存/重读及不命中/后续层级/兜底用例。最终无更新基线模式 56 项浏览器测试通过；另单独整卡焦点与组拖拽各 2 项通过。浏览器比较资源编辑与共享确认的 close 字体一致，检查新建空层级保存请求、删除引用保留层级、桌面/触摸/键盘层级排序、整卡焦点和控制位置。初轮发现设置外壳通用 focus 选择器优先级高于子按钮 outline:none，已提升局部子按钮选择器，保留父卡外框；组手柄移至顶部后，移动测试先滚动使两个目标都可见，避免向屏幕外派发触摸坐标。人工复核整卡焦点、中文移动组卡、紧凑过滤与三列层级；只更新本轮资源选择基线。文档 90 个 Markdown、git diff --check 通过。

运行版本：`.storage/runtime-refresh/resource-tiers-20261123T092948Z/`，保存前实例私有参数与离线 core/runtime 备份，quick_check 通过后启动新实例。verification.json 核对 healthz、实际首页与新构建一致、启动无 error/fatal/panic，8 条规则/2 组，配置字节未改，2301 个任务全部保留。core 10/runtime 9 无迁移，策略 v6。未执行真实站点搜索或手动下载，未提交/推送。


## 同日后续：标题重命名与添加图标

移除活跃组常驻蓝边框，保留正常 hover/focus 和状态按钮；组拖柄与名称使用同一 flex 行垂直居中，适应名称换行。规则/组通过 ResourceEditableTitle 在 Drawer 头部重命名，层级通过同组件在卡片头部重命名，正文不重复字段。共享 Drawer 增可选 titleContent，提供该 slot 时用实际字符串 title 作为 aria-label，其他调用方结构/字号不变。标题编辑沿原有自动保存，Enter 结束输入、Escape/关闭沿既有保留草稿契约。

ResourceRulePicker 复用 FontAwesome plus 与原生 select；透明 select 覆盖 32px 图标，保留系统键盘/移动选择与可访问名称，父容器显示 focus-within、禁用状态。准入/排除在各自右上，层级在删除左侧，不占单独行。互斥及候选选项语义未变，删除引用后焦点恢复至相应 picker。没有后端、配置结构或数据库改动。

验证：前端全量 83 文件/539 项通过，新增标题重命名用例后相关 39 项通过，最终标题行结构调整后相关 39 项再通过；i18n 10 项与最终生产构建通过。最终无更新基线模式 58 项浏览器全通过；单独标题/图标 2 项与最终卡片相关 20 项通过。覆盖窗口 accessible name 随重命名更新、头部/层级名称自动保存、图标真实选项/位置、活跃卡边框中性、手柄与名称中心差小于 2px、原有拖拽/焦点/互斥/删除。首次精确检查暴露绝对定位残余 3px 偏移，最终改为同一 flex 行；没有放宽对齐断言。人工检查中文桌面/英文移动编辑窗口、卡片、图标、三列层级。仅更新资源选择有意基线；文档 90 个 Markdown 与 git diff --check 通过。本轮无 Go 代码变化，保留上一运行版二进制，没有重复 Go 编译或领域测试。

运行目录 `.storage/runtime-refresh/resource-titles-20261123T101356Z/`：离线 core/runtime 备份并 quick_check，正常停止/重新启动后 healthz 成功，首页逐字匹配新构建，无启动 error/fatal/panic；二进制哈希与上一版相同。8 条规则/2 组和配置字节保持不变，2301 个任务全部保留，core 10/runtime 9、策略 v6 无迁移。证据见 verification.json 与私有备份/回退参数。未执行真实站点搜索或手动下载，未提交/推送。


## 同日后续：打开焦点、窄标题与方形图标

资源 Drawer 传 initialFocus=panel，useModalFocus 新增可选模式，默认 first 保持其他调用方行为。打开焦点在窗口容器，Tab 仍进入名称输入，Escape/嵌套确认与返回焦点沿用共享栈。失焦回收也遵循该窗口的初始策略。窗口 header 输入宽度 40%，层级的小标题保持原布局。添加条件移到条件标题右上 plus，保留添加/移除回退 ref。

资源区 icon-only 控件及引用菜单/X 统一 32×32，使用 square 的触摸/键盘命中区，长标签移除按钮居中；状态图文按钮仅固定高度。组 footer 缩至 4px/8px，状态靠左下，组头和优先级手柄用小负边距使图标与正文 inset 对齐。没有后端或配置变化；核实 Evaluate 的 admission combine(ANY, empty=Yes)，空准入表示不限制，最终仍受排除、层级和兜底约束。

验证：前端全量 83 文件/540 项通过；新增 panel 初始焦点、Tab 进入标题和关闭恢复用例后相关 40 项通过；i18n 10 项与最终生产构建通过。54 项资源浏览器基线更新通过，最终无更新基线模式 60 项全通过。新增断言覆盖两类编辑窗初始聚焦 dialog 而非标题、40% 宽度、添加条件图标的位置/真实新增、main/editor 所有 icon-only 操作的等宽高、组状态与拖柄/优先级拖柄左侧位置。人工复核中文桌面/移动规则窗、组卡与移动层级。文档 90 个 Markdown 与 git diff --check 通过。

运行目录 `.storage/runtime-refresh/resource-spacing-20261123T103526Z/`，复用上一版后端二进制，仅部署新 Web。正常停止、离线数据库备份及 quick_check 后启动；healthz 成功，实际首页与新构建一致，启动无 error/fatal/panic。8 条规则/2 组与配置字节保持不变，2301 个任务全部保留；策略 v6、core 10/runtime 9，无迁移。证据为 verification.json、database-before 与私有回退参数。未执行真实站点搜索/手动下载，未提交/推送。


## 同日后续：底栏引用与两列层级

规则编辑的引用信息从正文移入共享 Drawer footer 左侧，保留删除入口右侧，中文按笔误修正为“被以下规则组引用：”，英文同步。ResourceRuleEditor 去除不再使用的 usedBy prop；引用组仍由编排层根据 IDs 计算。长列表最多四行高，可聚焦滚动，避免占满窗口。

前置过滤 legend 13→16px、字重 600；准入/排除 row gap 8→4px。添加优先层级与 tiersHelp 说明共用一行，层级卡片 desktop 两列、640px 以下一列，保留拖拽/名称编辑/互斥功能。无业务或后端变化。

验证：相关前端 40 项通过，最后补充底栏列表高度/键盘访问后相关 40 项再次通过；i18n 10 项与最终生产构建通过。资源视觉基线更新 54 项、最终无更新模式 60 项浏览器回归通过。原有断言调整为引用组仅在 drawer footer、层级 desktop 宽度约一半，覆盖移动/深浅/pseudo/RTL、拖拽、删除与保存。人工检查中文规则窗底栏、中文组编辑、三张层级的两列排列。文档 90 个 Markdown 与 git diff --check 通过。本次低影响 UI/文案局部修改未重复全量 Go 或前端全量。

运行目录 `.storage/runtime-refresh/resource-density-20261123T120347Z/`，复用上一后端，仅更新 Web。正常停止、离线备份与 quick_check 后启动；healthz 成功，实际首页与构建一致，无启动 error/fatal/panic。8 条规则/2 组与配置字节保持不变，2301 个任务全部保留，策略 v6/core 10/runtime 9 无迁移。证据在 verification.json、database-before 与私有回退参数。未执行真实站点搜索或手动下载，未提交/推送。


## 同日后续：正向层级契约与库顺序标签

此节覆盖上文保留层级 negate、可拖前置标签的早期决定。提示与标题同属左侧 grid，2px gap，右上 plus 跨两行；前置标签无手柄/drag/keyboard move，按 rules 当前顺序映射成员，增删后也按库顺序写入；库重排或改名无需独立同步状态。ResourceRuleTag 供前置/层级复用，保留原 12px 字级，直接 X 移除，删除旧一项/两项菜单、反转文案及相关基线。

Policy/Parser 7；Tier.Rules []string，与 Require/Exclude 一样用规则 ID，验证重复/未知/跨排除冲突、空层级不命中语义不变。Go 求反分支和 RuleRef、API Reference/Input 类型、YAML rule_id 转换分支、前端 negate 状态/菜单/设置编码全移除，GraphQL 已重生。当前 JSON/YAML/API 不接受旧层级对象；旧 v6 配置必须显式转换，不做启动迁移。历史 snapshot 专用读取逻辑兼容旧对象，保留旧 version，旧人工选择报告原始 JSON 不重写，不能将旧策略作为当前配置验证通过。

只读本地核对为 8 规则、2 组、3 个层级引用、零反向引用。独立离线工具仅把版本 6→7、三个 false 对象→scalar IDs，比较完整 YAML 语义并用新 loader 验证；转换预演匹配含义和其他设置保持不变，运行更新前不改实际配置。此工具仅在私有运行产物留存，不进入应用或自动迁移。

自动审批曾拒绝递归批量替换测试的命令，因范围过宽；改为已逐段检查的明确文件/片段修改，未绕过拒绝。相关范围验证继续完成。

验证：GraphQL 后端/前端绑定已生成；前端全量 83 文件/541 项、i18n 10 项与最终构建通过。Go 全量通过；最终历史空字段补充后领域/配置/API/runtime 再通过；领域/配置/API race 通过。新增验证 v7 拒绝旧对象和重复 ID、历史 v6 含反向/空引用报告原样保留、规则 OR 的 27 组正向/未知组合、YAML 严格拒绝旧关系。54 项资源视觉更新通过，最终无更新模式 60 项浏览器全通过，覆盖无手柄标签、库重排同步、增删后排序、hint/title 真实间距 ≤3px、仅 scalar IDs 的保存与空层级、互斥与既有窗口/拖拽。人工复核中文桌面/移动组窗口；只更新本轮资源基线并移除不再引用的两张 tier menu 基线。文档 90 个 Markdown 与 git diff --check 通过。

运行目录 `.storage/runtime-refresh/positive-rules-20261123T153704Z/`。正常停止后备份原配置和 core/runtime、quick_check，私有离线工具写入 v7 配置，再启动新前后端；失败路径恢复原配置与旧程序。实际验收 healthz 成功、首页逐字匹配新构建、API policy 7 且旧 Reference/Input 类型不存在，无启动 error/fatal/panic。原 8 条规则/2 组保留，3 个正向引用无损转换，匹配含义和其他设置语义均核对不变；2301 个任务全部保留，core 10/runtime 9 无数据库迁移。证据为 verification.json、config-before.yaml、config-conversion.json、database-before 与私有回退参数。配置转换工具/源只保留在私有运行产物，本轮临时源码目录已清理；没有生产自动迁移。未执行真实站点搜索或手动下载，未提交/推送。
