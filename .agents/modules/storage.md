# 存储、迁移与恢复

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前生产接线。

主要源码：[Manager](../../internal/storage/manager.go)、[连接与写入](../../internal/storage/database.go)、[布局](../../internal/storage/layout.go)、[迁移](../../internal/storage/migrations.go)、[schema 注册](../../internal/storage/schema.go)、[离线 CLI](../../cmd/moji-storage/main.go)。操作说明以 [数据库运维](../../docs/database-operations.md) 为入口。

## 布局与所有权

默认 data root 为相对 CWD 的 `.storage/database`，可用 `-data-dir` 覆盖；配置路径独立选择。`CURRENT` 是只包含安全单目录名的纯文本文件，与 generation 同级。

```text
.storage/database/
  CURRENT
  .moji-storage.lock
  initial-example/
    moji.db
    runtime.db
    cache/stashbox/cache.db
    legacy-task-events.jsonl.gz   仅在核心库引用时必需
  backups/
```

OS 目录锁保证单实例/维护所有权。新目录完成初始化、验证、fsync 后才发布 CURRENT；已有目录缺指针、文件或有效 schema 时明确失败，不扫描目录猜 generation，不自动创建替代库。旧根目录平铺库和 JSON 指针不能直接启动。

生产 repository 使用唯一 Manager 注入的连接。每个文件默认一个 writer、四个 query-only readers，写 admission 默认容量 64；使用短 SQLite busy wait 和 Go context 重试预算。事务内只做数据库操作，不等待网络或媒体工具。

## 按恢复角色拆分

| 角色 | 内容 | 恢复要求 |
| --- | --- | --- |
| 核心 `moji.db`，FULL | 任务身份与 revision、冷附件、文件选择、持久路径、检查点、发行决定、检查 item、重要 audit、下载器策略和外部命令 | 保护备份与明确恢复；不能重建丢弃 |
| `runtime.db`，NORMAL | 可替换进度、qB 快照和诊断 | 显式离线 repair 后重观测 |
| cache，NORMAL | StashBox 实体与分页 generation | 显式离线 repair 后从上游重建 |

必须原子提交的决定留在同一个核心库。不依赖跨文件原子事务；不能把删除意图和确认来源放进可丢弃 snapshot。恢复 runtime 后，核心库里的已解析下载路径仍能支持待入库任务。

任务使用 CAS revision 拒绝旧写入、generation 隔离重启 worker。阶段决定与对应重要 audit 在同一事务；高频纯进度合并到 observation，避免核心重复写与历史膨胀。

## schema 的唯一生产来源

2026-11-01 核对 [schema 注册](../../internal/storage/schema.go)：当前为 core v1/v2/v3、runtime v1、cache v1，由该文件嵌入 `schema/` SQL。领域目录下旧 `sqlite_schema.sql` 仍存在于兼容/测试路径，不是生产 Manager 的迁移入口。

`schema_migrations` 保存有序版本、名称和不可变 checksum。未知/更新版本、未识别非空库、已应用迁移内容变化必须拒绝，不允许 reset/drop。新增迁移并更新注册、managed row mapping、索引和升级/恢复测试；DDL 与 ledger 同事务。

非空受支持升级前生成受保护备份。旧 schema-16 导入由 datamigration/离线 CLI 显式 prepare，逐字段核对任务、选择、复核和检查项，完整校验历史归档；activation 单独进行。

## 备份、清理和性能

备份使用 SQLite online backup API，验证完整性/FK、摘要和 manifest；初始及每小时一份，定期保留小时/日/周桶，保护备份不自动清理。包含配置与可用回执、引用的历史归档；不包含影片、Stash 数据库或图片缓存。`-backup-dir` 可以选择其他挂载点，默认本地备份不能证明具备离机灾难恢复。

restore/activate/compact/repair-derived 需要停实例并保留原数据，在独立 generation 准备验证后切换。CLI 不建立 qB/Stash 客户端，不重放命令。不能只复制运行中的主 DB 或手动删除 WAL。

诊断保留 14 天和最新 100000 IDs，每批最多 1000；重要 audit 不按诊断 TTL 清理。工作查询按 128 项 keyset 批次；事件统计用 SQL 聚合。缓存预算和 generation 规则见 [目录](catalog.md)。

现有 Manager 每五秒采样三个库的连接、写 admission、等待/执行、错误和空间；开发者模式开启后通过 `databasePerformance` 推送缓存结果，订阅不再打开 Manager 或做维护。失败保留上次数据，计数进程重启清零；界面平均执行时间不代表单 SQL 延迟或 p95。实现见 [performance.go](../../internal/storage/performance.go)。

## 修改时验证

- unknown/future/checksum 拒绝、支持版本升级与 preflight backup、初始化中断、坏指针、缺文件。
- managed repository、并发 CAS、选择与外部意图原子性、runtime 丢失后的恢复、关闭等待。
- backup manifest/资产完整性、离线恢复、保护备份保留；磁盘错误/busy/context 限期与失败不丢原状态。
- 数据修复、迁移或清理是独立运行操作。本轮文档核对和本地测试不授权再次对真实数据执行这些操作。

2026-10-31 核心 schema 新增 v3：[待发行迁移](../../internal/storage/schema/core_v3.sql)。subscription_upcoming_releases 保存可恢复生命周期及快照，subscription_upcoming_performers 独立保存演员关系，不级联依赖订阅检查状态。SQL 分页、发行日期/到期索引及 revision 写保护；发行决定和移交状态共用核心事务。纳入现有保护备份与恢复，回归见[待发行测试](../../internal/subscription/upcoming_test.go)。
