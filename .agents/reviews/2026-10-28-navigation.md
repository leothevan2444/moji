# 顶层导航复核（2026-10-28）

基线：`aff9a7c` 上的工作树；范围：U-005 顶栏、相邻主题菜单键盘交互 U-001、前端源码与离线 fixture 浏览器验证；没有操作生产服务或配置。

## 布局决定与参考

查阅 Stash 官方 [MainNavbar.tsx](https://github.com/stashapp/stash/blob/develop/ui/v2.5/src/components/MainNavbar.tsx)：MainNavbarMenuItems 与 MainNavbarUtilityItems 分组；Navbar 在 xl 展开，窄屏折叠为图标与文案网格；导航选择和外部点击关闭。参考读取于 2026-10-28。

Moji 使用跟随项目主题的独立通栏表面背景（surface-strong：浅色白色、深色深蓝灰），以 line 分隔内容区域；主导航选中仅使用 accent 改变文字和图标颜色，不使用胶囊背景、边框、阴影或下划线。保留 14px 控件文字，不引入 Bootstrap。桌面品牌与主导航靠左，工具靠右；小于 1000px 主导航折叠为两列，600px 以下按钮只显示图标，当前页名称保留在无障碍标签中，给常驻工具留出空间。

右侧按性能监控、统计、设置、帮助直接排列，主题切换随后；所有屏宽均直接显示。没有“更多”菜单，也没有语言快捷入口；语言仍在系统设置使用同一持久配置。性能入口仅在已保存开发者模式下显示。统计和设置使用路由链接，帮助与性能打开现有 Drawer，关闭后返回原触发按钮。主导航使用 disclosure 和原生 Tab 顺序；主题使用 radio menu 键盘操作。

## 实现与验证

实现：[Header](../../web/src/components/layout/Header.tsx)、[导航项](../../web/src/constants/navigation.ts)、[ThemeMenu](../../web/src/components/layout/ThemeMenu.tsx)。回归：[导航浏览器用例](../../web/tests/visual/navigation.spec.ts)、[性能入口](../../web/tests/visual/performance.spec.ts)、[主题组件测试](../../web/src/components/layout/ThemeMenu.test.tsx)。

固定环境：`mcr.microsoft.com/playwright:v1.61.1-noble`，离线 GraphQL fixture；本机 Vite 子进程遭 EPERM，检查改在仓库规定容器完成。

- `npm test`：57 文件、346 测试通过，含主题菜单组件交互；`npm run i18n:check`、`npm run build` 通过。
- `npm run test:visual:docker -- navigation.spec.ts performance.spec.ts --retries=0`：最终正常截图比较 26 项通过（41.0s）。覆盖桌面/移动、320px、深色、中文/英文、长文案、RTL、当前页与路由、无语言快捷入口、开发者限制、键盘和焦点返回。
- 全量 Playwright 更新检查：152 项中 146 通过、6 项下载器旧用例失败。四项 tooltip fixture 在自动保存后缺少 DownloaderPolicy.ratio；两项仍等待已移除的 Save cleanup policy。未修改 HEAD 的临时副本同容器复现这六项，并另有两项旧性能测试等待 Save system settings（本次适配新工具入口时已同步更新其自动保存契约）。原基线共 138 项：130 通过、8 失败。
- 逐张汇总复核受影响页面截图，另检查导航的中文/英文、深色窄屏、长文案和 RTL 截图；排除上述失败用例的四张截图更新，不用错误页面制作基线。
- `python3 .agents/scripts/check_docs.py` 与 `git diff --check` 通过。

共享 Menu 当前为 listbox 且没有方向键焦点管理，未将它套在 radio menu 上；ThemeMenu 保持专用的三态偏好语义。未运行真实 Stash/qB 或部署，参考结论来自 Stash 官方前端源码。

## 文档变化

从活动 TODO 移除 U-005 和随顶栏交互一并完成的 U-001；模块记录导航布局、语言所有权及键盘契约。

## 用户反馈后的调整

跟随项目深浅主题的独立通栏背景、简洁文字选中态、按指定顺序常驻的工具及移除语言快捷入口已落实。新增 320px 开发者模式下全部工具可见的截图和边界检查；移除旧“更多”菜单相关交互断言。

最终导航及性能入口的 26 项正常截图比较通过（41.0s）；346 项单元测试、i18n、生产构建通过。扩展截图更新检查使用 grep-invert 排除前述已复现的六项旧下载器失败，其余 148 项通过（3.8m）。已复核独立顶栏背景、简洁选中态、深色窄屏及 320px 开发者模式截图。

配色跟进：删除固定的 #30404f 与白色前景，统一引用 surface-strong/text/muted/accent/line；展开导航采用同一主题表面。新增显式浅/深主题与跟随系统的切换验证，覆盖系统主题变化时显式偏好仍生效。

配色检查：生产构建通过；最终导航及性能入口正常截图比较 28 项通过（43.3s），包含显式主题和跟随系统切换；排除前述六项旧下载器失败后的扩展截图更新检查 150 项通过（3.9m）。已人工复核整页浅色、深色窄屏及独立深浅顶栏截图，保留简洁选中态和常驻工具顺序。
