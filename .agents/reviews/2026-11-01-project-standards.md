# 架构与项目规范复核（2026-11-01）

基线：`71ef6fa30623903e15cb5f63e613ab9908ac569d`；开始时工作树干净。范围：当前生产依赖、领域/传输/存储边界、配置增量保存、前端状态与组件组织、主题与代表性视觉基线、现有测试/CI 及指引。本轮建立项目规范，不实施 UI、业务、配置、数据库或部署变更。

这是静态架构与规范复核，不替代 2026-10-27 全项目测试记录或后续功能专项的验收。未审计真实运行数据、生产性能或全部安全边界。

## 总体判断

现有单进程、按业务领域组织的结构基本合理，核心状态与恢复边界明确，值得继续沿用。没有发现需要以全仓重写、微服务化、替换 SQLite 或另建通用工作流平台解决的架构证据。需要处理的是已存在的验证/查询规模缺口，以及少数正在承载过多责任的修改热点。

本轮未确认 P0 阻断问题；这不表示运行系统无缺陷。下面的高等级项分别是验证覆盖风险和静态可确认的规模边界，不是已观察到的数据损坏或用户故障。

## 应保留的边界

| 能力 | 当前证据 | 维护判断 |
| --- | --- | --- |
| 创建意图与执行分离 | [taskflow](../../internal/taskflow/service.go)、[任务 resolver](../../internal/graphqlapi/task.resolvers.go) | 生产 GraphQL 创建经 TaskFlow，runtime 保有状态机；继续沿用 |
| 领域不依赖 Moji API 生成模型 | 当前 internal 业务包的生产 import；[GraphQL resolver 边界](../../internal/graphqlapi/resolver.go) | 领域与本项目的 GraphQL generated model 分离，传输负责映射；部分领域仍使用上游 Stash/StashBox 生成类型，属于既有客户端耦合，不混为反向依赖 |
| 持久角色与迁移 | [Manager](../../internal/storage/manager.go)、[迁移](../../internal/storage/migrations.go)、[注册](../../internal/storage/schema.go) | 单 owner、core/runtime/cache、保护备份与未知版本拒绝；当前 core 已为 v3，不回退旧 schema 描述 |
| 设置跨分区一致性 | [ApplyPatch](../../internal/config/patch.go)、[settings patch](../../internal/graphqlapi/settings_patch.go)、[autosave](../../web/src/components/settings/settingsAutosave.ts) | 最新配置锁内合并、验证后原子写；前端只确认已发送改动，失败保留。不是旧的整份 automation 保存问题 |
| 页面与异步恢复 | [任务工作台](../../internal/taskruntime/task_workspace.go)、[taskRealtime](../../web/src/graphql/taskRealtime.ts)、[设置 Provider](../../web/src/components/settings/SettingsAutosaveProvider.tsx) | 任务快照/事件、领域受控轮询和设置队列各有语义；不强行统一成一种刷新方式 |
| 视觉与共享交互 | [主题](../../web/src/styles/theme.css)、[tokens](../../web/src/styles/_tokens.scss)、[设置公共组件](../../web/src/components/settings/SettingsTabCommon.tsx)、[焦点管理](../../web/src/hooks/useModalFocus.ts) | 已有可沿用的设计语言和交互组件，新功能不应另起一套 |

## 问题、严重性与尺度

