# Stage1：实现与文档一致性治理（2026-11-19）

源码基线：`ec7b0935d6a76689641ab8c2005096b3f6eaea1f`；开始时工作区干净。范围：仓库源码、指引、用户文档、双语帮助、无消费者前端代码和 CI；不检查真实实例状态，不操作配置、数据库或上游。用户确认先完成一致性治理，明显不协调页面的视觉调整留给 Stage2。

## 阶段判断

保留现有业务与视觉，把任务入口、身份、持久化、索引与 Home 的当前事实收敛到按职责组织的文档。已有规范继续作为约束；本轮清理不引入新的业务层、UI 框架或运行配置。

## 旧结论校准

| 原描述或问题 | 当前源码与治理结果 |
| --- | --- |
| 手动番号/资源可直接新建；分隔符与补零是身份别名 | [taskflow](../../internal/taskflow/films.go)统一确认 StashBox 影片；[film](../../internal/film/film.go)保留完整番号。模块、README、媒体与双语帮助同步修正 |
| 文件计划位于 tasks.file_plan_json/schema 9 | 生产[核心 schema](../../internal/storage/schema/core.sql)使用 task_file_plans/task_files；旧 JSON 列是 standalone/查询适配。用户存储说明同步校准 |
| downloader 策略和观测存放含混 | [managed 存储](../../internal/downloader/storage.go)把外部命令/执行状态留核心、观测留 runtime；全局策略只在 YAML，core v9 删除策略表 |
| G-002 旧存储/寻源用户说明待收敛 | 已完成本次文档核对，退出 TODO；另发现并修正 schema 注册、缓存配置路径、v4 示例、首页订阅及完整入库判据 |
| A-001 四业务分区且重复 FieldLabel | 当前只剩 metadata/tasks/resources，FieldLabel 已共享；保留真实的跨业务编辑职责问题，移除过期证据 |
| 索引实施仍未完成、主页显示规则失败观测区 | 当前功能已落地；Home 显示站点搜索统计/执行记录，规则作业归设置；旧阶段文字退出当前模块 |
| 无消费者的 Menu、导出和配套样式 | 已核对源码、测试、静态/动态导入及构建入口；移除 Menu、三个 index.ts 和浅/深色配套 SCSS。保留 ThemeMenu 和有消费者的 hooks/测试设施 |

当前职责以[架构地图](../architecture.md)和八个模块为入口。metadata.NormalizeEndpoint 的注释不再声称会去掉结尾斜杠；实现不变。索引器内容从任务/运维/Web 的长流水中收敛到[当前索引模块](../modules/indexers.md)，UI 模块保留组件消费与交互契约，不复制后端完整预算。

## 文档生命周期

- 当前模块按职责改写，核对日期/源码基线置顶；取消按日期反复追加被后续覆盖的实现步骤。本轮 `.agents` Markdown 从 496166 字节减至约 382 KB，减少约 23%；这是文档字节量，不是模型 token 实测。
- 已完成的设置计划删除，长期契约归 Web/任务/用户设置，证据保留于原设置复核。
- 内置索引原 498 行计划压缩为[固定来源与采用边界快照](2026-11-01-builtin-indexers-design.md)，阶段/schema/实际验收仍由原复核保存；完整旧文档由 Git 保留。
- Widget 和清理方案移为[Widget 验收快照](2026-11-11-homepage-widget.md)、[清理验收快照](2026-11-13-downloader-cleanup.md)。开头注明历史日期及授权范围，不能据此重启或迁移当前实例。
- 数据库旧评估/重设计加历史身份和当前运维入口；维护已完成 TODO 及章节变动造成的历史链接，不改写旧测试结果。
- 文档检查覆盖 `.agents`、README、docs 和两种语言 Web help，验证本地链接、锚点和指引路由；CI Go job 在业务测试前执行检查器回归与文档检查。检查器不证明语义正确或远程链接可用。

## 验证

| 检查 | 命令与结果 |
| --- | --- |
| 检查器回归 | `python3 .agents/scripts/check_docs_test.py`：6 项通过；覆盖用户文档/双语帮助、失效锚点、越界链接、模块导航和必需入口 |
| 文档检查 | `python3 .agents/scripts/check_docs.py`：78 份 Markdown 的本地链接、锚点与路由通过 |
| 前端单元测试 | `npm --prefix web test`：79 文件、473 项通过 |
| 本地化 | `npm --prefix web run i18n:check`：严格审计/导出一致性及 4 文件、10 项测试通过 |
| 前端构建 | `npm --prefix web run build -- --outDir /tmp/moji-stage1-web-dist`：通过，未覆盖开发实例 web/dist |
| 导航/设置浏览器 | 固定 Playwright v1.61.1-noble 容器运行 navigation.spec.ts、settings-workspace.spec.ts：30 通过、6 截图失败；原始 ec7b093 同样 30/6，六张实际图逐字节相同，属于既有基线漂移；未更新基线 |
| diff | `git diff --check`：通过 |

原沙箱内 Vitest/Vite 因版本脚本 spawnSync sh EPERM 无法加载配置；相同命令在允许子进程的执行环境通过。Go 仅修改一处失实注释，未改业务，未重复运行 Go suite、真实站点、媒体集成或 Docker 产品镜像 smoke；不能把本轮前端/文档结果当作这些能力的验收。远端 Actions 未运行，未提交、推送或部署。

## 剩余边界

Stage2 的语义色/主题回退、页面协调、确认交互、日期格式调用及已有设置截图漂移见[U-006](../TODO.md#u-006)。设置编辑职责 A-001、运行接口 A-002、上游读取 S-001 和未实现产品需求仍按唯一 TODO 管理；本轮没有擅自改变其业务语义。

## 历史证据导航

仅在核对原决定或测试环境时读取：

- 全项目与架构：[2026-10-27](2026-10-27.md)、[2026-11-01 规范复核](2026-11-01-project-standards.md)。
- 设置：[分区与资源规则](2026-10-28-settings.md)、[文案](2026-10-29-settings-copy.md)、[布局](2026-10-29-settings-layout.md)。后续 LOCK 命中语义以当前任务模块为准。
- 交互：[导航](2026-10-28-navigation.md)、[候选与任务种子](2026-10-29-task-torrents.md)、[日志](2026-10-31-logs.md)、[待发行](2026-10-31-upcoming.md)。
- 索引：[请求与差分](2026-11-03-cardigann-requests.md)、[完成验收](2026-11-03-indexer-acceptance.md)、[统一执行记录](2026-11-08-indexer-executions.md)。
- 影片/入库：[识别设置](2026-11-07-stash-identify-parity.md)、[影片任务](2026-11-15-film-task-creation.md)、[检查诊断](2026-11-15-subscription-check-diagnostics.md)。
