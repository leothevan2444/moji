import { FormEvent, ReactNode, useDeferredValue, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faBookmark, faChartColumn, faCircleQuestion, faGear, faHeart } from "@fortawesome/free-solid-svg-icons";
import {
  AddTorrentDocumentDocument,
  DashboardDocumentDocument,
  FollowPerformerDocument,
  FollowingPerformersDocument,
  LogLevel,
  LogsDocumentDocument,
  RefreshFollowingNowDocument,
  RefreshFollowingPerformerDocument,
  SearchDocumentDocument,
  StashPerformersDocument,
  SyncTaskProgressDocumentDocument,
  TriggerTaskStashScanDocumentDocument,
  TriggerStashScansDocumentDocument,
  UnfollowPerformerDocument,
  UpdateFollowingSettingsDocumentDocument,
  UpdateJackettSettingsDocumentDocument,
  UpdateLoggingSettingsDocumentDocument,
  UpdateQBittorrentSettingsDocumentDocument,
  UpdateStashSettingsDocumentDocument,
  type AddTorrentDocumentMutation,
  type AddTorrentDocumentMutationVariables,
  type DashboardDocumentQuery,
  type DashboardDocumentQueryVariables,
  type FollowPerformerMutation,
  type FollowPerformerMutationVariables,
  type FollowingPerformersQuery,
  type JackettSearchInput,
  type LogsDocumentQuery,
  type LogsDocumentQueryVariables,
  type RefreshFollowingNowMutation,
  type RefreshFollowingNowMutationVariables,
  type RefreshFollowingPerformerMutation,
  type RefreshFollowingPerformerMutationVariables,
  type SearchDocumentQuery,
  type SearchDocumentQueryVariables,
  type StashPerformersQuery,
  type StashPerformersQueryVariables,
  type SyncTaskProgressDocumentMutation,
  type SyncTaskProgressDocumentMutationVariables,
  type TriggerTaskStashScanDocumentMutation,
  type TriggerTaskStashScanDocumentMutationVariables,
  type TriggerStashScansDocumentMutation,
  type TriggerStashScansDocumentMutationVariables,
  type UnfollowPerformerMutation,
  type UnfollowPerformerMutationVariables,
  type UpdateFollowingSettingsDocumentMutation,
  type UpdateFollowingSettingsDocumentMutationVariables,
  type UpdateJackettSettingsDocumentMutation,
  type UpdateJackettSettingsDocumentMutationVariables,
  type UpdateLoggingSettingsDocumentMutation,
  type UpdateLoggingSettingsDocumentMutationVariables,
  type UpdateQBittorrentSettingsDocumentMutation,
  type UpdateQBittorrentSettingsDocumentMutationVariables,
  type UpdateStashSettingsDocumentMutation,
  type UpdateStashSettingsDocumentMutationVariables
} from "./graphql/generated/graphql";
import { HELP_TOPICS, type HelpTopicId } from "./help";

type TabKey = "主页" | "任务" | "订阅" | "发现";
type DrawerKey = "stats" | "settings" | "help" | "task" | null;
type DashboardTask = DashboardDocumentQuery["tasks"][number];
type StashPerformerEntry = StashPerformersQuery["stashPerformers"]["items"][number];
type FollowingPerformerEntry = FollowingPerformersQuery["followingPerformers"][number];
type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type SettingsFeedback = { tone: "tone-success" | "tone-danger" | "tone-info"; message: string } | null;
type TaskGroupKey = "需处理" | "运行中" | "待入库" | "已完成";
type TaskFailureSummary = {
  title: string;
  detail: string;
  tone: "tone-danger" | "tone-warn" | "tone-info" | "tone-neutral";
};
type SettingsTab =
  | "Stash"
  | "索引器"
  | "下载器"
  | "任务"
  | "订阅"
  | "安全性"
  | "系统"
  | "日志"
  | "工具"
  | "更新历史"
  | "关于";

const NAV_TABS: TabKey[] = ["主页", "任务", "订阅", "发现"];
const SETTINGS_TABS: SettingsTab[] = [
  "Stash",
  "索引器",
  "下载器",
  "任务",
  "订阅",
  "安全性",
  "系统",
  "日志",
  "工具",
  "更新历史",
  "关于"
];

const ENABLED_SETTINGS_TABS: ReadonlySet<SettingsTab> = new Set([
  "Stash",
  "索引器",
  "下载器",
  "任务",
  "订阅",
  "日志",
  "系统"
]);

const EMPTY_STASH_FORM = {
  url: "",
  apiKey: "",
  libraryPath: ""
};

const EMPTY_JACKETT_FORM = {
  url: "",
  apiKey: ""
};

const EMPTY_QBITTORRENT_FORM = {
  url: "",
  username: "",
  password: "",
  defaultSavePath: "",
  category: "",
  tags: ""
};

const EMPTY_FOLLOWING_FORM = {
  store: "json",
  jsonPath: "",
  pollIntervalSeconds: "3600",
  javstashApiKey: ""
};

const EMPTY_LOGGING_FORM = {
  level: "info",
  filePath: "",
  maxEntries: "500",
  maxFileSizeBytes: String(10 * 1024 * 1024),
  maxFileBackups: "5"
};

const FOLLOWING_PAGE_SIZE_OPTIONS = [12, 24, 48, 96] as const;
const LOG_LEVEL_OPTIONS: LogLevel[] = [LogLevel.Debug, LogLevel.Info, LogLevel.Warning, LogLevel.Error];

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
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  }).format(date);
}

