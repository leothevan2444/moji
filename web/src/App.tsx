import { FormEvent, useDeferredValue, useEffect, useState } from "react";
import { HELP_TOPICS, type HelpTopicId } from "./help";
import { describeQueryError } from "./services/queryError";
import {
  useDashboard,
  useDrawerTransition,
  useJackettSearch,
  useSubscription,
  useTheme,
  useToast
} from "./hooks";
import { taskSummary, type TaskGroupKey } from "./utils";
import type { DrawerKey, SettingsTab, TabKey } from "./types";
import { Drawer, Header, ToastStack } from "./components/layout";
import { HelpDrawer, SettingsDrawer, StatsDrawer, TaskDrawer } from "./components/drawers";
import {
  DiscoveryPage,
  HomePage,
  SubscriptionPage,
  TasksPage,
  type TaskSortKey,
  type TaskStatusFilter
} from "./pages";
import type { SearchDocumentQuery } from "./graphql/generated/graphql";

function App() {
  // ── UI state ────────────────────────────────────────────────────────
  const [tab, setTab] = useState<TabKey>("主页");
  const [drawer, setDrawer] = useState<DrawerKey>(null);
  const [settingsTab, setSettingsTab] = useState<SettingsTab>("连接");
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

  // Discovery page state
  const [jackettQuery, setJackettQuery] = useState("");
  const [submittedJackettQuery, setSubmittedJackettQuery] = useState("");
  const [pendingAddId, setPendingAddId] = useState<string | null>(null);

  // Subscription page state
  const [subscriptionSearch, setSubscriptionSearch] = useState("");
  const [subscriptionPage, setSubscriptionPage] = useState(1);
  const [subscriptionPageSize, setSubscriptionPageSize] = useState<number>(24);
  const [pendingSubscriptionID, setPendingSubscriptionID] = useState<string | null>(null);

  // ── Data hooks ──────────────────────────────────────────────────────
  const { toasts, pushToast, dismissToast, copyText } = useToast();
  const theme = useTheme();
  const { renderedDrawer, drawerClosing, visibleDrawer } = useDrawerTransition(drawer);

  const deferredJackettQuery = useDeferredValue(submittedJackettQuery.trim());
  const deferredSubscriptionSearch = useDeferredValue(subscriptionSearch.trim());

  const {
    data,
    error,
    refreshDashboard,
    addTorrent,
    syncTaskProgress,
    triggerTaskStashScan,
    triggerStashScans
  } = useDashboard();

  const {
    results: searchResults,
    fetching: searching,
    error: searchError
  } = useJackettSearch(deferredJackettQuery);

  const {
    stashPerformerPage,
    stashPerformers,
    subscribedPerformers,
    fetchingStashPerformers,
    fetchingSubscription,
    stashPerformersError,
    subscriptionError,
    refreshingSubscriptionNow,
    subscribePerformer,
    unsubscribePerformer,
    refreshSubscribedPerformer,
    refreshSubscriptionsNow,
    reloadSubscription
  } = useSubscription({
    enabled: tab === "订阅",
    search: deferredSubscriptionSearch || null,
    page: subscriptionPage,
    pageSize: subscriptionPageSize
  });

  // ── Derived state ───────────────────────────────────────────────────
  const tasks = data?.tasks ?? [];
  const runtimeSettings = data?.settings ?? null;
  const runtimeStatus = data?.settingsStatus ?? null;
  const activeTask = selectedTaskId ? tasks.find((task) => task.id === selectedTaskId) ?? null : null;

  const metrics = {
    active: data?.dashboardStats.active ?? 0,
    completed: data?.dashboardStats.completed ?? 0,
    downloading: data?.dashboardStats.downloading ?? 0,
    pendingScans: data?.dashboardStats.pendingScans ?? 0,
    failed: data?.dashboardStats.failed ?? 0,
    total: data?.dashboardStats.total ?? 0
  };

  // ── Effects ─────────────────────────────────────────────────────────
  useEffect(() => {
    setSubscriptionPage(1);
  }, [deferredSubscriptionSearch]);

  useEffect(() => {
    setSubscriptionPage(1);
  }, [subscriptionPageSize]);

  // ── Action handlers ─────────────────────────────────────────────────
  const openTaskDetail = (taskId: string) => {
    setSelectedTaskId(taskId);
    setDrawer("task");
  };

  const handleSubmitJackettSearch = (event: FormEvent<HTMLFormElement>) => {
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
    setPendingTaskScanId(taskId);
    try {
      await triggerTaskStashScan({ id: taskId });
      await refreshDashboard({ requestPolicy: "network-only" });
    } finally {
      setPendingTaskScanId(null);
    }
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

    pushToast("tone-info", "已触发全部订阅对象的更新检查。");
    await reloadSubscription();
  };

  const handleToggleGroup = (group: TaskGroupKey) => {
    setTaskGroupOpen((current) => ({ ...current, [group]: !current[group] }));
  };

  // ── Drawer metadata ─────────────────────────────────────────────────
  const drawerKicker = (() => {
    if (visibleDrawer === "stats") return "统计";
    if (visibleDrawer === "settings") return "设置";
    if (visibleDrawer === "help") return "帮助";
    return "任务详情";
  })();

  const drawerTitle = (() => {
    if (visibleDrawer === "stats") return "运行概览";
    if (visibleDrawer === "settings") return "配置与系统";
    if (visibleDrawer === "help") return "Markdown 帮助";
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
          <HomePage
            tasks={tasks}
            runtimeSettings={runtimeSettings}
            runtimeStatus={runtimeStatus}
            lastCheckedAt={data?.tasks[0]?.updatedAt ?? null}
            pendingTaskScanId={pendingTaskScanId}
            onRefresh={() => refreshDashboard({ requestPolicy: "network-only" })}
            onOpenTask={openTaskDetail}
            onScanTask={(id) => void runTaskScan(id)}
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
            onSearchChange={setTaskSearch}
            onStatusChange={setTaskStatus}
            onSortChange={setTaskSort}
            onToggleGroup={handleToggleGroup}
            onRefresh={() => void refreshDashboard({ requestPolicy: "network-only" })}
            onSync={() => void runSync()}
            onScanAll={() => void runScan()}
            onOpenTask={openTaskDetail}
            onScanTask={(id) => void runTaskScan(id)}
          />
        ) : null}

        {tab === "发现" ? (
          <DiscoveryPage
            jackettQuery={jackettQuery}
            searching={searching}
            searchError={searchError ?? null}
            searchResults={searchResults}
            deferredJackettQuery={deferredJackettQuery}
            pendingAddId={pendingAddId}
            onQueryChange={setJackettQuery}
            onSubmit={handleSubmitJackettSearch}
            onAdd={(result) => void handleAddSearchResult(result)}
            onOpenHelp={() => setDrawer("help")}
          />
        ) : null}

        {tab === "订阅" ? (
          <SubscriptionPage
            runtimeSettings={runtimeSettings}
            stashPerformerPage={stashPerformerPage}
            stashPerformers={stashPerformers}
            subscribedPerformers={subscribedPerformers}
            fetchingStashPerformers={fetchingStashPerformers}
            fetchingSubscription={fetchingSubscription}
            refreshingSubscriptionNow={refreshingSubscriptionNow}
            subscriptionSearch={subscriptionSearch}
            subscriptionPageSize={subscriptionPageSize}
            pendingSubscriptionID={pendingSubscriptionID}
            subscriptionError={subscriptionError ?? null}
            stashPerformersError={stashPerformersError ?? null}
            onSearchChange={setSubscriptionSearch}
            onPageSizeChange={setSubscriptionPageSize}
            onReload={() => void reloadSubscription()}
            onRefreshAll={() => void handleRefreshAllSubscription()}
            onToggle={(performer) => void handleSubscriptionToggle(performer)}
            onRefreshOne={(performer) => void handleRefreshSubscribedPerformer(performer)}
            onPrevPage={() => setSubscriptionPage((current) => Math.max(1, current - 1))}
            onNextPage={() => setSubscriptionPage((current) => current + 1)}
          />
        ) : null}
      </main>

      {visibleDrawer ? (
        <Drawer
          visibleDrawer={visibleDrawer}
          closing={drawerClosing}
          kicker={drawerKicker}
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
              onCopy={copyText}
              onSyncAll={() => void runSync()}
              onScanTask={(id) => void runTaskScan(id)}
              onScanAll={() => void runScan()}
            />
          ) : null}
        </Drawer>
      ) : null}

    </div>
  );
}

export { App };
