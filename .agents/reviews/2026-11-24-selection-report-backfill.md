# 旧选种报告手动补全与旧展示移除

日期：2026-11-24；状态：实现、备份、手动补全、运行更新与核对完成，未提交。

## 用户授权与边界

用户明确授权备份后的手动补全，并删除旧展示代码。转换是本地一次性维护脚本，不加入服务启动或迁移注册表；任务选择、配置、核心/运行 schema 版本和已应用迁移保持。重构前备份仍有 389 份 v2 报告与当时完整 8 条规则快照，现存 v3 包含相同报告 ID、候选顺序、事实、匹配值和条件说明。依据冻结快照精确匹配说明/字段，唯一确定规则 ID；不按冒号拆名称，不读取当前配置，不重新运行匹配。

## 补全算法与保真

逐份核对选择结论、候选事实、状态、层级、排序、原因、未知列表、证据省略计数、匹配顺序和值，恢复条件的 RuleID/Expression。每组记录必须是冻结规则条件的有序前缀；命中条件只能出现在已保存前缀末尾。末尾 YES 则整体 YES，所有条件均有保存结果时按原任一语义恢复 NO/UNKNOWN；其他情况必须是最后一个且带证据省略的规则组，整体为 NOT_RECORDED，checked=null，总条件数保留。未保存的后续规则不补造结果。保留诊断行 ID/时间及 revision/generation/source 关联，只更新报告正文。共享规则/条件定义仍去重。

预演：389 份报告、2425 个候选、33709 条已保存条件均能唯一归组，8403 组整体结果可恢复，1343 组整体结果未记录。所有预演结果通过新 codec 解码、编码及二次解码。

## 清理范围

报告只使用分组 Item。移除前端 historical 平铺分支、表达式回退旧 attribute、历史文件 scope 文案及无效样式；清理旧展示测试/fixture。报告 codec 删除 v1/v2 历史策略投影及 Attribute 定义字段，存储只接受 v3；新解码验证规则 ID、条件归属、索引、三值/未记录值及 nullable checked 一致性。GraphQL checked 改为 nullable，操作不再获取报告 Attribute。预览仍使用 Attribute，它是现有预览契约的真实消费者。

32 条规则摘要预算下，引擎不记录无法附着于已记录摘要的条件证据，增加省略计数；规则评估不受预算限制。覆盖前 32 个空规则后仍命中第 33 个规则、但不留下孤立依据的场景。选择算法与结果不变。

## 验证

隔离源码完整 Go suite、相关规则/报告/API race 通过；完整前端 553 项、本地化 10 项及隔离构建通过。20 项报告浏览器回归无截图更新通过，覆盖未知条件和历史未记录整体结果同时存在，且不猜测未检查数量。96 份文档链接检查与 diff 格式检查通过。

日志：`/tmp/moji-backfill-go.log`、`/tmp/moji-backfill-race.log`、`/tmp/moji-backfill-web.log`、`/tmp/moji-backfill-i18n.log`、`/tmp/moji-backfill-visual-final.log`、`/tmp/moji-backfill-build.log`、`/tmp/moji-backfill-activate.log`。

## 实际交付

停机后用 SQLite backup API 备份核心库与运行库并校验，再显式更新 389 份报告。实际恢复 33709 条条件归属、9746 个候选规则分组，其中 8403 个整体结果可确认，1343 个为 NOT_RECORDED 且 checked=null。新 codec 逐份核对条件定义、值、文本及规则结果；388 份当前符合任务关联的报告全部通过扩展 GraphQL 核对。当前运行库无 Attribute 条件定义，运行健康检查通过。

核心报告记录仍为 0，2316 个任务保留，离线写入前后任务 revision/generation 不变；配置逐字节一致，core 10/runtime 9 迁移账本与校验和、规则版本 7 不变。原始选择、候选事实、状态、层级、原因及诊断关联均保留。报告正文由 2690052 增至 3322493 字节，增加来自恢复的规则上下文和结果，并非新增完整策略副本；保持共享定义和单份事实。

发布与备份目录：`.storage/runtime-refresh/backfilled-selection-reports-20261124T112555Z`（`database-before` 保存本次停机备份）；重构前完整规则快照保留于 `.storage/runtime-refresh/compact-selection-report-20261124T090003Z/database-before/runtime.db`，均为私有本地材料，不纳入提交。后端 SHA256：`610ac7c8a4027423b334c64462e74ac42640a71783cd7ceea83e4153dcff46df`。本次隔离发布不包含工作区下载器等无关改动。

后续用户要求删除全部匹配依据，本文补全的逐条件记录已经备份后从运行报告中移除，仅保留可确认的排除规则名称与候选摘要；当前实现和验收见[摘要报告复核](2026-11-24-selection-report-summary.md)。
