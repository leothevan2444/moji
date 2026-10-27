# Moji 数据库评估与重设计建议（2026-10-27）

已有修复已提交：`89c5f25 fix: isolate downloader monitoring and task event rendering`。提交后工作区干净。此轮只读调查，没有迁移、删除、VACUUM、checkpoint 或修改运行配置。

## 结论

继续使用 SQLite，保留进程内、文件式部署。当前问题主要在持久化职责、访问方式、迁移和运维机制，不是业务数据规模超出 SQLite 能力。DuckDB 的分析型定位不适合作为此次工作流数据库的替代品。

建议统一数据库基础设施；在同一个业务库中保留必须原子提交的重要状态，将可重建观测与外部缓存隔离。需要按访问模式拆表，不需要按演员、日期或任务编号横向分片。

## 一、调查范围和现状

检查了三个 SQL schema，以及全部 10 个直接操作 SQL 的生产文件：

- internal/taskruntime/sqlite_db.go、sqlite_store.go、home_overview.go
- internal/subscription/sqlite_db.go、sqlite_store.go、release_review_store.go、check_store.go、home_summary.go
- internal/downloader/service.go
- internal/stashboxcache/sqlite_store.go

同时检查调用层的任务事件、寻源队列、进度同步、文件选择、重启与交付检查点、订阅检查和发布审核，以及 config.yaml、图片缓存、旧 JSON store 的持久化路径。

实际运行进程 cwd 是项目根目录；业务库是根目录 moji.db，文件系统为本机 ext4。cmd/moji/moji.db 是另一个旧库，此次读到其中没有任务，不能与当前业务库合并统计。

当前主库有 9 张业务/元数据表，StashBox 缓存另有 6 张表。图片缓存当前使用文件，不应因旧库里有 image_cache_entries 空表而认定它仍使用 SQLite。

主库只读 quick_check 返回 ok，foreign_key_check 无违规。这里没有声称做过离线完整 integrity_check 或恢复演练。

### 现有设计值得保留的部分

1. SQLite WAL、STRICT 表、任务状态 CHECK、唯一约束和外键已经提供基本保护。
2. 任务、发布记录和人工审核决定有持久化模型，审核记录独立于演员展示窗口。
3. StashBox 可重建缓存已有独立库、TTL/清理与按小时合并访问时间更新。
4. 文件选择、重启/交付与删除意图中已经有先记录再执行、重启后核对的基础。
5. 最近提交已消除主要的不变任务重复写入和纯进度历史膨胀来源。

### 当前不合理的部分

#### P0：业务库存在破坏性升级路径

internal/taskruntime/sqlite_db.go:80-110 在 schema_version 不属于当前白名单时，进入 DROP tasks/task_events 的路径；这也会把未来更高版本视为不支持版本。internal/subscription/sqlite_db.go:38-49 有旧结构删表重建路径。

业务数据不能按缓存方式重置。遇到未知或更新版本应拒绝启动并保留原库；支持的旧版本必须通过明确的增量迁移升级。schema.sql 写版本号与后续补字段/补索引也不是统一的版本化事务。

这些 reset 还在事务内执行 PRAGMA foreign_keys=OFF，SQLite 明确规定此操作在事务内无效，不能作为可靠迁移机制。

#### P0：SQLite 引擎补丁需要升级

go.mod 使用 modernc.org/sqlite v1.38.2；运行中二进制依赖也相同，模块生成代码确认内置 SQLite 3.50.3。SQLite 官方披露 WAL-reset 缺陷可能影响该版本，修复版本包括 3.50.7、3.51.3 及后续版本。当前确有多个连接写同一 WAL 文件，符合需要评估的使用条件。

这是升级依据，不是当前已损坏的证据。升级应验证实际 SELECT sqlite_version()/sqlite_source_id()，而不仅检查 Go 模块标签。

#### P1：业务写入没有统一管理

任务、订阅、下载器分别打开同一个 moji.db，每个连接池 MaxOpenConns(1)。这仍是多个独立写者，并非整个应用单写者；任务和订阅甚至重复执行业务库 schema 初始化。

数据库配置也分散：下载器只对当前连接设置 busy_timeout；taskruntime 的 DSN 对重建连接设置 busy_timeout/foreign_keys，而其他入口并不完全一致。生命周期中没有统一停止 worker、等待任务、备份/checkpoint、关闭全部数据库的管理层。

#### P1：重要状态与高频观测混在一起

53 列 tasks 同时承载身份、业务阶段、qB 进度、选择结果、路径、检查点和可选 BLOB；更新使用全行 UPSERT。53 列本身不是错误，冷热混合与全量读写才是主要问题。

downloader_state 用整份 JSON 保存快照，每约 3 秒更新。特别注意：其中还包含删除来源/待确认删除意图，另有可能存在的 policy 键，不能直接把整个表当缓存迁走或清空。

