# 指引文档的组织与维护

主入口是 [AGENTS.md](AGENTS.md)。这里解释文档本身如何维护，不是每次任务的必读背景。

## 结构与单一职责

```text
AGENTS.md                       仓库根：发现转发
.agents/
  AGENTS.md                     唯一的规则正文与任务导航
  README.md                     文档组织和更新规则
  architecture.md               定位、运行链、所有权地图
  TODO.md                       唯一的未完成工作清单
  modules/                      当前实现及必须保持的模块约束
    tasks.md                    任务、工作台、事件恢复
    subscriptions.md            检查、策略与发行决定
    catalog.md                  演员、发现、元数据、缓存和图片
    media.md                    文件、交付和 Stash 入库
    storage.md                  持久化、迁移和恢复
    operations.md               下载器、配置、运行、发布
    web.md                      前端结构和交互约定
  workflows/
    development.md              开发命令和验证矩阵
    review.md                   阶段复核和跨会话交接方法
  reviews/2026-10-27.md          有日期与代码基线的复核快照
  plans/performer-full-scan.md   尚未实现的功能设计
  plans/settings-redesign.md     设置结构、统一设计与配置迁移要求
  references/agent-design.md     外部实践及本仓库的取舍
  scripts/check_docs.py          本地链接和结构检查
```

根目录 `AGENTS.md` 和 `.github/copilot-instructions.md` 只转发到同一正文，避免为不同工具各维护一套规则。实际目录名是 `.agents`，不另建 `.agent` 副本。

| 内容 | 唯一维护位置 | 更新时机 |
| --- | --- | --- |
| 全局产品边界、强约束、任务导航 | 主入口 | 长期约束或阅读路径改变 |
| 运行链、模块所有权 | 架构地图 | 依赖、生命周期或职责改变 |
| 已实现行为、数据契约、异常恢复 | 对应模块 | 改变该行为的同一变更 |
| 未实现需求与技术选择 | `plans/` | 需求澄清、设计调整或实施推进 |
| 未完成事项、优先级、验收、下一步 | TODO | 接手、受阻、完成或重新评估 |
| 某时点结论、命令结果、局限 | `reviews/` | 阶段结束或专项审计 |
| 安装、用户操作和运维教程 | 仓库 `docs/`、根 README、Web help | 用户可见能力或运维方式改变 |

模块文档链接现有用户文档和源码，不复制完整 schema、GraphQL 类型或大段实现。`docs/` 中有日期的评估和迁移验收属于历史证据，不能把里面的旧运行路径当作当前架构。

## 更新规则

1. 模块文档在开头写“核对日期、代码基线、状态、主要源码”。稳定约束与当前实现分开表述；修改时以相关源码再次核对。
2. TODO 使用稳定 ID，写明问题证据、完成条件和下一步。提案与已确认问题分开，用户没有要求时不擅自开展提案。
3. 不在阶段复核中维护第二份活动 checkbox；复核通过 TODO ID 引用问题。完成事项从活动 TODO 移除，证据进入新复核或 Git 提交记录。
4. 阶段复核有固定日期与基线。后续变更追加新复核；仅纠正错字、链接或明确事实错误可修改旧记录，并注明纠正。
5. 新复核生成时，同时更新主入口和 TODO 的“最近复核”链接。不要靠文件名排序猜当前复核。
6. 跨会话任务直接在对应 TODO 条目下留下简短交接：已改文件、已跑验证、剩余项、下一条可执行动作。大任务的实现细节写在一个 `plans/` 文件，通过 TODO 链接；不另建重复进度日志。
7. 提案落地后把长期决定吸收到模块文档；删除已失效的实施步骤。仅有历史解释价值的内容由 Git 保存。
8. 全局指引只保留高频规则。只有独立领域出现明显阅读负担时才拆模块，不按每个 Go 包或 React 组件建说明书。
9. 这些文件应随代码纳入版本管理。忽略的本地数据和工具个人设置继续保留在 `.gitignore`。

## 本轮校准

2026-10-27，以 `86f69e5` 为代码基线重写。旧 `.agents/docs` 的两份 7 月阶段复核和架构、存储、Web 说明已退出活动文档；相关长期边界吸收至新模块文档，已完成问题从 TODO 删除。

旧 `performer_full_scan_implementation.md` 的产品要求保留在 [历史全量检查提案](plans/performer-full-scan.md)，移除过时的共享 runtime 库方案和大段假定接口/SQL，明确当前没有该能力。

旧文档此前被 Git 忽略，本轮已保留临时原文供核对；长期产品约束和未实现需求以新体系保留，后续修改由 Git 记录。

## 文档验收

```sh
python3 .agents/scripts/check_docs.py
git diff --check
git status --short --untracked-files=all
```

检查器验证本体系中的本地 Markdown 链接和片段锚点、根入口、核心文件及模块导航；不检查远程页面是否可用，也不能证明文档语义与代码一致。语义核对按 [复核流程](workflows/review.md) 完成。

设计依据见 [外部实践与取舍](references/agent-design.md)。
