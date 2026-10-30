# 阶段复核：种子候选与任务详情（2026-10-29）

基线：`32144ec` 加本轮工作树改动；范围为源码、离线测试和固定 Playwright 容器截图，未操作生产服务。

## 阶段判断

U-003/U-004 已完成：所有手动选种候选入口提供用途明确的链接和做种数，任务详情的关联种子已整合到种子信息分区。

## 实现与边界

- [共享候选展示](../../web/src/components/tasks/TorrentCandidateLinks.tsx)用于任务寻源处理（独立抽屉及处理台）和发现页。magnetUri 为磁力，link 为下载地址（兼容返回 magnet），details 为站点详情；只展示对应协议的原始地址，不从下载地址猜测详情地址。无可用链接明确提示，并禁止选用没有下载/磁力地址的候选。
- HTTP(S) 链接按用户点击打开新页，使用 noopener/noreferrer 和 no-referrer，不预取、不展开原始下载 URL。拒绝相对地址、脚本协议和带 userinfo 的 URL。上游下载 URL 仍可能携带 Jackett API key 或站点 passkey，保持已有 GraphQL 的授权边界，不声称这些地址已匿名化；磁力使用本机协议处理程序。链接和提交按钮是独立交互元素。
- [Jackett 解码](../../pkg/jackett/client.go)将未提供、null 或负数做种数统一为 -1，保留真实 0；沿用整数 API，两个手动入口均将负数显示为未知。此为索引器报告值，不是实时 qB 做种连接数。既有历史记录没有的观测无法追溯补造。
- [TaskDetail](../../web/src/components/tasks/TaskDetail.tsx)将唯一的 DownloadManager 通过 TaskFiles 插槽置于静态种子信息与文件组之间，[TaskDrawer](../../web/src/components/drawers/TaskDrawer.tsx)不再额外渲染。内嵌模式不再套第二层卡片或回链当前任务；保留真实 qB 名称、官方状态、进度、刷新、详情、失联/缺失提示和危险操作确认。仍使用原有一组下载器查询与可见性轮询，独立文件选择抽屉不增加下载器查询。
- 发现页 Jackett 卡片改用自然高度及正常流的底部操作区；长标题两行截断并保留 title，避免原固定高度和绝对定位在移动端裁切、重叠新增链接。

## 验证

| 检查 | 命令或证据 | 结果 |
| --- | --- | --- |
| Go 受影响包 | `GOCACHE=/tmp/moji-go-cache go test ./pkg/jackett ./internal/graphqlapi ./internal/taskruntime` | 通过；默认离线测试，未启用真实服务集成 |
| Web 自动化 | 固定容器中 `npm test`；新增链接边界测试单独运行 | 全量 348 项和新增 7 项通过；覆盖链接不触发提交、缺失/0/未知做种数、协议限制、单一轮询及删除确认 |
| 本地化 | 固定容器中 `npm run i18n:check` | 通过；已运行 i18n:export |
| 生成 | `npm --prefix web run codegen` | 已更新手动候选 details 查询及 Web bindings；无 schema 结构变化 |
| 构建及浏览器 | 固定 `mcr.microsoft.com/playwright:v1.61.1-noble` 容器；`npm run build`，`npm run test:visual -- task-torrents.spec.ts task-workspace.spec.ts --retries=0` | 构建通过；28 项通过，包括 18 项新种子用例和 10 项原工作台回归 |

最终检查：构建及全部 28 项浏览器用例通过。新增 [task-torrents.spec.ts](../../web/tests/visual/task-torrents.spec.ts) 覆盖中英文、深浅色、桌面/移动、未知及零做种数、无链接、长标题、分区内控制及危险操作确认；18 张新基线均已逐张查看。发现页另加标题区和操作区边界不重叠断言。没有更新无关旧基线。

本机初始测试受 spawnSync sh EPERM 和 httptest 端口限制阻挡；Go 改在允许本机临时监听的权限下运行，Web 使用仓库规定容器通过。浏览器依赖均为 fixture；未验证真实站点链接和外部协议程序，也未执行真实 qB 控制。

## 文档变化

从活动 TODO 移除 U-003/U-004，并在 [Web 模块](../modules/web.md)记录共享展示、URL 边界和详情组合契约。尚未提交、推送或部署。
