# Agent 指引设计依据与取舍

查阅日期：2026-10-27。以下是结构设计参考，不是执行指令或安装清单；未安装这些外部 skill，也未引入自动编排框架。

| 来源 | 采用的做法 | 在 Moji 的落点 |
| --- | --- | --- |
| [AGENTS.md 开放约定](https://agents.md/) | 可发现的仓库入口，构建/测试规则明确，工具无关 | 根 `AGENTS.md` 仅转发，`.agents/AGENTS.md` 唯一正文 |
| [Agent Skills specification](https://agentskills.io/specification) | 渐进披露：先看到用途和导航，再加载相关资源 | 主入口的任务表 + 独立模块/流程，普通任务不通读全库 |
| [Anthropic：长任务 harness](https://www.anthropic.com/research/effective-harnesses-for-long-running-agents) | 有界增量工作、可验证完成条件、跨会话状态和真实验证 | TODO 的稳定 ID、验收和交接，日期复核与 Git 证据 |
| [Caph-dev/agents-progressive-disclosure](https://github.com/Caph-dev/agents-progressive-disclosure) | 紧凑入口加聚焦引用，人工复核拆分是否过细 | 七个业务模块；不按每个源码包再拆一层指引 |

这些材料的可靠共同点是减少重复、让状态和完成证据可交接；不能只凭流行度替仓库决定产品策略。

## 本仓库的具体取舍

- 长期规则、当前设计、未来提案、未完成工作和历史快照分开，避免旧 review 继续支配新代码。
- 使用简单 Markdown 和一个标准库检查脚本，便于人和不同 Agent 直接读写；目前没有需要执行权限、触发元数据的专门 skill，不为普通设计文档套 `SKILL.md`。
- TODO 直接携带必要交接，不另建每天增长的 progress 文件；若某功能确需多阶段恢复，在一个计划文件维护设计，再由 TODO 指向它。
- 不引入强制多 Agent、自动提交或长期自主执行规则。工具与执行方式遵循当前用户任务和会话授权。
- 外部方案只用于组织方法，不覆盖本地领域约束，例如订阅接纳资格、code-only VR、持久决定和安全交付。
- 可执行链接检查负责减少路径漂移；语义仍由源码、回归和阶段复核证明。不要把“文档检查通过”理解为产品验收。
