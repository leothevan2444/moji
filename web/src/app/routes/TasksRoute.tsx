import { lazy, Suspense, useState } from "react";
import { useLocation, useNavigate, useOutletContext, useParams, useSearchParams } from "react-router";
import { useQuery } from "urql";
import { Drawer } from "../../components/layout/Drawer";
import { TasksPage } from "../../pages/TasksPage";
import { useTaskMutations } from "../../hooks/useTaskMutations";
import { describeQueryError } from "../../services/queryError";
import { parseTaskSearchParams, serializeTaskSearchParams } from "../searchParams";
import { taskSummary, type TaskGroupKey } from "../../utils/taskUtils";
import {
  TaskDeletePolicy,
  TaskDetailDocumentDocument,
  TasksOverviewDocumentDocument
} from "../../graphql/generated/graphql";
import type { TaskSortKey, TaskStatusFilter } from "../../types";
import type { AppOutletContext } from "../AppLayout";
import { useTranslation } from "react-i18next";

const TaskDrawer = lazy(() => import("../../components/drawers/TaskDrawer").then((m) => ({ default: m.TaskDrawer })));
const ConfirmDeleteDrawer = lazy(() => import("../../components/drawers/ConfirmDeleteDrawer").then((m) => ({ default: m.ConfirmDeleteDrawer })));
const SourcingResolutionDrawer = lazy(() => import("../../components/drawers/SourcingResolutionDrawer").then((m) => ({ default: m.SourcingResolutionDrawer })));

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
  const filter = parseTaskSearchParams(searchParams);
  const [groups, setGroups] = useState<Record<TaskGroupKey, boolean>>({ attention: true, active: true, ingestPending: false, completed: false });
  const [pendingScan, setPendingScan] = useState<string | null>(null);
  const [pendingRetry, setPendingRetry] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [retryingBlocked, setRetryingBlocked] = useState(false);

  const [{ data, fetching, error }, refresh] = useQuery({ query: TasksOverviewDocumentDocument, requestPolicy: "cache-and-network" });
  const [{ data: detailData, fetching: detailFetching, error: detailError }, refreshDetail] = useQuery({
    query: TaskDetailDocumentDocument,
    variables: { id: taskId ?? "" },
    pause: !taskId,
    requestPolicy: "cache-and-network"
  });
  const { deleteTask, retryTask, syncTaskProgress, triggerTaskStashScan, triggerStashScans } = useTaskMutations();
  const tasks = data?.tasks ?? [];
  const activeTask = detailData?.task ?? (taskId ? tasks.find((task) => task.id === taskId) ?? null : null);
  const deleteTarget = confirmDelete ? tasks.find((task) => task.id === confirmDelete) ?? null : null;
  const deletePolicy = data?.settings.system.taskDeletePolicy ?? TaskDeletePolicy.KeepOnly;
  const isResolution = Boolean(taskId && location.pathname.endsWith("/resolve"));

  const requery = () => refresh({ requestPolicy: "network-only" });
  const openTask = (id: string, resolve = false) => navigate(`/tasks/${encodeURIComponent(id)}${resolve ? "/resolve" : ""}`, { state: { backgroundLocation: location } });
  const closeTask = () => {
    navigate(taskCloseTarget(location.state, searchParams));
  };
  const updateFilter = (patch: Partial<{ q: string; status: TaskStatusFilter; sort: TaskSortKey }>) => {
    setSearchParams(serializeTaskSearchParams({ ...filter, ...patch }));
  };

  const runSync = async () => { await syncTaskProgress({}); await requery(); if (taskId) await refreshDetail({ requestPolicy: "network-only" }); };
  const runScanAll = async () => { await triggerStashScans({}); await requery(); if (taskId) await refreshDetail({ requestPolicy: "network-only" }); };
  const runScan = async (id: string) => {
    setPendingScan(id);
    try { const result = await triggerTaskStashScan({ id }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); await requery(); if (taskId === id) await refreshDetail({ requestPolicy: "network-only" }); }
    finally { setPendingScan(null); }
  };
  const runRetry = async (id: string) => {
    setPendingRetry(id);
    try {
      const result = await retryTask({ id });
      if (result.error) pushToast("tone-danger", describeQueryError(result.error));
      else if (!result.data?.retryTask?.id) pushToast("tone-danger", t("taskRoute.retryNoResult"));
      else pushToast("tone-success", t("taskRoute.retried", { task: tasks.find((task) => task.id === id) ? taskSummary(tasks.find((task) => task.id === id)!) : id }));
      await requery(); if (taskId === id) await refreshDetail({ requestPolicy: "network-only" });
    } finally { setPendingRetry(null); }
  };
  const retryBlocked = async () => {
    const blocked = tasks.filter((task) => task.stageStatus === "BLOCKED");
    if (!blocked.length) { pushToast("tone-info", t("taskRoute.noneBlocked")); return; }
    setRetryingBlocked(true); let succeeded = 0;
    try { for (const task of blocked) { setPendingRetry(task.id); const result = await retryTask({ id: task.id }); if (!result.error && result.data?.retryTask?.id) succeeded += 1; } await requery(); pushToast("tone-info", t("taskRoute.retrySummary", { succeeded, failed: blocked.length - succeeded })); }
    finally { setPendingRetry(null); setRetryingBlocked(false); }
  };
  const runDelete = async (id: string) => {
    setPendingDelete(id);
    try {
      const result = await deleteTask({ id });
      if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
      if (!result.data?.deleteTask?.id) { pushToast("tone-danger", t("taskRoute.deleteNoResult")); return; }
      pushToast("tone-success", t("taskRoute.deleted", { task: deleteTarget ? taskSummary(deleteTarget) : id }));
      setConfirmDelete(null); await requery(); if (taskId === id) navigate("/tasks", { replace: true });
    } finally { setPendingDelete(null); }
  };

  if (error && !data) return <div className="empty-card"><h3>{t("taskRoute.loadFailed")}</h3><p>{describeQueryError(error)}</p><button type="button" onClick={requery}>{t("common.retry")}</button></div>;
  const metrics = data?.dashboardStats;
  return <>
    <TasksPage tasks={tasks} metrics={{ active: metrics?.active ?? 0, completed: metrics?.completed ?? 0, pendingScans: metrics?.pendingScans ?? 0, failed: metrics?.failed ?? 0 }}
      taskSearch={filter.q} taskStatus={filter.status} taskSort={filter.sort} taskGroupOpen={groups}
      pendingTaskScanId={pendingScan} pendingTaskRetryId={pendingRetry} pendingTaskDeleteId={pendingDelete} retryingBlockedTasks={retryingBlocked}
      onSearchChange={(q) => updateFilter({ q })} onStatusChange={(status) => updateFilter({ status })} onSortChange={(sort) => updateFilter({ sort })}
      onToggleGroup={(group) => setGroups((current) => ({ ...current, [group]: !current[group] }))}
      onRefresh={requery} onSync={() => void runSync()} onScanAll={() => void runScanAll()} onOpenTask={(id) => openTask(id)}
      onScanTask={(id) => void runScan(id)} onRetryTask={(id) => void runRetry(id)} onResolveTask={(id) => openTask(id, true)}
      onRetryBlockedTasks={() => void retryBlocked()} onDeleteTask={setConfirmDelete} />
    {taskId ? <Drawer visibleDrawer={isResolution ? "task-resolution" : "task"} closing={false} title={isResolution ? t("taskRoute.resolutionTitle", { task: activeTask ? taskSummary(activeTask) : t("taskRoute.task") }) : activeTask ? taskSummary(activeTask) : t("taskRoute.details")} onClose={closeTask}>
      <Suspense fallback={<div className="skeleton skeleton-card" />}>
        {detailFetching && !activeTask ? <p>{t("taskRoute.loading")}</p> : detailError ? <div className="empty-card"><h3>{t("taskRoute.loadFailed")}</h3><p>{describeQueryError(detailError)}</p></div> : !activeTask ? <div className="empty-card"><h3>{t("taskRoute.notFound")}</h3></div> : isResolution ?
          <SourcingResolutionDrawer task={activeTask} onResolved={async () => { await requery(); await refreshDetail({ requestPolicy: "network-only" }); navigate(`/tasks/${encodeURIComponent(taskId)}`); }} /> :
          <TaskDrawer task={activeTask} pendingScan={pendingScan === taskId} pendingRetry={pendingRetry === taskId} pendingDelete={pendingDelete === taskId} onCopy={copyText} onSyncAll={() => void runSync()} onScanTask={(id) => void runScan(id)} onRetryTask={(id) => void runRetry(id)} onScanAll={() => void runScanAll()} onDeleteTask={setConfirmDelete} />}
      </Suspense>
    </Drawer> : null}
    {confirmDelete ? <Drawer visibleDrawer="confirm" closing={false} title={t("taskRoute.deleteTitle")} onClose={() => { if (!pendingDelete) setConfirmDelete(null); }}><Suspense fallback={null}><ConfirmDeleteDrawer taskLabel={deleteTarget ? taskSummary(deleteTarget) : confirmDelete} destructive={deletePolicy === TaskDeletePolicy.RemoveTorrentAndFiles} pending={pendingDelete === confirmDelete} onConfirm={() => void runDelete(confirmDelete)} onCancel={() => { if (!pendingDelete) setConfirmDelete(null); }} /></Suspense></Drawer> : null}
  </>;
}
