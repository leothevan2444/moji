# Homepage Widget 首版设计与验收

日期：2026-11-11；核对基线：`47ec468` 后的功能变更；状态：已实现并通过隔离验收，未部署。用户在设计后明确要求执行，采用原建议的接入与访问方案。

## 已采用的设计

用户路径：在 Homepage 配置 Moji 服务卡片 → 查看四项任务数 → 点击卡片进入 Moji。首版使用 Homepage 内置 `customapi`，没有原生 `type: moji`；四项数字由单个只读接口返回，推荐 30 秒刷新。配置与排错见 [用户文档](../../docs/homepage-widget.md)。

依据：[Homepage Custom API](https://gethomepage.dev/widgets/services/customapi/) 的 JSON 字段映射、数字格式、刷新间隔和请求头能力，以及 [Services](https://gethomepage.dev/configs/services/) 的服务卡片配置约定。

| 显示名称 | field | 口径 |
| --- | --- | --- |
| 待处理 | `attention` | 非 COMPLETED 且 BLOCKED |
| 进行中 | `active` | 未完成且既不属于待处理、也不属于等待资源，包含已排队任务 |
| 等待资源 | `waiting` | SOURCING、PENDING 且持久 sourcing_next_check_at 非空；到期未接手仍算等待 |
| 24h 入库 | `completedLast24Hours` | 当前 COMPLETED 且持久 ingest_completed_at 位于服务器 `[now-24h, now]` 闭区间 |

计数单位为现存 Moji 任务。前三项互斥；订阅、待发行、发行复核记录不加入统计。入库不等于下载或搬运完成；历史缺失完成时间不补猜。删除和重新选种后随当前任务投影变化，不作为不可变事件报表。

## 已实现的接口与边界

`GET /api/v1/widget/homepage` 返回四个非负 JSON 整数，不含任务标题、ID、路径、凭据或其他详情。URL 为版本契约，无筛选、请求体或写操作。标准 HEAD 沿 net/http GET 路由处理。

```json
{"attention":3,"active":8,"waiting":12,"completedLast24Hours":5}
```

成功及应用错误均设置 `Cache-Control: no-store`。有效空库返回四个零；依赖缺失返回 503/WIDGET_UNAVAILABLE，读失败返回 500/WIDGET_QUERY_FAILED，5 秒查询截止返回 504/WIDGET_QUERY_TIMEOUT。错误体为 `error` 对象，包含 `code`、通用 `message`、UUID `correlationId`，不输出内部错误原文；日志记录相同 ID。请求取消传播给查询，不返回假成功或旧数据。

沿用现有 Moji 网络访问边界，无专用 key、功能开关或设置页。能够访问该路由的客户端可读四项数量。代理认证通过 Homepage 的 headers/Basic Auth 配置；不使用 Stash 或 Jackett 凭据。浏览器 href 与 Homepage 服务端 widget.url 独立配置，无需新增 CORS。

## 实现所有权

- [home_stats.go](../../internal/taskruntime/home_stats.go)：领域计数及聚合；生产 Relational/旧 SQLite 读取一次聚合，不加载任务详情、payload、观测 overlay，不访问外部服务。
- [home_overview.go](../../internal/taskruntime/home_overview.go)：首页复用聚合，仍在同一个只读事务中读取有界样本；开发内存路径共享任务分类函数。
- [homepage.go](../../internal/controller/api/homepage.go)：窄只读依赖、HTTP 映射、截止和错误日志；[main.go](../../cmd/moji/main.go) 注入运行服务或未配置下载器时的本地存储。
- 未支持统计的存储直接报错，不能静默回退全量 List。没有新增缓存、worker、数据库迁移或前端/GraphQL 契约。

长期契约已吸收至 [运维模块](../modules/operations.md#homepage-widget)、[任务模块](../modules/tasks.md#首页与-widget-共享统计) 和 [Home 文档](../../docs/home-workspace.md)。

## 验收证据

- 完整 `GOCACHE=/tmp/moji-go-cache go test ./...` 通过；本次 HomeStats/HomeOverview/Widget 测试的 race 检查通过。现有测试需要本地监听 socket，完整 suite 在允许 socket 的权限环境执行。
- 回归覆盖生产/旧 SQLite/内存与首页一致性、24h 两端及偏移时区、过期/未来/缺失时间、等待到期、删除和重新选种、冷 payload 版本异常、取消、空库、HTTP 错误和无下载器应用装配。额外验证不支持统计的 store 不回退 List。
- 实际 Homepage 使用本地镜像 v2.7.0，镜像 ID `sha256:0193309878fc07a9016fc8bb36a54206bc1b1d17226049d7e0250bceef2d4078`。隔离 Moji 仅使用 `/tmp` 数据及合成任务，无真实上游配置。
- 28 条合成任务经接口和真实 Homepage 显示 `3 / 8 / 12 / 5`。新增一条待处理任务后，观察到约 29.9 秒自动刷新为 `4 / 8 / 12 / 5`；卡片 href 可打开临时 Moji。
- 删除两项 mappings 后，实际只显示待处理和 24h 入库。临时制造读取失败后，接口返回 500/WIDGET_QUERY_FAILED，Homepage 显示中文「API 错误」，没有四个假零值；恢复后接口再次返回正常计数。
- 最终使用两个临时容器及专用 Docker 网络，直接按文档以 `http://moji:10000/api/v1/widget/homepage` 查询，浏览器确认四项正常显示；href 使用独立主机端口。验收容器和专用网络在结束时移除。
- 在 28 条任务上，直接执行相同 SQL 100 次，平均约 0.039 ms；该观测不是大规模或 HTTP 延迟承诺。
- 截图与临时测试日志保留在 `/tmp/moji-widget-acceptance`，不纳入版本管理。生产实例、配置与媒体未修改。

本轮仅验收上述 Homepage 版本；不承诺其他版本的错误文案或旧值保留策略。新增规模优化、原生 Homepage 集成与 Moji 统一鉴权不在首版范围。
