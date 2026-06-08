import { FormEvent, ReactNode, useDeferredValue, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faChartColumn, faGear, faCircleQuestion } from "@fortawesome/free-solid-svg-icons";
import {
  AddTorrentDocumentDocument,
  DashboardDocumentDocument,
  SearchDocumentDocument,
  SyncTaskProgressDocumentDocument,
  TriggerStashScansDocumentDocument,
  type AddTorrentDocumentMutation,
  type AddTorrentDocumentMutationVariables,
  type DashboardDocumentQuery,
  type DashboardDocumentQueryVariables,
  type JackettSearchInput,
  type SearchDocumentQuery,
  type SearchDocumentQueryVariables,
  type SyncTaskProgressDocumentMutation,
  type SyncTaskProgressDocumentMutationVariables,
  type TriggerStashScansDocumentMutation,
  type TriggerStashScansDocumentMutationVariables
} from "./graphql/generated/graphql";
import { HELP_TOPICS, type HelpTopicId } from "./help";

type TabKey = "主页" | "任务" | "following" | "发现";
type DrawerKey = "stats" | "settings" | "help" | "task" | null;
type DashboardTask = DashboardDocumentQuery["tasks"][number];
type SettingsTab =
  | "Stash"
  | "索引器"
  | "下载器"
  | "任务"
  | "安全性"
  | "系统"
  | "日志"
  | "工具"
  | "更新历史"
  | "关于";

const NAV_TABS: TabKey[] = ["主页", "任务", "following", "发现"];
const SETTINGS_TABS: SettingsTab[] = [
  "Stash",
  "索引器",
  "下载器",
  "任务",
  "安全性",
  "系统",
  "日志",
  "工具",
  "更新历史",
  "关于"
];

const FOLLOWING_PLACEHOLDERS = [
  {
    name: "未接入追踪源 01",
    alias: "alias pending",
    status: "尚未追踪",
    updatedAt: "规划中",
    works: "0",
    note: "暂无头像"
  },
  {
    name: "未接入追踪源 02",
    alias: "alias pending",
    status: "等待导入",
    updatedAt: "规划中",
    works: "0",
    note: "统一占位头像"
  },
  {
    name: "未接入追踪源 03",
    alias: "alias pending",
    status: "未启用",
    updatedAt: "规划中",
    works: "0",
    note: "筛选与搜索预留"
  }
] as const;

function formatBytes(size: number) {
  if (!size) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(size) / Math.log(1024)), units.length - 1);
  const value = size / 1024 ** index;
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function formatDateTime(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(date);
}

function formatRelativeDate(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "short",
    day: "numeric"
  }).format(date);
}

function normalizeStatus(value: string) {
  return value.trim().toLowerCase();
}

function isStatus(task: DashboardTask, ...values: string[]) {
  const status = normalizeStatus(task.status);
  return values.some((value) => status === value || status.includes(value));
}

function isTaskActive(task: DashboardTask) {
  return !isStatus(task, "completed", "failed", "cancelled", "canceled", "paused");
}

function isScanPending(task: DashboardTask) {
  const status = task.stashScanStatus.trim().toLowerCase();
  if (!status) return false;
  return !["completed", "done", "failed", "skipped", "idle"].includes(status);
}

function taskSummary(task: DashboardTask) {
  return task.torrentName || task.query || task.id;
}

function statusTone(status: string) {
  const normalized = status.toLowerCase();
  if (normalized.includes("complete")) return "tone-success";
  if (normalized.includes("fail")) return "tone-danger";
  if (normalized.includes("download") || normalized.includes("sync")) return "tone-info";
  if (normalized.includes("pending") || normalized.includes("wait")) return "tone-warn";
  return "tone-neutral";
}

function describeQueryError(error: unknown) {
  if (!error || typeof error !== "object") return "unknown error";

  const combined = error as {
    message?: string;
    graphQLErrors?: Array<{ message?: string }>;
    networkError?: { message?: string };
  };

  const pieces = [combined.message];
  if (combined.networkError?.message) {
    pieces.push(`network: ${combined.networkError.message}`);
  }
  if (combined.graphQLErrors?.length) {
    pieces.push(
      `graphql: ${combined.graphQLErrors
        .map((item) => item.message)
        .filter(Boolean)
        .join(" | ")}`
    );
  }

  return pieces.filter(Boolean).join(" · ") || "unknown error";
}

