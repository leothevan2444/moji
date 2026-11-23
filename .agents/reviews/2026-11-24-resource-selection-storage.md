# 资源选择决定与报告分库复核

日期：2026-11-24。状态：已实现并更新本地运行实例，未提交。

同日后续已按用户要求移除核心独立精简决定；下方首轮实现属于历史过程，当前状态以末尾“报告完全归运行库”为准。

## 授权与边界

用户要求按核心库/运行库职责调整资源选择报告，并明确所有资源选择引擎重构尚未提交，不重复递增版本。本次保持策略/parser 7、报告 payload 2、core 10、runtime 9；不改写任何已应用迁移。当前职责见[存储](../modules/storage.md)、[任务](../modules/tasks.md)和[资源选择设计](../plans/resource-selection.md)。

## 实现

生产 RelationalTaskStore 显式注入 Manager.Runtime。核心既有 task_checkpoints.selection_report_json 物理列只保存 selectionDecision：结论、时间、选定标题、组/层级、策略/parser 版本、报告主键和 SHA-256。任务候选、接受来源、附件、精确身份及恢复路径继续由现有核心关系记录保存；决定和任务/audit 同事务提交，保留 revision/CAS。

完整 report 放运行库现有 diagnostics，component=resource-selection-report。写入预算 250ms；先尝试运行写入，再提交核心，运行失败只记录脱敏 warning，不阻断核心任务。失败核心事务可能留下未引用诊断，按现有保留策略清理。读取按主键、任务、component 和摘要核对；runtime repair 重用整数 ID 也不会串报告。报告单份最多 256 KiB，沿用 diagnostics 的 14 天/最新 100000 条清理，受既有每批 1000 条的清理边界约束。

报告缺失保留核心决定；过期后的人工历史报告不使用当前规则重建。明确换源时清空旧决定，形成新报告。GraphQL 仍返回可空 selectionReport，报告不存在时任务详情不显示说明。独立 SQLite 旧 store 仅为兼容/测试入口，不作为生产分库方案。

## 验证

- 完整 Go suite 通过：`/tmp/moji-report-go-final.log`。
- taskruntime/storage 完整 race 通过：`/tmp/moji-report-race-final.log`；最终报告边界与恢复 race 通过：`/tmp/moji-report-race-boundaries.log`。
- 回归覆盖核心 audit 回滚、CAS、运行写入失败、读取失败、实际 TTL 清理、运行记录 ID 重用、人工报告过期不重解释、核心备份恢复和运行库 repair 后继续入库。
- 文档检查及 git diff --check 通过。没有前端 schema/交互变化，复用现有前端产物。

## 本地运行交付

私有 release：`.storage/runtime-refresh/report-storage-20261123T164250Z`。启动信息与两库停机备份保存在其私有目录，未纳入 Git。

停实例后通过 SQLite backup API 备份两库并 quick_check；单独将 388 份既有完整报告原样写到运行 diagnostics，在核心替换为精简决定，逐份校验 SHA-256。转换前后任务 ID/revision/generation 与完整 schema_migrations 账本一致；核心 FK/完整性通过。无启动自动转换、配置自动迁移或版本递增。

新实例 healthz 正常，前端仍与原产物一致，历史报告 GraphQL 读取通过，无启动错误。2316 个任务全部保留，配置原字节不变；core 10/runtime 9/策略 7 与原实例一致。后端 SHA-256：`55c3dd4b4b613c2f6f785dfd3ef55dddee00a085570277af04ce8db2643fd483`。没有提交或推送。


## 同日后续：报告完全归运行库

用户明确执行移除独立核心“精简决定”。现有核心任务候选、来源身份、附件、文件计划、路径、恢复检查点和状态审计继续负责执行/恢复；不再新增选择摘要、报告 ID 或摘要引用。

已删除 selectionDecision 类型、Task 私有决定字段、核心相关读写、恢复依赖及配套旧测试。core v10 的历史 selection_report_json 列保持物理定义以保护已应用迁移校验和，当前全部 NULL，生产关系查询和写入不再访问。报告全部属于运行 diagnostics，按 task_id 最新记录关联；运行包装包含 revision/generation/来源摘要及报告本体，防止显示未来状态或旧来源说明。没有新增 schema、版本或生产自动迁移/旧数据回退。

核心提交成功后才尝试写运行报告，失败不撤销核心操作；仅报告变化不写核心、不增加 revision/audit。核心事务失败与过期 CAS 不发布新报告。详情读取报告，摘要/分页不读；TaskStore.List 按 128 个任务 ID 批量取报告，避免每任务扫描。报告限额与运行保留策略不变。获取/提交不重建人工报告，明确换源才新建，避免过期后使用当前策略解释历史。

验证：完整 Go suite `/tmp/moji-runtime-only-all-final.log`、taskruntime/storage race `/tmp/moji-runtime-only-race.log` 通过；最终报告关联、未来 revision、源变更写失败、过期和恢复回归见 `/tmp/moji-runtime-only-boundaries.log`。文档检查及 git diff --check 通过。恢复测试覆盖核心备份恢复、runtime repair 后无报告仍能继续执行。

本地交付目录：`.storage/runtime-refresh/runtime-only-report-20261123T170543Z`。停实例后 SQLite backup API 备份两库并核对完整性；388 份报告本体逐份摘要校验不变，包装为运行任务关联记录；核心 388 条摘要全部清空，剩余独立选择记录 0。离线转换前后任务 ID/revision/generation、完整迁移账本一致，核心完整性/FK 通过。

新实例 healthz 正常，全部 388 份历史报告逐份 GraphQL end/selected 校验通过；2316 个任务保留，配置原字节不变，core 10/runtime 9/策略 7 不变，启动无错误。二进制 SHA-256：`00776b0967eda3c0dc889f4051d4be27db262923fefe554a134b9dd87e2e6a78`。没有前端契约或布局变化，沿用原前端产物。未提交、未推送。