| 问题 | 严重性 | 证据与影响 | 处理边界 |
| --- | --- | --- | --- |
| CI 后端测试覆盖不完整 | P1 高：验证风险 | [ci.yml](../../.github/workflows/ci.yml) 的 Go job 仅跑 graphqlapi/taskruntime/cmd/moji，storage/subscription/downloader 等包自身测试不因被依赖而执行 | 当时延续 G-001；后续已[补全包配置与本地验证](2026-11-03-cardigann-requests.md#ci-go-coverage)，未声称远端运行 |
| 演员及关联列表仍依赖上游全量读取 | P1 高：规模边界 | [performer.List](../../internal/performer/service.go) 的 AllPerformers、[场景读取](../../internal/performer/performer_detail.go) 的 per_page=-1；[upcoming](../../internal/subscription/upcoming.go) 的分页查询前也全量取演员判断订阅关系 | 延续并补充 [S-001](../TODO.md#s-001)。这是请求量/内存/延迟随库增长的边界，尚无本轮负载测量；不能靠延迟订阅真值换性能而不复核产品取舍 |
| 设置策略编辑器同时承担多个独立领域 | P2 中：修改影响面 | [PolicySettingsEditor](../../web/src/components/settings/PolicySettingsEditor.tsx) 约 1171 行，真实调用横跨 indexers/metadata/tasks/resources，共有大表单、编码与 area 分支，且重复定义 FieldLabel；settings 样式分布在多个作用域/导入文件 | 新增 [A-001](../TODO.md#a-001)。按领域拆编辑与编码，复用公共控件及有所有权的样式，保持已验证队列/配置语义；不是仅因行数要求立即重写 |
| 任务运行时消费接口过宽 | P2 中：边界约束不足 | [TaskRuntimeService](../../internal/graphqlapi/resolver.go) 同时暴露读、创建、重试、批量、同步和入库，虽然当前创建仍走 TaskFlow，类型层未限制绕过它；main/service 的组装与执行路径也较集中 | 新增 [A-002](../TODO.md#a-002)。按实际消费能力增量收窄；runtime 状态机保持单一所有者，不拆成互相复制阶段的服务 |

没有将命名偏好、行数门槛或每个小型工具函数列成架构任务。当前较长的 main/taskruntime 文件是调查线索，只有真实责任冲突、共享状态或不相关改动波及时才提取职责。已有兼容 store/接口也需查实际消费者再处理，不一概删除或继续扩散。

## 规范落点

- [产品体验规范](../standards/product.md)：源码事实转成用户表达；只保留范围、非直观行为与后果说明；沿用现有主题/控件，复核整页色彩与层次；模拟完整用户路径与失败/返回；真正产品取舍向用户复核。
- [工程规范](../standards/engineering.md)：职责与类型边界、分区配置保存、数据/副作用/恢复、生命周期、有界查询、诊断/依赖、验证和交付，及严重性尺度。
- 主入口提前引导规范；开发/阶段复核加入验收方法；模块继续描述当前事实，TODO 是唯一活动清单。规范约束后续变更，不等于本轮已经修复表中问题。

用户本轮明确要求未来遇到产品取舍先复核，已写入项目规则；没有更改既定名称、默认值、规则语义或交互流程，未产生需要用户选择的新产品分叉。

## 验证与限制

| 项目 | 本轮证据 |
| --- | --- |
| 源码审阅 | 当前 commit、生产 import/调用方、配置/存储/生命周期/前端控制点与实际 CI 文件；不是仅依据旧指引推断 |
| 视觉参照 | 人工查看已存基线：settings-workspace 的中文桌面资源页、Home 深色桌面、任务中文移动；同时核对当前主题/组件。部分旧快照仍有已退休文案，不能据截图覆盖现行 LOCK 等语义 |
| 文档完整性 | `python3 .agents/scripts/check_docs.py` 通过，覆盖 29 份 Markdown 的相对路径、锚点、入口/规范路由及可达性；格式和新增文件空白检查通过；临时最小文档树校验正常入口及缺少规范/入口导航时的失败路径 |
| 代码验证 | 本轮无业务、UI、schema 或运行配置改动，不重新跑 Go/Web/媒体/视觉 suite。历史测试数字保留在各自 dated review，不能作为本轮结果 |
| 未验证 | 本轮未操作真实服务、运行端到端流程、性能负载测试或远端 Actions；静态责任风险不是故障复现 |

上述脚本验证结构完整性，语义与体验仍由源码审阅及后续具体功能验收保证。
