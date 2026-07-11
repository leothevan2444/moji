import { FormEvent, lazy, Suspense, useCallback, useDeferredValue, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate, useParams, useSearchParams } from "react-router";
import { useQuery } from "urql";
import { HELP_TOPICS, type HelpTopicId } from "./help";
import { describeQueryError } from "./services/queryError";
import {
  useDashboard,
  useDiscoverScenes,
  useDrawerTransition,
  useGlobalSlashShortcut,
  useJackettIndexers,
  usePreviewJackettSelection,
  useJackettSearch,
  useSearchHistory,
  useSubscription,
  useTheme,
  useToast
} from "./hooks";
import { taskSummary, type TaskGroupKey } from "./utils";
import type { DrawerKey, SettingsTab } from "./types";
import { Drawer } from "./components/layout/Drawer";
import { Header } from "./components/layout/Header";
import { ToastStack } from "./components/layout/ToastStack";
import { JackettFilterPanel } from "./components/drawers/JackettFilterPanel";
import { SortAndPagination } from "./components/drawers/SortAndPagination";
import type { TaskSortKey, TaskStatusFilter } from "./types";
import {
  DISCOVERY_PAGE_SIZE,
  DISCOVER_SORT_OPTIONS,
  JACKETT_SORT_OPTIONS,
  type DiscoveryMode
} from "./constants";
import {
  LibraryFilter,
  SceneSourceFilter,
  type DiscoverScenesDocumentQuery,
  type SearchDocumentQuery,
  DiscoverSortBy,
  JackettSortBy,
  TaskDeletePolicy,
  TaskDetailDocumentDocument,
  type TaskDetailDocumentQuery,
  type TaskDetailDocumentQueryVariables
} from "./graphql/generated/graphql";
import { parseDiscoverSearchParams, parsePerformerSearchParams, parseTaskSearchParams, serializeDiscoverSearchParams, serializePerformerSearchParams, serializeTaskSearchParams } from "./app/searchParams";

const PREVIEW_FAST_RULES_STORAGE_KEY = "moji.discovery.previewFastRules";
const PREVIEW_FILE_RULES_STORAGE_KEY = "moji.discovery.previewFileRules";

const HomePage = lazy(() => import("./pages/HomePage").then((module) => ({ default: module.HomePage })));
const TasksPage = lazy(() => import("./pages/TasksPage").then((module) => ({ default: module.TasksPage })));
const DiscoveryPage = lazy(() => import("./pages/DiscoveryPage").then((module) => ({ default: module.DiscoveryPage })));
const SubscriptionPage = lazy(() => import("./pages/SubscriptionPage").then((module) => ({ default: module.SubscriptionPage })));
const ConfirmDeleteDrawer = lazy(() => import("./components/drawers/ConfirmDeleteDrawer").then((module) => ({ default: module.ConfirmDeleteDrawer })));
const DiscoveryDrawer = lazy(() => import("./components/drawers/DiscoveryDrawer").then((module) => ({ default: module.DiscoveryDrawer })));
const HelpDrawer = lazy(() => import("./components/drawers/HelpDrawer").then((module) => ({ default: module.HelpDrawer })));
const SettingsDrawer = lazy(() => import("./components/drawers/SettingsDrawer").then((module) => ({ default: module.SettingsDrawer })));
const SourcingResolutionDrawer = lazy(() => import("./components/drawers/SourcingResolutionDrawer").then((module) => ({ default: module.SourcingResolutionDrawer })));
const StatsDrawer = lazy(() => import("./components/drawers/StatsDrawer").then((module) => ({ default: module.StatsDrawer })));
const TaskDrawer = lazy(() => import("./components/drawers/TaskDrawer").then((module) => ({ default: module.TaskDrawer })));

function RouteFallback() {
  return <div className="skeleton skeleton-card" aria-label="页面加载中" />;
}

function readStoredBoolean(key: string): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(key) === "true";
  } catch {
    return false;
  }
}

function writeStoredBoolean(key: string, value: boolean) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, String(value));
  } catch {
    // ignore localStorage failures
  }
}

