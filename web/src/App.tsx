import { FormEvent, useCallback, useDeferredValue, useEffect, useMemo, useState } from "react";
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
import type { DrawerKey, SettingsTab, TabKey } from "./types";
import { Drawer, Header, ToastStack } from "./components/layout";
import { ConfirmDeleteDrawer, DiscoveryDrawer, HelpDrawer, SettingsDrawer, StatsDrawer, TaskDrawer } from "./components/drawers";
import { JackettFilterPanel } from "./components/drawers/JackettFilterPanel";
import { SortAndPagination } from "./components/drawers/SortAndPagination";
import {
  DiscoveryPage,
  HomePage,
  SubscriptionPage,
  TasksPage
} from "./pages";
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
  TaskDeletePolicy
} from "./graphql/generated/graphql";

const PREVIEW_FAST_RULES_STORAGE_KEY = "moji.discovery.previewFastRules";
const PREVIEW_FILE_RULES_STORAGE_KEY = "moji.discovery.previewFileRules";

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
  // ── UI state ────────────────────────────────────────────────────────
  const [tab, setTab] = useState<TabKey>("主页");
  const [drawer, setDrawer] = useState<DrawerKey>(null);
  const [settingsTab, setSettingsTab] = useState<SettingsTab>("连接");
  const openSettings = useCallback((tab: SettingsTab) => {
    setSettingsTab(tab);
    setDrawer("settings");
  }, []);
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
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [pendingTaskScanId, setPendingTaskScanId] = useState<string | null>(null);
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
  const [selectedPerformerId, setSelectedPerformerId] = useState<string | null>(null);
  const [performerSceneSearch, setPerformerSceneSearch] = useState("");
  const [performerSceneSourceFilter, setPerformerSceneSourceFilter] = useState<SceneSourceFilter>(SceneSourceFilter.All);
  const [performerSceneLibraryFilter, setPerformerSceneLibraryFilter] = useState<LibraryFilter>(LibraryFilter.All);
  const [performerScenePageIndex, setPerformerScenePageIndex] = useState(1);
  const [performerScenePageSize, setPerformerScenePageSize] = useState<number>(24);
  const [selectedSceneKeys, setSelectedSceneKeys] = useState<string[]>([]);

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
    syncTaskProgress,
    triggerTaskStashScan,
    triggerStashScans
  } = useDashboard();

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
  const activeTask = selectedTaskId ? tasks.find((task) => task.id === selectedTaskId) ?? null : null;
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
    setDrawer("task");
  };

  const handleSubmitDiscoverSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = discoveryQuery.trim();
    if (trimmed === "") return;
    searchHistory.push(trimmed);
    setSubmittedDiscoveryQuery(trimmed);
    setHistoryVisible(false);
    setDrawer("discovery");
    setTab("发现");
  };

  const handlePickHistory = (entry: string) => {
    setDiscoveryQuery(entry);
    setSubmittedDiscoveryQuery(entry);
    setHistoryVisible(false);
    setDrawer("discovery");
    setTab("发现");
  };

  const handleRemoveHistory = (entry: string) => {
    searchHistory.remove(entry);
  };

  const handleClearHistory = () => {
    searchHistory.clear();
  };

  const handleToggleTracker = (id: string) => {
    setSelectedTrackerIDs((current) =>
      current.includes(id) ? current.filter((entry) => entry !== id) : [...current, id]
    );
  };

  const handleClearTrackers = () => {
    setSelectedTrackerIDs([]);
  };

  const handleSwitchMode = (next: DiscoveryMode) => {
    setDiscoveryMode(next);
  };

  const handleChangeStashboxSort = (next: DiscoverSortBy) => {
    setDiscoveryStashboxSort(next);
  };

  const handleChangeJackettSort = (next: JackettSortBy) => {
    setDiscoveryJackettSort(next);
  };

  const handlePrevPage = () => {
    setDiscoveryPage((current) => Math.max(1, current - 1));
  };

  const handleNextPage = () => {
    setDiscoveryPage((current) => Math.min(totalPages, current + 1));
  };

  const handleSwitchToJackettFromEmpty = () => {
    setDiscoveryMode("jackett");
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
      setDrawer("task");
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
      setDrawer("task");
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
  };

  const handleBackToPerformers = () => {
    setSelectedPerformerId(null);
    setPerformerSceneSearch("");
    setPerformerSceneSourceFilter(SceneSourceFilter.All);
    setPerformerSceneLibraryFilter(LibraryFilter.All);
    setPerformerScenePageIndex(1);
    setPerformerScenePageSize(24);
    setSelectedSceneKeys([]);
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

  const handleToggleGroup = (group: TaskGroupKey) => {
    setTaskGroupOpen((current) => ({ ...current, [group]: !current[group] }));
  };

  // ── Drawer metadata ─────────────────────────────────────────────────
  const drawerTitle = (() => {
    if (visibleDrawer === "stats") return "运行概览";
    if (visibleDrawer === "settings") return "配置与系统";
    if (visibleDrawer === "help") return "Markdown 帮助";
    if (visibleDrawer === "discovery") return discoveryMode === "stashbox" ? "StashBox 搜索结果" : "Jackett 搜索结果";
    return activeTask ? taskSummary(activeTask) : "任务详情";
  })();

  // ── Render ──────────────────────────────────────────────────────────
  return (
    <div className="app-shell">
      <ToastStack toasts={toasts} onDismiss={dismissToast} />
      <div className="ambient ambient-a" />
      <div className="ambient ambient-b" />
      <div className="ambient ambient-c" />

      <Header tab={tab} onTabChange={setTab} onOpenDrawer={setDrawer} theme={theme} />

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
        {tab === "主页" ? (
          <HomePage
            tasks={tasks}
            runtimeSettings={runtimeSettings}
            runtimeStatus={runtimeStatus}
            pendingTaskScanId={pendingTaskScanId}
            onOpenTask={openTaskDetail}
            onScanTask={(id) => void runTaskScan(id)}
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
            pendingTaskDeleteId={pendingTaskDeleteId}
            onSearchChange={setTaskSearch}
            onStatusChange={setTaskStatus}
            onSortChange={setTaskSort}
            onToggleGroup={handleToggleGroup}
            onRefresh={() => void refreshDashboard({ requestPolicy: "network-only" })}
            onSync={() => void runSync()}
            onScanAll={() => void runScan()}
            onOpenTask={openTaskDetail}
            onScanTask={(id) => void runTaskScan(id)}
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
            pendingSubscriptionID={pendingSubscriptionID}
            subscriptionError={subscriptionError ?? null}
            stashPerformersError={stashPerformersError ?? null}
            performerDetailError={performerDetailError ?? null}
            performerScenesError={performerScenesError ?? null}
            onSearchChange={setSubscriptionSearch}
            onPageSizeChange={setSubscriptionPageSize}
            onReload={() => void reloadSubscription()}
            onRefreshAll={() => void handleRefreshAllSubscription()}
            onToggle={(performer) => void handleSubscriptionToggle(performer)}
            onRefreshOne={(performer) => void handleRefreshSubscribedPerformer(performer)}
            onPrevPage={() => setSubscriptionPage((current) => Math.max(1, current - 1))}
            onNextPage={() => setSubscriptionPage((current) => current + 1)}
            onOpenPerformer={handleOpenPerformer}
            onBackToList={handleBackToPerformers}
            onPerformerSceneSearchChange={setPerformerSceneSearch}
            onPerformerSceneSourceChange={setPerformerSceneSourceFilter}
            onPerformerSceneLibraryChange={setPerformerSceneLibraryFilter}
            onPerformerScenePageSizeChange={setPerformerScenePageSize}
            onPrevPerformerScenePage={() => setPerformerScenePageIndex((current) => Math.max(1, current - 1))}
            onNextPerformerScenePage={() => setPerformerScenePageIndex((current) => current + 1)}
            onToggleSceneSelection={handleToggleSceneSelection}
            onSelectCurrentScenePage={handleSelectCurrentScenePage}
            onClearSceneSelection={handleClearSceneSelection}
            onQueueSelectedScenes={() => void handleQueueSelectedScenes()}
          />
        ) : null}
      </main>

      {visibleDrawer ? (
        <Drawer
          visibleDrawer={visibleDrawer}
          closing={drawerClosing}
          title={drawerTitle}
          onClose={() => setDrawer(null)}
        >
          {visibleDrawer === "stats" ? (
            <StatsDrawer
              active={metrics.active}
              completed={metrics.completed}
              pendingScans={metrics.pendingScans}
              failed={metrics.failed}
            />
          ) : null}

          {visibleDrawer === "settings" ? (
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

          {visibleDrawer === "help" ? (
            <HelpDrawer topicId={helpTopicId} onTopicChange={setHelpTopicId} />
          ) : null}

          {visibleDrawer === "task" ? (
            <TaskDrawer
              task={activeTask}
              pendingScan={activeTask ? pendingTaskScanId === activeTask.id : false}
              pendingDelete={activeTask ? pendingTaskDeleteId === activeTask.id : false}
              onCopy={copyText}
              onSyncAll={() => void runSync()}
              onScanTask={(id) => void runTaskScan(id)}
              onScanAll={() => void runScan()}
              onDeleteTask={requestDeleteTask}
            />
          ) : null}

          {visibleDrawer === "discovery" ? (
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
                            onChange={(event) => setPreviewFastRules(event.target.checked)}
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
                            onChange={(event) => setPreviewFileRules(event.target.checked)}
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
  );
}

export { App };
