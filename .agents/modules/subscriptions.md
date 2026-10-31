# 订阅检查与发行决策

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前实现。

主要源码：[服务](../../internal/subscription/service.go)、[检查作业](../../internal/subscription/checks.go)、[reservation 投影](../../internal/subscription/check_activity.go)、[发行策略](../../internal/subscription/release_policy.go)、[持久复核](../../internal/subscription/release_review.go)。

## 订阅与检查是不同操作

订阅标记位于 Stash performer `custom_fields`，默认 key 为 `moji_subscribed`。Moji 核心库存自己的检查状态、发行实体及关系和处理决定，不通过本地布尔值替代上游真值。取消订阅不删除持久复核决定。

`startSubscriptionCheck` 先保存 run/reservation，再返回，执行使用应用 context 和 30 分钟期限。省略 IDs 表示全部订阅演员，显式空列表无效，所选检查最多 100 个不同演员。

| 入口 | 接纳资格 |
| --- | --- |
| 定时检查、检查全部 | 接纳时只选已订阅演员 |
| 手动显式选择演员 | 可以包含未订阅演员；UI 确认，执行不自动订阅 |
| 已接纳 run 的 item | 执行时不再次要求订阅，避免中途取消订阅导致既有承诺静默落空 |

reservation 覆盖完整输入并持续到 run 结束。已有 run 完全覆盖的新后台请求可复用；部分交叠报 `SUBSCRIPTION_CHECK_IN_PROGRESS`；互不交叠可以执行。后端 reservation 是多页面/多客户端下的最终判断，按钮 pending 只是展示。

进程启动会将遗留运行和未完成 item 标记 `INTERRUPTED/SERVICE_RESTARTED`，不会自动接着执行。保留最近 100 次、所有活动 run 和最近一次定时 run。浏览器刷新只恢复查看，不改变生命周期。

## 增量目录与调度

目录分页目前每页 40 条：首次 backfill 最多 6 页，后续 poll 最多 2 页，遇到已知/日期边界可能更早结束。手动检查未订阅演员也不等于“全历史扫描”。完整枚举是另一份 [未实现提案](../plans/performer-full-scan.md)。

调度 worker 每五秒校准设置和下一 deadline；部分 reservation 冲突时在 30 秒内重试调度，并显示实际下次时间。不要偷偷增大分页或并发来满足另一类功能。

## 策略与复核

- 分类考虑单人、多人、合集特征、未知元数据及日期范围。现有纯分类代码仍在 subscription 内，尚无 `internal/releasepolicy` 包。
- VR 只看 `strings.Contains(strings.ToUpper(scene.Code), "VR")`，可以收紧其他策略，不能放宽。
- `DOWNLOAD` 尝试经 taskflow 创建/关联任务；`REVIEW` 建立独立持久待复核记录；`BLOCK` 策略跳过不建任务。历史内部 `QUEUED` 决策并不表示已有下载任务，要看复核状态和 task linkage。
- 同一发行按来源 endpoint 和 scene ID 形成身份，跨演员关联；实际任务另按番号去重。不能靠相似标题合并影片。
- 人工“下载此片”已表达本片授权，不重新套用自动策略；服务器重读决定、任务和媒体库状态，必要时关联并发创建的原任务。
- 人工拒绝持久保存，后续轮询不重新询问；改全局策略也不自动撤销。只有显式“重新考虑”恢复待复核。
- 决策、历史保留和每演员最近 25 条展示不是同一层；展示截断不能删除决定。批量最多 100，返回逐条结果，失败项保持可重试。

完整产品交互见 [发行复核](../../docs/release-review-design.md)。

## UI 数据边界

Home 投影只返回有限活动/历史摘要；详情通过 run ID 按需读。演员按钮只查可见和已选 ID 的 reservation，不拿 Home 历史充当所有演员的状态表。

检查状态采用可见性受控轮询：活动时两秒、空闲十秒，隐藏标签页暂停；页面挂载和恢复焦点读取快照。订阅实体结果继续通过已有订阅事件更新。不能为了统一“实时”而移除当前有意存在的轮询，或让每张卡片各开 timer。

检查图标沿用 `faBinoculars`。状态读取中或失败时阻止重复提交；明确说明已有 run、失败原因和跳转入口。概要见 [检查作业](../../docs/subscription-checks.md)，其中旧的全部演员 reservation 叙述需结合当前接纳时过滤逻辑理解。

## 修改时验证

- 定时/全部/手动未订阅入口分别覆盖；检查过程中订阅变化不破坏已接纳输入。
- 重复请求、交叠、部分失败、期限、关闭和重启中断；持久 run 与 item 一致。
- 策略测试必须包含 code-only VR、策略收紧、REVIEW 不建任务、拒绝保持、取消订阅后决定保留。
- GraphQL 错误码、本地化、Home 有界摘要、刷新恢复与按钮 reservation 一起验证。

## 待发行影片（2026-10-31）

[待发行服务](../../internal/subscription/upcoming.go) 在现有发行决策前保存未来影片到核心库专表，不创建任务或发行复核。原本允许自动下载而日期未知的作品进入复核，包括“全部时间”和合集。

[到期 worker](../../cmd/moji/upcoming_worker.go) 接入应用生命周期，启动及每分钟扫描最多 32 条到期记录，独立于演员目录分页，服从订阅自动检查开关。未来影片最多每日核验一次，临近发行日按业务时区零点检查。来源失联、影片不存在或归属失配保留最近快照，每小时重试；无有效订阅的记录保留、暂停并从展示隐藏。

`system.release_timezone` 留空或 auto 时跟随 Moji 运行环境系统时区（含容器 TZ）；支持从系统时区文件解析，保留 Local 自定义 TZif 回退。设置 > 系统 > 常规可手动填 IANA 时区，也支持 YAML。来源日期按来源日历日期比较，原值另存；列表返回服务端 today，浏览器无需猜测 Local 的 IANA 名称。到期核验在进入下载前重新确认当前订阅，Stash 失联不创建任务。

到期核验最新元数据、当前订阅、入库情况、人工决定和下载策略；先保存 HANDOFF_PENDING，发行决定与移交状态共用核心事务。任务创建继续走 taskflow，重启通过任务身份去重恢复关联；HANDED_OFF 保留历史。失败自动创建重试前应用当前策略，人工拒绝保持有效。未知日期转交复核。

发现/演员所选场景校验未来日期，人工复核下载重读有 scene ID 的来源日期。原始种子和仅番号入口不额外猜测日期，旧任务不自动修改。验证见[待发行复核](../reviews/2026-10-31-upcoming.md)。

再次检查遇到已有非 HANDED_OFF 待发行记录时，captureUpcoming 更新原记录的元数据、日期和演员关联，保留首次发现时间，不写入普通任务/复核流程；日期到期或变为未知时将 NextCheckAt 设为当前时间，由到期流程重新核验处理。UpcomingReleaseStates 是仅按请求 key 读取状态的有界投影，包含已暂停记录、排除 HANDED_OFF，不查询订阅或上游元数据，供演员影片卡片展示。
