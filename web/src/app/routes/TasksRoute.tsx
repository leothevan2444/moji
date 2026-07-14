import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useOutletContext, useParams, useSearchParams } from "react-router";
import { useQuery } from "urql";
import { Drawer } from "../../components/layout/Drawer";
import { TasksPage } from "../../pages/TasksPage";
import { useTaskMutations } from "../../hooks/useTaskMutations";
import { useTaskSnapshotRecoveryGeneration } from "../../graphql/taskRecovery";
import { describeQueryError } from "../../services/queryError";
import { parseTaskSearchParams, serializeTaskSearchParams } from "../searchParams";
import { mergeTaskSelection, taskSummary, type TaskGroupKey } from "../../utils/taskUtils";
import {
  TaskDeletePolicy,
  TaskDetailDocumentDocument,
  TasksOverviewDocumentDocument
} from "../../graphql/generated/graphql";
import type { TaskSortKey, TaskStatusFilter } from "../../types";
import type { AppOutletContext } from "../AppLayout";
import { useTranslation } from "react-i18next";
import type { TaskBatchResultView } from "../../components/drawers/TaskBatchResultDrawer";

const TaskDrawer = lazy(() => import("../../components/drawers/TaskDrawer").then((m) => ({ default: m.TaskDrawer })));
const ConfirmDeleteDrawer = lazy(() => import("../../components/drawers/ConfirmDeleteDrawer").then((m) => ({ default: m.ConfirmDeleteDrawer })));
const SourcingResolutionDrawer = lazy(() => import("../../components/drawers/SourcingResolutionDrawer").then((m) => ({ default: m.SourcingResolutionDrawer })));
const TaskBatchResultDrawer = lazy(() => import("../../components/drawers/TaskBatchResultDrawer").then((m) => ({ default: m.TaskBatchResultDrawer })));

export function taskCloseTarget(state: unknown, searchParams: URLSearchParams) {
  const background = (state as { backgroundLocation?: { pathname: string; search?: string } } | null)?.backgroundLocation;
  return background ? `${background.pathname}${background.search ?? ""}` : `/tasks${searchParams.size ? `?${searchParams}` : ""}`;
}

