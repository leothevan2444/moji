export const resources = {
  "zh-CN": { translation: {
    common: { auto: "跟随浏览器", language: "语言", loading: "页面加载中", helpLoading: "帮助加载中", collapse: "收起", expand: "展开", close: "关闭", retry: "重试", reload: "重新加载", items: "{{count}} 项", refresh: "刷新" },
    navigation: { label: "主导航", home: "主页", tasks: "任务", performers: "演员", discover: "发现", stats: "统计", settings: "设置", help: "帮助" },
    settings: { title: "配置与系统", state: "当前状态", waiting: "等待后端返回设置数据", saving: "保存中...", tabs: { connections: "连接", ingest: "入库", automation: "自动化", system: "系统", logs: "日志", about: "关于" }, connections: { save: "保存 {{service}} 连接", saved: "{{service}} 设置已保存。", dashboardPassword: "Dashboard 密码", dashboardPasswordPlaceholder: "Jackett 管理界面登录密码", username: "用户名", password: "密码", defaultSavePath: "默认保存路径", defaultCategory: "默认分类", defaultTags: "默认标签" }, ingest: { saved: "入库设置已保存。", save: "保存入库设置", mode: "入库方式", pathMap: "路径映射", transfer: "文件交付", qbRoot: "qB 下载根目录", stashRoot: "Stash 媒体库根目录", action: "交付动作", copy: "复制", move: "移动", symlink: "符号链接", mojiDownloadRoot: "Moji 下载根目录", mojiLibraryRoot: "Moji 媒体库根目录", pathMapInfo: "Moji 只把任务里的 qB 下载路径翻译成 Stash 扫描路径，不直接搬运文件。", transferInfo: "Moji 先把 qB 下载路径翻译成自己的可操作源路径，再交付到媒体库，并把同一相对路径翻译成 Stash 扫描路径。", qbRootInfo: "填写 qBittorrent 视角下的下载根目录。任务里的 ContentPath / SavePath 会先基于这个根路径计算相对路径。", mojiDownloadInfo: "填写 Moji 视角下的下载根目录。TRANSFER 模式会把上一步得到的相对路径拼到这里，得到 Moji 实际读取的源路径。", mojiLibraryInfo: "填写 Moji 视角下的媒体库根目录。TRANSFER 模式会把相对路径拼到这里，得到 Moji 实际写入的交付目标。", stashInfo: "填写 Stash 视角下的媒体库根目录。无论使用 PATH_MAP 还是 TRANSFER，Moji 最终都会把相对路径拼到这里并通知 Stash 扫描。", actionInfo: "COPY 会保留下载区原文件，MOVE 会把文件迁移进媒体库，SYMLINK 会在媒体库里创建指向源文件的符号链接。目标已存在同名文件或链接时会直接失败。", useQbDefault: "使用 qB 默认下载目录", noQbDefault: "当前未配置 qB 默认下载目录", initializeWith: "使用 {{path}} 初始化 qB 下载根目录", selectStashRoot: "请选择 Stash 媒体库根目录", noStashRoot: "暂无可用媒体库路径", initTitle: "初始化 qB 下载根目录", emptyTitle: "qB 下载根目录当前为空", initQuestion: "是否使用 qB 默认下载目录 {{path}} 初始化？", useAndSave: "使用默认目录并保存", keepEmpty: "保持为空并保存" } },
    help: { title: "Markdown 帮助", navigation: "帮助文档" },
    errors: { notFoundTitle: "页面不存在", notFoundDetail: "这个地址没有对应的 Moji 页面。", returnHome: "返回主页", routeTitle: "页面加载失败", moduleLoad: "页面模块加载失败" },
    theme: { label: "主题：{{theme}}", choose: "选择主题", light: "浅色", dark: "深色", auto: "自动", resolved: "（当前显示：{{theme}}）" },
    stats: { title: "运行概览", loadFailed: "统计加载失败" },
    home: {
      services: "外部服务", ingestPolicy: "入库策略", todos: "待办任务", todosNote: "失败项、待扫描项和长时间停滞项都放在这里。",
      noTodos: "暂无待处理项", noTodosDetail: "这里会优先显示失败、待扫和异常任务。", configure: "去配置", adjust: "去调整", noData: "暂无数据", loadFailed: "首页加载失败", retryNoResult: "任务重试失败，后端没有返回任务记录。", retried: "已重试任务：{{task}}。",
      status: { unconfigured: "未配置", stale: "数据陈旧", enabled: "已启用", error: "运行异常", pending: "待检测", configuredDisabled: "已配置未启用" },
      diagnostics: {
        stashError: "已配置但最近未联通，请检查 Stash 服务状态。", stashMissing: "Stash 必须先配置，否则入库与演员更新流程无法展开。", stashPending: "Stash 已配置，等待首次探测或最近状态已过期。",
        jackettError: "已配置但最近未联通，请检查 Jackett 服务状态。", jackettMissing: "Jackett 必须先配置，否则任务搜索与演员更新没有上游数据。", jackettPending: "Jackett 已配置，等待首次探测或最近状态已过期。",
        qbError: "已配置但最近未联通，请检查 qBittorrent 服务状态。", qbMissing: "qBittorrent 必须先配置，否则下载与落地流程无法展开。", qbPending: "qBittorrent 已配置，等待首次探测或最近状态已过期。"
      },
      blockers: { jackettSearch: "任务搜索无索引源", jackettPerformers: "演员更新无上游数据", qbDownload: "任务无法启动下载", qbLanding: "下载完成后无客户端落地" },
      stats: { scenes: "{{count}} 部影片", stashMissing: "Stash 尚未回报数据", pendingScans: "Moji 待扫任务 {{count}} 项", indexers: "索引器 {{configured}} / {{total}} 已配置", slowest: "上次搜索最慢 {{latency}} ms", transfer: "下载 {{download}} · 上传 {{upload}}", active: "活跃任务 {{count}} · 连接 {{status}}" },
      config: { user: "用户", savePath: "保存路径", mode: "入库方式", action: "交付动作", qbRoot: "qB 下载根", mojiDownloadRoot: "Moji 下载根", mojiLibraryRoot: "Moji 媒体库根", stashRoot: "Stash 媒体库根" },
      ingest: { title: "入库", missing: "缺少：{{fields}}", incomplete: "工作方式已选择，但路径映射未填完整。", mode: "入库方式：{{mode}}", none: "未选择", pathMap: "路径映射", transfer: "文件交付", copy: "复制", move: "移动", symlink: "符号链接", guidePathMap: "Moji 只负责把 qB 下载路径翻译成 Stash 扫描路径，不直接搬运文件。", cautionPathMap: "要求先配置 qB 下载根路径和 Stash 媒体库根路径；两者使用各自命名空间。", guideTransfer: "Moji 先把 qB 下载路径翻译成自己的可操作路径，再交付到媒体库并换算为 Stash 扫描路径。", cautionTransfer: "要求同时配置 qB、Moji、Stash 三套根路径；目标已有同名文件或目录时会直接失败。", guideNone: "请选择入库策略后再继续。", blockerQb: "qB 下载根路径未映射", blockerStash: "Stash 媒体库根路径未映射", blockerScan: "任务完成后无法闭环换算扫描路径" }
    },
    tasks: {
      title: "工作台", metrics: "活跃 {{active}} · 完成 {{completed}} · 待扫 {{pendingScans}} · 失败 {{failed}}",
      searchPlaceholder: "搜索任务、番号、tracker、状态",
      filters: { all: "全部", running: "运行中", completed: "完成", failed: "失败", scanPending: "待扫描" },
      sorts: { createdAt: "最新", updatedAt: "更新时间", progress: "进度" },
      actions: { sync: "同步进度", scanAll: "触发扫描", scanGroup: "全部触发扫描", retryBlocked: "重试受阻任务", retryingBlocked: "正在重试受阻任务..." },
      empty: { title: "没有匹配的任务", detail: "换个过滤条件，或者先去发现区创建任务。" },
      groups: {
        attention: { label: "需处理", description: "失败、扫描报错或需要人工回看的任务。" },
        active: { label: "运行中", description: "仍在下载、同步或等待外部状态推进。" },
        ingestPending: { label: "待入库", description: "下载已完成，但 Stash 扫描尚未收口。" },
        completed: { label: "已完成", description: "流程已闭环的任务。" }
      }
    },
    titles: { tasks: "任务 · Moji", performers: "演员 · Moji", discover: "发现 · Moji", settings: "设置 · Moji", stats: "统计 · Moji", home: "Moji" }
  } },
  en: { translation: {
    common: { auto: "Use browser language", language: "Language", loading: "Loading page", helpLoading: "Loading help", collapse: "Collapse", expand: "Expand", close: "Close", retry: "Retry", reload: "Reload", items: "{{count}} item", items_other: "{{count}} items", refresh: "Refresh" },
    navigation: { label: "Main navigation", home: "Home", tasks: "Tasks", performers: "Performers", discover: "Discover", stats: "Statistics", settings: "Settings", help: "Help" },
    settings: { title: "Configuration & system", state: "Current status", waiting: "Waiting for settings data from the server", saving: "Saving...", tabs: { connections: "Connections", ingest: "Ingest", automation: "Automation", system: "System", logs: "Logs", about: "About" }, connections: { save: "Save {{service}} connection", saved: "{{service}} settings saved.", dashboardPassword: "Dashboard password", dashboardPasswordPlaceholder: "Jackett administration password", username: "Username", password: "Password", defaultSavePath: "Default save path", defaultCategory: "Default category", defaultTags: "Default tags" }, ingest: { saved: "Ingest settings saved.", save: "Save ingest settings", mode: "Ingest mode", pathMap: "Path mapping", transfer: "File delivery", qbRoot: "qB download root", stashRoot: "Stash library root", action: "Delivery action", copy: "Copy", move: "Move", symlink: "Symbolic link", mojiDownloadRoot: "Moji download root", mojiLibraryRoot: "Moji library root", pathMapInfo: "Moji translates the task's qB path into a Stash scan path without moving files.", transferInfo: "Moji translates the qB path into a locally accessible source, delivers it to the library, and maps the same relative path to Stash.", qbRootInfo: "The download root from qBittorrent's perspective. ContentPath and SavePath are made relative to this root.", mojiDownloadInfo: "The download root from Moji's perspective. TRANSFER appends the relative path here to locate the readable source.", mojiLibraryInfo: "The media root from Moji's perspective. TRANSFER appends the relative path here to create the delivery destination.", stashInfo: "The media root from Stash's perspective. Both modes append the relative path here and request a Stash scan.", actionInfo: "COPY preserves the source, MOVE relocates it, and SYMLINK creates a library link. Delivery fails when the destination already exists.", useQbDefault: "Use qB default download directory", noQbDefault: "No qB default download directory is configured", initializeWith: "Initialize the qB download root with {{path}}", selectStashRoot: "Select a Stash library root", noStashRoot: "No library paths available", initTitle: "Initialize qB download root", emptyTitle: "The qB download root is empty", initQuestion: "Initialize it with qB's default directory, {{path}}?", useAndSave: "Use default and save", keepEmpty: "Keep empty and save" } },
    help: { title: "Markdown help", navigation: "Help documentation" },
    errors: { notFoundTitle: "Page not found", notFoundDetail: "There is no Moji page at this address.", returnHome: "Return home", routeTitle: "Page failed to load", moduleLoad: "Page module failed to load" },
    theme: { label: "Theme: {{theme}}", choose: "Choose theme", light: "Light", dark: "Dark", auto: "Automatic", resolved: "(Currently showing: {{theme}})" },
    stats: { title: "Runtime overview", loadFailed: "Statistics failed to load" },
    home: {
      services: "External services", ingestPolicy: "Ingest policy", todos: "Tasks requiring attention", todosNote: "Failed, pending-scan, and stalled tasks appear here.",
      noTodos: "Nothing requires attention", noTodosDetail: "Failed, pending-scan, and abnormal tasks are prioritized here.", configure: "Configure", adjust: "Adjust", noData: "No data", loadFailed: "Home failed to load", retryNoResult: "The retry failed because the server returned no task record.", retried: "Retried task: {{task}}.",
      status: { unconfigured: "Not configured", stale: "Stale data", enabled: "Enabled", error: "Runtime error", pending: "Pending check", configuredDisabled: "Configured but disabled" },
      diagnostics: {
        stashError: "Configured but recently unreachable. Check the Stash service.", stashMissing: "Configure Stash before using ingest and performer updates.", stashPending: "Stash is configured and waiting for its first check, or its status is stale.",
        jackettError: "Configured but recently unreachable. Check the Jackett service.", jackettMissing: "Configure Jackett to provide upstream data for task searches and performer updates.", jackettPending: "Jackett is configured and waiting for its first check, or its status is stale.",
        qbError: "Configured but recently unreachable. Check the qBittorrent service.", qbMissing: "Configure qBittorrent before downloads and delivery can run.", qbPending: "qBittorrent is configured and waiting for its first check, or its status is stale."
      },
      blockers: { jackettSearch: "No indexer source for task searches", jackettPerformers: "No upstream data for performer updates", qbDownload: "Tasks cannot start downloads", qbLanding: "No client can finish downloaded content" },
      stats: { scenes: "{{count}} scene", scenes_other: "{{count}} scenes", stashMissing: "Stash has not reported data", pendingScans: "{{count}} Moji task pending scan", pendingScans_other: "{{count}} Moji tasks pending scan", indexers: "{{configured}} / {{total}} indexers configured", slowest: "Slowest recent search: {{latency}} ms", transfer: "Download {{download}} · Upload {{upload}}", active: "{{count}} active task · {{status}}", active_other: "{{count}} active tasks · {{status}}" },
      config: { user: "User", savePath: "Save path", mode: "Ingest mode", action: "Delivery action", qbRoot: "qB download root", mojiDownloadRoot: "Moji download root", mojiLibraryRoot: "Moji library root", stashRoot: "Stash library root" },
      ingest: { title: "Ingest", missing: "Missing: {{fields}}", incomplete: "A workflow is selected, but its path mapping is incomplete.", mode: "Ingest mode: {{mode}}", none: "Not selected", pathMap: "Path mapping", transfer: "File delivery", copy: "Copy", move: "Move", symlink: "Symbolic link", guidePathMap: "Moji translates the qB download path into a Stash scan path without moving files.", cautionPathMap: "Configure both the qB download root and Stash library root; each uses its own namespace.", guideTransfer: "Moji translates the qB path into a local source, delivers it to the library, and then calculates the Stash scan path.", cautionTransfer: "Configure qB, Moji, and Stash roots. Delivery fails if the destination already contains the same file or directory.", guideNone: "Select an ingest policy to continue.", blockerQb: "qB download root is not mapped", blockerStash: "Stash library root is not mapped", blockerScan: "Completed tasks cannot resolve a scan path" }
    },
    tasks: {
      title: "Workspace", metrics: "Active {{active}} · Completed {{completed}} · Pending scans {{pendingScans}} · Failed {{failed}}",
      searchPlaceholder: "Search tasks, codes, trackers, or statuses",
      filters: { all: "All", running: "Running", completed: "Completed", failed: "Failed", scanPending: "Pending scan" },
      sorts: { createdAt: "Newest", updatedAt: "Recently updated", progress: "Progress" },
      actions: { sync: "Sync progress", scanAll: "Trigger scans", scanGroup: "Scan all", retryBlocked: "Retry blocked tasks", retryingBlocked: "Retrying blocked tasks..." },
      empty: { title: "No matching tasks", detail: "Try different filters or create a task from Discover." },
      groups: {
        attention: { label: "Needs attention", description: "Failed tasks, scan errors, and tasks requiring review." },
        active: { label: "Running", description: "Tasks downloading, syncing, or waiting for an external state change." },
        ingestPending: { label: "Pending ingest", description: "Downloads completed but the Stash scan has not finished." },
        completed: { label: "Completed", description: "Tasks whose workflows have completed." }
      }
    },
    titles: { tasks: "Tasks · Moji", performers: "Performers · Moji", discover: "Discover · Moji", settings: "Settings · Moji", stats: "Statistics · Moji", home: "Moji" }
  } }
} as const;