function MarkdownBlock({ markdown }: { markdown: string }) {
  const nodes: ReactNode[] = [];
  const lines = markdown.replace(/\r\n/g, "\n").split("\n");
  let paragraph: string[] = [];
  let listItems: string[] = [];
  let codeLines: string[] = [];
  let inCode = false;

  const flushParagraph = () => {
    if (!paragraph.length) return;
    nodes.push(
      <p key={`p-${nodes.length}`}>
        {paragraph.join(" ").trim()}
      </p>
    );
    paragraph = [];
  };

  const flushList = () => {
    if (!listItems.length) return;
    nodes.push(
      <ul key={`ul-${nodes.length}`}>
        {listItems.map((item, index) => (
          <li key={`${item}-${index}`}>{item}</li>
        ))}
      </ul>
    );
    listItems = [];
  };

  const flushCode = () => {
    if (!codeLines.length) return;
    nodes.push(
      <pre key={`pre-${nodes.length}`}>
        <code>{codeLines.join("\n")}</code>
      </pre>
    );
    codeLines = [];
  };

  for (const line of lines) {
    if (line.trim().startsWith("```")) {
      if (inCode) {
        flushCode();
      } else {
        flushParagraph();
        flushList();
      }
      inCode = !inCode;
      continue;
    }

    if (inCode) {
      codeLines.push(line);
      continue;
    }

    if (!line.trim()) {
      flushParagraph();
      flushList();
      continue;
    }

    if (line.startsWith("# ")) {
      flushParagraph();
      flushList();
      nodes.push(<h2 key={`h2-${nodes.length}`}>{line.slice(2).trim()}</h2>);
      continue;
    }

    if (line.startsWith("## ")) {
      flushParagraph();
      flushList();
      nodes.push(<h3 key={`h3-${nodes.length}`}>{line.slice(3).trim()}</h3>);
      continue;
    }

    if (/^[-*]\s+/.test(line)) {
      flushParagraph();
      listItems.push(line.replace(/^[-*]\s+/, ""));
      continue;
    }

    if (/^\d+\.\s+/.test(line)) {
      flushParagraph();
      listItems.push(line.replace(/^\d+\.\s+/, ""));
      continue;
    }

    paragraph.push(line.trim());
  }

  flushParagraph();
  flushList();
  flushCode();

  return <>{nodes}</>;
}

