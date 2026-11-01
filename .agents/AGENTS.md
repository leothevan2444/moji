# Moji Agent 主入口

Moji 是自托管的 Stash 自动化伴侣：跟踪演员与发行，通过 Jackett 寻源、qBittorrent 下载，整理并交付影片，再编排 Stash 扫描、生成和识别。最终目标是让自动化长期可靠运行，让用户只在需要判断或处理故障时介入，主要媒体库体验由 Stash 提供。

## 开始工作

1. 查看 `git status --short` 和近期提交，保留已有改动；核对任务涉及的实现与测试。
2. 所有改动遵守 [工程规范](standards/engineering.md)；涉及界面、文案或用户流程，先读 [产品体验规范](standards/product.md)。规范约束新增与重构，不能以旧实现中的不良模式作为新增依据。
3. 阅读 [架构地图](architecture.md)，再按下表读取相关模块。普通局部修改无需通读全部历史复核。
4. 需要选择后续工作时读 [TODO](TODO.md)；架构判断与严重性见 [2026-11-01 规范复核](reviews/2026-11-01-project-standards.md)，早期全项目验证见 [2026-10-27 复核](reviews/2026-10-27.md)，功能专项由模块导航。用户当前任务优先于 TODO 的建议顺序。
5. 按 [开发与验证](workflows/development.md) 完成改动和适当检查；跨会话任务按 [阶段复核与交接](workflows/review.md) 留下证据和下一步。

| 任务 | 按需阅读 |
| --- | --- |
| 文案、视觉、交互、用户路径与产品取舍 | [产品体验规范](standards/product.md) |
| 代码组织、接口、数据、异步、测试与交付 | [工程规范](standards/engineering.md) |
| 新建任务、寻源、重试、任务列表、实时事件 | [任务与实时工作台](modules/tasks.md) |
| 订阅、检查作业、VR 策略、发行复核 | [订阅与发行决策](modules/subscriptions.md) |
| 演员、场景选片、发现、StashBox 缓存、头像 | [演员与发现](modules/catalog.md) |
| 文件选择、合并、字幕、交付、Stash 入库 | [媒体交付](modules/media.md) |
| SQLite、迁移、备份、恢复、性能观测 | [存储](modules/storage.md) |
| qB 管理、清理、配置、客户端、运行和发布 | [集成与运维](modules/operations.md) |
| 路由、交互、缓存、设置、本地化、无障碍 | [Web UI](modules/web.md) |
| 设置页重构、配置结构调整与实际旧配置迁移 | [设置契约](plans/settings-redesign.md)；[实施复核](reviews/2026-10-28-settings.md) |
| 演员历史全量检查 | [未实现的设计提案](plans/performer-full-scan.md)；不要将其当作已有能力 |
| 整理或新增指引 | [文档组织与维护](README.md) |

## 始终保持的边界

- 当前实现、测试和配置是核对事实的依据；设计文档描述约束，提案描述未来。发现冲突时核实并修正文档，不为迎合旧文档改坏代码。
- 所有生产任务创建经过 `internal/taskflow`；任务执行与身份去重归 `internal/taskruntime`。有效番号是任务身份门槛，不按模糊标题猜测。
- Stash `custom_fields` 是订阅标记真值；Moji 持久保存检查、发行决定、任务和恢复意图。`REVIEW` 产生独立复核记录，策略跳过和拒绝不创建任务。
- VR 仅依据 `scene.Code` 转为大写后是否包含 `VR`；不从标题、标签或其他属性推断。
- 重要决定、文件选择、删除和外部操作意图先持久化，再执行副作用。上游失联保留最近确认状态；不能把超时视为操作失败或把断连视为资源消失。
- 生产数据库统一由 `internal/storage.Manager` 管理。未知 schema 拒绝启动；新增有序迁移，不能删库重建或改写已应用迁移。
- 长任务属于应用生命周期，浏览器刷新或关闭不丢进度；进程重启后如何恢复由具体领域定义。现有订阅检查会标记中断，不会自动续跑。
- 后端、GraphQL、Web 和必要的持久化改动同步交付。GraphQL 修改后生成绑定，界面文案走 i18n，用户错误采用稳定代码及 correlation ID。
- 用户界面优先显示需要处理的事项、真实进度、原因和下一步；复用现有组件。检查操作沿用 `faBinoculars`；qB 状态文案统一使用官方映射。
- 文案先核实实际能力再用用户语言表达；删除重复说明和无关技术叙述。新增与重构沿用现有设计语言，走查完整用户路径及异常/返回，不以“技术无感”隐藏失败。
- 产品取舍、默认行为或主要流程存在选择时向用户复核，说明建议与影响；已经确认的决定直接沿用。常规技术实现自行推进，未回复不代表同意。
- 不提交本地配置、API key、Cookie、数据库、日志或备份；测试使用临时目录、fake/httptest，真实服务测试按显式环境开关运行。
- 按变更范围验证，区分静态检查、自动化通过、外部服务实测和未验证项。文档修改不会自动授权操作正在运行的服务或生产数据。

## 常用验证

从仓库根目录执行，具体适用范围见 [开发与验证](workflows/development.md)：

```sh
GOCACHE=/tmp/moji-go-cache go test ./...
npm --prefix web test
npm --prefix web run build
npm --prefix web run i18n:check
python3 .agents/scripts/check_docs.py
git diff --check
```
