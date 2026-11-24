# 选种报告精简为候选摘要

日期：2026-11-24；状态：实现、备份、报告精简、运行更新与核对完成，未提交。

用户要求删除匹配依据，每个候选只显示标题、体积、做种数、资格状态和原因，并注明命中的排除规则。窗口改为直接显示静态候选条目，状态采用图标加完整文字；不再折叠，不再显示候选内选中标签、独立层级字段、未知规则列表、逐规则/逐条件结果。报告顶层冻结组、结果及实际选中资源保留。命中层级附名称，兜底单独标注，排除附实际规则名称；身份/准入失败等实际原因不隐藏。

删除 SelectionEvidence 组件及测试、样式、翻译；删除 RuleResult/RulesOmitted 与条件归属字段。引擎仅新增 ExcludedRules，在排除 ANY 短路后记录实际命中规则的当时名称。预览真实使用的 Evidence/Attribute/EvidenceOmitted 保留，32 条预览证据预算不限制排除原因采集。资源选择结果及排序不变。

GraphQL 报告使用独立 ResourceSelectionCandidate，只投影标题/大小/做种/状态/层级/原因/排除规则名称；不再复用预览 ResourceEvaluation 或取报告 selectedIndex。持久化 v3 仅保留候选事实和结果摘要，规则/条件定义池、匹配结果、偏好排序副本、未知规则及证据省略信息从报告移除。保留选中索引、报告元数据和包装关联。当前仍是未提交重构，不改应用版本、规则/parser 7、core 10/runtime 9。

历史数据预演：389 份报告、2425 个候选；1119 个候选被排除，其中 1082 个能从已保存结果与冻结排除列表确定名称，37 个无法唯一确定，显示“具体规则未记录”，不使用当前配置猜测或重评估。删除 33709 条条件条目。全部 389 份精简结果通过新 codec 验证。维护脚本仅位于 /tmp，不引入启动迁移。

验证：隔离源码完整 Go suite，相关选择/报告/API race；完整前端 553 项，本地化 10 项及隔离构建通过。浏览器报告回归 18 项最终无截图更新通过；已检查中文浅色桌面、英文深色移动截图。覆盖真实排除名称、缺失名称、兜底、无需展开、未知与零及报告不保存条件细节。

日志：`/tmp/moji-summary-report-go-all.log`、`/tmp/moji-summary-report-race.log`、`/tmp/moji-summary-report-web-final.log`、`/tmp/moji-summary-report-i18n.log`、`/tmp/moji-summary-report-visual-final.log`、`/tmp/moji-summary-report-build.log`、`/tmp/moji-summary-report-activate.log`。

## 运行验收

停机备份核心库和运行库后，精简全部 389 份报告、2425 个候选，删除 33709 条逐条件记录。保留 1082 个候选可确认的排除规则名，另 37 个明确名称未记录。全部现存报告无规则/条件定义池、无逐条件结果；388 份符合任务关联的可见报告通过 GraphQL 核对标题、体积、做种、状态、层级、原因与排除名称，原选择结论保留。

报告正文由 3322493 → 1310281 字节；运行库离线 VACUUM 后文件 126099456 → 122486784 字节（含既有空闲页回收，不能全归于报告减少）。核心报告记录仍为 0，2316 个任务保留，离线任务 revision/generation、配置及迁移账本不变；core 10/runtime 9、规则/parser 7 保持，健康检查通过，启动错误 0。

发布及私有备份目录：`.storage/runtime-refresh/summary-selection-reports-20261124T115553Z`，备份位于 `database-before`。后端 SHA256：`984fd38bc251e0bf7c3918ab4aff585f5e2d7c3725c33da6d7194476d8e21057`。报告相关源文件隔离构建，不包含工作区下载器等无关改动。文档与 diff 检查通过；仍未提交。

## 选中候选与信息行布局