#### P1：历史占用失控，缺少容量和生命周期管理

已有任务历史及索引占主库约 99.70%。新提交阻止了主要的重复增长，但没有治理既有历史、清理策略和备份/空间维护。

#### P2：读写放大与查询职责

- EventingTaskStore 每次发布事件都 List 全部任务计算统计，并在自身全局锁内执行。
- SQLiteTaskStore.List 总是读取任务所有字段、全部行并排序。
- subscription.SQLiteStore.List 先查演员 ID，再逐个 Get，每个 Get 又查状态和发布列表，属于 N+1。
- 每次订阅 Put 删除演员全部关联，再逐条重写发布和关联，最后做全局孤儿扫描。
- 人工审核数据只有 release_key + record_json；筛选/排序字段埋在 JSON，服务层读取全量后处理，审核读取还可能触发迁移/核对写入。
- 检查运行状态与整个 Items 数组重复存储在 payload 中；目前有 100 条保留上限，是合理基础，但 item 应能独立更新。
- StashBox putSnapshot 每次删掉该 generation 的全部 pages/items 后重写；未来大演员目录会形成多次累积重写。缓存目前只有约 3 MB，不是本轮主因。
- 时间格式混用 RFC3339Nano 可变精度和固定九位小数，依赖文本排序时应处理同一秒内的边界。新结构应统一精度和比较方式。

## 二、真实数据构成与更新重要性

### 主库空间

以下为只读 dbstat 页占用，包含各类相关索引；MB 为十进制。在线库各查询时点略有不同。

| 数据 | 数量 | 表及索引占用 | 主库已分配空间占比 |
|---|---:|---:|---:|
| task_events | 2,592,229 条 | 922.88 MB | 99.6973% |
| tasks | 277 个 | 1.585 MB | 0.1712% |
| subscription_release_reviews | 435 条 | 0.598 MB | 0.0646% |
| 订阅状态、发布、关联、检查记录 | 11 / 291 / 337 / 8 条 | 0.401 MB | 0.0434% |
| downloader_state | 1 个 snapshot 键 | 0.188 MB | 0.0204% |
| 元数据及其他 | — | 0.029 MB | 0.0031% |

主库文件约 925.86 MB，WAL 文件约 1218.55 MB。WAL 是版本/事务页面，不能当作独立业务内容再加进数据类别占比。其文件大小也不等于尚未 checkpoint 的数据量；已 checkpoint 的空间可以留着复用，需读维护指标判断。

task_events 表本体约 334.62 MB，两条历史索引合计约 588.26 MB。约 99.96% 的事件没有发生 stage 变化，且消息统一为 task updated；这不等于每条都无业务意义，但显然不是优质的审计记录模型。不能仅用 old_stage=new_stage 一条规则直接删除所有数据。

StashBox 独立缓存约 2.97 MB，有 722 个 scene、16 个 performer、16 个 snapshot、961 个 snapshot item。cmd/moji/moji.db 旧空库约 0.156 MB，不计入当前业务库。

### 更新频率

30.33 秒只读抽样（每 2 秒对比行内容）：

| 数据 | 观测到的变化 | 含义 |
|---|---:|---|
| tasks | 至少 156 次行变化，涉及 27 个任务 | 目前最频繁的业务表变化，主要涉及执行/下载观测 |
| downloader_state | 10 次同一行变化 | 约每 3 秒重写一次快照 |
| 订阅状态、发布、审核、检查运行 | 本窗口 0 次 | 由检查、人工操作等触发，不能解释为永不更新 |

这是采样下界，不是 SQL tracing 的精确 UPDATE 次数。期间多次变化、或内容不变的重复 SQL，可能被采样遗漏。历史 append 数也不能用作当前写频率。

### 重要性排序

1. **不可丢的意图和决定**：人工文件选择/播放顺序、审核拒绝/跳过/下载决定、创建任务请求、重试/重启世代、删除命令及意图、已确认的交付/入库结果。必须持久化成功后才报告成功或执行不可逆外部操作。
2. **恢复必需的检查点**：队列到期时间、执行 lease、上传种子原件、Stash job ID、交付路径/回执引用。低频，但重要性高。
3. **可重新观测的状态**：速度、当前 qB state、进度、剩余空间和在线统计。写得最频繁，但可合并、延迟和重建。
4. **诊断历史和外部缓存**：可按预算保留/淘汰。

订阅开关实际还依赖 Stash 自定义字段；设置存于 config.yaml，交付回执也存在媒体目录。因此备份 moji.db 不等于完成整个 Moji 工作流的恢复准备。

## 三、建议的数据结构

建议三个本地 SQLite 文件，保持现有核心文件名：

