import { workspaceToolbarIcons } from "../components/common/workspace-toolbar/icons";
import { WorkspaceToolbar } from "../components/common/workspace-toolbar/WorkspaceToolbar";
import { Pagination } from "../components/common/pagination/Pagination";
import { FilterCountHeading } from "../components/layout/FilterCountHeading";
import { faBarsProgress, faHourglassHalf, faCheckDouble, faTriangleExclamation, faLayerGroup } from "@fortawesome/free-solid-svg-icons";
import { useAppTimezone } from "../hooks/useAppTimezone";
import { TaskGroupSection } from "../components/tasks";
import {
  formatTime,
  taskGroup,
  taskGroupTone,
  taskBatchEligibility,
  taskSummary,
  type DashboardTask,
  type TaskGroupKey
} from "../utils";
import { useDeferredValue, useMemo, useState } from "react";
import type { TaskSortKey, TaskStatusFilter } from "../types";
import { useTranslation } from "react-i18next";
import { qbittorrentStateLabel } from "../utils/qbittorrentState";

interface TasksPageProps {
  serverPage?: { filterCounts?: { total: number; active: number; waiting: number; completed: number; attention: number } | null; page: number; pageSize: number; totalPages: number; totalCount: number; stats: { total: number; active: number; completed: number; attention: number; waiting: number } } | null;
  selectedTasks?: DashboardTask[];
  selectionLoading?: boolean;
  updatesPending?: boolean;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (size: number) => void;
  tasks: DashboardTask[];
  completedWithin24Hours?: boolean;
  onClearCompletedWindow?: () => void;
  taskSearch: string;
  taskStatus: TaskStatusFilter;
  taskSort: TaskSortKey;
  taskGroupOpen: Record<TaskGroupKey, boolean>;
  pendingTaskRetryId: string | null;
  pendingTaskDeleteId: string | null;
  selectedTaskIds: string[];
  batchPending: boolean;
  refreshing: boolean;
  lastRefreshedAt: Date | null;
  onSearchChange: (value: string) => void;
  onStatusChange: (status: TaskStatusFilter) => void;
  onSortChange: (sort: TaskSortKey) => void;
  onToggleGroup: (group: TaskGroupKey) => void;
  onRefresh: () => void;
  onOpenQueue: () => void;
  onOpenTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onResolveTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
  onToggleTaskSelection: (taskId: string) => void;
  onSelectVisibleTasks: (taskIds: string[]) => void;
  onClearTaskSelection: () => void;
  onBatchRetry: (taskIds: string[]) => void;
  onBatchDelete: (taskIds: string[]) => void;
}