用户要求去掉顶层选中资源区块，改为候选淡绿色背景与背景水印，并确认水印“已选中”。通过/未通过状态移除可见文案，图标统一置于标题左侧，保留 aria-label 与原生 title；信息不足文案保留。体积/做种靠左，简短原因与其同一信息行靠右，长内容自然换行且不横向溢出。装饰水印 aria-hidden、无指针事件，内容在其上层；选中背景沿用 success/surface 主题变量，不等同通过资格。

GraphQL ResourceSelectionCandidate 新增 selected 布尔值，唯一依据报告现有 SelectedIndex，避免同名候选或人工选择被错误标识。持久化及规则算法不变，无数据转换。覆盖同名候选中仅一项选中、选中项可以未通过条件，完整前端 554 项、API suite、本地化 10 项、构建与最终无截图更新浏览器 18 项通过。已查看中英、深浅色、桌面和移动截图。运行更新已核对：388 份当前可展示报告的候选 selected 标志与保存索引逐项一致；2316 个任务保留，配置及 core 10/runtime 9、规则 7 不变，无数据转换，健康检查通过。

日志：`/tmp/moji-selected-report-api.log`、`/tmp/moji-selected-report-web-all.log`、`/tmp/moji-selected-report-i18n.log`、`/tmp/moji-selected-report-visual-final.log`、`/tmp/moji-selected-report-release-build.log`。

选中布局发布目录：`.storage/runtime-refresh/selected-report-emphasis-20261124T130548Z`；后端 SHA256：`5eb6ecb8bf42731235d0e7eace3922569e1829e118c434dab58cefa1d205a9fc`。停机备份位于该目录 `database-before`，实际交付记录见 `verification.json`，均为私有本地材料。工作区无关下载器改动未纳入隔离发布，尚未提交。

## 2026-11-25 居中、同行与冗余核对

水印水平/垂直居中；多个原因保留语义列表但 inline 排列、使用中点分隔，窄屏自然换行。浏览器断言水印几何中心、原因 inline 和分隔符，覆盖中英文/主题/屏幕。关闭自动及人工报告场景的未使用预览证据采集，EvaluateSummary 和 Evaluate 共用同一个内核；等价性测试覆盖结果一致与摘要无 Evidence，预览消费者继续保留。删除未使用的报告 selected 标题 API 与对应旧翻译。

只读存储抽样实际全量 717 份报告约 2268627 字节，逐条件/逐规则字段为 0；站点诊断约 755723 字节，其余主要为候选事实与摘要。保留站点、时间、版本及关联信息的诊断/一致性作用，不为节省约数百 KiB 删除必要上下文。无 schema/config/payload 转换，不改规则版本和应用版本。

相关三个 Go 包回归、554 项前端、本地化与构建通过。浏览器最终 18 项无截图更新回归通过，运行更新已完成。日志 `/tmp/moji-report-center-go.log`、`/tmp/moji-report-center-web.log`、`/tmp/moji-report-center-i18n.log`、`/tmp/moji-report-center-visual-final.log`、`/tmp/moji-report-center-build.log`。

2026-11-25 实际交付：发布目录 `.storage/runtime-refresh/centered-report-20261124T161554Z`；388 份当前可见报告通过 API 核对，任务 2316 个保留、配置及 core 10/runtime 9 不变，无数据库转换，健康和前端产物核对通过，启动错误 0。私有备份位于该目录 database-before；后端 SHA256 `7d6e6ffb5ea29fe6f31ad7d5fb8f614504006399223c8203238810193b954e49`。工作区无关改动保持，尚未提交。

## 2026-11-25 提交归档

用户要求提交全部未提交改动并分批。下载器共享文件核验修复独立一批，报告存储/摘要 UI、接口生成物与视觉基线为另一批；共享翻译按命名空间分开暂存。提交前完整工作区 `GOCACHE=/tmp/moji-go-cache go test ./...` 通过（包含下载器），日志 `/tmp/moji-precommit-go.log`；97 份文档与 diff 检查通过，前端/浏览器验证沿用上一节已完成证据。配置、数据库、日志、发布产物、备份与维护脚本均不纳入提交。上文“未提交”描述为各阶段当时状态，最终提交以 Git 历史为准；本轮不推送、不新增部署。