| 文件 | 职责 | 持久性与清理 |
|---|---|---|
| moji.db | 任务、决策、选择、恢复检查点、命令和关键审计 | 业务主库，FULL 同步，备份与迁移保护 |
| runtime.db（新增） | qB 观测、进度、可重建汇总、有限诊断 | 可重建，合并写入，允许 NORMAL；丢失后重新观测 |
| cache/stashbox/cache.db（保留） | 外部内容缓存 | TTL + 容量预算，可重建 |

这是按持久性边界拆文件，而非业务分片。当前 277 个任务没有按演员/日期横向分库的需要。

### 业务库的逻辑表组

- `schema_migrations`：版本、校验和、应用版本、执行时间；一个库只有一个迁移入口。
- `tasks`：稳定 ID、code、来源、业务 stage/status、错误码、revision、世代和业务时间。保持当前唯一性语义，扩展多版本下载时再显式调整。
- `task_details` / `task_attachments`：候选、来源描述和人工上传 BLOB 等冷字段，避免进入每次进度查询。
- `task_file_selections` + `task_files`：确认状态、选择版本、各文件 kind/selected/order。按 task + attempt/generation + index 唯一，避免重启后旧 UI 文件选择应用到新种子。
- `task_checkpoints`：少量版本化 JSON，用于具体执行器内部检查点；不要把检索用 stage/status/due_at 藏进去。
- `work_items`：持久执行队列，kind、owner、attempt、due_at、lease_until、错误与重试。它只管理调度，不成为另一套重复的业务任务状态。
- 发布实体、发布—演员关系、`release_decisions`：把经常筛选的状态、code、task 引用、日期等提升为列；人工决定与演员订阅/展示窗口解耦，取消订阅不丢决定。
- `subscription_check_runs` + `subscription_check_items`：当前运行与每个演员结果独立持久化，避免每次重写整个数组。
- `download_policies` + `external_commands`：保存清理策略与需要执行/核对的 qB/Stash 命令。删除意图在核心事务中持久化后才能发送。
- `audit_events`：只记录用户决定、阶段变更、重要错误和外部操作结果，带操作 ID 和版本化详情；不记录每次进度采样。

常用查询字段用有类型的列、约束与索引；稳定关联用外键；只读外部快照和执行器私有载荷可以保留 JSON，并带 payload_version/json_valid。不会为了形式把所有 JSON 拆成键值表。

SQLite WAL 下跨多个文件的提交不具有整体原子性，所以任务、选择、命令、关键审计必须在同一 moji.db 事务中。runtime/cache 的引用不承担跨库外键约束，允许重建。

命令与外部 API 无法做一个数据库事务：采用“记录意图 -> 发命令 -> 观测结果 -> 确认”的可重试协议，依靠操作 ID 和精确对象身份核对，不能宣称 exactly once。

## 四、性能和可靠性基础设施

1. 统一 StoreManager：负责路径、连接、迁移、备份、维护与关闭；业务模块只接收 repository/事务接口，不自行打开同一个文件。
2. 每个库一个受控写入口（MaxOpenConns=1），核心操作使用有界队列/明确优先级和回压；重要意图写失败应明确失败，不能静默丢弃。2–4 个只读连接起步，query_only 保护；根据实测调整，不无限加连接。
3. 核心读改写使用短事务与预期 revision 的条件更新。网络、媒体处理、JSON 大对象构造放在事务外；失败重试整个幂等事务，不单独重放后半段。
4. 每条新连接统一配置 busy_timeout、foreign_keys、同步级别和只读限制；WAL 按库初始化并验证。核心库 FULL，runtime/cache 可 NORMAL；不为速度把重要决定改成掉电可丢。
5. 进度/速度等优先维护内存观测快照，合并成每几秒一个批量事务；不因无变化而落盘。具体 3/5/10 秒周期通过负载和 UI 新鲜度验证，不改变重要业务提交的即时性。
6. 统计使用 SQL 聚合或可重建的增量计数，避免每次任务事件读取全部任务。队列按 due_at/lease 取有限工作，订阅关联差量写，避免重复全表扫描与长事务。
7. 索引服务真实查询：到期任务用状态/时间的部分复合索引；任务列表保留匹配 created_at/id 或 updated_at/id 的排序索引；重要唯一键保留。高频 progress 不建无实际查询用途的索引。先 EXPLAIN 和基准再删索引。
8. 统一维护任务观测 WAL 总帧/已 checkpoint 帧、大小、最老读事务、写队列、SQL 延迟和错误码。优先使用受控 PASSIVE checkpoint；RESTART/TRUNCATE/VACUUM 放维护窗口。journal_size_limit 不是任意时刻的硬上限，不能靠删除 WAL 回收空间。
9. 正确生命周期：停止接单、取消/等待 worker、落关键检查点、完成允许的维护、关闭连接池。数据目录固定、可配置，启动记录实际绝对路径；防止第二个 Moji 实例同时执行业务迁移和 worker。
10. 定义可测试目标：在约定并发下核心事务 p95 < 50 ms、简单读 p95 < 20 ms、无未处理 SQLITE_BUSY、无丢失用户意图；这些是建议验收目标，不是当前已达到的承诺。