function App() {
  const location = useLocation();
  const navigate = useNavigate();
  const params = useParams();
  const [urlSearchParams, setURLSearchParams] = useSearchParams();
  const pathname = location.pathname;
  const tab = pathname.startsWith("/tasks")
    ? "任务"
    : pathname.startsWith("/performers")
      ? "演员"
      : pathname.startsWith("/discover")
        ? "发现"
        : "主页";
  // ── UI state ────────────────────────────────────────────────────────
  const [drawer, setDrawer] = useState<DrawerKey>(null);
  const [settingsTab, setSettingsTab] = useState<SettingsTab>("连接");
  const openSettings = useCallback((tab: SettingsTab) => {
    const slug: Record<SettingsTab, string> = { 连接: "connections", 入库: "ingest", 自动化: "automation", 系统: "system", 日志: "logs", 关于: "about" };
    navigate(`/settings/${slug[tab]}`);
  }, [navigate]);
  const [helpTopicId, setHelpTopicId] = useState<HelpTopicId>(HELP_TOPICS[0].id);

  // Tasks page state
  const [taskSearch, setTaskSearch] = useState("");
  const [taskStatus, setTaskStatus] = useState<TaskStatusFilter>("全部");
  const [taskSort, setTaskSort] = useState<TaskSortKey>("最新");
  const [taskGroupOpen, setTaskGroupOpen] = useState<Record<TaskGroupKey, boolean>>({
    需处理: true,
    运行中: true,
    待入库: false,
    已完成: false
  });
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(params.taskId ?? null);
  const [pendingTaskScanId, setPendingTaskScanId] = useState<string | null>(null);
  const [pendingTaskRetryId, setPendingTaskRetryId] = useState<string | null>(null);
  const [retryingBlockedTasks, setRetryingBlockedTasks] = useState(false);
  const [pendingTaskDeleteId, setPendingTaskDeleteId] = useState<string | null>(null);
  const [confirmDeleteTaskId, setConfirmDeleteTaskId] = useState<string | null>(null);

  // Discovery page state
  const [discoveryQuery, setDiscoveryQuery] = useState("");
  const [submittedDiscoveryQuery, setSubmittedDiscoveryQuery] = useState("");
  const [discoveryMode, setDiscoveryMode] = useState<DiscoveryMode>("stashbox");
  const [discoveryStashboxSort, setDiscoveryStashboxSort] = useState<DiscoverSortBy>(DiscoverSortBy.Relevance);
  const [discoveryJackettSort, setDiscoveryJackettSort] = useState<JackettSortBy>(JackettSortBy.Relevance);
  const [discoveryPage, setDiscoveryPage] = useState(1);
  const [selectedTrackerIDs, setSelectedTrackerIDs] = useState<string[]>([]);
  const [previewFastRules, setPreviewFastRules] = useState<boolean>(() => readStoredBoolean(PREVIEW_FAST_RULES_STORAGE_KEY));
  const [previewFileRules, setPreviewFileRules] = useState<boolean>(() => readStoredBoolean(PREVIEW_FILE_RULES_STORAGE_KEY));
  const [discoveryInputFocused, setDiscoveryInputFocused] = useState(false);
  const [historyVisible, setHistoryVisible] = useState(false);
  const [pendingAddId, setPendingAddId] = useState<string | null>(null);

  const searchHistory = useSearchHistory();

  // Subscription page state
  const [subscriptionSearch, setSubscriptionSearch] = useState("");
  const [subscriptionPage, setSubscriptionPage] = useState(1);
  const [subscriptionPageSize, setSubscriptionPageSize] = useState<number>(24);
  const [pendingSubscriptionID, setPendingSubscriptionID] = useState<string | null>(null);
  const [selectedPerformerId, setSelectedPerformerId] = useState<string | null>(params.performerId ?? null);
  const [performerSceneSearch, setPerformerSceneSearch] = useState("");
  const [performerSceneSourceFilter, setPerformerSceneSourceFilter] = useState<SceneSourceFilter>(SceneSourceFilter.All);
  const [performerSceneLibraryFilter, setPerformerSceneLibraryFilter] = useState<LibraryFilter>(LibraryFilter.All);
  const [performerScenePageIndex, setPerformerScenePageIndex] = useState(1);
  const [performerScenePageSize, setPerformerScenePageSize] = useState<number>(24);
  const [selectedSceneKeys, setSelectedSceneKeys] = useState<string[]>([]);
  const [pendingPerformerSceneKeys, setPendingPerformerSceneKeys] = useState<string[]>([]);

  useEffect(() => {
    if (pathname.startsWith("/tasks")) {
      const state = parseTaskSearchParams(urlSearchParams);
      setTaskSearch(state.q);
      setTaskStatus(state.status);
      setTaskSort(state.sort);
      setSelectedTaskId(params.taskId ?? null);
    }
    if (pathname.startsWith("/performers")) {
      const state = parsePerformerSearchParams(urlSearchParams);
      setSubscriptionSearch(state.q);
      setSubscriptionPage(state.page);
      setSubscriptionPageSize(state.pageSize);
      setSelectedPerformerId(params.performerId ?? null);
      setPerformerSceneSearch(state.sceneQ);
      setPerformerSceneSourceFilter(state.source === "stash" ? SceneSourceFilter.Stash : state.source === "stashbox" ? SceneSourceFilter.Stashbox : SceneSourceFilter.All);
      setPerformerSceneLibraryFilter(state.library === "in-library" ? LibraryFilter.InLibrary : state.library === "not-in-library" ? LibraryFilter.NotInLibrary : LibraryFilter.All);
      setPerformerScenePageIndex(state.scenePage);
      setPerformerScenePageSize(state.scenePageSize);
    }
    if (pathname.startsWith("/discover")) {
      const state = parseDiscoverSearchParams(urlSearchParams);
      setDiscoveryQuery(state.q);
      setSubmittedDiscoveryQuery(state.q);
      setDiscoveryMode(state.source);
      setDiscoveryPage(state.page);
      setSelectedTrackerIDs(state.trackers);
      const sortToken = state.sort.toUpperCase().replaceAll("-", "_");
      if (state.source === "stashbox") setDiscoveryStashboxSort(sortToken as DiscoverSortBy);
      else setDiscoveryJackettSort(sortToken as JackettSortBy);
      if (state.fastRules !== undefined) setPreviewFastRules(state.fastRules);
      if (state.fileRules !== undefined) setPreviewFileRules(state.fileRules);
    }
  }, [pathname, params.taskId, params.performerId, urlSearchParams]);

  useEffect(() => {
    let canonical: URLSearchParams | null = null;
    if (pathname.startsWith("/tasks")) canonical = serializeTaskSearchParams(parseTaskSearchParams(urlSearchParams));
    if (pathname.startsWith("/performers")) canonical = serializePerformerSearchParams(parsePerformerSearchParams(urlSearchParams));
    if (pathname === "/discover") canonical = serializeDiscoverSearchParams(parseDiscoverSearchParams(urlSearchParams));
    if (canonical && canonical.toString() !== urlSearchParams.toString()) {
      setURLSearchParams(canonical, { replace: true });
    }
  }, [pathname, urlSearchParams, setURLSearchParams]);

  useEffect(() => {
    if (pathname.startsWith("/settings/")) {
      const sectionToTab: Record<string, SettingsTab> = {
        connections: "连接", ingest: "入库", automation: "自动化",
        system: "系统", logs: "日志", about: "关于"
      };
      setSettingsTab(sectionToTab[params.section ?? ""] ?? "连接");
      if (!sectionToTab[params.section ?? ""]) navigate("/settings/connections", { replace: true });
    }
    if (pathname === "/discover" && urlSearchParams.get("q")) {
      setDrawer("discovery");
    } else if (drawer === "discovery" && !pathname.startsWith("/discover")) {
      setDrawer(null);
    }
  }, [pathname, params.section, urlSearchParams]);

  useEffect(() => {
    const titles: Array<[boolean, string]> = [
      [pathname.startsWith("/tasks"), "任务 · Moji"],
      [pathname.startsWith("/performers"), "演员 · Moji"],
      [pathname.startsWith("/discover"), "发现 · Moji"],
      [pathname.startsWith("/settings"), "设置 · Moji"],
      [pathname === "/stats", "统计 · Moji"],
      [pathname.startsWith("/help"), "帮助 · Moji"]
    ];
    document.title = titles.find(([matches]) => matches)?.[1] ?? "Moji";
  }, [pathname]);

  // ── Data hooks ──────────────────────────────────────────────────────
  const { toasts, pushToast, dismissToast, copyText } = useToast();
  const theme = useTheme();
  const { renderedDrawer, drawerClosing, visibleDrawer } = useDrawerTransition(drawer);

  const deferredDiscoveryQuery = useDeferredValue(submittedDiscoveryQuery.trim());
  const deferredSubscriptionSearch = useDeferredValue(subscriptionSearch.trim());
  const deferredPerformerSceneSearch = useDeferredValue(performerSceneSearch.trim());

  const {
    data,
    error,
    fetching: dashboardFetching,
    refreshDashboard,
    addTorrent,
    deleteTask,
    retryTask,
    syncTaskProgress,
    triggerTaskStashScan,
    triggerStashScans
  } = useDashboard();

  const [{ data: taskDetailData, fetching: taskDetailFetching, error: taskDetailError }, refreshTaskDetail] = useQuery<
    TaskDetailDocumentQuery,
    TaskDetailDocumentQueryVariables
  >({
    query: TaskDetailDocumentDocument,
    variables: { id: params.taskId ?? "" },
    pause: !params.taskId,
    requestPolicy: "cache-and-network"
  });

  const discoveryDrawerOpen = drawer === "discovery";

  const {
    result: discoverResult,
    results: discoveredScenes,
    fetching: searchingDiscoverScenes,
    error: discoverScenesError,
    queueDiscoveredScene
  } = useDiscoverScenes(deferredDiscoveryQuery, {
    enabled: discoveryDrawerOpen && discoveryMode === "stashbox",
    sortBy: discoveryStashboxSort
  });

  const {
    results: searchResults,
    fetching: searchingJackett,
    error: searchError
  } = useJackettSearch(deferredDiscoveryQuery, {
    enabled: discoveryDrawerOpen && discoveryMode === "jackett",
    trackers: selectedTrackerIDs,
    sortBy: discoveryJackettSort
  });

  const inspectionCandidateLimit = Number.parseInt(
    String(data?.settings?.automation.torrentSelection.inspectionCandidateLimit ?? 5),
    10
  );

  const {
    results: previewSearchResults,
    previewMeta,
    fetching: previewingJackett,
    error: previewSearchError
  } = usePreviewJackettSelection(deferredDiscoveryQuery, searchResults, {
    enabled: discoveryDrawerOpen && discoveryMode === "jackett",
    applyFastRules: previewFastRules,
    applyFileRules: previewFileRules,
    inspectionCandidateLimit: Number.isNaN(inspectionCandidateLimit) ? 5 : inspectionCandidateLimit
  });

  // Jackett 索引器列表：仅在 Jackett 模式下拉取，避免无谓请求。
  const { indexers: jackettIndexers, fetching: fetchingIndexers } = useJackettIndexers(
    discoveryDrawerOpen && discoveryMode === "jackett"
  );

  const {
    stashPerformerPage,
    stashPerformers,
    performerDetail,
    performerScenePage: performerSceneResultPage,
    performerScenes,
    subscribedPerformers,
    fetchingStashPerformers,
    fetchingPerformerDetail,
    fetchingPerformerScenes,
    fetchingSubscription,
    stashPerformersError,
    performerDetailError,
    performerScenesError,
    subscriptionError,
    refreshingSubscriptionNow,
    queueingPerformerScenes,
    subscribePerformer,
    unsubscribePerformer,
    queuePerformerScenes,
    queueSinglePerformerScene,
    refreshSubscribedPerformer,
    refreshSubscriptionsNow,
    reloadSubscription
  } = useSubscription({
    enabled: tab === "演员",
    search: deferredSubscriptionSearch || null,
    page: subscriptionPage,
    pageSize: subscriptionPageSize,
    performerId: selectedPerformerId,
    performerSceneSearch: deferredPerformerSceneSearch || null,
    performerSceneSource: performerSceneSourceFilter,
    performerSceneLibrary: performerSceneLibraryFilter,
    performerScenePage: performerScenePageIndex,
    performerScenePageSize
  });

  // ── Derived state ───────────────────────────────────────────────────
  const tasks = data?.tasks ?? [];
  const runtimeSettings = data?.settings ?? null;
  const runtimeStatus = data?.settingsStatus ?? null;
  const activeTask = taskDetailData?.task ?? (selectedTaskId ? tasks.find((task) => task.id === selectedTaskId) ?? null : null);
  const confirmDeleteTask = confirmDeleteTaskId ? tasks.find((task) => task.id === confirmDeleteTaskId) ?? null : null;
  const deletePolicy = runtimeSettings?.system.taskDeletePolicy ?? TaskDeletePolicy.KeepOnly;

  const metrics = {
    active: data?.dashboardStats.active ?? 0,
    completed: data?.dashboardStats.completed ?? 0,
    downloading: data?.dashboardStats.downloading ?? 0,
    pendingScans: data?.dashboardStats.pendingScans ?? 0,
    failed: data?.dashboardStats.failed ?? 0,
    total: data?.dashboardStats.total ?? 0
  };

  useEffect(() => {
    writeStoredBoolean(PREVIEW_FAST_RULES_STORAGE_KEY, previewFastRules);
  }, [previewFastRules]);

  useEffect(() => {
    writeStoredBoolean(PREVIEW_FILE_RULES_STORAGE_KEY, previewFileRules);
  }, [previewFileRules]);

  // 当前模式下可见的总条数 + 当前页切片结果。前端按 pageSize 切片，单页请求固定返回 50 条。
  const activeJackettResults = useMemo(() => {
    if (!previewFastRules && !previewFileRules) return searchResults;
    return previewSearchResults.length > 0 ? previewSearchResults : searchResults;
  }, [previewFastRules, previewFileRules, previewSearchResults, searchResults]);

  const visibleResults = useMemo(() => {
    return discoveryMode === "stashbox" ? discoveredScenes : activeJackettResults;
  }, [discoveryMode, discoveredScenes, activeJackettResults]);

  const totalPages = Math.max(1, Math.ceil(visibleResults.length / DISCOVERY_PAGE_SIZE));

  // 切片分别保存为 narrow 类型，避免把 union 数组传给两个不同的 prop。
  const stashboxPagedResults = useMemo(() => {
    if (discoveryMode !== "stashbox") return [] as DiscoverScenesDocumentQuery["discoverScenes"]["items"];
    const start = (discoveryPage - 1) * DISCOVERY_PAGE_SIZE;
    return discoveredScenes.slice(start, start + DISCOVERY_PAGE_SIZE);
  }, [discoveryMode, discoveredScenes, discoveryPage]);

  const jackettPagedResults = useMemo(() => {
    if (discoveryMode !== "jackett") return [] as SearchDocumentQuery["jackettSearch"];
    const start = (discoveryPage - 1) * DISCOVERY_PAGE_SIZE;
    return activeJackettResults.slice(start, start + DISCOVERY_PAGE_SIZE);
  }, [discoveryMode, activeJackettResults, discoveryPage]);

  // ── Effects ─────────────────────────────────────────────────────────
  useEffect(() => {
    setSubscriptionPage(1);
  }, [deferredSubscriptionSearch]);

  useEffect(() => {
    setSubscriptionPage(1);
  }, [subscriptionPageSize]);

  useEffect(() => {
    setPerformerScenePageIndex(1);
  }, [deferredPerformerSceneSearch, performerSceneSourceFilter, performerSceneLibraryFilter, performerScenePageSize, selectedPerformerId]);

  useEffect(() => {
    setSelectedSceneKeys([]);
  }, [selectedPerformerId]);

  // 切 mode / 切 sort / 切 tracker 时回到第 1 页，结果来自同一份缓存不需要重发请求。
  useEffect(() => {
    setDiscoveryPage(1);
  }, [discoveryMode, discoveryStashboxSort, discoveryJackettSort, selectedTrackerIDs, previewFastRules, previewFileRules]);

  // ── Action handlers ─────────────────────────────────────────────────
  const openTaskDetail = (taskId: string) => {
    setSelectedTaskId(taskId);
    navigate(`/tasks/${encodeURIComponent(taskId)}`, { state: { backgroundLocation: location } });
  };

  const openTaskResolution = (taskId: string) => {
    setSelectedTaskId(taskId);
    navigate(`/tasks/${encodeURIComponent(taskId)}/resolve`, { state: { backgroundLocation: location } });
  };

  const handleSourcingResolved = async () => {
    await refreshDashboard({ requestPolicy: "network-only" });
    if (selectedTaskId) navigate(`/tasks/${encodeURIComponent(selectedTaskId)}`);
  };

  const handleSubmitDiscoverSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = discoveryQuery.trim();
    if (trimmed === "") return;
    searchHistory.push(trimmed);
    setSubmittedDiscoveryQuery(trimmed);
    setHistoryVisible(false);
    const next = new URLSearchParams(urlSearchParams);
    next.set("q", trimmed);
    if (discoveryMode !== "stashbox") next.set("source", discoveryMode);
    next.delete("page");
    navigate(`/discover?${next.toString()}`);
  };

  const handlePickHistory = (entry: string) => {
    setDiscoveryQuery(entry);
    setSubmittedDiscoveryQuery(entry);
    setHistoryVisible(false);
    const next = new URLSearchParams(urlSearchParams);
    next.set("q", entry);
    next.delete("page");
    navigate(`/discover?${next.toString()}`);
  };

  const handleRemoveHistory = (entry: string) => {
    searchHistory.remove(entry);
  };

  const handleClearHistory = () => {
    searchHistory.clear();
  };

  const handleToggleTracker = (id: string) => {
    const trackers = selectedTrackerIDs.includes(id) ? selectedTrackerIDs.filter((entry) => entry !== id) : [...selectedTrackerIDs, id];
    updateDiscoverURL({ trackers, page: 1 });
  };

  const handleClearTrackers = () => {
    updateDiscoverURL({ trackers: [], page: 1 });
  };

  const handleSwitchMode = (next: DiscoveryMode) => {
    setDiscoveryMode(next);
    const search = new URLSearchParams(urlSearchParams);
    if (next === "stashbox") search.delete("source"); else search.set("source", next);
    search.delete("sort");
    search.delete("page");
    setURLSearchParams(search);
  };

  const handleChangeStashboxSort = (next: DiscoverSortBy) => {
    setDiscoveryStashboxSort(next);
    updateDiscoverURL({ sort: String(next).toLowerCase().replaceAll("_", "-"), page: 1 });
  };

  const handleChangeJackettSort = (next: JackettSortBy) => {
    setDiscoveryJackettSort(next);
    updateDiscoverURL({ sort: String(next).toLowerCase().replaceAll("_", "-"), page: 1 });
  };

  const handlePrevPage = () => {
    updateDiscoverURL({ page: Math.max(1, discoveryPage - 1) });
  };

  const handleNextPage = () => {
    updateDiscoverURL({ page: Math.min(totalPages, discoveryPage + 1) });
  };

  const handleSwitchToJackettFromEmpty = () => {
    handleSwitchMode("jackett");
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
    setPendingTaskScanId(taskId);
    try {
      await triggerTaskStashScan({ id: taskId });
      await refreshDashboard({ requestPolicy: "network-only" });
    } finally {
      setPendingTaskScanId(null);
    }
  };

  const runRetryTask = async (taskId: string) => {
    const task = tasks.find((entry) => entry.id === taskId);
    setPendingTaskRetryId(taskId);
    try {
      const response = await retryTask({ id: taskId });
      if (response.error) {
        pushToast("tone-danger", describeQueryError(response.error));
        return;
      }
      if (!response.data?.retryTask?.id) {
        pushToast("tone-danger", "任务重试失败，后端没有返回任务记录。");
        return;
      }
      pushToast("tone-success", `已重试任务：${task ? taskSummary(task) : taskId}。`);
    } finally {
      await refreshDashboard({ requestPolicy: "network-only" });
      setPendingTaskRetryId(null);
    }
  };

  const runRetryBlockedTasks = async () => {
    const blockedTasks = tasks.filter((task) => task.stageStatus === "BLOCKED");
    if (blockedTasks.length === 0) {
      pushToast("tone-info", "当前没有受阻任务需要重试。");
      return;
    }

    setRetryingBlockedTasks(true);
    let succeeded = 0;
    let failed = 0;
    try {
      for (const task of blockedTasks) {
        setPendingTaskRetryId(task.id);
        const response = await retryTask({ id: task.id });
        if (response.error || !response.data?.retryTask?.id) {
          failed += 1;
        } else {
          succeeded += 1;
        }
      }

      if (failed === 0) {
        pushToast("tone-success", `已重试 ${succeeded} 个受阻任务。`);
      } else {
        pushToast("tone-info", `受阻任务重试完成：成功 ${succeeded} 个，失败 ${failed} 个。`);
      }
    } finally {
      setPendingTaskRetryId(null);
      await refreshDashboard({ requestPolicy: "network-only" });
      setRetryingBlockedTasks(false);
    }
  };

  const runDeleteTask = async (taskId: string) => {
    const task = tasks.find((entry) => entry.id === taskId);
    setPendingTaskDeleteId(taskId);

    try {
      const response = await deleteTask({ id: taskId });
      if (response.error) {
        pushToast("tone-danger", describeQueryError(response.error));
        return;
      }

      if (!response.data?.deleteTask?.id) {
        pushToast("tone-danger", "任务删除失败，后端没有返回已删除的任务记录。");
        return;
      }

      if (selectedTaskId === taskId) {
        setSelectedTaskId(null);
        setDrawer(null);
        navigate("/tasks", { replace: true });
      }

      setConfirmDeleteTaskId(null);
      pushToast("tone-success", `已删除任务：${task ? taskSummary(task) : taskId}。`);
      await refreshDashboard({ requestPolicy: "network-only" });
    } finally {
      setPendingTaskDeleteId(null);
    }
  };

  const requestDeleteTask = (taskId: string) => {
    setConfirmDeleteTaskId(taskId);
  };

  const handleAddSearchResult = async (result: SearchDocumentQuery["jackettSearch"][number]) => {
    const taskURL = result.magnetUri || result.link;
    setPendingAddId(result.link);

    try {
      const response = await addTorrent({
        input: {
          url: taskURL
        }
      });

      if (response.error) {
        pushToast("tone-danger", describeQueryError(response.error));
        return;
      }

      if (!response.data?.addTorrent?.id) {
        pushToast("tone-danger", "任务创建失败，后端没有返回新的任务记录。");
        return;
      }

      setSelectedTaskId(response.data.addTorrent.id);
      await refreshDashboard({ requestPolicy: "network-only" });
      navigate(`/tasks/${encodeURIComponent(response.data.addTorrent.id)}`);
    } finally {
      setPendingAddId(null);
    }
  };

  const handleQueueDiscoveredScene = async (
    result: DiscoverScenesDocumentQuery["discoverScenes"]["items"][number]
  ) => {
    setPendingAddId(result.key);

    try {
      const response = await queueDiscoveredScene({
        input: {
          sceneId: result.sceneId,
          stashBoxEndpoint: result.stashBoxEndpoint
        }
      });

      if (response.error) {
        pushToast("tone-danger", describeQueryError(response.error));
        return;
      }

      if (!response.data?.queueDiscoveredScene?.id) {
        pushToast("tone-danger", "任务创建失败，后端没有返回新的任务记录。");
        return;
      }

      pushToast("tone-success", `已将 ${result.title} 加入任务队列。`);
      setSelectedTaskId(response.data.queueDiscoveredScene.id);
      await refreshDashboard({ requestPolicy: "network-only" });
      navigate(`/tasks/${encodeURIComponent(response.data.queueDiscoveredScene.id)}`);
    } finally {
      setPendingAddId(null);
    }
  };

  const handleSubscriptionToggle = async (performer: { id: string; name: string; subscribed: boolean }) => {
    setPendingSubscriptionID(performer.id);

    const result = performer.subscribed
      ? await unsubscribePerformer({ stashPerformerID: performer.id })
      : await subscribePerformer({ stashPerformerID: performer.id });

    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      setPendingSubscriptionID(null);
      return;
    }

    pushToast(
      "tone-success",
      performer.subscribed
        ? `已取消订阅 ${performer.name}。`
        : `已订阅 ${performer.name}，Moji 会通过 custom_fields 记录状态。`
    );
    await reloadSubscription();
    setPendingSubscriptionID(null);
  };

  const handleRefreshSubscribedPerformer = async (performer: { id: string; name: string }) => {
    setPendingSubscriptionID(performer.id);

    const result = await refreshSubscribedPerformer({ stashPerformerID: performer.id });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      setPendingSubscriptionID(null);
      return;
    }

    pushToast("tone-info", `已检查 ${performer.name} 的最新发行信息。`);
    await reloadSubscription();
    setPendingSubscriptionID(null);
  };

  const handleRefreshAllSubscription = async () => {
    const result = await refreshSubscriptionsNow({});
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }

    pushToast("tone-info", "已触发全部演员的更新检查。");
    await reloadSubscription();
  };

  const handleOpenPerformer = (performerId: string) => {
    setSelectedPerformerId(performerId);
    navigate(`/performers/${encodeURIComponent(performerId)}?${urlSearchParams.toString()}`, { state: { backgroundLocation: location } });
  };

  const handleBackToPerformers = () => {
    setSelectedPerformerId(null);
    setPerformerSceneSearch("");
    setPerformerSceneSourceFilter(SceneSourceFilter.All);
    setPerformerSceneLibraryFilter(LibraryFilter.All);
    setPerformerScenePageIndex(1);
    setPerformerScenePageSize(24);
    setSelectedSceneKeys([]);
    navigate(`/performers?${urlSearchParams.toString()}`);
  };

  const handleToggleSceneSelection = (key: string) => {
    setSelectedSceneKeys((current) =>
      current.includes(key) ? current.filter((item) => item !== key) : [...current, key]
    );
  };

  const handleSelectCurrentScenePage = (keys: string[]) => {
    setSelectedSceneKeys((current) => Array.from(new Set([...current, ...keys])));
  };

  const handleClearSceneSelection = () => {
    setSelectedSceneKeys([]);
  };

  const handleQueueSelectedScenes = async () => {
    if (!selectedPerformerId || selectedSceneKeys.length === 0) return;

    const selectedScenes = performerScenes.filter((scene) => selectedSceneKeys.includes(scene.key));
    if (selectedScenes.length === 0) {
      pushToast("tone-danger", "当前没有可提交的已选作品，请刷新列表后重试。");
      return;
    }

    const result = await queuePerformerScenes({
      input: {
        performerId: selectedPerformerId,
        scenes: selectedScenes.map((scene) => ({
          key: scene.key,
          sourceSceneId: scene.sourceSceneId,
          stashBoxSceneId: scene.stashBoxSceneId ?? undefined,
          stashBoxEndpoint: scene.stashBoxEndpoint ?? undefined,
          code: scene.code ?? undefined,
          title: scene.title ?? undefined,
          inLibrary: scene.inLibrary
        }))
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }

    const payload = result.data?.queuePerformerScenes;
    if (!payload) {
      pushToast("tone-danger", "批量下载失败，后端没有返回结果。");
      return;
    }

    const { summary, results } = payload;
    if (summary.queuedCount > 0 && summary.skippedCount === 0 && summary.failedCount === 0) {
      pushToast("tone-success", `已创建 ${summary.queuedCount} 个下载任务。`);
    } else if (summary.queuedCount > 0) {
      pushToast("tone-info", `已创建 ${summary.queuedCount} 个任务，跳过 ${summary.skippedCount} 个，失败 ${summary.failedCount} 个。`);
    } else if (summary.skippedCount > 0 && summary.failedCount === 0) {
      pushToast("tone-info", "没有创建新任务，所选影片均被跳过。");
    } else {
      pushToast("tone-danger", `没有创建新任务，失败 ${summary.failedCount} 个。`);
    }

    const queuedKeys = new Set(results.filter((item) => item.status === "QUEUED").map((item) => item.key));
    if (queuedKeys.size > 0) {
      setSelectedSceneKeys((current) => current.filter((key) => !queuedKeys.has(key)));
    }

    await reloadSubscription();
  };

  const handleQueueSinglePerformerScene = async (scene: (typeof performerScenes)[number]) => {
    if (!selectedPerformerId || scene.inLibrary || scene.mojiTask || pendingPerformerSceneKeys.includes(scene.key)) return;

    setPendingPerformerSceneKeys((current) => [...current, scene.key]);
    try {
      const result = await queueSinglePerformerScene({
        input: {
          performerId: selectedPerformerId,
          scenes: [{
            key: scene.key,
            sourceSceneId: scene.sourceSceneId,
            stashBoxSceneId: scene.stashBoxSceneId ?? undefined,
            stashBoxEndpoint: scene.stashBoxEndpoint ?? undefined,
            code: scene.code ?? undefined,
            title: scene.title ?? undefined,
            inLibrary: scene.inLibrary
          }]
        }
      });
      if (result.error) {
        pushToast("tone-danger", describeQueryError(result.error));
        return;
      }

      const item = result.data?.queuePerformerScenes.results[0];
      if (!item) {
        pushToast("tone-danger", "创建任务失败，后端没有返回结果。");
        return;
      }
      if (item.status === "QUEUED") {
        pushToast("tone-success", `已为 ${item.resolvedCode || scene.code || scene.title || "影片"} 创建下载任务。`);
        setSelectedSceneKeys((current) => current.filter((key) => key !== scene.key));
      } else if (item.status === "SKIPPED") {
        pushToast("tone-info", item.message);
        setSelectedSceneKeys((current) => current.filter((key) => key !== scene.key));
      } else {
        pushToast("tone-danger", item.message);
      }
      await reloadSubscription();
    } finally {
      setPendingPerformerSceneKeys((current) => current.filter((key) => key !== scene.key));
    }
  };

  const handleToggleGroup = (group: TaskGroupKey) => {
    setTaskGroupOpen((current) => ({ ...current, [group]: !current[group] }));
  };

  const updatePerformerURL = (patch: Partial<ReturnType<typeof parsePerformerSearchParams>>) => {
    const current = parsePerformerSearchParams(urlSearchParams);
    const next = serializePerformerSearchParams({ ...current, ...patch });
    const base = params.performerId ? `/performers/${encodeURIComponent(params.performerId)}` : "/performers";
    navigate(`${base}${next.size ? `?${next}` : ""}`, { replace: true });
  };

  const updateDiscoverURL = (patch: Partial<ReturnType<typeof parseDiscoverSearchParams>>) => {
    const current = parseDiscoverSearchParams(urlSearchParams);
    const next = serializeDiscoverSearchParams({ ...current, ...patch });
    navigate(`/discover${next.size ? `?${next}` : ""}`, { replace: true });
  };

  const routeDrawer: DrawerKey = params.taskId
    ? pathname.endsWith("/resolve") ? "task-resolution" : "task"
    : null;
  const displayedDrawer = routeDrawer ?? visibleDrawer;
  const closeDisplayedDrawer = () => {
    if (routeDrawer) {
      const background = (location.state as { backgroundLocation?: { pathname: string; search?: string } } | null)?.backgroundLocation;
      navigate(background ? `${background.pathname}${background.search ?? ""}` : "/tasks");
      return;
    }
    setDrawer(null);
  };

  // ── Drawer metadata ─────────────────────────────────────────────────
  const drawerTitle = (() => {
    if (displayedDrawer === "stats") return "运行概览";
    if (displayedDrawer === "settings") return "配置与系统";
    if (displayedDrawer === "help") return "Markdown 帮助";
    if (displayedDrawer === "discovery") return discoveryMode === "stashbox" ? "StashBox 搜索结果" : "Jackett 搜索结果";
    if (displayedDrawer === "task-resolution") return activeTask ? `人工处理：${taskSummary(activeTask)}` : "人工处理受阻任务";
    return activeTask ? taskSummary(activeTask) : "任务详情";
  })();

  // ── Render ──────────────────────────────────────────────────────────
  return (
    <Suspense fallback={<RouteFallback />}>
      <div className="app-shell">
      <ToastStack toasts={toasts} onDismiss={dismissToast} />
      <div className="ambient ambient-a" />
      <div className="ambient ambient-b" />
      <div className="ambient ambient-c" />

      <Header onOpenHelp={() => setDrawer("help")} theme={theme} />

      {error ? (
        <section className="surface surface--alert">
          <div>
            <h2>{data ? "GraphQL 返回错误" : "GraphQL 当前不可用"}</h2>
            <p>{describeQueryError(error)}</p>
          </div>
          <button
            type="button"
            className="ghost-button"
            onClick={() => void refreshDashboard({ requestPolicy: "network-only" })}
            disabled={dashboardFetching}
          >
            {dashboardFetching ? "重试中..." : "重试"}
          </button>
        </section>
      ) : null}

      <main className="content">
        {pathname === "/" ? (
          <HomePage
            tasks={tasks}
            runtimeSettings={runtimeSettings}
            runtimeStatus={runtimeStatus}
            pendingTaskScanId={pendingTaskScanId}
            pendingTaskRetryId={pendingTaskRetryId}
            onOpenTask={openTaskDetail}
            onScanTask={(id) => void runTaskScan(id)}
            onRetryTask={(id) => void runRetryTask(id)}
            onResolveTask={openTaskResolution}
            onOpenSettings={openSettings}
          />
        ) : null}

        {tab === "任务" ? (
          <TasksPage
            tasks={tasks}
            metrics={{
              active: metrics.active,
              completed: metrics.completed,
              pendingScans: metrics.pendingScans,
              failed: metrics.failed
            }}
            taskSearch={taskSearch}
            taskStatus={taskStatus}
            taskSort={taskSort}
            taskGroupOpen={taskGroupOpen}
            pendingTaskScanId={pendingTaskScanId}
            pendingTaskRetryId={pendingTaskRetryId}
            pendingTaskDeleteId={pendingTaskDeleteId}
            retryingBlockedTasks={retryingBlockedTasks}
            onSearchChange={(q) => setURLSearchParams(serializeTaskSearchParams({ q, status: taskStatus, sort: taskSort }))}
            onStatusChange={(status) => setURLSearchParams(serializeTaskSearchParams({ q: taskSearch, status, sort: taskSort }))}
            onSortChange={(sort) => setURLSearchParams(serializeTaskSearchParams({ q: taskSearch, status: taskStatus, sort }))}
            onToggleGroup={handleToggleGroup}
            onRefresh={() => void refreshDashboard({ requestPolicy: "network-only" })}
            onSync={() => void runSync()}
            onScanAll={() => void runScan()}
            onOpenTask={openTaskDetail}
            onScanTask={(id) => void runTaskScan(id)}
            onRetryTask={(id) => void runRetryTask(id)}
            onResolveTask={openTaskResolution}
            onRetryBlockedTasks={() => void runRetryBlockedTasks()}
            onDeleteTask={requestDeleteTask}
          />
        ) : null}

        {tab === "发现" ? (
          <DiscoveryPage
            query={discoveryQuery}
            searching={discoveryMode === "stashbox" ? searchingDiscoverScenes : searchingJackett}
            inputFocused={discoveryInputFocused}
            mode={discoveryMode}
            history={searchHistory.history}
            historyVisible={historyVisible}
            onQueryChange={(value) => {
              setDiscoveryQuery(value);
              if (value.trim() === "") {
                setHistoryVisible(discoveryInputFocused);
              } else {
                setHistoryVisible(false);
              }
            }}
            onInputFocus={() => {
              setDiscoveryInputFocused(true);
              setHistoryVisible(discoveryQuery.trim() === "");
            }}
            onInputBlur={() => {
              setDiscoveryInputFocused(false);
              setHistoryVisible(false);
            }}
            onSubmit={handleSubmitDiscoverSearch}
            onModeChange={handleSwitchMode}
            onPickHistory={handlePickHistory}
            onRemoveHistory={handleRemoveHistory}
            onClearHistory={handleClearHistory}
            onDismissHistory={() => setHistoryVisible(false)}
            onOpenHelp={() => setDrawer("help")}
          />
        ) : null}

        {tab === "演员" ? (
          <SubscriptionPage
            runtimeSettings={runtimeSettings}
            stashPerformerPage={stashPerformerPage}
            stashPerformers={stashPerformers}
            subscribedPerformers={subscribedPerformers}
            fetchingStashPerformers={fetchingStashPerformers}
            fetchingSubscription={fetchingSubscription}
            performerDetail={performerDetail}
            performerScenePage={performerSceneResultPage}
            performerScenes={performerScenes}
            fetchingPerformerDetail={fetchingPerformerDetail}
            fetchingPerformerScenes={fetchingPerformerScenes}
            refreshingSubscriptionNow={refreshingSubscriptionNow}
            queueingPerformerScenes={queueingPerformerScenes}
            subscriptionSearch={subscriptionSearch}
            subscriptionPageSize={subscriptionPageSize}
            selectedPerformerId={selectedPerformerId}
            performerSceneSearch={performerSceneSearch}
            performerSceneSourceFilter={performerSceneSourceFilter}
            performerSceneLibraryFilter={performerSceneLibraryFilter}
            performerScenePageSize={performerScenePageSize}
            selectedSceneKeys={selectedSceneKeys}
            pendingSceneKeys={pendingPerformerSceneKeys}
            pendingSubscriptionID={pendingSubscriptionID}
            subscriptionError={subscriptionError ?? null}
            stashPerformersError={stashPerformersError ?? null}
            performerDetailError={performerDetailError ?? null}
            performerScenesError={performerScenesError ?? null}
            onSearchChange={(q) => updatePerformerURL({ q, page: 1 })}
            onPageSizeChange={(pageSize) => updatePerformerURL({ pageSize, page: 1 })}
            onReload={() => void reloadSubscription()}
            onRefreshAll={() => void handleRefreshAllSubscription()}
            onToggle={(performer) => void handleSubscriptionToggle(performer)}
            onRefreshOne={(performer) => void handleRefreshSubscribedPerformer(performer)}
            onPrevPage={() => updatePerformerURL({ page: Math.max(1, subscriptionPage - 1) })}
            onNextPage={() => updatePerformerURL({ page: subscriptionPage + 1 })}
            onOpenPerformer={handleOpenPerformer}
            onOpenTask={openTaskDetail}
            onBackToList={handleBackToPerformers}
            onPerformerSceneSearchChange={(sceneQ) => updatePerformerURL({ sceneQ, scenePage: 1 })}
            onPerformerSceneSourceChange={(source) => updatePerformerURL({ source: source === SceneSourceFilter.Stash ? "stash" : source === SceneSourceFilter.Stashbox ? "stashbox" : "all", scenePage: 1 })}
            onPerformerSceneLibraryChange={(library) => updatePerformerURL({ library: library === LibraryFilter.InLibrary ? "in-library" : library === LibraryFilter.NotInLibrary ? "not-in-library" : "all", scenePage: 1 })}
            onPerformerScenePageSizeChange={(scenePageSize) => updatePerformerURL({ scenePageSize, scenePage: 1 })}
            onPrevPerformerScenePage={() => updatePerformerURL({ scenePage: Math.max(1, performerScenePageIndex - 1) })}
            onNextPerformerScenePage={() => updatePerformerURL({ scenePage: performerScenePageIndex + 1 })}
            onToggleSceneSelection={handleToggleSceneSelection}
            onSelectCurrentScenePage={handleSelectCurrentScenePage}
            onClearSceneSelection={handleClearSceneSelection}
            onQueueSelectedScenes={() => void handleQueueSelectedScenes()}
            onQueueScene={(scene) => void handleQueueSinglePerformerScene(scene)}
          />
        ) : null}

        {pathname.startsWith("/settings/") ? (
          <section className="section-band">
            <div className="band-head"><div><h2 tabIndex={-1}>配置与系统</h2></div></div>
            <SettingsDrawer
              settingsTab={settingsTab}
              onSettingsTabChange={(next) => openSettings(next)}
              runtimeSettings={runtimeSettings}
              runtimeStatus={runtimeStatus}
              appVersion={data?.version ?? ""}
              drawer="settings"
              renderedDrawer="settings"
              pushToast={pushToast}
              refreshDashboard={refreshDashboard}
            />
          </section>
        ) : null}

        {pathname === "/stats" ? (
          <section className="section-band">
            <div className="band-head"><div><h2 tabIndex={-1}>运行概览</h2></div></div>
            <StatsDrawer active={metrics.active} completed={metrics.completed} pendingScans={metrics.pendingScans} failed={metrics.failed} />
          </section>
        ) : null}

      </main>

      {displayedDrawer ? (
        <Drawer
          visibleDrawer={displayedDrawer}
          closing={routeDrawer ? false : drawerClosing}
          title={drawerTitle}
          onClose={closeDisplayedDrawer}
        >
          {displayedDrawer === "stats" ? (
            <StatsDrawer
              active={metrics.active}
              completed={metrics.completed}
              pendingScans={metrics.pendingScans}
              failed={metrics.failed}
            />
          ) : null}

          {displayedDrawer === "settings" ? (
            <SettingsDrawer
              settingsTab={settingsTab}
              onSettingsTabChange={setSettingsTab}
              runtimeSettings={runtimeSettings}
              runtimeStatus={runtimeStatus}
              appVersion={data?.version ?? ""}
              drawer={drawer}
              renderedDrawer={renderedDrawer}
              pushToast={pushToast}
              refreshDashboard={refreshDashboard}
            />
          ) : null}

          {displayedDrawer === "help" ? (
            <HelpDrawer topicId={helpTopicId} onTopicChange={setHelpTopicId} />
          ) : null}

          {displayedDrawer === "task" ? (
            taskDetailFetching && !activeTask ? <p>正在加载任务...</p> : taskDetailError ? (
              <div className="empty-card"><h3>任务加载失败</h3><p>{describeQueryError(taskDetailError)}</p><button type="button" onClick={() => refreshTaskDetail({ requestPolicy: "network-only" })}>重试</button></div>
            ) : !activeTask ? (
              <div className="empty-card"><h3>任务不存在</h3><p>该任务可能已被删除。</p></div>
            ) : <TaskDrawer
              task={activeTask}
              pendingScan={activeTask ? pendingTaskScanId === activeTask.id : false}
              pendingRetry={activeTask ? pendingTaskRetryId === activeTask.id : false}
              pendingDelete={activeTask ? pendingTaskDeleteId === activeTask.id : false}
              onCopy={copyText}
              onSyncAll={() => void runSync()}
              onScanTask={(id) => void runTaskScan(id)}
              onRetryTask={(id) => void runRetryTask(id)}
              onScanAll={() => void runScan()}
              onDeleteTask={requestDeleteTask}
            />
          ) : null}

          {displayedDrawer === "task-resolution" ? (
            <SourcingResolutionDrawer task={activeTask} onResolved={handleSourcingResolved} />
          ) : null}

          {displayedDrawer === "discovery" ? (
            <div className="drawer-stack">
              <article className="drawer-card">
                <div className="drawer-card__head">
                  <div>
                    <h3>{deferredDiscoveryQuery ? `搜索词：${deferredDiscoveryQuery}` : "未提供搜索词"}</h3>
                  </div>
                  {discoveryMode === "stashbox" && discoverResult?.usedStashBox ? (
                    <span className="status-chip tone-info">{discoverResult.usedStashBox.name}</span>
                  ) : null}
                  {discoveryMode === "jackett" && selectedTrackerIDs.length > 0 ? (
                    <span className="status-chip tone-info">已应用 {selectedTrackerIDs.length} 个过滤</span>
                  ) : null}
                </div>

                {discoveryMode === "jackett" && (
                  <JackettFilterPanel
                    indexers={jackettIndexers}
                    fetching={fetchingIndexers}
                    enabledIds={selectedTrackerIDs}
                    onToggle={handleToggleTracker}
                    onClear={handleClearTrackers}
                  />
                )}

                <SortAndPagination
                  sortValue={discoveryMode === "stashbox" ? discoveryStashboxSort : discoveryJackettSort}
                  sortOptions={discoveryMode === "stashbox" ? DISCOVER_SORT_OPTIONS : JACKETT_SORT_OPTIONS}
                  onSortChange={(value) =>
                    discoveryMode === "stashbox"
                      ? handleChangeStashboxSort(value as DiscoverSortBy)
                      : handleChangeJackettSort(value as JackettSortBy)
                  }
                  page={discoveryPage}
                  totalPages={totalPages}
                  total={visibleResults.length}
                  onPrevPage={handlePrevPage}
                  onNextPage={handleNextPage}
                  extraContent={discoveryMode === "jackett" ? (
                    <div className="discovery-toolbar__preview">
                      <label className="switch-row">
                        <span className="switch-row__label">快速规则预览</span>
                        <span className="switch" role="switch" aria-checked={previewFastRules}>
                          <input
                            type="checkbox"
                            checked={previewFastRules}
                            onChange={(event) => updateDiscoverURL({ fastRules: event.target.checked, page: 1 })}
                          />
                          <span className="switch__track" aria-hidden="true" />
                          <span className="switch__thumb" aria-hidden="true" />
                        </span>
                      </label>
                      <label className="switch-row">
                        <span className="switch-row__label">文件结构规则预览</span>
                        <span className="switch" role="switch" aria-checked={previewFileRules}>
                          <input
                            type="checkbox"
                            checked={previewFileRules}
                            onChange={(event) => updateDiscoverURL({ fileRules: event.target.checked, page: 1 })}
                          />
                          <span className="switch__track" aria-hidden="true" />
                          <span className="switch__thumb" aria-hidden="true" />
                        </span>
                      </label>
                      {previewFileRules ? (
                        <span className="discovery-toolbar__preview-note">
                          {previewingJackett
                            ? "正在检查文件结构..."
                            : `已检查 ${previewMeta?.inspectedCount ?? 0} / ${previewMeta?.inspectableCount ?? 0} 条可检查候选`}
                        </span>
                      ) : (
                        <span className="discovery-toolbar__preview-note">仅影响结果展示。</span>
                      )}
                    </div>
                  ) : null}
                />

                <DiscoveryDrawer
                  mode={discoveryMode}
                  query={deferredDiscoveryQuery}
                  searching={discoveryMode === "stashbox" ? searchingDiscoverScenes : (searchingJackett || previewingJackett)}
                  error={(discoveryMode === "stashbox" ? discoverScenesError : (previewSearchError ?? searchError)) ?? null}
                  pendingAddId={pendingAddId}
                  discoverResult={discoverResult}
                  discoverItems={stashboxPagedResults}
                  jackettItems={jackettPagedResults}
                  hasAnyResults={visibleResults.length > 0}
                  usedStashBoxName={discoverResult?.usedStashBox?.name ?? null}
                  onQueueDiscovered={(result) => void handleQueueDiscoveredScene(result)}
                  onAddJackett={(result) => void handleAddSearchResult(result)}
                  onTryJackett={handleSwitchToJackettFromEmpty}
                  onClearTrackers={handleClearTrackers}
                  hasActiveTrackers={selectedTrackerIDs.length > 0}
                />
              </article>
            </div>
          ) : null}
        </Drawer>
      ) : null}

      {confirmDeleteTaskId ? (
        <Drawer
          visibleDrawer="confirm"
          closing={false}
          title="删除确认"
          onClose={() => {
            if (!pendingTaskDeleteId) {
              setConfirmDeleteTaskId(null);
            }
          }}
        >
          <ConfirmDeleteDrawer
            taskLabel={confirmDeleteTask ? taskSummary(confirmDeleteTask) : confirmDeleteTaskId}
            destructive={deletePolicy === TaskDeletePolicy.RemoveTorrentAndFiles}
            pending={pendingTaskDeleteId === confirmDeleteTaskId}
            onConfirm={() => void runDeleteTask(confirmDeleteTaskId)}
            onCancel={() => {
              if (!pendingTaskDeleteId) {
                setConfirmDeleteTaskId(null);
              }
            }}
          />
        </Drawer>
      ) : null}

      </div>
    </Suspense>
  );
}

export { App };