function formatLogEntries(entries: LogsDocumentQuery["logs"]) {
  return entries
    .map((entry) => `${entry.time} [${entry.level}] ${entry.message}`)
    .join("\n");
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

function performerInitials(name: string) {
  return name
    .trim()
    .slice(0, 2)
    .toUpperCase();
}

function performerImageURL(imagePath?: string | null, stashURL?: string | null) {
  if (!imagePath) return null;
  try {
    if (/^https?:\/\//i.test(imagePath)) {
      return imagePath;
    }
    if (!stashURL) return imagePath;
    return new URL(imagePath, stashURL.endsWith("/") ? stashURL : `${stashURL}/`).toString();
  } catch {
    return imagePath;
  }
}

function statusTone(status: string) {
  const normalized = status.toLowerCase();
  if (normalized.includes("complete")) return "tone-success";
  if (normalized.includes("fail")) return "tone-danger";
  if (normalized.includes("download") || normalized.includes("sync")) return "tone-info";
  if (normalized.includes("pending") || normalized.includes("wait")) return "tone-warn";
  return "tone-neutral";
}

function taskGroup(task: DashboardTask): TaskGroupKey {
  if (isStatus(task, "failed") || task.stashScanError) return "需处理";
  if (isTaskActive(task)) return "运行中";
  if (isScanPending(task)) return "待入库";
  return "已完成";
}

function taskGroupTone(group: TaskGroupKey) {
  if (group === "需处理") return "tone-danger";
  if (group === "运行中") return "tone-info";
  if (group === "待入库") return "tone-warn";
  return "tone-success";
}

function taskGroupDescription(group: TaskGroupKey) {
  if (group === "需处理") return "失败、扫描报错或需要人工回看的任务。";
  if (group === "运行中") return "仍在下载、同步或等待外部状态推进。";
  if (group === "待入库") return "下载已完成，但 Stash 扫描尚未收口。";
  return "流程已闭环的任务。";
}

function canTriggerTaskStashScan(task: DashboardTask) {
  return task.status.trim().toLowerCase() === "completed" && task.stashScanStatus.trim().toLowerCase() !== "started";
}

function simplifyMessage(message: string) {
  return message.replace(/^downloader:\s*/i, "").replace(/^stashsync:\s*/i, "").trim();
}

function taskFailureSummary(task: DashboardTask): TaskFailureSummary {
  const stashError = simplifyMessage(task.stashScanError || "");
  const taskError = simplifyMessage(task.error || "");
  const status = normalizeStatus(task.status);
  const scanStatus = normalizeStatus(task.stashScanStatus || "");
  const qbtState = normalizeStatus(task.qbittorrentState || "");

  if (stashError) {
    if (stashError.includes("at least one scan path is required")) {
      return {
        title: "缺少扫描路径",
        detail: "任务没有可用于 Stash 扫描的内容路径或保存路径。",
        tone: "tone-danger"
      };
    }
    if (stashError.includes("not configured")) {
      return {
        title: "Stash 未配置",
        detail: "当前任务需要触发 Stash 扫描，但后端未启用对应连接。",
        tone: "tone-danger"
      };
    }
    return {
      title: "Stash 扫描失败",
      detail: stashError,
      tone: "tone-danger"
    };
  }

  if (taskError) {
    if (taskError.includes("no downloadable torrent candidate found")) {
      return {
        title: "没有可下载候选",
        detail: "搜索返回了结果，但没有可直接提交的 magnet 或种子链接。",
        tone: "tone-warn"
      };
    }
    if (taskError.includes("tracker is not configured")) {
      return {
        title: "索引器未配置",
        detail: "当前下载链路无法访问 Jackett 或其他搜索后端。",
        tone: "tone-danger"
      };
    }
    if (taskError.includes("torrent url is required")) {
      return {
        title: "缺少种子地址",
        detail: "手动添加任务时没有提供有效的磁链或下载地址。",
        tone: "tone-warn"
      };
    }
    if (taskError.includes("qBittorrent client is required") || taskError.includes("qBittorrent client is not configured")) {
      return {
        title: "下载器未启用",
        detail: "任务无法提交到 qBittorrent，需先补齐下载器配置。",
        tone: "tone-danger"
      };
    }
    if (taskError.includes("add torrent")) {
      return {
        title: "提交下载失败",
        detail: taskError,
        tone: "tone-danger"
      };
    }
    return {
      title: "任务执行失败",
      detail: taskError,
      tone: "tone-danger"
    };
  }

  if (status.includes("failed")) {
    return {
      title: "任务状态失败",
      detail: task.status || "任务被标记为失败，但没有更多错误上下文。",
      tone: "tone-danger"
    };
  }

  if (scanStatus) {
    return {
      title: "等待扫描收口",
      detail: task.stashScanStatus || "下载已完成，等待 Stash 扫描继续推进。",
      tone: "tone-warn"
    };
  }

  if (qbtState) {
    return {
      title: "下载进行中",
      detail: task.qbittorrentState || "任务仍在等待下载状态变化。",
      tone: "tone-info"
    };
  }

  return {
    title: "状态正常",
    detail: "当前任务没有显式错误，等待下一次同步。",
    tone: "tone-neutral"
  };
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

function boolState(value: boolean, positive = "已配置", negative = "未配置") {
  return value ? positive : negative;
}

function serviceStatus(configured: boolean, enabled: boolean) {
  if (enabled) return { label: "已启用", tone: "tone-success" };
  if (configured) return { label: "待启用", tone: "tone-warn" };
  return { label: "未配置", tone: "tone-neutral" };
}

function taskSyncStatus(settings: RuntimeSettings["tasks"]) {
  return settings.progressSyncEnabled ? "已启用" : "未启用";
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
  const [followingSearch, setFollowingSearch] = useState("");
  const [followingPage, setFollowingPage] = useState(1);
  const [followingPageSize, setFollowingPageSize] = useState<number>(24);
  const [logsLevel, setLogsLevel] = useState<LogLevel>(LogLevel.Info);
  const [logsActionFeedback, setLogsActionFeedback] = useState<SettingsFeedback>(null);
  const [downloadingLogFile, setDownloadingLogFile] = useState(false);
  const [followingFeedback, setFollowingFeedback] = useState<SettingsFeedback>(null);
  const [pendingFollowingID, setPendingFollowingID] = useState<string | null>(null);
  const [pendingAddId, setPendingAddId] = useState<string | null>(null);
  const [stashForm, setStashForm] = useState(EMPTY_STASH_FORM);
  const [jackettForm, setJackettForm] = useState(EMPTY_JACKETT_FORM);
  const [qbittorrentForm, setQBittorrentForm] = useState(EMPTY_QBITTORRENT_FORM);
  const [followingForm, setFollowingForm] = useState(EMPTY_FOLLOWING_FORM);
  const [loggingForm, setLoggingForm] = useState(EMPTY_LOGGING_FORM);
  const [settingsFeedback, setSettingsFeedback] = useState<Record<"Stash" | "索引器" | "下载器" | "订阅" | "系统", SettingsFeedback>>({
    Stash: null,
    索引器: null,
    下载器: null,
    订阅: null,
    系统: null
  });

  const deferredTaskSearch = useDeferredValue(taskSearch.trim().toLowerCase());
  const deferredJackettQuery = useDeferredValue(submittedJackettQuery.trim());
  const deferredFollowingSearch = useDeferredValue(followingSearch.trim());

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
  const [{ data: stashPerformersData, fetching: fetchingStashPerformers, error: stashPerformersError }, refreshStashPerformers] =
    useQuery<StashPerformersQuery, StashPerformersQueryVariables>({
      query: StashPerformersDocument,
      variables: {
        search: deferredFollowingSearch || null,
        page: followingPage,
        pageSize: followingPageSize
      },
      requestPolicy: "cache-and-network",
      pause: tab !== "订阅"
    });
  const [{ data: followingData, fetching: fetchingFollowing, error: followingError }, refreshFollowing] = useQuery<
    FollowingPerformersQuery,
    Record<string, never>
  >({
    query: FollowingPerformersDocument,
    requestPolicy: "cache-and-network",
    pause: tab !== "订阅"
  });
  const [{ data: logsData, fetching: fetchingLogs, error: logsError }, refreshLogs] = useQuery<
    LogsDocumentQuery,
    LogsDocumentQueryVariables
  >({
    query: LogsDocumentDocument,
    variables: {
      limit: 200,
      minLevel: logsLevel
    },
    requestPolicy: "cache-and-network",
    pause: settingsTab !== "日志" || (drawer !== "settings" && renderedDrawer !== "settings")
  });

  const [, addTorrent] = useMutation<
    AddTorrentDocumentMutation,
    AddTorrentDocumentMutationVariables
  >(AddTorrentDocumentDocument);
  const [, syncTaskProgress] = useMutation<
    SyncTaskProgressDocumentMutation,
    SyncTaskProgressDocumentMutationVariables
  >(SyncTaskProgressDocumentDocument);
  const [{ fetching: triggeringTaskScan }, triggerTaskStashScan] = useMutation<
    TriggerTaskStashScanDocumentMutation,
    TriggerTaskStashScanDocumentMutationVariables
  >(TriggerTaskStashScanDocumentDocument);
  const [, triggerStashScans] = useMutation<
    TriggerStashScansDocumentMutation,
    TriggerStashScansDocumentMutationVariables
  >(TriggerStashScansDocumentDocument);
  const [{ fetching: updatingStash }, updateStashSettings] = useMutation<
    UpdateStashSettingsDocumentMutation,
    UpdateStashSettingsDocumentMutationVariables
  >(UpdateStashSettingsDocumentDocument);
  const [{ fetching: updatingJackett }, updateJackettSettings] = useMutation<
    UpdateJackettSettingsDocumentMutation,
    UpdateJackettSettingsDocumentMutationVariables
  >(UpdateJackettSettingsDocumentDocument);
  const [{ fetching: updatingQBittorrent }, updateQBittorrentSettings] = useMutation<
    UpdateQBittorrentSettingsDocumentMutation,
    UpdateQBittorrentSettingsDocumentMutationVariables
  >(UpdateQBittorrentSettingsDocumentDocument);
  const [{ fetching: updatingFollowing }, updateFollowingSettings] = useMutation<
    UpdateFollowingSettingsDocumentMutation,
    UpdateFollowingSettingsDocumentMutationVariables
  >(UpdateFollowingSettingsDocumentDocument);
  const [{ fetching: updatingLogging }, updateLoggingSettings] = useMutation<
    UpdateLoggingSettingsDocumentMutation,
    UpdateLoggingSettingsDocumentMutationVariables
  >(UpdateLoggingSettingsDocumentDocument);
  const [, followPerformer] = useMutation<FollowPerformerMutation, FollowPerformerMutationVariables>(FollowPerformerDocument);
  const [, unfollowPerformer] = useMutation<UnfollowPerformerMutation, UnfollowPerformerMutationVariables>(UnfollowPerformerDocument);
  const [, refreshFollowingPerformer] = useMutation<
    RefreshFollowingPerformerMutation,
    RefreshFollowingPerformerMutationVariables
  >(RefreshFollowingPerformerDocument);
  const [{ fetching: refreshingFollowingNow }, refreshFollowingNow] = useMutation<
    RefreshFollowingNowMutation,
    RefreshFollowingNowMutationVariables
  >(RefreshFollowingNowDocument);

  const tasks = data?.tasks ?? [];
  const logs = logsData?.logs ?? [];
  const runtimeSettings = data?.settings ?? null;
  const stashPerformerPage = stashPerformersData?.stashPerformers ?? null;
  const stashPerformers = stashPerformerPage?.items ?? [];
  const followedPerformers = followingData?.followingPerformers ?? [];
  const activeTask = selectedTaskId ? tasks.find((task) => task.id === selectedTaskId) ?? null : null;
  const activeTaskFailure = activeTask ? taskFailureSummary(activeTask) : null;

  useEffect(() => {
    setFollowingPage(1);
  }, [deferredFollowingSearch]);

  useEffect(() => {
    setFollowingPage(1);
  }, [followingPageSize]);

  useEffect(() => {
    if (!runtimeSettings) return;

    setStashForm({
      url: runtimeSettings.stash.url || "",
      apiKey: "",
      libraryPath: runtimeSettings.stash.libraryPath || ""
    });
    setJackettForm({
      url: runtimeSettings.jackett.url || "",
      apiKey: ""
    });
    setQBittorrentForm({
      url: runtimeSettings.qbittorrent.url || "",
      username: runtimeSettings.qbittorrent.username || "",
      password: "",
      defaultSavePath: runtimeSettings.qbittorrent.defaultSavePath || "",
      category: runtimeSettings.qbittorrent.category || "",
      tags: runtimeSettings.qbittorrent.tags || ""
    });
    setFollowingForm({
      store: runtimeSettings.following.store || "json",
      jsonPath: runtimeSettings.following.jsonPath || "",
      pollIntervalSeconds: String(runtimeSettings.following.pollIntervalSeconds || 3600),
      javstashApiKey: ""
    });
    setLoggingForm({
      level: runtimeSettings.logging.level || "info",
      filePath: runtimeSettings.logging.filePath || "",
      maxEntries: String(runtimeSettings.logging.maxEntries || 500),
      maxFileSizeBytes: String(runtimeSettings.logging.maxFileSizeBytes || 10 * 1024 * 1024),
      maxFileBackups: String(runtimeSettings.logging.maxFileBackups || 5)
    });
  }, [runtimeSettings]);

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
    active: data?.dashboardStats.active ?? 0,
    completed: data?.dashboardStats.completed ?? 0,
    downloading: data?.dashboardStats.downloading ?? 0,
    pendingScans: data?.dashboardStats.pendingScans ?? 0,
    failed: data?.dashboardStats.failed ?? 0,
    total: data?.dashboardStats.total ?? 0,
    versions: data?.version ?? "unknown"
  };

  const taskGroups = useMemo(() => {
    const order: TaskGroupKey[] = ["需处理", "运行中", "待入库", "已完成"];
    return order.map((group) => ({
      group,
      tone: taskGroupTone(group),
      description: taskGroupDescription(group),
      tasks: visibleTasks.filter((task) => taskGroup(task) === group)
    }));
  }, [visibleTasks]);
  const followedByID = useMemo(() => {
    return new Map(followedPerformers.map((item) => [item.performer.id, item]));
  }, [followedPerformers]);

  const dependencyCards = runtimeSettings
    ? [
        {
          name: "Stash",
          ...serviceStatus(runtimeSettings.stash.configured, runtimeSettings.stash.enabled),
          detail: runtimeSettings.stash.enabled
            ? `媒体库路径: ${runtimeSettings.stash.libraryPath || "未设置"}`
            : runtimeSettings.stash.configured
              ? "配置已存在，但运行时尚未启用"
              : "缺少 Stash URL 或库路径"
        },
        {
          name: "Jackett",
          ...serviceStatus(runtimeSettings.jackett.configured, runtimeSettings.jackett.enabled),
          detail: runtimeSettings.jackett.enabled
            ? `索引地址: ${runtimeSettings.jackett.url || "未设置"}`
            : "缺少 URL 或 API key"
        },
        {
          name: "qBittorrent",
          ...serviceStatus(runtimeSettings.qbittorrent.configured, runtimeSettings.qbittorrent.enabled),
          detail: runtimeSettings.qbittorrent.enabled
            ? `默认保存路径: ${runtimeSettings.qbittorrent.defaultSavePath || "未设置"}`
            : runtimeSettings.qbittorrent.configured
              ? "配置完整，但运行时未连接成功"
              : "缺少 URL、用户名或密码"
        },
        {
          name: "订阅",
          label: runtimeSettings.following.pollEnabled ? "已启用" : "未启用",
          tone: runtimeSettings.following.pollEnabled ? "tone-success" : "tone-neutral",
          detail: runtimeSettings.following.javstashEnabled
            ? `轮询间隔: ${runtimeSettings.following.pollIntervalSeconds} 秒`
            : "缺少 JAVStash API key，暂时只能手动检查"
        }
      ]
    : [];

  const settingsStatus = (() => {
    if (!runtimeSettings) {
      return { label: "载入中", tone: "tone-neutral" as const };
    }
    if (settingsTab === "Stash") {
      return serviceStatus(runtimeSettings.stash.configured, runtimeSettings.stash.enabled);
    }
    if (settingsTab === "索引器") {
      return serviceStatus(runtimeSettings.jackett.configured, runtimeSettings.jackett.enabled);
    }
    if (settingsTab === "下载器") {
      return serviceStatus(runtimeSettings.qbittorrent.configured, runtimeSettings.qbittorrent.enabled);
    }
    if (settingsTab === "任务") {
      return {
        label: taskSyncStatus(runtimeSettings.tasks),
        tone: runtimeSettings.tasks.progressSyncEnabled ? "tone-success" as const : "tone-neutral" as const
      };
    }
    if (settingsTab === "订阅") {
      return {
        label: runtimeSettings.following.pollEnabled ? "已启用" : "未启用",
        tone: runtimeSettings.following.pollEnabled ? "tone-success" as const : "tone-neutral" as const
      };
    }
    if (settingsTab === "系统") {
      return { label: "已接线", tone: "tone-info" as const };
    }
    return { label: "规划中", tone: "tone-neutral" as const };
  })();

  const saveStashSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSettingsFeedback((current) => ({ ...current, Stash: null }));

    const result = await updateStashSettings({
        input: {
        url: stashForm.url.trim(),
        apiKey: stashForm.apiKey.trim() || null,
        libraryPath: stashForm.libraryPath.trim()
      }
    });

    if (result.error) {
      setSettingsFeedback((current) => ({
        ...current,
        Stash: { tone: "tone-danger", message: describeQueryError(result.error) }
      }));
      return;
    }

    setStashForm((current) => ({ ...current, apiKey: "" }));
    setSettingsFeedback((current) => ({
      ...current,
      Stash: { tone: "tone-success", message: "Stash 设置已保存，配置文件与运行时快照已刷新。" }
    }));
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveJackettSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSettingsFeedback((current) => ({ ...current, 索引器: null }));

    const result = await updateJackettSettings({
      input: {
        url: jackettForm.url.trim(),
        apiKey: jackettForm.apiKey.trim() || null
      }
    });

    if (result.error) {
      setSettingsFeedback((current) => ({
        ...current,
        索引器: { tone: "tone-danger", message: describeQueryError(result.error) }
      }));
      return;
    }

    setJackettForm((current) => ({ ...current, apiKey: "" }));
    setSettingsFeedback((current) => ({
      ...current,
      索引器: { tone: "tone-success", message: "索引器设置已保存，后端配置已同步。" }
    }));
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveQBittorrentSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSettingsFeedback((current) => ({ ...current, 下载器: null }));

    const result = await updateQBittorrentSettings({
      input: {
        url: qbittorrentForm.url.trim(),
        username: qbittorrentForm.username.trim(),
        password: qbittorrentForm.password.trim() || null,
        defaultSavePath: qbittorrentForm.defaultSavePath.trim(),
        category: qbittorrentForm.category.trim(),
        tags: qbittorrentForm.tags.trim()
      }
    });

    if (result.error) {
      setSettingsFeedback((current) => ({
        ...current,
        下载器: { tone: "tone-danger", message: describeQueryError(result.error) }
      }));
      return;
    }

    setQBittorrentForm((current) => ({ ...current, password: "" }));
    setSettingsFeedback((current) => ({
      ...current,
      下载器: { tone: "tone-success", message: "下载器设置已保存，新的默认值已同步到后端。" }
    }));
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveFollowingSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSettingsFeedback((current) => ({ ...current, 订阅: null }));

    const pollIntervalSeconds = Number.parseInt(followingForm.pollIntervalSeconds.trim(), 10);
    const normalizedPollIntervalSeconds = Number.isNaN(pollIntervalSeconds) ? 0 : pollIntervalSeconds;

    const result = await updateFollowingSettings({
      input: {
        store: followingForm.store.trim() || "json",
        jsonPath: followingForm.jsonPath.trim(),
        pollIntervalSeconds: normalizedPollIntervalSeconds,
        javstashApiKey: followingForm.javstashApiKey.trim() || null
      }
    });

    if (result.error) {
      setSettingsFeedback((current) => ({
        ...current,
        订阅: { tone: "tone-danger", message: describeQueryError(result.error) }
      }));
      return;
    }

    setFollowingForm((current) => ({ ...current, javstashApiKey: "" }));
    setSettingsFeedback((current) => ({
      ...current,
      订阅: { tone: "tone-success", message: "订阅设置已保存，轮询与 JAVStash 凭据已同步到后端。" }
    }));
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveLoggingSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSettingsFeedback((current) => ({ ...current, 系统: null }));

    const maxEntries = Number.parseInt(loggingForm.maxEntries.trim(), 10);
    const maxFileSizeBytes = Number.parseInt(loggingForm.maxFileSizeBytes.trim(), 10);
    const maxFileBackups = Number.parseInt(loggingForm.maxFileBackups.trim(), 10);

    const result = await updateLoggingSettings({
      input: {
        level: loggingForm.level.trim() || "info",
        filePath: loggingForm.filePath.trim(),
        maxEntries: Number.isNaN(maxEntries) ? 500 : maxEntries,
        maxFileSizeBytes: Number.isNaN(maxFileSizeBytes) ? 10 * 1024 * 1024 : maxFileSizeBytes,
        maxFileBackups: Number.isNaN(maxFileBackups) ? 5 : maxFileBackups
      }
    });

    if (result.error) {
      setSettingsFeedback((current) => ({
        ...current,
        系统: { tone: "tone-danger", message: describeQueryError(result.error) }
      }));
      return;
    }

    setSettingsFeedback((current) => ({
      ...current,
      系统: { tone: "tone-success", message: "日志设置已保存，并已即时重载到当前进程。" }
    }));
    await refreshDashboard({ requestPolicy: "network-only" });
    await refreshLogs({ requestPolicy: "network-only" });
  };

  const reloadFollowing = async () => {
    await Promise.all([
      refreshFollowing({ requestPolicy: "network-only" }),
      refreshStashPerformers({ requestPolicy: "network-only" })
    ]);
  };

  const handleFollowToggle = async (performer: StashPerformerEntry) => {
    setPendingFollowingID(performer.id);
    setFollowingFeedback(null);

    const result = performer.followed
      ? await unfollowPerformer({ stashPerformerID: performer.id })
      : await followPerformer({ stashPerformerID: performer.id });

    if (result.error) {
      setFollowingFeedback({
        tone: "tone-danger",
        message: describeQueryError(result.error)
      });
      setPendingFollowingID(null);
      return;
    }

    setFollowingFeedback({
      tone: "tone-success",
      message: performer.followed ? `已取消订阅 ${performer.name}。` : `已订阅 ${performer.name}，Moji 会通过 custom_fields 记录状态。`
    });
    await reloadFollowing();
    setPendingFollowingID(null);
  };

  const handleRefreshPerformer = async (performer: StashPerformerEntry) => {
    setPendingFollowingID(performer.id);
    setFollowingFeedback(null);

    const result = await refreshFollowingPerformer({ stashPerformerID: performer.id });
    if (result.error) {
      setFollowingFeedback({
        tone: "tone-danger",
        message: describeQueryError(result.error)
      });
      setPendingFollowingID(null);
      return;
    }

    setFollowingFeedback({
      tone: "tone-info",
      message: `已检查 ${performer.name} 的最新发行信息。`
    });
    await reloadFollowing();
    setPendingFollowingID(null);
  };

  const handleRefreshAllFollowing = async () => {
    setFollowingFeedback(null);
    const result = await refreshFollowingNow({});
    if (result.error) {
      setFollowingFeedback({
        tone: "tone-danger",
        message: describeQueryError(result.error)
      });
      return;
    }

    setFollowingFeedback({
      tone: "tone-info",
      message: "已触发全部订阅对象的更新检查。"
    });
    await reloadFollowing();
  };

  const handleCopyLogs = async () => {
    if (!logs.length) {
      setLogsActionFeedback({ tone: "tone-info", message: "当前列表里还没有可复制的日志。" });
      return;
    }

    try {
      await navigator.clipboard.writeText(formatLogEntries(logs));
      setLogsActionFeedback({ tone: "tone-success", message: `已复制 ${logs.length} 条日志。` });
    } catch {
      setLogsActionFeedback({ tone: "tone-danger", message: "复制失败，请检查浏览器剪贴板权限。" });
    }
  };

  const handleDownloadCurrentLogFile = async () => {
    setLogsActionFeedback(null);
    setDownloadingLogFile(true);

    try {
      const response = await fetch("/api/logs/current");
      if (!response.ok) {
        const message = (await response.text()) || `下载失败 (${response.status})`;
        throw new Error(message);
      }

      const blob = await response.blob();
      const downloadURL = window.URL.createObjectURL(blob);
      const disposition = response.headers.get("Content-Disposition") || "";
      const match = disposition.match(/filename="([^"]+)"/i);
      const filename = match?.[1] || "moji.log";
      const anchor = document.createElement("a");
      anchor.href = downloadURL;
      anchor.download = filename;
      document.body.append(anchor);
      anchor.click();
      anchor.remove();
      window.URL.revokeObjectURL(downloadURL);
      setLogsActionFeedback({ tone: "tone-success", message: `已开始下载 ${filename}。` });
    } catch (error) {
      setLogsActionFeedback({
        tone: "tone-danger",
        message: error instanceof Error ? error.message : "下载当前日志文件失败。"
      });
    } finally {
      setDownloadingLogFile(false);
    }
  };

  const renderSettingsPanel = () => {
    if (!runtimeSettings) {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <dl className="settings-grid">
            <div>
              <dt>当前状态</dt>
              <dd>等待后端返回配置状态</dd>
            </div>
            <div>
              <dt>说明</dt>
              <dd>设置面板会在 dashboard 查询完成后显示实时状态。</dd>
            </div>
          </dl>
        </article>
      );
    }

    if (settingsTab === "Stash") {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <form className="settings-form" onSubmit={(event) => void saveStashSettings(event)}>
            <label className="settings-field">
              <span>Stash URL</span>
              <input
                value={stashForm.url}
                onChange={(event) => setStashForm((current) => ({ ...current, url: event.target.value }))}
                placeholder="http://localhost:9999"
              />
            </label>
            <label className="settings-field">
              <span>Library path</span>
              <input
                value={stashForm.libraryPath}
                onChange={(event) => setStashForm((current) => ({ ...current, libraryPath: event.target.value }))}
                placeholder="/data/library"
              />
            </label>
            <label className="settings-field">
              <span>API key</span>
              <input
                type="password"
                value={stashForm.apiKey}
                onChange={(event) => setStashForm((current) => ({ ...current, apiKey: event.target.value }))}
                placeholder={runtimeSettings.stash.apiKeyConfigured ? "留空则保留现有 API key" : "输入新的 API key"}
              />
            </label>
            <div className="settings-meta">
              <span>当前 API key: {boolState(runtimeSettings.stash.apiKeyConfigured)}</span>
            </div>
            {settingsFeedback.Stash ? <p className={`settings-feedback ${settingsFeedback.Stash.tone}`}>{settingsFeedback.Stash.message}</p> : null}
            <div className="settings-actions">
              <button type="submit" disabled={updatingStash}>保存 Stash 设置</button>
            </div>
          </form>
        </article>
      );
    }

    if (settingsTab === "索引器") {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <form className="settings-form" onSubmit={(event) => void saveJackettSettings(event)}>
            <label className="settings-field">
              <span>Jackett URL</span>
              <input
                value={jackettForm.url}
                onChange={(event) => setJackettForm((current) => ({ ...current, url: event.target.value }))}
                placeholder="http://localhost:9117"
              />
            </label>
            <label className="settings-field">
              <span>API key</span>
              <input
                type="password"
                value={jackettForm.apiKey}
                onChange={(event) => setJackettForm((current) => ({ ...current, apiKey: event.target.value }))}
                placeholder={runtimeSettings.jackett.apiKeyConfigured ? "留空则保留现有 API key" : "输入新的 API key"}
              />
            </label>
            <div className="settings-meta">
              <span>当前 API key: {boolState(runtimeSettings.jackett.apiKeyConfigured)}</span>
              <span>后续可继续扩展 tracker 分组与默认搜索策略。</span>
            </div>
            {settingsFeedback.索引器 ? <p className={`settings-feedback ${settingsFeedback.索引器.tone}`}>{settingsFeedback.索引器.message}</p> : null}
            <div className="settings-actions">
              <button type="submit" disabled={updatingJackett}>保存索引器设置</button>
            </div>
          </form>
        </article>
      );
    }

    if (settingsTab === "下载器") {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <form className="settings-form" onSubmit={(event) => void saveQBittorrentSettings(event)}>
            <label className="settings-field">
              <span>qBittorrent URL</span>
              <input
                value={qbittorrentForm.url}
                onChange={(event) => setQBittorrentForm((current) => ({ ...current, url: event.target.value }))}
                placeholder="http://localhost:8080"
              />
            </label>
            <label className="settings-field">
              <span>用户名</span>
              <input
                value={qbittorrentForm.username}
                onChange={(event) => setQBittorrentForm((current) => ({ ...current, username: event.target.value }))}
                placeholder="admin"
              />
            </label>
            <label className="settings-field">
              <span>密码</span>
              <input
                type="password"
                value={qbittorrentForm.password}
                onChange={(event) => setQBittorrentForm((current) => ({ ...current, password: event.target.value }))}
                placeholder={runtimeSettings.qbittorrent.passwordConfigured ? "留空则保留现有密码" : "输入新的登录密码"}
              />
            </label>
            <label className="settings-field">
              <span>默认保存路径</span>
              <input
                value={qbittorrentForm.defaultSavePath}
                onChange={(event) => setQBittorrentForm((current) => ({ ...current, defaultSavePath: event.target.value }))}
                placeholder="/downloads"
              />
            </label>
            <label className="settings-field">
              <span>默认分类</span>
              <input
                value={qbittorrentForm.category}
                onChange={(event) => setQBittorrentForm((current) => ({ ...current, category: event.target.value }))}
                placeholder="moji"
              />
            </label>
            <label className="settings-field">
              <span>默认标签</span>
              <input
                value={qbittorrentForm.tags}
                onChange={(event) => setQBittorrentForm((current) => ({ ...current, tags: event.target.value }))}
                placeholder="auto"
              />
            </label>
            <div className="settings-meta">
              <span>当前密码: {boolState(runtimeSettings.qbittorrent.passwordConfigured)}</span>
              <span>用户名会直接回显，密码仍只支持覆盖更新。</span>
            </div>
            {settingsFeedback.下载器 ? <p className={`settings-feedback ${settingsFeedback.下载器.tone}`}>{settingsFeedback.下载器.message}</p> : null}
            <div className="settings-actions">
              <button type="submit" disabled={updatingQBittorrent}>保存下载器设置</button>
            </div>
          </form>
        </article>
      );
    }

    if (settingsTab === "任务") {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <dl className="settings-grid">
            <div>
              <dt>存储类型</dt>
              <dd>{runtimeSettings.tasks.store || "json"}</dd>
            </div>
            <div>
              <dt>JSON 路径</dt>
              <dd>{runtimeSettings.tasks.jsonPath || "moji-tasks.json"}</dd>
            </div>
            <div>
              <dt>同步间隔</dt>
              <dd>{runtimeSettings.tasks.progressSyncIntervalSeconds} 秒</dd>
            </div>
            <div>
              <dt>进度同步</dt>
              <dd>{taskSyncStatus(runtimeSettings.tasks)}</dd>
            </div>
            <div>
              <dt>说明</dt>
              <dd>当前同步开关由任务配置和下载链路是否启用共同决定。</dd>
            </div>
          </dl>
        </article>
      );
    }

    if (settingsTab === "订阅") {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <form className="settings-form" onSubmit={(event) => void saveFollowingSettings(event)}>
            <label className="settings-field">
              <span>存储类型</span>
              <input
                value={followingForm.store}
                onChange={(event) => setFollowingForm((current) => ({ ...current, store: event.target.value }))}
                placeholder="json"
              />
            </label>
            <label className="settings-field">
              <span>状态文件路径</span>
              <input
                value={followingForm.jsonPath}
                onChange={(event) => setFollowingForm((current) => ({ ...current, jsonPath: event.target.value }))}
                placeholder="moji-following.json"
              />
            </label>
            <label className="settings-field">
              <span>轮询间隔（秒）</span>
              <input
                value={followingForm.pollIntervalSeconds}
                onChange={(event) => setFollowingForm((current) => ({ ...current, pollIntervalSeconds: event.target.value }))}
                placeholder="3600"
                inputMode="numeric"
              />
            </label>
            <label className="settings-field">
              <span>JAVStash API key</span>
              <input
                type="password"
                value={followingForm.javstashApiKey}
                onChange={(event) => setFollowingForm((current) => ({ ...current, javstashApiKey: event.target.value }))}
                placeholder={runtimeSettings.following.javstashApiKeyConfigured ? "留空则保留现有 API key" : "输入新的 API key"}
              />
            </label>
            <div className="settings-meta">
              <span>当前存储: {runtimeSettings.following.store || "json"}</span>
              <span>JAVStash key: {boolState(runtimeSettings.following.javstashApiKeyConfigured)}</span>
              <span>轮询状态: {runtimeSettings.following.pollEnabled ? "已启用" : "未启用"}</span>
            </div>
            {settingsFeedback.订阅 ? <p className={`settings-feedback ${settingsFeedback.订阅.tone}`}>{settingsFeedback.订阅.message}</p> : null}
            <div className="settings-actions">
              <button type="submit" disabled={updatingFollowing}>保存订阅设置</button>
            </div>
          </form>
        </article>
      );
    }

    if (settingsTab === "系统") {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <form className="settings-form" onSubmit={(event) => void saveLoggingSettings(event)}>
            <label className="settings-field">
              <span>日志级别</span>
              <input
                value={loggingForm.level}
                onChange={(event) => setLoggingForm((current) => ({ ...current, level: event.target.value }))}
                placeholder="info"
              />
            </label>
            <label className="settings-field">
              <span>日志文件路径</span>
              <input
                value={loggingForm.filePath}
                onChange={(event) => setLoggingForm((current) => ({ ...current, filePath: event.target.value }))}
                placeholder="moji.log"
              />
            </label>
            <label className="settings-field">
              <span>内存保留条数</span>
              <input
                value={loggingForm.maxEntries}
                onChange={(event) => setLoggingForm((current) => ({ ...current, maxEntries: event.target.value }))}
                inputMode="numeric"
                placeholder="500"
              />
            </label>
            <label className="settings-field">
              <span>单文件大小上限（字节）</span>
              <input
                value={loggingForm.maxFileSizeBytes}
                onChange={(event) => setLoggingForm((current) => ({ ...current, maxFileSizeBytes: event.target.value }))}
                inputMode="numeric"
                placeholder={String(10 * 1024 * 1024)}
              />
            </label>
            <label className="settings-field">
              <span>滚动备份份数</span>
              <input
                value={loggingForm.maxFileBackups}
                onChange={(event) => setLoggingForm((current) => ({ ...current, maxFileBackups: event.target.value }))}
                inputMode="numeric"
                placeholder="5"
              />
            </label>
            <div className="settings-meta">
              <span>版本: {runtimeSettings.system.appVersion || "dev"}</span>
              <span>当前日志文件: {runtimeSettings.logging.filePath}</span>
              <span>当前缓存: {runtimeSettings.logging.maxEntries} 条</span>
            </div>
            {settingsFeedback.系统 ? <p className={`settings-feedback ${settingsFeedback.系统.tone}`}>{settingsFeedback.系统.message}</p> : null}
            <div className="settings-actions">
              <button type="submit" disabled={updatingLogging}>保存系统设置</button>
            </div>
          </form>
        </article>
      );
    }

    if (settingsTab === "日志") {
      return (
        <article className="drawer-card">
          <div className="drawer-card__head">
            <h3>{settingsTab}</h3>
            <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
          </div>
          <div className="toolbar-inline toolbar-inline--logs">
            <select value={logsLevel} onChange={(event) => setLogsLevel(event.target.value as LogLevel)}>
              {LOG_LEVEL_OPTIONS.map((level) => (
                <option key={level} value={level}>
                  {level}
                </option>
              ))}
            </select>
            <button type="button" className="ghost-button" onClick={() => void refreshLogs({ requestPolicy: "network-only" })}>
              {fetchingLogs ? "刷新中..." : "刷新日志"}
            </button>
            <button type="button" className="ghost-button" onClick={() => void handleCopyLogs()}>
              复制当前列表
            </button>
            <button type="button" className="ghost-button" onClick={() => void handleDownloadCurrentLogFile()} disabled={downloadingLogFile}>
              {downloadingLogFile ? "下载中..." : "下载当前日志"}
            </button>
          </div>
          <div className="settings-meta">
            <span>级别过滤: {logsLevel}</span>
            <span>已加载: {logs.length}</span>
            <span>状态: {fetchingLogs ? "同步中" : "已就绪"}</span>
            <span>文件: {runtimeSettings.logging.filePath}</span>
          </div>
          {logsActionFeedback ? <p className={`settings-feedback ${logsActionFeedback.tone}`}>{logsActionFeedback.message}</p> : null}
          {logsError ? <p className="settings-feedback tone-danger">{describeQueryError(logsError)}</p> : null}
          {!logs.length && !fetchingLogs ? (
            <article className="empty-card empty-card--wide">
              <h3>暂无日志</h3>
              <p>当前过滤条件下没有最近日志记录。</p>
            </article>
          ) : (
            <div className="log-stream" role="log" aria-live="polite">
              {logs.map((entry, index) => (
                <div
                  key={`${entry.time}-${index}`}
                  className={`log-line ${
                    entry.level === LogLevel.Error
                      ? "log-line--error"
                      : entry.level === LogLevel.Warning
                        ? "log-line--warn"
                        : entry.level === LogLevel.Debug
                          ? "log-line--debug"
                          : "log-line--info"
                  }`}
                >
                  <span className="log-line__time">{formatDateTime(entry.time)}</span>
                  <span className="log-line__level">[{entry.level}]</span>
                  <span className="log-line__message">{entry.message}</span>
                </div>
              ))}
            </div>
          )}
        </article>
      );
    }

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <dl className="settings-grid">
          <div>
            <dt>当前状态</dt>
            <dd>该分区尚未接入后端契约</dd>
          </div>
          <div>
            <dt>敏感值</dt>
            <dd>前端不展示明文</dd>
          </div>
          <div>
            <dt>接入方式</dt>
            <dd>后续会扩展为真实查询或操作面板</dd>
          </div>
          <div>
            <dt>说明</dt>
            <dd>
              {settingsTab === "安全性"
                ? "这里会放访问控制、CORS 和未来登录策略。"
                : settingsTab === "工具"
                    ? "这里会放重新同步、重新探测和修复动作。"
                    : settingsTab === "更新历史"
                      ? "这里会放版本记录和升级提示。"
                      : "这里会放项目定位、许可证和作者信息。"}
            </dd>
          </div>
        </dl>
      </article>
    );
  };

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

  const runTaskScan = async (taskId: string) => {
    await triggerTaskStashScan({ id: taskId });
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

  useEffect(() => {
    const logsTabActive = settingsTab === "日志" && (drawer === "settings" || renderedDrawer === "settings");
    if (!logsTabActive) {
      return;
    }

    const timer = window.setInterval(() => {
      void refreshLogs({ requestPolicy: "network-only" });
    }, 5000);

    return () => window.clearInterval(timer);
  }, [drawer, renderedDrawer, refreshLogs, settingsTab]);

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
                      <span className={`status-chip ${card.tone}`}>
                        {card.label}
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

              {!visibleTasks.length ? (
                <div className="task-grid">
                  <article className="empty-card empty-card--wide">
                    <h3>没有匹配的任务</h3>
                    <p>换个过滤条件，或者先去发现区创建任务。</p>
                  </article>
                </div>
              ) : null}

              {taskGroups.filter((item) => item.tasks.length > 0).map((item) => (
                <section key={item.group} className="task-group-section">
                  <div className="task-group-section__head">
                    <div>
                      <h3>{item.group}</h3>
                      <p>{item.description}</p>
                    </div>
                    <span className={`status-chip ${item.tone}`}>{item.tasks.length} 项</span>
                  </div>

                  <div className="task-grid">
                    {item.tasks.map((task) => {
                      const failure = taskFailureSummary(task);
                      return (
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
                          <div className={`task-issue ${failure.tone}`}>
                            <strong>{failure.title}</strong>
                            <span>{failure.detail}</span>
                          </div>
                        </article>
                      );
                    })}
                  </div>
                </section>
              ))}
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

        {tab === "订阅" ? (
          <>
            <section className="section-band">
              <div className="band-head">
                <div>
                  <p className="section-kicker">订阅</p>
                  <h2>订阅更新</h2>
                </div>
              </div>

              <div className="toolbar-inline toolbar-inline--following">
                <input placeholder="按名称或别名搜索 Stash performer" value={followingSearch} onChange={(event) => setFollowingSearch(event.target.value)} />
                <select value={followingPageSize} onChange={(event) => setFollowingPageSize(Number(event.target.value))}>
                  {FOLLOWING_PAGE_SIZE_OPTIONS.map((size) => (
                    <option key={size} value={size}>
                      每页 {size} 条
                    </option>
                  ))}
                </select>
                <button type="button" className="ghost-button" onClick={() => void reloadFollowing()}>
                  刷新列表
                </button>
                <button
                  type="button"
                  className="ghost-button"
                  disabled={refreshingFollowingNow || followedPerformers.length === 0}
                  onClick={() => void handleRefreshAllFollowing()}
                >
                  {refreshingFollowingNow ? "检查中..." : "检查全部订阅"}
                </button>
              </div>

              <div className="settings-meta">
                <span>Stash 候选: {stashPerformerPage?.totalCount ?? 0}</span>
                <span>当前页: {stashPerformerPage?.page ?? 1} / {stashPerformerPage?.totalPages ?? 0}</span>
                <span>每页: {stashPerformerPage?.pageSize ?? followingPageSize}</span>
                <span>已订阅: {followedPerformers.length}</span>
                <span>载入状态: {fetchingStashPerformers || fetchingFollowing ? "同步中" : "已就绪"}</span>
              </div>
              {followingFeedback ? <p className={`settings-feedback ${followingFeedback.tone}`}>{followingFeedback.message}</p> : null}
              {followingError || stashPerformersError ? (
                <p className="settings-feedback tone-danger">{describeQueryError(followingError || stashPerformersError)}</p>
              ) : null}

              <div className="profile-grid">
                {stashPerformers.length === 0 && !fetchingStashPerformers ? (
                  <article className="empty-card empty-card--wide">
                    <h3>没有找到匹配的 performer</h3>
                    <p>可以尝试修改关键词，或先确认 Stash 已正确返回 performer 数据。</p>
                  </article>
                ) : null}
                {stashPerformers.map((performer, index) => {
                  const followingEntry = followedByID.get(performer.id) ?? null;
                  const latestRelease = followingEntry?.recentReleases[0] ?? null;
                  const imageURL = performerImageURL(performer.imagePath, runtimeSettings?.stash.url);

                  return (
                  <article key={performer.id} className="profile-card" style={{ animationDelay: `${index * 80}ms` }}>
                    {imageURL ? (
                      <img className="avatar avatar--image" src={imageURL} alt={performer.name} loading="lazy" />
                    ) : (
                      <div className="avatar avatar--placeholder">{performerInitials(performer.name)}</div>
                    )}
                    <div className="profile-card__body">
                      <div className="profile-card__head">
                        <div>
                          <h3>{performer.name}</h3>
                        </div>
                        <div className="profile-card__icons">
                          {performer.favorite ? (
                            <span
                              className="profile-icon profile-icon--favorite is-active"
                              title="Stash 已收藏"
                              aria-label="Stash 已收藏"
                            >
                              <FontAwesomeIcon icon={faHeart} />
                            </span>
                          ) : null}
                          <button
                            type="button"
                            className={`profile-icon profile-icon--follow ${performer.followed ? "is-active" : ""}`}
                            title={performer.followed ? "取消订阅" : "订阅"}
                            aria-label={performer.followed ? "取消订阅" : "订阅"}
                            disabled={pendingFollowingID === performer.id}
                            onClick={() => void handleFollowToggle(performer)}
                          >
                            <FontAwesomeIcon icon={faBookmark} />
                          </button>
                        </div>
                      </div>
                      <dl className="profile-facts">
                        <div>
                          <dt>作品</dt>
                          <dd>{performer.sceneCount}</dd>
                        </div>
                        <div>
                          <dt>检查</dt>
                          <dd>{formatDateTime(followingEntry?.lastCheckedAt)}</dd>
                        </div>
                      </dl>
                      <p className="profile-note">
                        {followingEntry?.lastError
                          ? `最近错误: ${followingEntry.lastError}`
                          : latestRelease
                            ? `最近记录: ${latestRelease.code || latestRelease.title} · ${formatRelativeDate(latestRelease.date || latestRelease.seenAt)}`
                            : performer.followed
                              ? "已订阅，等待首次检查结果。"
                              : "尚未订阅。"}
                      </p>
                      <div className="profile-actions">
                        <button
                          type="button"
                          className="ghost-button"
                          disabled={!performer.followed || pendingFollowingID === performer.id}
                          onClick={() => void handleRefreshPerformer(performer)}
                        >
                          立即检查
                        </button>
                      </div>
                    </div>
                  </article>
                )})}
              </div>

              {stashPerformerPage && stashPerformerPage.totalPages > 1 ? (
                <div className="pagination-bar">
                  <button
                    type="button"
                    className="ghost-button"
                    disabled={!stashPerformerPage.hasPrevPage || fetchingStashPerformers}
                    onClick={() => setFollowingPage((current) => Math.max(1, current - 1))}
                  >
                    上一页
                  </button>
                  <span className="status-chip tone-neutral">
                    第 {stashPerformerPage.page} / {stashPerformerPage.totalPages} 页
                  </span>
                  <button
                    type="button"
                    className="ghost-button"
                    disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers}
                    onClick={() => setFollowingPage((current) => current + 1)}
                  >
                    下一页
                  </button>
                </div>
              ) : null}
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
                        className={`chip ${settingsTab === item ? "is-active" : ""} ${!ENABLED_SETTINGS_TABS.has(item) ? "is-disabled" : ""}`}
                        onClick={() => setSettingsTab(item)}
                        disabled={!ENABLED_SETTINGS_TABS.has(item)}
                      >
                        {item}
                      </button>
                    ))}
                  </div>

                  {renderSettingsPanel()}
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
                        <div className={`task-issue ${activeTaskFailure?.tone ?? "tone-neutral"}`}>
                          <strong>{activeTaskFailure?.title ?? "状态正常"}</strong>
                          <span>{activeTaskFailure?.detail ?? "当前任务没有显式错误，等待下一次同步。"}</span>
                        </div>
                      </article>

                      <div className="inline-actions">
                        <button type="button" className="ghost-button" onClick={() => void runSync()}>
                          同步进度
                        </button>
                        {canTriggerTaskStashScan(activeTask) ? (
                          <button
                            type="button"
                            className="ghost-button"
                            onClick={() => void runTaskScan(activeTask.id)}
                            disabled={triggeringTaskScan}
                          >
                            {triggeringTaskScan ? "触发中" : "扫描当前任务"}
                          </button>
                        ) : null}
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
