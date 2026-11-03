# 演员、发现与元数据目录

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前实现。

主要源码：[performer](../../internal/performer/service.go)、[演员详情](../../internal/performer/performer_detail.go)、[所选场景解析](../../internal/performer/performer_batch_queue.go)、[discovery](../../internal/discovery/service.go)、[metadata](../../internal/metadata/service.go)、[registry](../../internal/metadata/stashbox_registry.go)。

## 所有权与读取

演员身份来自 Stash。metadata 从 Stash 已配置的 StashBox endpoint 构建 registry，并依据演员 stash IDs 解析远程演员；不能相信浏览器传来的任意 endpoint、演员名或场景元数据。

演员 profile 与场景列表独立读取，打开 profile 不应先加载远程目录。场景视图合并本地 Stash 和 StashBox，暴露 `loaded/remote count`、`complete/stale` 等状态。StashBox 按需增量加载；本地演员场景目前仍用 `per_page=-1` 读取，再合并筛选。

演员列表仍调用 `AllPerformers`，在服务/GraphQL 侧过滤排序分页。因此“页面分页”不等于上游读取已经有界。这是当前 TODO 中的规模问题，不要重新把已修复的批量选片当作同一个问题。

发现页通过 registry 查询 StashBox 目录，并提供统一内置/Jackett 资源搜索；服务器保存 run 及候选身份，选中的远程引用在服务端解析后进入 taskflow。发现服务不负责实现第二套下载逻辑。

## 演员所选场景下载

输入仅含 performer ID 和稳定 scene keys，最多 100 项，空/重复/超限先拒绝。客户端不提交 `code/inLibrary/title` 作为业务依据。

服务只解析选中的 key，验证 endpoint 属于当前映射、场景确实属于目标演员；不枚举整个演员目录，不生成图片，不为了创建任务再二次拉同一个场景。Stash 本地选择作为已入库跳过；远程有效 code/title 通过 `QueueResolvedScene` 进入 taskflow。

每个输入按原顺序返回 `QUEUED/SKIPPED/FAILED` 及稳定 reason code。未知、陈旧或伪造 key 必须产生可解释结果，不能因为 UI 曾经展示过就信任它。

Web 跨页保存稳定 key 的集合，变更演员时清空；提交全部选中 key，不能只提交当前页。仅 `QUEUED` 成功项从选择中移除，失败、跳过及缺失响应项保留供查看或重试。

相关回归：[选中项定向查询](../../internal/performer/performer_batch_queue_test.go)、[跨页选择](../../web/src/app/routes/performerSceneSelection.ts)、[路由测试](../../web/src/app/routes/PerformersRoute.test.tsx)。

## StashBox 可重建缓存

[stashboxcache](../../internal/stashboxcache/service.go) 保存演员实体与分页场景快照。缓存有 generation、完整性和新鲜度含义，旧的完整 generation 可以为刷新失败提供回退，但不完整集合不能冒充完整历史。

生产使用 storage Manager 的 cache owner。默认 TTL 24 小时、stale window 30 天、容量预算 512 MiB；保留 active 与最近 inactive generation，清理有界且包含索引/关联页。容量是 live-page 预算，删记录后文件长度不必马上缩小。

读取 generation 和关联数据要在同一 SQLite 快照完成，避免刷新/淘汰时拼接出不存在的组合；保存采用差量写而非整库刷新。使用现有 freshness policy，不为绕过上游压力另建缓存。

## 图片代理与头像

[imagecache](../../internal/imagecache/service.go) 管理带鉴权的服务端图片代理和文件缓存，真实凭据不暴露给浏览器。图片缓存不是 SQL 业务状态，不受 core 恢复覆盖。

演员列表使用 `/api/images/{key}?variant=avatar`；详情仍用原图。头像限于 192×320、不放大小图；透明 PNG，其余通常 JPEG；AVIF 保持原图回退。派生版本共享缓存预算、清理和原图获取，同一派生请求合并，最多两个源图同时解码，过大像素图在解码前拒绝。

具体格式及回归见 [头像说明](../../docs/performer-thumbnails.md)。不要通过把列表换回原图来回避派生失败，也不要改变所有图片消费者的默认 URL。

## 修改时验证

- 目录与创建分别测：局部 profile 不应触发场景枚举，选 N 项只允许与 N 成比例的解析。
- 场景归属、endpoint 碰撞、陈旧 key、请求顺序与部分成功。
- 缓存新旧 generation、stale 回退、清理并发与完整性，变更持久化时测 managed 路径。
- 头像的透明度、并发与超大图拒绝；浏览器测试验证列表/详情的不同请求。

2026-10-31 发现页新增与搜索、推荐平级的待发行区块，默认最近 8 条，/discover?section=upcoming 展开服务端分页。数据归订阅领域核心库，只展示仍有关联订阅演员的记录。演员所选场景与发现页远程场景创建均校验未来日期，返回 RELEASE_NOT_AVAILABLE；原始种子与仅番号入口保持显式用户意图。见[待发行复核](../reviews/2026-10-31-upcoming.md)。

2026-10-31 演员影片待发行状态：演员列表分页在任务关联前，通过注入的 UpcomingStateReader 按当前可见 StashBox 场景 key 批量读取持久状态（至多本页 100 项），不遍历待发行列表或额外请求上游。仅非 HANDED_OFF 记录投影为 upcomingState；已入库与已有任务优先显示。WAITING/UNCONFIRMED/HANDOFF_PENDING 分别显示待发行/待确认/发行处理中，不提供创建和选择；订阅事件合并后刷新当前分页，重连校准。
