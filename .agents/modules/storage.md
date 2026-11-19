# 存储、迁移与恢复

核对日期：2026-11-19；代码基线：`ec7b093`；状态：当前源码接线，不表示运行实例状态。

主要源码：[Manager](../../internal/storage/manager.go)、[连接](../../internal/storage/database.go)、[布局](../../internal/storage/layout.go)、[迁移](../../internal/storage/migrations.go)、[schema 注册](../../internal/storage/schema.go)、[离线 CLI](../../cmd/moji-storage/main.go)。用户操作统一见[数据库运维](../../docs/database-operations.md)。

## 布局与所有权

默认 data root 为相对 CWD 的 `.storage/database`，`-data-dir` 可覆盖；配置路径独立。CURRENT 是安全的单目录名纯文本指针，与 generation 目录同级。generation 内有 moji.db、runtime.db、cache/stashbox/cache.db，以及仅在核心库引用时必需的 legacy-task-events.jsonl.gz。

OS 锁保证单实例/维护所有权。新目录完成初始化、验证、fsync 才发布 CURRENT；已有目录缺指针、文件或有效 schema 时失败，不扫描猜 generation、不自动替换派生库。旧平铺布局和 JSON 指针不能直接启动。

生产 repository 统一由 Manager 注入，每文件默认一个 writer、四个 query-only readers，写 admission 默认 64，短 SQLite busy wait 配合 context 重试预算。事务内不做网络/媒体操作；关闭 worker 和 HTTP 后才关闭数据库。

| 真值 | 存放与恢复角色 |
| --- | --- |
| 影片/任务身份、revision、附件、选择、路径、恢复检查点、发行决定、检查 item、重要 audit、外部命令与清理执行状态 | moji.db，FULL；必须保护备份 |
| 最新进度、qB 观测、交互搜索、索引会话/资源、冷却及执行记录 | runtime.db，NORMAL；丢失后显式离线 repair，各领域定义重建/中断 |
| StashBox 实体和分页 generation | cache 库，NORMAL；显式离线 repair 后回源 |
| 应用全局配置，包括自动清理策略 | 所选 config.yaml，经 config store 验证和原子保存；无数据库配置回退 |

重要决定不依赖跨文件事务。任务更新使用 revision/CAS 和 generation fence，阶段决定与重要 audit 同事务；高频纯进度合并到 observation。runtime 重建不能丢掉核心已确认路径、来源或删除意图。

## 生产 schema 与领域数据

唯一生产注册表是 schema.go：当前 core 1–9、runtime 1–7、cache 1。领域内旧 sqlite_schema.sql/独立构造器用于兼容或测试，不是 Manager 的迁移入口。

| 数据 | 所有权与关键约束 |
| --- | --- |
| 任务影片、文件计划、附件与检查点 | 核心关系表；task_file_plans/task_files 保存计划，task_attachments/task_ingest 等保存附件与入库数据；旧 file_plan_json 只属旧 schema/查询适配 |
| 订阅、复核和待发行 | 核心库；发行决定与移交状态同事务，关系不随取消订阅丢失 |
| 活跃度和截止时间 | subscription_performer_state；activity_json 与独立 next_check_at/check_failures，专用更新不重写影片关联，到期 keyset 每批最多 128 |
| 规则包与选定来源 | core v4–v6；manifest、active/previous、持久 pin 和获取前私密来源，同任务接收原子提交 |
| 清理执行 | downloader_cleanup_state 与 external_commands；core v9 已删除 download_policies，策略只在 YAML |
| 交互搜索 | runtime v5；取消落盘保留结果，重启标记中断；具体预算归[索引器](indexers.md) |
| 网络冷却 | runtime v6；私有 origin/出口摘要，不保存 URL/凭据；容量不驱逐有效冷却 |
| 索引执行 | runtime v7；indexer_executions JSON 为真值，索引/统计列派生；无旧 diagnostics 自动转换 |

获取前私密来源单份上限 256 KiB，不对外投影。接受任务与规则 pin 同事务；原始 torrent/精确 magnet 持久化后清除私密来源并解除 pin，保留非敏感审计。未知版本、身份不一致或缺少恢复来源拒绝读取。规则与缓存细节归[索引器](indexers.md)，执行记录保留规则也由该模块唯一维护。

schema_migrations 保存有序版本、名称与不可变 checksum。未知/未来版本、非法非空库或已应用内容变化拒绝启动，不 reset/drop。新增迁移更新注册、managed mapping 和升级/恢复测试；DDL/ledger 同事务，非空受支持升级前创建保护备份。

影片身份与活跃度开发期曾经由用户单独授权修改基础 schema 并人工转换本地数据；这是历史例外，不是新变更的迁移规则。[影片转换复核](../reviews/2026-11-15-film-task-creation.md)记录了当时范围。旧 schema-16 的 prepare 导入不能替代后来影片关联要求；所有任务必须通过启动时有界身份验证。

## 备份、恢复与资产

备份用 SQLite online backup API，核对完整性/FK、摘要与 manifest；启动及每小时一份，保留小时/日/周桶，保护备份不自动清理。包含配置、可用交付回执和引用的历史归档，不含影片、Stash 数据库或图片缓存。`-backup-dir` 可选其他挂载点，本地默认备份不证明离机恢复能力。

规则资产从已完成的数据库快照枚举并核对 manifest/文件摘要，VerifyBackup 校验资产，RestoreCore 恢复到新 generation。WithAssetMaintenance 协调快照至复制与规则清理/修复的窗口，不占 SQL 事务。清理旧规则先移除核心登记再删目录，active、previous、持久 pin、在途 lease 和更新目标受保护。

restore/activate/compact/repair-derived 要停实例、保留原数据，在独立 generation 验证后切换。CLI 不组装 qB/Stash 客户端，不重放外部命令；不能复制运行中的主库而忽略 WAL。配置/回执的恢复位置及命令见用户运维文档。

## 保留与性能

通用 diagnostics 保留 14 天及最新 100000 IDs，每批最多 1000；重要 audit 不按该 TTL 清理，索引执行表有独立规则。工作查询使用 128 项 keyset，事件统计用 SQL 聚合。StashBox generation/预算见[目录](catalog.md)。

Manager 每五秒采样三个库的连接、admission、等待/执行、错误及空间。databasePerformance 推送现有缓存，不重新打开 Manager 或执行维护；失败保留上次结果，进程重启清零计数。界面平均执行耗时不是单 SQL 延迟或 p95。

## 修改时验证

覆盖 unknown/future/checksum 拒绝、preflight backup、初始化中断与坏布局；managed repository、CAS/持久意图、runtime 丢失恢复与关闭等待；备份资产、保护保留、busy/context/磁盘故障。真实迁移、修复或清理是独立运行操作，文档治理不授权执行。