export function Component() {
  const { t } = useTranslation();
  const { taskId } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { pushToast, copyText } = useOutletContext<AppOutletContext>();
  const recoveryGeneration = useTaskSnapshotRecoveryGeneration();
  const observedRecoveryGeneration = useRef(recoveryGeneration);
  const filter = parseTaskSearchParams(searchParams);
  const [groups, setGroups] = useState<Record<TaskGroupKey, boolean>>({ attention: true, active: true, ingestPending: false, completed: false });
  const [pendingScan, setPendingScan] = useState<string | null>(null);
  const [pendingRetry, setPendingRetry] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);
  const [confirmDeleteIds, setConfirmDeleteIds] = useState<string[]>([]);
  const [selectedTaskIds, setSelectedTaskIds] = useState<string[]>([]);
  const [batchResult, setBatchResult] = useState<TaskBatchResultView | null>(null);
  const [lastRefreshedAt, setLastRefreshedAt] = useState<Date | null>(null);

  const [{ data, fetching, error }, refresh] = useQuery({ query: TasksOverviewDocumentDocument, requestPolicy: "cache-and-network" });
  const [{ data: detailData, fetching: detailFetching, error: detailError }, refreshDetail] = useQuery({
    query: TaskDetailDocumentDocument,
    variables: { id: taskId ?? "" },
    pause: !taskId,
    requestPolicy: "cache-and-network"
  });
  const { deleteTask, retryTask, syncTaskProgress, syncingTaskProgress, triggerTaskStashScan, triggerStashScans, retryTasks, retryingTasks, processTaskIngest, processingTaskIngest, deleteTasks, deletingTasks } = useTaskMutations();
  const tasks = useMemo(() => data?.tasks ?? [], [data?.tasks]);
  const activeTask = detailData?.task ?? (taskId ? tasks.find((task) => task.id === taskId) ?? null : null);
  const deleteTarget = confirmDeleteIds.length === 1 ? tasks.find((task) => task.id === confirmDeleteIds[0]) ?? null : null;
  const deletePolicy = data?.settings.system.taskDeletePolicy ?? TaskDeletePolicy.KeepOnly;
  const isResolution = Boolean(taskId && location.pathname.endsWith("/resolve"));

  const requery = () => refresh({ requestPolicy: "network-only" });
  useEffect(() => { if (!fetching && data) setLastRefreshedAt(new Date()); }, [data, fetching]);
  useEffect(() => { setSelectedTaskIds((current) => current.filter((id) => tasks.some((task) => task.id === id))); }, [tasks]);
  useEffect(() => {
    if (observedRecoveryGeneration.current === recoveryGeneration) return;
    observedRecoveryGeneration.current = recoveryGeneration;
    if (taskId) void refreshDetail({ requestPolicy: "network-only" });
  }, [recoveryGeneration, refreshDetail, taskId]);
  const openTask = (id: string, resolve = false) => navigate(`/tasks/${encodeURIComponent(id)}${resolve ? "/resolve" : ""}`, { state: { backgroundLocation: location } });
  const closeTask = () => {
    navigate(taskCloseTarget(location.state, searchParams));
  };
  const updateFilter = (patch: Partial<{ q: string; status: TaskStatusFilter; sort: TaskSortKey }>) => {
    setSearchParams(serializeTaskSearchParams({ ...filter, ...patch }));
  };

  const runSync = async () => { const result = await syncTaskProgress({}); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else pushToast("tone-success", t("taskBatch.syncComplete")); };
  const runScanAll = async () => { await triggerStashScans({}); };
  const runScan = async (id: string) => {
    setPendingScan(id);
    try { const result = await triggerTaskStashScan({ id }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); }
    finally { setPendingScan(null); }
  };
  const runRetry = async (id: string) => {
    setPendingRetry(id);
    try {
      const result = await retryTask({ id });
      if (result.error) pushToast("tone-danger", describeQueryError(result.error));
      else if (!result.data?.retryTask?.id) pushToast("tone-danger", t("taskRoute.retryNoResult"));
      else pushToast("tone-success", t("taskRoute.retried", { task: tasks.find((task) => task.id === id) ? taskSummary(tasks.find((task) => task.id === id)!) : id }));
    } finally { setPendingRetry(null); }
  };
  const runDelete = async (id: string) => {
    setPendingDelete(id);
    try {
      const result = await deleteTask({ id });
      if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
      if (!result.data?.deleteTask?.id) { pushToast("tone-danger", t("taskRoute.deleteNoResult")); return; }
      pushToast("tone-success", t("taskRoute.deleted", { task: deleteTarget ? taskSummary(deleteTarget) : id }));
      setConfirmDeleteIds([]); if (taskId === id) navigate("/tasks", { replace: true });
    } finally { setPendingDelete(null); }
  };
  const showBatchResult = (payload: TaskBatchResultView | undefined) => {
    if (!payload) return;
    if (payload.summary.failedCount || payload.summary.skippedCount) setBatchResult(payload);
    else pushToast("tone-success", t("taskBatch.batchComplete", { count: payload.summary.succeededCount }));
  };
  const runBatchRetry = async (ids: string[]) => { const result = await retryTasks({ ids }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else showBatchResult(result.data?.retryTasks); };
  const runBatchIngest = async (ids: string[]) => { const result = await processTaskIngest({ ids }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else showBatchResult(result.data?.processTaskIngest); };
  const runBatchDelete = async (ids: string[]) => {
    const result = await deleteTasks({ ids });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    const payload = result.data?.deleteTasks;
    if (payload) {
      const deleted = new Set(payload.results.filter((item) => item.status === "SUCCEEDED").map((item) => item.taskId));
      setSelectedTaskIds((current) => current.filter((id) => !deleted.has(id)));
      showBatchResult(payload);
    }
    setConfirmDeleteIds([]);
  };
  const toggleSelection = (id: string) => setSelectedTaskIds((current) => current.includes(id) ? current.filter((item) => item !== id) : current.length >= 100 ? (pushToast("tone-info", t("taskBatch.limit", { count: 100 })), current) : [...current, id]);
  const selectVisible = (ids: string[]) => {
    const requestedCount = new Set([...selectedTaskIds, ...ids]).size;
    if (requestedCount > 100) pushToast("tone-info", t("taskBatch.limit", { count: 100 }));
    setSelectedTaskIds((current) => mergeTaskSelection(current, ids));
  };

  if (error && !data) return <div className="empty-card"><h3>{t("taskRoute.loadFailed")}</h3><p>{describeQueryError(error)}</p><button type="button" onClick={requery}>{t("common.retry")}</button></div>;
  const metrics = data?.dashboardStats;
  return <>
    <TasksPage tasks={tasks} metrics={{ active: metrics?.active ?? 0, completed: metrics?.completed ?? 0, pendingScans: metrics?.pendingScans ?? 0, failed: metrics?.failed ?? 0 }}
      taskSearch={filter.q} taskStatus={filter.status} taskSort={filter.sort} taskGroupOpen={groups}
      pendingTaskScanId={pendingScan} pendingTaskRetryId={pendingRetry} pendingTaskDeleteId={pendingDelete} selectedTaskIds={selectedTaskIds}
      batchPending={retryingTasks || processingTaskIngest || deletingTasks} refreshing={fetching} syncing={syncingTaskProgress}
      autoSyncEnabled={data?.settingsStatus.automation.taskProgressSyncEnabled ?? false} autoSyncIntervalSeconds={data?.settingsStatus.automation.taskProgressSyncIntervalSeconds ?? 0} lastRefreshedAt={lastRefreshedAt}
      onSearchChange={(q) => updateFilter({ q })} onStatusChange={(status) => updateFilter({ status })} onSortChange={(sort) => updateFilter({ sort })}
      onToggleGroup={(group) => setGroups((current) => ({ ...current, [group]: !current[group] }))}
      onRefresh={requery} onSync={() => void runSync()} onOpenTask={(id) => openTask(id)}
      onScanTask={(id) => void runScan(id)} onRetryTask={(id) => void runRetry(id)} onResolveTask={(id) => openTask(id, true)}
      onDeleteTask={(id) => setConfirmDeleteIds([id])} onToggleTaskSelection={toggleSelection} onSelectVisibleTasks={selectVisible} onClearTaskSelection={() => setSelectedTaskIds([])}
      onBatchRetry={(ids) => void runBatchRetry(ids)} onBatchIngest={(ids) => void runBatchIngest(ids)} onBatchDelete={setConfirmDeleteIds} />
    {taskId ? <Drawer visibleDrawer={isResolution ? "task-resolution" : "task"} closing={false} title={isResolution ? t("taskRoute.resolutionTitle", { task: activeTask ? taskSummary(activeTask) : t("taskRoute.task") }) : activeTask ? taskSummary(activeTask) : t("taskRoute.details")} onClose={closeTask}>
      <Suspense fallback={<div className="skeleton skeleton-card" />}>
        {detailFetching && !activeTask ? <p>{t("taskRoute.loading")}</p> : detailError ? <div className="empty-card"><h3>{t("taskRoute.loadFailed")}</h3><p>{describeQueryError(detailError)}</p></div> : !activeTask ? <div className="empty-card"><h3>{t("taskRoute.notFound")}</h3></div> : isResolution ?
          <SourcingResolutionDrawer task={activeTask} onResolved={async () => { navigate(`/tasks/${encodeURIComponent(taskId)}`); }} /> :
          <TaskDrawer task={activeTask} pendingScan={pendingScan === taskId} pendingRetry={pendingRetry === taskId} pendingDelete={pendingDelete === taskId} onCopy={copyText} onSyncAll={() => void runSync()} onScanTask={(id) => void runScan(id)} onRetryTask={(id) => void runRetry(id)} onScanAll={() => void runScanAll()} onDeleteTask={(id) => setConfirmDeleteIds([id])} />}
      </Suspense>
    </Drawer> : null}
    {confirmDeleteIds.length ? <Drawer visibleDrawer="confirm" closing={false} title={t("taskRoute.deleteTitle")} onClose={() => { if (!pendingDelete && !deletingTasks) setConfirmDeleteIds([]); }}><Suspense fallback={null}><ConfirmDeleteDrawer taskLabel={deleteTarget ? taskSummary(deleteTarget) : undefined} count={confirmDeleteIds.length > 1 ? confirmDeleteIds.length : undefined} deletePolicy={deletePolicy} pending={Boolean(pendingDelete) || deletingTasks} onConfirm={() => confirmDeleteIds.length === 1 ? void runDelete(confirmDeleteIds[0]) : void runBatchDelete(confirmDeleteIds)} onCancel={() => { if (!pendingDelete && !deletingTasks) setConfirmDeleteIds([]); }} /></Suspense></Drawer> : null}
    {batchResult ? <Drawer visibleDrawer="task-batch-result" closing={false} title={t("taskBatch.resultTitle")} onClose={() => setBatchResult(null)}><Suspense fallback={null}><TaskBatchResultDrawer payload={batchResult} onOpenTask={(id) => { setBatchResult(null); openTask(id); }} /></Suspense></Drawer> : null}
  </>;
}