function App() {
  const [tab, setTab] = useState<TabKey>("主页");
  const [drawer, setDrawer] = useState<DrawerKey>(null);
  const [renderedDrawer, setRenderedDrawer] = useState<Exclude<DrawerKey, null> | null>(null);
  const [drawerClosing, setDrawerClosing] = useState(false);
  const [settingsTab, setSettingsTab] = useState<SettingsTab>("Stash");
  const [helpTopicId, setHelpTopicId] = useState<HelpTopicId>(HELP_TOPICS[0].id);
  const [taskSearch, setTaskSearch] = useState("");
  const [taskStatus, setTaskStatus] = useState<"全部" | "运行中" | "完成" | "失败" | "待扫描">("全部");
  const [taskSort, setTaskSort] = useState<"最新" | "更新时间" | "进度">("最新");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [jackettQuery, setJackettQuery] = useState("");
  const [submittedJackettQuery, setSubmittedJackettQuery] = useState("");
  const [pendingAddId, setPendingAddId] = useState<string | null>(null);

  const deferredTaskSearch = useDeferredValue(taskSearch.trim().toLowerCase());
  const deferredJackettQuery = useDeferredValue(submittedJackettQuery.trim());

  const [{ data, fetching, error }, refreshDashboard] = useQuery<
    DashboardDocumentQuery,
    DashboardDocumentQueryVariables
  >({
    query: DashboardDocumentDocument,
    requestPolicy: "cache-and-network"
  });

  const [{ data: searchData, fetching: searching, error: searchError }] = useQuery<
    SearchDocumentQuery,
    SearchDocumentQueryVariables
  >({
    query: SearchDocumentDocument,
    variables: {
      input: {
        query: deferredJackettQuery,
        limit: 18
      } satisfies JackettSearchInput
    },
    pause: deferredJackettQuery.length === 0
  });

  const [, addTorrent] = useMutation<
    AddTorrentDocumentMutation,
    AddTorrentDocumentMutationVariables
  >(AddTorrentDocumentDocument);
  const [, syncTaskProgress] = useMutation<
    SyncTaskProgressDocumentMutation,
    SyncTaskProgressDocumentMutationVariables
  >(SyncTaskProgressDocumentDocument);
  const [, triggerStashScans] = useMutation<
    TriggerStashScansDocumentMutation,
    TriggerStashScansDocumentMutationVariables
  >(TriggerStashScansDocumentDocument);

  const tasks = data?.tasks ?? [];
  const activeTask = selectedTaskId ? tasks.find((task) => task.id === selectedTaskId) ?? null : null;

  const visibleTasks = useMemo(() => {
    const search = deferredTaskSearch;
    let next = tasks.filter((task) => {
      if (!search) return true;
      const haystack = [
        taskSummary(task),
        task.status,
        task.qbittorrentState,
        task.stashScanStatus,
        task.torrentHash,
        task.contentPath,
        task.query
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(search);
    });

    if (taskStatus === "运行中") {
      next = next.filter(isTaskActive);
    } else if (taskStatus === "完成") {
      next = next.filter((task) => isStatus(task, "completed"));
    } else if (taskStatus === "失败") {
      next = next.filter((task) => isStatus(task, "failed"));
    } else if (taskStatus === "待扫描") {
      next = next.filter((task) => isScanPending(task) || isStatus(task, "completed"));
    }

    const sorters: Record<typeof taskSort, (a: DashboardTask, b: DashboardTask) => number> = {
      最新: (a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
      更新时间: (a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt),
      进度: (a, b) => b.progress - a.progress
    };

    return [...next].sort(sorters[taskSort]);
  }, [deferredTaskSearch, taskSort, taskStatus, tasks]);

  const metrics = {
    active: tasks.filter(isTaskActive).length,
    completed: tasks.filter((task) => isStatus(task, "completed")).length,
    downloading: tasks.filter((task) => /download|sync|queued|stalled/i.test(task.status)).length,
    pendingScans: tasks.filter(isScanPending).length,
    failed: tasks.filter((task) => isStatus(task, "failed") || isScanPending(task) === false && isStatus(task, "error")).length,
    versions: data?.version ?? "unknown"
  };

  const dependencyCards = [
    {
      name: "Stash",
      state: tasks.some((task) => task.stashScanStatus) ? "已接入" : "未观测",
      detail: tasks.some((task) => task.stashScanStatus)
        ? "扫描链路已经出现在任务数据里"
        : "等待任务完成后触发扫描"
    },
    {
      name: "Jackett",
      state: deferredJackettQuery ? "可搜索" : "待输入",
      detail: deferredJackettQuery
        ? `最近一次搜索: ${deferredJackettQuery}`
        : "在任务页输入关键词后可以发现候选"
    },
    {
      name: "qBittorrent",
      state: tasks.some((task) => task.qbittorrentState) ? "已接入" : "待观测",
      detail: tasks.some((task) => task.qbittorrentState)
        ? "任务状态已能同步"
        : "当前界面没有观察到 qBittorrent 任务数据"
    }
  ];

  const selectedHelpTopic =
    HELP_TOPICS.find((topic) => topic.id === helpTopicId) ?? HELP_TOPICS[0];
  const visibleDrawer = renderedDrawer ?? drawer;

  const submitJackettSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmittedJackettQuery(jackettQuery.trim());
    setTab("发现");
  };

  const runSync = async () => {
    await syncTaskProgress({});
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const runScan = async () => {
    await triggerStashScans({});
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const addSearchResult = async (result: SearchDocumentQuery["jackettSearch"][number]) => {
    setPendingAddId(result.link);
    const response = await addTorrent({
      input: {
        url: result.magnetUri || result.link
      }
    });
    if (response.data?.addTorrent?.id) {
      setSelectedTaskId(response.data.addTorrent.id);
    }
    await refreshDashboard({ requestPolicy: "network-only" });
    setPendingAddId(null);
    setDrawer("task");
  };

  useEffect(() => {
    if (drawer) {
      setRenderedDrawer(drawer);
      setDrawerClosing(false);
      return;
    }

    if (!renderedDrawer) return;

    setDrawerClosing(true);
    const timer = window.setTimeout(() => {
      setRenderedDrawer(null);
      setDrawerClosing(false);
    }, 240);

    return () => window.clearTimeout(timer);
  }, [drawer, renderedDrawer]);

  return (
    <div className="app-shell">
      <div className="ambient ambient-a" />
      <div className="ambient ambient-b" />
      <div className="ambient ambient-c" />

      <header className="masthead">
        <div className="masthead__brand">
          <div className="title-row">
            <h1>Moji</h1>
          </div>
        </div>
        <div className="masthead__actions" aria-label="主导航">
          <div className="masthead__navgroup">
            <div className="tab-group">
              {NAV_TABS.map((item) => (
                <button
                  key={item}
                  type="button"
                  className={`nav-tab ${tab === item ? "is-active" : ""}`}
                  onClick={() => setTab(item)}
                >
                  {item}
                </button>
              ))}
            </div>
          </div>

          <div className="masthead__toolgroup">
            <div className="utility-group">
              <button
                type="button"
                className="utility-button utility-icon-button"
                onClick={() => setDrawer("stats")}
                aria-label="统计"
                title="统计"
              >
                <FontAwesomeIcon icon={faChartColumn} aria-hidden="true" />
              </button>
              <button
                type="button"
                className="utility-button utility-icon-button"
                onClick={() => setDrawer("settings")}
                aria-label="设置"
                title="设置"
              >
                <FontAwesomeIcon icon={faGear} aria-hidden="true" />
              </button>
              <button
                type="button"
                className="utility-button utility-icon-button"
                onClick={() => setDrawer("help")}
                aria-label="帮助"
                title="帮助"
              >
                <FontAwesomeIcon icon={faCircleQuestion} aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>
      </header>

      {error ? (
        <section className="surface surface--alert">
          <div>
            <p className="section-kicker">运行异常</p>
            <h2>{data ? "GraphQL 返回错误" : "GraphQL 当前不可用"}</h2>
            <p>{describeQueryError(error)}</p>
          </div>
          <button type="button" className="ghost-button" onClick={() => refreshDashboard({ requestPolicy: "network-only" })}>
            重试
          </button>
        </section>
      ) : null}

      <main className="content">
        {tab === "主页" ? (
          <>
            {/* <section className="section-band section-band--hero">
              <div className="band-head">
                <div>
                  <p className="section-kicker">主页</p>
                  <h2>概览</h2>
                </div>
                <p className="band-note">主页只保留依赖、待办和任务入口。</p>
              </div>
            </section>
 */}
            <section className="section-band section-band--hero">
              <div className="band-head">
                <div>
                  <p className="section-kicker">依赖状态</p>
                  <h2>外部服务</h2>
                </div>
              </div>

              <div className="card-grid card-grid--deps">
                {dependencyCards.map((card) => (
                  <article key={card.name} className="service-card">
                    <div className="service-card__head">
                      <div>
                        <h3>{card.name}</h3>
                      </div>
                      <span className={`status-chip ${card.state === "正常" || card.state === "已接入" ? "tone-success" : card.state === "异常" ? "tone-danger" : "tone-neutral"}`}>
                        {card.state}
                      </span>
                    </div>
                    <p className="service-card__detail">{card.detail}</p>
                    <div className="service-card__actions">
                      <span>上次检测: {formatDateTime(data?.tasks[0]?.updatedAt ?? null)}</span>
                      <button type="button" className="ghost-button" onClick={() => refreshDashboard({ requestPolicy: "network-only" })}>
                        重试
                      </button>
                    </div>
                  </article>
                ))}
              </div>
            </section>

            <section className="section-band">
              <div className="band-head">
                <div>
                  <p className="section-kicker">待办</p>
                  <h2>需要人工确认的任务</h2>
                </div>
                <p className="band-note">失败项、待扫描项和长时间停滞项都放在这里。</p>
              </div>

              <div className="card-grid">
                {tasks.filter((task) => isStatus(task, "failed") || isScanPending(task)).slice(0, 4).map((task) => (
                  <article key={task.id} className={`task-card ${statusTone(task.status)}`}>
                    <div className="task-card__head">
                      <div>
                        <h3>{taskSummary(task)}</h3>
                        <p>{task.query}</p>
                      </div>
                      <button
                        type="button"
                        className="ghost-button"
                        onClick={() => {
                          setSelectedTaskId(task.id);
                          setDrawer("task");
                        }}
                      >
                        详情
                      </button>
                    </div>
                    <div className="progress-shell" aria-hidden="true">
                      <div className="progress-fill" style={{ width: `${Math.round(task.progress * 100)}%` }} />
                    </div>
                    <dl className="task-meta">
                      <div>
                        <dt>下载状态</dt>
                        <dd>{task.qbittorrentState || "待同步"}</dd>
                      </div>
                      <div>
                        <dt>扫描状态</dt>
                        <dd>{task.stashScanStatus || "未开始"}</dd>
                      </div>
                    </dl>
                  </article>
                ))}
                {!tasks.some((task) => isStatus(task, "failed") || isScanPending(task)) ? (
                  <article className="empty-card">
                    <h3>暂无待处理项</h3>
                    <p>这里会优先显示失败、待扫和异常任务。</p>
                  </article>
                ) : null}
              </div>
            </section>

          </>
        ) : null}

        {tab === "任务" ? (
          <>
            <section className="section-band">
              <div className="band-head">
                <div>
                  <p className="section-kicker">任务</p>
                  <h2>工作台</h2>
                </div>
                <p className="band-note">活跃 {metrics.active} · 完成 {metrics.completed} · 待扫 {metrics.pendingScans} · 失败 {metrics.failed}</p>
              </div>

              <div className="toolbar-inline">
                <input
                  value={taskSearch}
                  onChange={(event) => setTaskSearch(event.target.value)}
                  placeholder="搜索任务、番号、tracker、状态"
                />
                <select value={taskStatus} onChange={(event) => setTaskStatus(event.target.value as typeof taskStatus)}>
                  <option value="全部">全部</option>
                  <option value="运行中">运行中</option>
                  <option value="完成">完成</option>
                  <option value="失败">失败</option>
                  <option value="待扫描">待扫描</option>
                </select>
                <select value={taskSort} onChange={(event) => setTaskSort(event.target.value as typeof taskSort)}>
                  <option value="最新">最新</option>
                  <option value="更新时间">更新时间</option>
                  <option value="进度">进度</option>
                </select>
                <button type="button" className="ghost-button" onClick={() => void refreshDashboard({ requestPolicy: "network-only" })}>
                  刷新
                </button>
                <button type="button" className="ghost-button" onClick={() => void runSync()}>
                  同步进度
                </button>
                <button type="button" className="ghost-button" onClick={() => void runScan()}>
                  触发扫描
                </button>
              </div>

              <div className="chip-row">
                {["全部", "运行中", "完成", "失败", "待扫描"].map((value) => (
                  <button
                    key={value}
                    type="button"
                    className={`chip ${taskStatus === value ? "is-active" : ""}`}
                    onClick={() => setTaskStatus(value as typeof taskStatus)}
                  >
                    {value}
                  </button>
                ))}
              </div>

              <div className="task-grid">
                {visibleTasks.map((task) => (
                  <article
                    key={task.id}
                    className={`task-card ${statusTone(task.status)}`}
                    onClick={() => {
                      setSelectedTaskId(task.id);
                      setDrawer("task");
                    }}
                    role="button"
                    tabIndex={0}
                  >
                    <div className="task-card__head">
                      <div>
                        <h3>{taskSummary(task)}</h3>
                        <p>{task.query || "无查询文本"}</p>
                      </div>
                      <span className={`status-chip ${statusTone(task.status)}`}>{task.status}</span>
                    </div>
                    <div className="progress-shell">
                      <div className="progress-fill" style={{ width: `${Math.round(task.progress * 100)}%` }} />
                    </div>
                    <dl className="task-meta">
                      <div>
                        <dt>qBittorrent</dt>
                        <dd>{task.qbittorrentState || "待同步"}</dd>
                      </div>
                      <div>
                        <dt>Stash</dt>
                        <dd>{task.stashScanStatus || "未开始"}</dd>
                      </div>
                      <div>
                        <dt>更新时间</dt>
                        <dd>{formatDateTime(task.updatedAt)}</dd>
                      </div>
                      <div>
                        <dt>完成时间</dt>
                        <dd>{formatDateTime(task.completedAt)}</dd>
                      </div>
                    </dl>
                    {task.stashScanError ? <p className="task-error">{task.stashScanError}</p> : null}
                  </article>
                ))}
                {!visibleTasks.length ? (
                  <article className="empty-card empty-card--wide">
                    <h3>没有匹配的任务</h3>
                    <p>换个过滤条件，或者先去发现区创建任务。</p>
                  </article>
                ) : null}
              </div>
            </section>
          </>
        ) : null}

        {tab === "发现" ? (
          <>
            <section className="section-band">
              <div className="band-head">
                <div>
                  <p className="section-kicker">发现</p>
                  <h2>Jackett 搜索</h2>
                </div>
                <p className="band-note">搜索候选后直接创建 Moji task。</p>
              </div>

              <form className="discovery-bar" onSubmit={submitJackettSearch}>
                <input
                  value={jackettQuery}
                  onChange={(event) => setJackettQuery(event.target.value)}
                  placeholder="输入番号、标题、女优或关键词"
                />
                <button type="submit" disabled={searching || jackettQuery.trim() === ""}>
                  {searching ? "搜索中" : "搜索"}
                </button>
              </form>

              {searchError ? <p className="inline-error">{searchError.message}</p> : null}

              <div className="discovery-results">
                {(searchData?.jackettSearch ?? []).map((result) => (
                  <article key={`${result.tracker}-${result.link}`} className="candidate-card">
                    <div className="candidate-card__head">
                      <div>
                        <h3>{result.title}</h3>
                        <p>
                          {result.tracker} · {formatBytes(Number(result.size) || 0)} · {result.seeders} seeders
                        </p>
                      </div>
                      <span className="status-chip tone-info">{result.categoryDesc || "候选"}</span>
                    </div>
                    <div className="candidate-card__foot">
                      <span>{formatRelativeDate(result.publishDate)}</span>
                      <div className="inline-actions">
                        <a href={result.link} target="_blank" rel="noreferrer">
                          原始链接
                        </a>
                        <button type="button" onClick={() => void addSearchResult(result)} disabled={pendingAddId === result.link}>
                          {pendingAddId === result.link ? "添加中" : "创建任务"}
                        </button>
                      </div>
                    </div>
                  </article>
                ))}
                {deferredJackettQuery && !searching && (searchData?.jackettSearch ?? []).length === 0 ? (
                  <article className="empty-card empty-card--wide">
                    <h3>没有候选</h3>
                    <p>Jackett 没有返回结果，换个关键词再试。</p>
                  </article>
                ) : null}
                {!deferredJackettQuery ? (
                  <article className="empty-card empty-card--wide">
                    <h3>先搜索</h3>
                    <p>输入关键词后会在这里列出候选项。</p>
                  </article>
                ) : null}
              </div>
            </section>

            <section className="section-band section-band--preview">
              <div className="band-head">
                <div>
                  <p className="section-kicker">推荐</p>
                  <h2>推荐系统占位区</h2>
                </div>
                <p className="band-note">后续可接入推荐、通知和批量操作。</p>
              </div>
              <div className="preview-panel">
                <div>
                  <h3>推荐系统未启用</h3>
                  <p>先把健康、任务和扫描闭环跑顺，再把推荐位接进来。</p>
                </div>
                <button type="button" className="ghost-button" onClick={() => setDrawer("help")}>
                  看帮助
                </button>
              </div>
            </section>
          </>
        ) : null}

        {tab === "following" ? (
          <>
            <section className="section-band">
              <div className="band-head">
                <div>
                  <p className="section-kicker">following</p>
                  <h2>追踪列表</h2>
                </div>
                <p className="band-note">未来会接 Stash favorite / tracked performer。</p>
              </div>

              <div className="toolbar-inline toolbar-inline--following">
                <input placeholder="按名称搜索" />
                <select defaultValue="全部">
                  <option>全部</option>
                  <option>尚未追踪</option>
                  <option>已追踪</option>
                  <option>待更新</option>
                </select>
                <select defaultValue="全部">
                  <option>全部</option>
                  <option>A-Z</option>
                  <option>最近更新</option>
                  <option>作品数量</option>
                </select>
              </div>

              <div className="profile-grid">
                {FOLLOWING_PLACEHOLDERS.map((item, index) => (
                  <article key={item.name} className="profile-card" style={{ animationDelay: `${index * 80}ms` }}>
                    <div className="avatar avatar--placeholder">{item.name.slice(0, 2)}</div>
                    <div className="profile-card__body">
                      <div className="profile-card__head">
                        <div>
                          <h3>{item.name}</h3>
                          <p>{item.alias}</p>
                        </div>
                        <span className="status-chip tone-neutral">{item.status}</span>
                      </div>
                      <dl>
                        <div>
                          <dt>最近更新</dt>
                          <dd>{item.updatedAt}</dd>
                        </div>
                        <div>
                          <dt>相关作品</dt>
                          <dd>{item.works}</dd>
                        </div>
                      </dl>
                      <p className="profile-note">{item.note}</p>
                    </div>
                  </article>
                ))}
              </div>
            </section>
          </>
        ) : null}
      </main>

      {visibleDrawer ? (
        <div
          className={`drawer-scrim ${visibleDrawer === "task" ? "drawer-scrim--task" : "drawer-scrim--modal"} ${drawerClosing ? "is-closing" : ""}`}
          onClick={() => setDrawer(null)}
        >
          <aside
            className={`drawer ${visibleDrawer === "task" ? "drawer--task" : "drawer--modal"} ${drawerClosing ? "is-closing" : ""}`}
            onClick={(event) => event.stopPropagation()}
          >
            <div className="drawer__head">
              <div>
                <p className="section-kicker">
                  {visibleDrawer === "stats" ? "统计" : visibleDrawer === "settings" ? "设置" : visibleDrawer === "help" ? "帮助" : "任务详情"}
                </p>
                <h2>
                  {visibleDrawer === "stats"
                    ? "运行概览"
                    : visibleDrawer === "settings"
                      ? "配置与系统"
                      : visibleDrawer === "help"
                        ? "Markdown 帮助"
                        : activeTask
                          ? taskSummary(activeTask)
                          : "任务详情"}
                </h2>
              </div>
              <button type="button" className="ghost-button" onClick={() => setDrawer(null)}>
                关闭
              </button>
            </div>

            <div className="drawer-body">
              {visibleDrawer === "stats" ? (
                <div className="drawer-stack">
                  <div className="stat-strip">
                    <article className="stat-card">
                      <span>活跃任务</span>
                      <strong>{metrics.active}</strong>
                    </article>
                    <article className="stat-card">
                      <span>完成任务</span>
                      <strong>{metrics.completed}</strong>
                    </article>
                    <article className="stat-card">
                      <span>待扫描</span>
                      <strong>{metrics.pendingScans}</strong>
                    </article>
                    <article className="stat-card">
                      <span>失败</span>
                      <strong>{metrics.failed}</strong>
                    </article>
                  </div>

                  <article className="drawer-card">
                    <h3>指标占位</h3>
                    <p>后续可在这里接入速度、队列、成功率和时段趋势图。</p>
                    <div className="mini-bars" aria-hidden="true">
                      <span style={{ height: "35%" }} />
                      <span style={{ height: "65%" }} />
                      <span style={{ height: "50%" }} />
                      <span style={{ height: "80%" }} />
                      <span style={{ height: "42%" }} />
                      <span style={{ height: "70%" }} />
                    </div>
                  </article>
                </div>
              ) : null}

              {visibleDrawer === "settings" ? (
                <div className="drawer-stack">
                  <div className="settings-tabs">
                    {SETTINGS_TABS.map((item) => (
                      <button
                        key={item}
                        type="button"
                        className={`chip ${settingsTab === item ? "is-active" : ""}`}
                        onClick={() => setSettingsTab(item)}
                      >
                        {item}
                      </button>
                    ))}
                  </div>

                  <article className="drawer-card">
                    <div className="drawer-card__head">
                      <h3>{settingsTab}</h3>
                      <span className="status-chip tone-neutral">规划中</span>
                    </div>
                    <dl className="settings-grid">
                      <div>
                        <dt>当前状态</dt>
                        <dd>由后端配置驱动</dd>
                      </div>
                      <div>
                        <dt>敏感值</dt>
                        <dd>前端不展示明文</dd>
                      </div>
                      <div>
                        <dt>接入方式</dt>
                        <dd>后续可扩展可编辑表单</dd>
                      </div>
                      <div>
                        <dt>说明</dt>
                        <dd>
                          {settingsTab === "Stash"
                            ? "这里会放 GraphQL URL、API key、library path 和扫描策略。"
                            : settingsTab === "索引器"
                              ? "这里会放 Jackett URL、API key 和 tracker 分组。"
                              : settingsTab === "下载器"
                                ? "这里会放 qBittorrent URL、认证和默认保存路径。"
                                : settingsTab === "任务"
                                  ? "这里会放任务存储、同步间隔和扫描策略。"
                                  : settingsTab === "安全性"
                                    ? "这里会放访问控制、CORS 和未来登录策略。"
                                    : settingsTab === "系统"
                                      ? "这里会放版本、构建信息和运行环境。"
                                      : settingsTab === "日志"
                                        ? "这里会放最近日志和错误过滤器。"
                                        : settingsTab === "工具"
                                          ? "这里会放重新同步、重新探测和修复动作。"
                                          : settingsTab === "更新历史"
                                            ? "这里会放版本记录和升级提示。"
                                            : "这里会放项目定位、许可证和作者信息。"}
                        </dd>
                      </div>
                    </dl>
                  </article>
                </div>
              ) : null}

              {visibleDrawer === "help" ? (
                <div className="help-layout">
                  <div className="help-tabs">
                    {HELP_TOPICS.map((topic) => (
                      <button
                        key={topic.id}
                        type="button"
                        className={`help-tab ${helpTopicId === topic.id ? "is-active" : ""}`}
                        onClick={() => setHelpTopicId(topic.id)}
                      >
                        {topic.title}
                      </button>
                    ))}
                  </div>
                  <article className="drawer-card help-card">
                    <MarkdownBlock markdown={selectedHelpTopic.markdown} />
                  </article>
                </div>
              ) : null}

              {visibleDrawer === "task" ? (
                <div className="drawer-stack">
                  {activeTask ? (
                    <>
                      <article className="drawer-card">
                        <div className="drawer-card__head">
                          <div>
                            <h3>{taskSummary(activeTask)}</h3>
                            <p>{activeTask.query}</p>
                          </div>
                          <span className={`status-chip ${statusTone(activeTask.status)}`}>{activeTask.status}</span>
                        </div>
                        <dl className="settings-grid">
                          <div>
                            <dt>保存路径</dt>
                            <dd>{activeTask.savePath || "—"}</dd>
                          </div>
                          <div>
                            <dt>分类</dt>
                            <dd>{activeTask.category || "—"}</dd>
                          </div>
                          <div>
                            <dt>标签</dt>
                            <dd>{activeTask.tags || "—"}</dd>
                          </div>
                          <div>
                            <dt>保存内容</dt>
                            <dd>{activeTask.contentPath || "—"}</dd>
                          </div>
                          <div>
                            <dt>创建时间</dt>
                            <dd>{formatDateTime(activeTask.createdAt)}</dd>
                          </div>
                          <div>
                            <dt>更新时间</dt>
                            <dd>{formatDateTime(activeTask.updatedAt)}</dd>
                          </div>
                        </dl>
                      </article>

                      <article className="drawer-card">
                        <h3>下载与扫描</h3>
                        <dl className="settings-grid">
                          <div>
                            <dt>qBittorrent</dt>
                            <dd>{activeTask.qbittorrentState || "待同步"}</dd>
                          </div>
                          <div>
                            <dt>进度</dt>
                            <dd>{Math.round(activeTask.progress * 100)}%</dd>
                          </div>
                          <div>
                            <dt>Stash job</dt>
                            <dd>{activeTask.stashJobId || "—"}</dd>
                          </div>
                          <div>
                            <dt>扫描状态</dt>
                            <dd>{activeTask.stashScanStatus || "未开始"}</dd>
                          </div>
                        </dl>
                        {activeTask.stashScanError ? <p className="task-error">{activeTask.stashScanError}</p> : null}
                      </article>

                      <div className="inline-actions">
                        <button type="button" className="ghost-button" onClick={() => void runSync()}>
                          同步进度
                        </button>
                        <button type="button" className="ghost-button" onClick={() => void runScan()}>
                          触发扫描
                        </button>
                      </div>
                    </>
                  ) : (
                    <article className="drawer-card">
                      <h3>还没有选中任务</h3>
                      <p>点击任务卡片后，这里会显示详细信息和操作。</p>
                    </article>
                  )}
                </div>
              ) : null}
            </div>
          </aside>
        </div>
      ) : null}
    </div>
  );
}

export { App };
