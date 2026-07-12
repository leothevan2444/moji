export const resources = {
  "zh-CN": { translation: {
    common: { auto: "跟随浏览器", language: "语言", loading: "页面加载中", helpLoading: "帮助加载中", collapse: "收起", expand: "展开", close: "关闭", retry: "重试", reload: "重新加载", items: "{{count}} 项", refresh: "刷新" },
    navigation: { label: "主导航", home: "主页", tasks: "任务", performers: "演员", discover: "发现", stats: "统计", settings: "设置", help: "帮助" },
    settings: { title: "配置与系统", tabs: { connections: "连接", ingest: "入库", automation: "自动化", system: "系统", logs: "日志", about: "关于" } },
    help: { title: "Markdown 帮助", navigation: "帮助文档" },
    errors: { notFoundTitle: "页面不存在", notFoundDetail: "这个地址没有对应的 Moji 页面。", returnHome: "返回主页", routeTitle: "页面加载失败", moduleLoad: "页面模块加载失败" },
    theme: { label: "主题：{{theme}}", choose: "选择主题", light: "浅色", dark: "深色", auto: "自动", resolved: "（当前显示：{{theme}}）" },
    stats: { title: "运行概览", loadFailed: "统计加载失败" },
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
    settings: { title: "Configuration & system", tabs: { connections: "Connections", ingest: "Ingest", automation: "Automation", system: "System", logs: "Logs", about: "About" } },
    help: { title: "Markdown help", navigation: "Help documentation" },
    errors: { notFoundTitle: "Page not found", notFoundDetail: "There is no Moji page at this address.", returnHome: "Return home", routeTitle: "Page failed to load", moduleLoad: "Page module failed to load" },
    theme: { label: "Theme: {{theme}}", choose: "Choose theme", light: "Light", dark: "Dark", auto: "Automatic", resolved: "(Currently showing: {{theme}})" },
    stats: { title: "Runtime overview", loadFailed: "Statistics failed to load" },
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