export function TasksPage({
  tasks, serverPage, selectedTasks: selectedSnapshots, selectionLoading = false, updatesPending = false, onPageChange, onPageSizeChange,
  completedWithin24Hours = false,
  onClearCompletedWindow,
  taskSearch,
  taskStatus,
  taskSort,
  taskGroupOpen,
  pendingTaskRetryId,
  pendingTaskDeleteId,
  selectedTaskIds,
  batchPending,
  refreshing,
  lastRefreshedAt,
  onSearchChange,
  onStatusChange,
  onSortChange,
  onToggleGroup,
  onRefresh,
  onOpenQueue,
  onOpenTask,
  onRetryTask,
  onResolveTask,
  onDeleteTask,
  onToggleTaskSelection,
  onSelectVisibleTasks,
  onClearTaskSelection,
  onBatchRetry,
  onBatchDelete
}: TasksPageProps) {
  useAppTimezone();
  const { t } = useTranslation();
  const deferredTaskSearch = useDeferredValue(taskSearch.trim().toLowerCase());
  const [selectionMode, setSelectionMode] = useState(false);

  const scopeTasks = useMemo(() => {
    if (serverPage) return tasks;
    const search = deferredTaskSearch;
    let next = tasks.filter((task) => {
      if (!search) return true;
      const haystack = [
        taskSummary(task),
        task.torrentName,
        task.stage,
        task.stageStatus,
        task.stageLabel,
        task.stageStatusLabel,
        task.qbittorrentState,
        task.qbittorrentState ? qbittorrentStateLabel(task.qbittorrentState, t) : "",
        task.stashScanJobId,
        task.torrentHash,
        task.contentPath,
        task.code,
        task.torrentUrl
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(search);
    });

    if (completedWithin24Hours) {
      const now = Date.now();
      next = next.filter(task => task.stage === "COMPLETED" && task.ingestCompletedAt && Date.parse(task.ingestCompletedAt) >= now - 86400000 && Date.parse(task.ingestCompletedAt) <= now);
    }
    return next;
  }, [deferredTaskSearch, tasks, completedWithin24Hours, t, serverPage]);
  const visibleTasks = useMemo(() => {
    if (serverPage) return tasks;
    let next = scopeTasks;
    const groupByFilter = { running: "active", completed: "completed", failed: "attention", waiting: "waiting" } as const;
    if (taskStatus !== "all") next = next.filter(task => taskGroup(task) === groupByFilter[taskStatus]);

    const sorters: Record<TaskSortKey, (a: DashboardTask, b: DashboardTask) => number> = {
      createdAt: (a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
      updatedAt: (a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt),
      progress: (a, b) => b.progress - a.progress
    };

    return [...next].sort(sorters[taskSort]);
  }, [scopeTasks, serverPage, tasks, taskStatus, taskSort]);

  const taskGroups = useMemo(() => {
    const order: TaskGroupKey[] = ["attention", "waiting", "active", "completed"];
    return order.map((group) => ({
      group,
      tone: taskGroupTone(group),
      description: t(`tasks.groups.${group}.description`),
      tasks: visibleTasks.filter((task) => taskGroup(task) === group)
    }));
  }, [t, visibleTasks]);
  const metrics = useMemo(() => {
    if (serverPage) return { active: serverPage.stats.total - serverPage.stats.completed - serverPage.stats.attention - serverPage.stats.waiting, waiting: serverPage.stats.waiting, completed: serverPage.stats.completed, failed: serverPage.stats.attention };
    const counts = { active: 0, waiting: 0, completed: 0, failed: 0 };
    for (const task of tasks) {
      const group = taskGroup(task);
      counts[group === "attention" ? "failed" : group]++;
    }
    return counts;
  }, [tasks, serverPage]);
  const filterCounts = useMemo(() => {
    if (serverPage !== undefined) return serverPage?.filterCounts;
    const counts = { total: scopeTasks.length, active: 0, waiting: 0, completed: 0, attention: 0 };
    for (const task of scopeTasks) {
      const group = taskGroup(task);
      counts[group === "attention" ? "attention" : group]++;
    }
    return counts;
  }, [serverPage, scopeTasks]);
  const selectedTasks = selectedSnapshots ?? tasks.filter((task) => selectedTaskIds.includes(task.id));
  const { retryIds: retryableIds } = taskBatchEligibility(selectedTasks);
  const hiddenSelectedCount = selectedTaskIds.filter((id) => !visibleTasks.some((task) => task.id === id)).length;

  return (
    <section className="section-band section-band--tasks">
      <FilterCountHeading title={t("tasks.title")} kind="tasks" value={taskStatus} loading={refreshing}
        onChange={value => onStatusChange(value as TaskStatusFilter)} items={[
          { id: "all", label: t("tasks.filters.all"), icon: faLayerGroup, count: filterCounts?.total },
          { id: "running", label: t("tasks.filters.running"), icon: faBarsProgress, count: filterCounts?.active },
          { id: "waiting", label: t("tasks.filters.waiting"), icon: faHourglassHalf, count: filterCounts?.waiting },
          { id: "completed", label: t("tasks.filters.completed"), icon: faCheckDouble, count: filterCounts?.completed },
          { id: "failed", label: t("tasks.filters.failed"), icon: faTriangleExclamation, count: filterCounts?.attention },
        ]} />

      {completedWithin24Hours ? <p className="home-muted">{t("workbench.recentFilter")} · <button type="button" className="ghost-button" onClick={onClearCompletedWindow}>{t("workbench.clearRecentFilter")}</button></p> : null}
      <WorkspaceToolbar selecting={selectionMode} toggleLabel={t(selectionMode ? "taskBatch.exitMultiSelect" : "taskBatch.multiSelect")}
        toggleDisabled={batchPending} onToggleSelection={() => { if (selectionMode) onClearTaskSelection(); setSelectionMode(current => !current); }}
        selectionLabel={t("taskBatch.selectionActions")}
        normal={{
          search: { label: t("tasks.searchPlaceholder"), value: taskSearch, onChange: onSearchChange }, filters: [
            { id: "sort", label: t("tasks.sortLabel"), value: taskSort, onChange: value => onSortChange(value as TaskSortKey), options: (["createdAt", "updatedAt", "progress"] as const).map(value => ({ value, label: t(`tasks.sorts.${value}`) })) },
          ], actions: [
            { id: "refresh", label: t("common.refresh"), icon: workspaceToolbarIcons.refresh, onClick: onRefresh, disabled: refreshing, busy: refreshing, spinning: refreshing },
            { id: "queue", label: t("taskProcessingQueue.open", { count: metrics.failed }), title: t("taskProcessingQueue.entryHelp"), icon: workspaceToolbarIcons.queue, onClick: onOpenQueue, disabled: metrics.failed === 0, showLabel: true },
          ]
        }}
        selection={{
          summary: t("taskBatch.selected", { count: selectedTaskIds.length }) + (hiddenSelectedCount ? ` · ${t(serverPage ? "taskList.hiddenSelection" : "taskBatch.hidden", { count: hiddenSelectedCount })}` : ""), actions: [
            { id: "select-page", label: t(serverPage ? "taskList.selectPage" : "taskBatch.selectVisible", { count: visibleTasks.length }), icon: workspaceToolbarIcons.selectPage, showLabel: true, disabled: batchPending || !visibleTasks.length, onClick: () => onSelectVisibleTasks(visibleTasks.map(task => task.id)) },
            { id: "retry", label: t("taskBatch.retry", { count: retryableIds.length }), icon: workspaceToolbarIcons.retry, showLabel: true, disabled: batchPending || selectionLoading || !retryableIds.length, onClick: () => onBatchRetry(retryableIds) },
            { id: "delete", label: t("taskBatch.delete", { count: selectedTaskIds.length }), icon: workspaceToolbarIcons.delete, showLabel: true, danger: true, disabled: batchPending || !selectedTaskIds.length, onClick: () => onBatchDelete(selectedTaskIds) },
            { id: "clear", label: t("taskBatch.clear"), icon: workspaceToolbarIcons.clearSelection, showLabel: true, disabled: batchPending || !selectedTaskIds.length, onClick: onClearTaskSelection },
          ]
        }} />

      {serverPage ? <Pagination label={t("tasks.title")} position="top" page={serverPage.page} pageSize={serverPage.pageSize}
        totalPages={serverPage.totalPages} totalCount={serverPage.totalCount} loading={refreshing}
        onPageChange={page => onPageChange?.(page)} onPageSizeChange={size => onPageSizeChange?.(size)} /> : null}
      {updatesPending ? <p className="task-sync-meta"><button type="button" className="ghost-button" disabled={refreshing} onClick={onRefresh}>{t("taskList.updates")}</button></p> : null}
      {lastRefreshedAt ? <p className="task-sync-meta">
        {lastRefreshedAt ? t("taskBatch.updatedAt", { time: formatTime(lastRefreshedAt) }) : ""}
      </p> : null}

      {!visibleTasks.length ? (
        <div className="task-grid">
          <article className="empty-card empty-card--wide">
            <h3>{t("tasks.empty.title")}</h3>
            <p>{t("tasks.empty.detail")}</p>
          </article>
        </div>
      ) : null}

      {taskGroups
        .filter((item) => item.tasks.length > 0)
        .map((item) => (
          <TaskGroupSection
            key={item.group}
            group={item.group}
            tone={item.tone}
            description={item.description}
            tasks={item.tasks}
            open={taskGroupOpen[item.group]}
            pendingTaskRetryId={pendingTaskRetryId}
            pendingTaskDeleteId={pendingTaskDeleteId}
            onToggle={onToggleGroup}
            onOpenTask={onOpenTask}
            onRetryTask={onRetryTask}
            onResolveTask={onResolveTask}
            onDeleteTask={onDeleteTask}
            selectionMode={selectionMode}
            selectedTaskIds={selectedTaskIds}
            onToggleTaskSelection={onToggleTaskSelection}
          />
        ))}

      {serverPage ? <Pagination label={t("tasks.title")} position="bottom" page={serverPage.page} pageSize={serverPage.pageSize}
        totalPages={serverPage.totalPages} totalCount={serverPage.totalCount} loading={refreshing}
        onPageChange={page => onPageChange?.(page)} onPageSizeChange={size => onPageSizeChange?.(size)} /> : null}

    </section>
  );
}

export type { TaskStatusFilter, TaskSortKey };
