import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useOutletContext, useSearchParams } from "react-router";
import { useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { HomePage } from "../../pages/HomePage";
import { Drawer } from "../../components/layout/Drawer";
import { useTaskMutations } from "../../hooks/useTaskMutations";
import { useServiceStatusEvents } from "../../hooks/useServiceStatusEvents";
import { useTaskEvents } from "../../hooks/useTaskEvents";
import { usePerformerSubscriptionEvents } from "../../hooks/usePerformerSubscriptionEvents";
import { describeQueryError } from "../../services/queryError";
import { taskSummary } from "../../utils/taskUtils";
import { useFragment as readFragment } from "../../graphql/generated/fragment-masking";
import { HomePageDocumentDocument, HomeServiceStatusDocument, HomeSubscriptionSummaryDocument, HomeTaskFragmentDoc, TaskDetailDocumentDocument, TaskDeletePolicy } from "../../graphql/generated/graphql";
import type { AppOutletContext } from "../AppLayout";

const TaskFiles = lazy(() => import("../../components/tasks/TaskFiles").then(m => ({ default: m.TaskFiles })));
const TaskDrawer = lazy(() => import("../../components/drawers/TaskDrawer").then(m => ({ default: m.TaskDrawer })));
const SourcingResolutionDrawer = lazy(() => import("../../components/drawers/SourcingResolutionDrawer").then(m => ({ default: m.SourcingResolutionDrawer })));
const ConfirmDeleteDrawer = lazy(() => import("../../components/drawers/ConfirmDeleteDrawer").then(m => ({ default: m.ConfirmDeleteDrawer })));

export function Component() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const taskId = params.get("task");
  const resolve = params.get("resolve") === "1";
  const filesOnly = params.get("files") === "1";
  const [filesBusy, setFilesBusy] = useState(false);
  const { pushToast, copyText } = useOutletContext<AppOutletContext>();
  const [pendingRetry, setPendingRetry] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [{ data, error, fetching }, refresh] = useQuery({ query: HomePageDocumentDocument, requestPolicy: "cache-and-network" });
  const [{ data: status, error: statusError, fetching: statusFetching }, refreshStatus] = useQuery({ query: HomeServiceStatusDocument, requestPolicy: "cache-and-network" });
  const [{ data: subscriptions, error: subscriptionError }, refreshSubscriptions] = useQuery({ query: HomeSubscriptionSummaryDocument, requestPolicy: "cache-and-network" });
  const [{ data: detail, error: detailError, fetching: detailFetching }, refreshDetail] = useQuery({ query: TaskDetailDocumentDocument, variables: { id: taskId ?? "" }, pause: !taskId, requestPolicy: "cache-and-network" });
  const { retryTask, deleteTask, deletingTask } = useTaskMutations();
  const timer = useRef<number | null>(null);
  const refreshRef = useRef(() => {});
  refreshRef.current = () => { refresh({ requestPolicy: "network-only" }); if (taskId) refreshDetail({ requestPolicy: "network-only" }); };
  const scheduleRefresh = useCallback(() => {
    if (timer.current !== null) return;
    timer.current = window.setTimeout(() => { timer.current = null; refreshRef.current(); }, 800);
  }, []);
  const taskEvents = useTaskEvents({ onEvent: scheduleRefresh, onReconnect: scheduleRefresh, onSequenceGap: scheduleRefresh });
  const serviceEvents = useServiceStatusEvents({ onRefresh: () => refreshStatus({ requestPolicy: "network-only" }) });
  const subscriptionEvents = usePerformerSubscriptionEvents({ enabled: true, onRefresh: () => refreshSubscriptions({ requestPolicy: "network-only" }) });
  useEffect(() => { if (subscriptionEvents.event) refreshSubscriptions({ requestPolicy: "network-only" }); }, [subscriptionEvents.event, refreshSubscriptions]);
  useEffect(() => () => { if (timer.current !== null) window.clearTimeout(timer.current); }, []);
  // Keep the bounded snapshot current even when a subscription drops silently.
  useEffect(() => {
    const interval = window.setInterval(() => {
      if (document.visibilityState === "hidden") return;
      refreshRef.current();
      refreshStatus({ requestPolicy: "network-only" });
      refreshSubscriptions({ requestPolicy: "network-only" });
    }, 30000);
    const onVisible = () => { if (document.visibilityState === "visible") { refreshRef.current(); refreshStatus({ requestPolicy: "network-only" }); refreshSubscriptions({ requestPolicy: "network-only" }); } };
    document.addEventListener("visibilitychange", onVisible);
    return () => { window.clearInterval(interval); document.removeEventListener("visibilitychange", onVisible); };
  }, [refreshStatus, refreshSubscriptions]);
  const overview = useMemo(() => data?.homeOverview ? {
    ...data.homeOverview,
    attentionTasks: data.homeOverview.attentionTasks.map(task => readFragment(HomeTaskFragmentDoc, task)),
    activeTasks: data.homeOverview.activeTasks.map(task => readFragment(HomeTaskFragmentDoc, task)),
    recentTasks: data.homeOverview.recentTasks.map(task => readFragment(HomeTaskFragmentDoc, task))
  } : null, [data?.homeOverview]);
  // Opening a task preserves the home list and scroll context behind its drawer.
  const [displayed, setDisplayed] = useState(overview);
  const [listUpdatePending, setListUpdatePending] = useState(false);
  useEffect(() => {
    if (taskId || pendingRetry || overview === displayed) return;
    const signature = (value: typeof overview) => value ? [value.attentionTasks, value.activeTasks, value.recentTasks].map(tasks => tasks.map(task => task.id).join(",")).join(";") : "";
    if (displayed && document.activeElement?.closest(".home-task-list") && signature(displayed) !== signature(overview)) {
      setListUpdatePending(true);
      return;
    }
    setDisplayed(overview);
    setListUpdatePending(false);
  }, [overview, displayed, taskId, pendingRetry]);
  const tasks = overview ? [...overview.attentionTasks, ...overview.activeTasks, ...overview.recentTasks] : [];
  const task = detail ? detail.task ?? null : tasks.find(entry => entry.id === taskId) ?? null;
  const openTask = (id: string, resolution = false, files = false) => { const next = new URLSearchParams(params); next.set("task", id); if (resolution) next.set("resolve", "1"); else next.delete("resolve"); if (files) next.set("files", "1"); else next.delete("files"); setParams(next, { preventScrollReset: true }); };
  const closeTask = () => { if (pendingRetry || deletingTask || filesBusy) return; const next = new URLSearchParams(params); next.delete("task"); next.delete("resolve"); next.delete("files"); setParams(next, { replace: true, preventScrollReset: true }); };
  const retry = async (id: string) => {
    if (pendingRetry) return;
    setPendingRetry(id);
    try {
      const result = await retryTask({ id });
      if (result.error) pushToast("tone-danger", describeQueryError(result.error));
      else if (!result.data?.retryTask?.id) pushToast("tone-danger", t("home.retryNoResult"));
      else if (result.data.retryTask.stageStatus === "BLOCKED") { pushToast("tone-danger", t("taskProcessingQueue.stillBlocked")); refreshRef.current(); }
      else { pushToast("tone-success", t("home.retried", { task: tasks.find(entry => entry.id === id)?.code || id })); refreshRef.current(); }
    } catch { pushToast("tone-danger", t("home.retryNoResult")); }
    finally { setPendingRetry(null); }
  };
  const confirmDelete = async () => {
    if (!deleteId) return;
    try {
      const result = await deleteTask({ id: deleteId });
      if (result.error) pushToast("tone-danger", describeQueryError(result.error));
      else if (!result.data?.deleteTask?.id) pushToast("tone-danger", t("taskRoute.deleteNoResult"));
      else {
        setDeleteId(null);
        const next = new URLSearchParams(params);
        next.delete("task"); next.delete("resolve"); next.delete("files");
        setParams(next, { replace: true, preventScrollReset: true });
        pushToast("tone-success", t("taskRoute.deleted", { task: task ? taskSummary(task) : deleteId }));
        refreshRef.current();
      }
    } catch { pushToast("tone-danger", t("taskRoute.deleteNoResult")); }
  };
  const refreshAll = () => { refreshRef.current(); refreshStatus({ requestPolicy: "network-only" }); refreshSubscriptions({ requestPolicy: "network-only" }); };
  return <>
    <HomePage listUpdatePending={listUpdatePending} onApplyUpdates={() => { setDisplayed(overview); setListUpdatePending(false); }} interactionOpen={Boolean(taskId)} overview={displayed ?? overview} runtimeSettings={data?.settings ?? null} runtimeStatus={status?.settingsStatus ?? null} subscription={subscriptions?.homeSubscriptionSummary ?? null}
      overviewError={error ? describeQueryError(error) : null} statusError={statusError ? describeQueryError(statusError) : null} subscriptionError={subscriptionError ? describeQueryError(subscriptionError) : null}
      fetching={fetching || statusFetching} connectionStatus={taskEvents.error || serviceEvents.error || subscriptionEvents.error ? "error" : taskEvents.connectionStatus} pendingTaskRetryId={pendingRetry}
      onRefresh={refreshAll} onRefreshStatus={() => refreshStatus({ requestPolicy: "network-only" })} onRefreshSubscription={() => refreshSubscriptions({ requestPolicy: "network-only" })}
      onSearch={(q, source) => navigate(`/discover?${new URLSearchParams({ q, source })}`)} onOpenTask={id => openTask(id)} onConfirmFiles={id => openTask(id, false, true)} onRetryTask={id => void retry(id)} onResolveTask={id => openTask(id, true)}
      onOpenQueue={() => navigate("/tasks?status=failed&workbench=1", { state: { backgroundLocation: { pathname: "/", search: "" } } })} onOpenSettings={tab => navigate(`/settings/${tab}`)} />
    {taskId ? <Drawer visibleDrawer={resolve ? "task-resolution" : "task"} closing={false} title={task ? taskSummary(task) : t("taskRoute.details")} onClose={closeTask} closeDisabled={Boolean(pendingRetry) || deletingTask || filesBusy}>
      <Suspense fallback={<div className="skeleton skeleton-card" aria-label={t("common.loading")} />}>
        {detailError && !task ? <div className="home-module-error"><p>{describeQueryError(detailError)}</p><button type="button" onClick={() => refreshDetail({ requestPolicy: "network-only" })}>{t("common.retry")}</button></div> : detailFetching && !task ? <div className="skeleton skeleton-card" aria-label={t("common.loading")} /> : !task ? <p>{t("taskRoute.notFound")}</p> : filesOnly ? <TaskFiles key={task.id} task={task} onCopy={copyText} onBusyChange={setFilesBusy} onResolved={() => { setFilesBusy(false); pushToast("tone-success", t("mediaFiles.selectionSaved")); openTask(task.id); refreshRef.current(); }} /> : resolve ? <SourcingResolutionDrawer task={task} onNotify={pushToast} onResolved={async () => { openTask(task.id); refreshRef.current(); }} /> : <TaskDrawer task={task} pendingRetry={pendingRetry === task.id} pendingDelete={deletingTask} onCopy={copyText} onResolveTask={id => openTask(id, true)} onRetryTask={id => void retry(id)} onDeleteTask={setDeleteId} />}
        {task ? <LinkToTask id={task.id} /> : null}
      </Suspense>
    </Drawer> : null}
    {deleteId ? <Drawer visibleDrawer="confirm" closing={false} title={t("taskRoute.deleteTitle")} onClose={() => { if (!deletingTask) setDeleteId(null); }} closeDisabled={deletingTask}><Suspense fallback={null}><ConfirmDeleteDrawer taskLabel={task ? taskSummary(task) : deleteId} deletePolicy={data?.settings.system.taskDeletePolicy ?? TaskDeletePolicy.KeepOnly} pending={deletingTask} onConfirm={() => void confirmDelete()} onCancel={() => { if (!deletingTask) setDeleteId(null); }} /></Suspense></Drawer> : null}
  </>;
}

function LinkToTask({ id }: { id: string }) {
  const { t } = useTranslation();
  return <Link className="home-more" to={`/tasks/${encodeURIComponent(id)}`}>{t("workbench.openTaskCenter")} →</Link>;
}
