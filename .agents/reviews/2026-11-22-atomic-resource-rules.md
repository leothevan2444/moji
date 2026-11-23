# 单项资源规则与规则组组合

历史记录：同日用户后续明确允许多个条件并固定 OR；当前契约及验收见[多条件规则复核](2026-11-22-any-condition-rules.md)。

日期：2026-11-22。状态：已实现、验证并更新本机运行实例。用户批准路线一：规则只描述单项特征，过滤与优先级归规则组。当前契约见[设计](../plans/resource-selection.md)及[用户指引](../../docs/resource-selection.md)。

## 实现与取舍

规则为 ID、名称、字段，以及 value 或 minimum/maximum；删除条件数组、CONTAINS/NOT_CONTAINS 等操作符、规则内 ALL/ANY。普通文本为不区分大小写的字面子串，完整 /…/ 表示 Go RE2 正则，非法正则保存失败；数字闭区间允许一端开放，至少一端非空。大小输入 GiB、API/YAML 为精确整数 bytes 字符串。

组前置过滤保留准入/排除用途，分别拥有 ALL/ANY，默认 ALL/ANY。该组合能力支持原本跨标题和文件路径的任一条件，不迫使用户建立可嵌套规则。空准入通过、空排除不触发；决定性组合结果不受无关未知影响。优先层级仍支持 ALL/ANY 与局部反转；反转使用标签动作菜单，提供键盘移动、Escape、焦点返回和恢复匹配。前置标签仍可拖动和删除。

策略及解析版本改为 3；同步 Go/YAML/GraphQL/Web 与生成产物，不自动转换旧 YAML。历史报告保留原始解释，旧策略不重新执行；没有新增数据库 schema。共享规则编辑/删除保护、活跃组、检查范围与既有持久预览均保留。

## 配置转换与验证

本机旧配置 7 条 ANY 复合规则、22 个正向 CONTAINS 条件；拆分为 22 条特征，原组合移到组层，保留当前组、过滤和优先层级。本机一次性工具先验证可展平的结构，拒绝不可等价展平的嵌套；使用 20,000 组确定性输入对照旧/新资格、优先层级及未知状态，其他配置完整语义相同。正式写入前使用新 config loader 验证，持久备份后原子写入；工具不进入生产实现。

## 验证结果

- Go 全量 `go test ./...` 通过；resourceselection/config/graphqlapi race 通过，后续补充的 GraphQL patch→原子保存→重开配置的闭环回归也在 race 下通过。覆盖字面与正则区分、无效正则拒绝、数值边界和未知、ALL/ANY 短路、否定、复用、稳定排序，以及 v2 人工历史报告原样保留。
- GraphQL 两端绑定已重新生成；前端 83 个文件、524 项测试通过，生产构建、i18n 10 项、86 篇文档检查及 diff whitespace 通过。
- 固定 Playwright v1.61.1-noble 最终 32 项不更新基线回归通过，覆盖中英/深浅色/桌面/移动/长文案/RTL、规则表达式编辑、前置组合实际保存、层级反转/恢复与菜单 Escape/焦点、前置拖动和索引器共享排序、预览/任务与通用设置布局。只更新资源选择相关基线，新增层级标签/菜单截图并人工复核。
- 首次浏览器检查发现移除旧文案后层级组合标签缺失，已换为独立双语文案；实际保存检查补齐 group requireMode/excludeMode 到 YAML require_mode/exclude_mode 的转换，并加入真实 Store 保存重读回归。最终通过结果来自修正后的独立运行。

## 本机运行验收

沿用已授权实例更新；独立发布 `.storage/runtime-refresh/atomic-rules-20261122T152725Z`，优雅停止后离线备份 core/runtime，两份 SQLite quick_check 均为 ok。旧配置持久备份 `backups/config-atomic-rules-20261122T153236Z.yaml`，0600 且忽略。通过新 loader、20,000 组决定对照及其他配置完整语义对比后，原子应用 7→22 条规则转换；一次性源码已清理，无生产自动迁移。

新实例 `/healthz` 为 200/ok，主页与独立新前端产物一致；实际 GraphQL 策略版本 3，规则类型只有 id/name/scope/value/minimum/maximum，组组合和层级 negate 正常。更新前后 2,288 个任务均完整保留，核心 schema 10/runtime schema 8 未变，启动无 error/panic/fatal。发布目录保留 verification.json、旧启动信息、配置和数据库备份；没有提交或推送，也没有主动执行真实站点搜索或媒体操作。