## 五、无损迁移、清理与备份

### 迁移

- 首先取消所有业务库自动 reset 路径，统一增量 migration 机制。新于程序支持版本的库拒绝写入/启动，不尝试降级。
- 迁移前使用在线 backup API 形成一致副本，记录 schema/应用版本、摘要和行数。先在副本运行完整演练。
- 小型加字段/索引在明确事务中执行；表结构重建则新建表、转换复制、校验，再事务内切换。每成功一个版本才记录版本与 checksum。
- 第一次跨文件冷热拆分采用短维护窗口：停 worker/写入，备份核心库，拆出策略和删除意图等重要部分，再复制可重建观测，按全量核心校验/外键校验验证；保留原库到验收结束。无需为了家庭应用构建复杂线上双写系统。
- 旧任务 ID、code、选择顺序、审核决定、下载身份、外部命令与恢复检查点全部核对，不只比行数。时间精度变更需有明确转换规则。
- 失败时恢复整个旧副本；若新版本已接受新业务写入，不盲目用旧库覆盖，优先向前修复或进行受控数据回灌。

### 清理建议（尚未执行）

- 旧 task_events 先完整导出/压缩并校验，再按有意义的变更、近期记录和每任务最小历史集制定保留；不能直接清空同阶段记录。
- 将低价值诊断移到 runtime.db，建议 7–30 天 + 总容量上限；批次删除 1000–5000 行，主动让出写者。
- 检查运行建议保留最近 100 次及需要的时间窗口，保护当前运行和必要关联。重要阶段/人工审核采用长期精简审计。
- 完成任务不直接当垃圾；保留任务摘要、下载去重身份、用户决定与审核关联。较老重型细节可归档。
- 缓存延续 24h TTL/30d stale retention，加容量预算和过期 generation 清理；不需要备份每个缓存。
- DELETE 后空间通常先变成 freelist，可复用但文件不立即缩小。现有约 926 MB 库可在维护窗口通过 VACUUM INTO 生成压实新副本并核对切换；注意临时空间、WAL 和备份都需要预算。此轮没有执行。

### 备份

- 核心库采用 SQLite 在线 backup API，分块生成一致副本。不要在 WAL 运行时只复制主 .db，也不要随意手动删/分离 -wal。
- 建议默认每小时一份、保留 24 份；每日 14 份、每周 8 份。迁移和历史清理前额外创建固定备份。该默认最多约一小时的数据恢复点损失，需要按实际业务要求调小。
- 至少一份离机保存并加密，限制包含凭据/签名链接的数据库和 config.yaml 的访问权限。缓存无需进入关键备份链。
- 备份后校验，定期在隔离实例恢复并验证任务、审核和恢复检查点；恢复演练必须禁止真实 qB/Stash 写操作。
- config.yaml 当前直接 WriteFile，需作为相关持久化改进采用临时文件、fsync、原子替换，并纳入备份；订阅标记在 Stash、交付回执在文件系统，需分别说明恢复范围。无需把所有设置强制迁进数据库。

## 六、执行顺序与完成标准

1. 升级并验证 SQLite 引擎、禁用破坏性 reset、建立一致备份和恢复演练。
2. 建立统一 StoreManager、读写连接策略、版本化迁移与可观测指标。
3. 拆分重要命令/策略与可重建观测，按访问模式整理表结构和局部写入。
4. 在副本演练旧历史归档/压实，随后维护窗口执行。
5. 测试并发写、长读、持续进度、重启/强杀、磁盘写入失败、迁移失败、备份恢复和真实数据规模回放。

只有“无损升级 + 正确恢复 + 数据保留受控 + 约定负载下达标 + 外部操作不会重复或遗漏关键决定”均验收，才把数据库专项标为闭环。本次是方案评估，尚未执行这套数据库重构。

## 官方依据

- SQLite WAL、单写者、跨文件事务边界及 WAL-reset 修复版本：https://www.sqlite.org/wal.html
- SQLite 在线备份：https://www.sqlite.org/backup.html
- SQLite 表结构迁移：https://www.sqlite.org/lang_altertable.html
- SQLite 外键开关的连接/事务规则：https://www.sqlite.org/foreignkeys.html
- SQLite WAL 文件保留与 journal_size_limit：https://www.sqlite.org/pragma.html#pragma_journal_size_limit
- DuckDB 分析型并发模型：https://duckdb.org/docs/2.0/connect/concurrency

原始只读空间/结构统计：/tmp/moji-db-inventory.json；30 秒变化抽样：/tmp/moji-db-write-sample.json。
