# 存储、迁移与恢复

2026-11-03 新增 [runtime v6](../../internal/storage/schema/runtime_v6.sql)：indexer_network_cooldowns 保存来源/代理出口的私有 SHA-256 key、失败次数（1–7）、冷却与过期毫秒时间，不保存 URL/账号/凭据。最多 8192 条，仅回收过期项，不驱逐有效冷却；普通成功请求不写该表，失败/已有失败后的成功/挑战解除才更新。到期冷却保留失败计数至额外 30 分钟，最长 Retry-After 24 小时；这不是查询缓存或任务真值。v5→v6 有序升级保留已有资源和搜索 run，重复迁移验证通过；未对实际库执行迁移。

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前生产接线。

主要源码：[Manager](../../internal/storage/manager.go)、[连接与写入](../../internal/storage/database.go)、[布局](../../internal/storage/layout.go)、[迁移](../../internal/storage/migrations.go)、[schema 注册](../../internal/storage/schema.go)、[离线 CLI](../../cmd/moji-storage/main.go)。操作说明以 [数据库运维](../../docs/database-operations.md) 为入口。

## 布局与所有权

2026-11-02 新增 [runtime v5](../../internal/storage/schema/runtime_v5.sql)：交互搜索 run、逐站报告与可分页候选。过期只拒绝读取，不隐式重跑；新接收时清理过期 run，级联删除其展示结果。候选仅保存内部 opaque resource ID，不接收客户端定位符；候选删除不影响已转 core 的任务来源快照。查询/站点/候选共用 64 MiB 预算，事务提交后通知；重启将未完成 run/站点标为中断。实际 v4→v5 升级保留旧资源表与健康表的验证见 [迁移测试](../../internal/storage/torrent_search_migration_test.go)。

2026-11-02 内置索引工作区新增 [runtime v4](../../internal/storage/schema/runtime_v4.sql)：最近搜索/取种观测独立保存，按实例、配置指纹、活动规则修订和操作隔离；只含稳定错误代码、起止时间、关联标识与观测到的等待时间，不含凭据、URL 或原始错误。后发请求优先于迟到结果，调用方取消不记为站点失败；查询缓存命中不生成新的上游验证。容量限 8192 个命名空间/操作记录，淘汰后只能按无匹配观测表达；它们不是下载任务恢复真值。生产由 Manager.Runtime 注入，网络不在事务中；v3→v4 使用实际旧 schema 验证候选引用不丢失。见 [健康观测测试](../../internal/indexer/health_test.go)。

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

2026-11-02 核对 [schema 注册](../../internal/storage/schema.go)：工作区为 core v1 至 v6、runtime v1 至 v6、cache v1，由该文件嵌入 `schema/` SQL；内置索引相关迁移尚未对生产库执行升级。core v5 的 task_indexer_sources 保存非敏感来源身份与规则/配置修订；[core v6](../../internal/storage/schema/core_v6.sql) 的 task_indexer_selections 保存接受意图、资源 ID 及获取前的完整来源（单份上限 256 KiB，含凭据/定位符，不对外投影）。接收与 task-resource 规则 pin 同事务；原始 torrent 或精确 magnet 检查点提交后，私密来源设空并解除 pin，保留选择身份；任务删除触发清理 pin。未知记录版本、身份不一致或缺少恢复来源拒绝读取。实际 v5 schema 在隔离 generation 中升级，保留任务及来源审计；不以删 ledger 冒充升级。备份中的未获取来源同样按敏感数据保护，规则目录继续纳入 Manager 附属资产。领域目录下旧 `sqlite_schema.sql` 仍存在于兼容/测试路径，不是生产 Manager 的迁移入口。

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

2026-11-01 内置索引工作区新增 [core v4](../../internal/storage/schema/core_v4.sql)：规则库 manifest/文件摘要清单、active/previous 修订、更新作业、持久引用和激活 audit。完整规则目录先 fsync，核心事务再原子写入引用、活动修订、作业完成与 audit；路径为当前 generation 的 `indexers/bundles/<commit>/`。`indexerdefs.Service` 复用注入的 Manager.Core，启动核对损坏时仅将内置规则能力标为不可用，保留活动记录；已在 cmd/moji 组装。

备份从已经完成的 SQLite 快照枚举该快照登记的规则 manifest/文件，核对 inventory 与摘要后复制资产；VerifyBackup 验证资产齐全，既有 RestoreCore 将其恢复到新 generation，可离线重新加载。旧版没有规则表的快照仍可验证。`Database.WithAssetMaintenance` 协调备份快照至资产复制的窗口与规则清理/修复；不占 SQL 事务，普通数据库写入仍可进行。无引用旧版先删核心登记再删目录；active、previous、持久引用、在途 lease 和更新目标均受保护。验证见 [规则管理测试](../../internal/indexerdefs/service_test.go)。

2026-11-02 工作区新增 [runtime v2](../../internal/storage/schema/runtime_v2.sql) 的账号隔离派生会话（24 小时、512 条、单条 256 KiB）与 [runtime v3](../../internal/storage/schema/runtime_v3.sql) 的服务器端候选引用（30 分钟、5000 条/128 MiB、单条 256 KiB）。会话按账号配置及认证相关规则派生 key，保存 Cookie/UA，人工挑战与答案不落库。候选引用冻结配置和定位信息，恢复注册表时以轻量 Retain 保护未过期引用对应的规则版本；实际取种仍须 Acquire 验证文件。Resolve 返回独立在途 lease，清理到期记录不撤销已经开始的获取。Resources.Maintain 由应用生命周期运行/等待，关闭定义更新 worker 后再释放注册表引用；已在 cmd/moji 组装。已选任务的来源/原始 torrent 仍必须另行写 core，不能把这些 runtime 行当恢复真值。验证见 [资源引用测试](../../internal/indexer/resources_test.go) 与 [私有站会话测试](../../internal/indexer/builtin_test.go)。
