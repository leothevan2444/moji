import { Pagination } from "../components/common/pagination/Pagination";
import { ResponsivePageHeading } from "../components/layout/ResponsivePageHeading";
import { faBarsProgress, faHourglassHalf, faCheckDouble, faTriangleExclamation } from "@fortawesome/free-solid-svg-icons";
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
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowsRotate } from "@fortawesome/free-solid-svg-icons/faArrowsRotate";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons/faTrashCan";
import { faXmark } from "@fortawesome/free-solid-svg-icons/faXmark";
import { faListCheck } from "@fortawesome/free-solid-svg-icons/faListCheck";
import { faClipboardCheck } from "@fortawesome/free-solid-svg-icons/faClipboardCheck";

interface TasksPageProps {
  serverPage?: { page: number; pageSize: number; totalPages: number; totalCount: number; stats: { total: number; active: number; completed: number; attention: number; waiting: number } };
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

  const visibleTasks = useMemo(() => {
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

    const groupByFilter = { running: "active", completed: "completed", failed: "attention", waiting: "waiting" } as const;
    if (taskStatus !== "all") next = next.filter(task => taskGroup(task) === groupByFilter[taskStatus]);

    const sorters: Record<TaskSortKey, (a: DashboardTask, b: DashboardTask) => number> = {
      createdAt: (a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
      updatedAt: (a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt),
      progress: (a, b) => b.progress - a.progress
    };

    if (completedWithin24Hours) {
      const now = Date.now();
      next = next.filter(task => task.stage === "COMPLETED" && task.ingestCompletedAt && Date.parse(task.ingestCompletedAt) >= now - 86400000 && Date.parse(task.ingestCompletedAt) <= now);
    }
    return [...next].sort(sorters[taskSort]);
  }, [deferredTaskSearch, taskSort, taskStatus, tasks, completedWithin24Hours, t, serverPage]);

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
  const selectedTasks = selectedSnapshots ?? tasks.filter((task) => selectedTaskIds.includes(task.id));
  const { retryIds: retryableIds } = taskBatchEligibility(selectedTasks);
  const hiddenSelectedCount = selectedTaskIds.filter((id) => !visibleTasks.some((task) => task.id === id)).length;

  return (
    <section className="section-band section-band--tasks">
      <ResponsivePageHeading title={t("tasks.title")} variant="metrics" items={[
        { id: "active", label: t("tasks.filters.running"), icon: faBarsProgress, count: metrics.active },
        { id: "waiting", label: t("tasks.filters.waiting"), icon: faHourglassHalf, count: metrics.waiting },
        { id: "completed", label: t("tasks.filters.completed"), icon: faCheckDouble, count: metrics.completed },
        { id: "attention", label: t("tasks.filters.failed"), icon: faTriangleExclamation, count: metrics.failed },
      ]} />

      {completedWithin24Hours ? <p className="home-muted">{t("workbench.recentFilter")} · <button type="button" className="ghost-button" onClick={onClearCompletedWindow}>{t("workbench.clearRecentFilter")}</button></p> : null}
      <div className="toolbar-inline toolbar-inline--tasks">
        <input
          aria-label={t("tasks.searchPlaceholder")}
          value={taskSearch}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={t("tasks.searchPlaceholder")}
        />
        <select aria-label={t("tasks.statusLabel")} value={taskStatus} onChange={(event) => onStatusChange(event.target.value as TaskStatusFilter)}>
          {(["all", "failed", "waiting", "running", "completed"] as const).map((value) => <option key={value} value={value}>{t(`tasks.filters.${value}`)}</option>)}
        </select>
        <select aria-label={t("tasks.sortLabel")} value={taskSort} onChange={(event) => onSortChange(event.target.value as TaskSortKey)}>
          {(["createdAt", "updatedAt", "progress"] as const).map((value) => <option key={value} value={value}>{t(`tasks.sorts.${value}`)}</option>)}
        </select>
        <div className="task-toolbar-actions">
          <button
            type="button"
            className={`ghost-button task-icon-button task-icon-button--bordered${selectionMode ? " is-active" : ""}`}
            onClick={() => {
              if (selectionMode) onClearTaskSelection();
              setSelectionMode((current) => !current);
            }}
            aria-pressed={selectionMode}
            aria-label={selectionMode ? t("taskBatch.exitMultiSelect") : t("taskBatch.multiSelect")}
            title={selectionMode ? t("taskBatch.exitMultiSelect") : t("taskBatch.multiSelect")}
          >
            <FontAwesomeIcon icon={faListCheck} />
          </button>
          <button type="button" className="ghost-button task-icon-button task-icon-button--bordered" onClick={onRefresh} disabled={refreshing} aria-label={t("common.refresh")} title={t("common.refresh")}>
            <FontAwesomeIcon icon={faRotate} className={refreshing ? "is-spinning" : undefined} />
          </button>
        </div>

        <button type="button" className="ghost-button task-queue-entry" onClick={onOpenQueue}
          disabled={metrics.failed === 0} title={t("taskProcessingQueue.entryHelp")} aria-label={t("taskProcessingQueue.open", { count: metrics.failed })}>
          <FontAwesomeIcon icon={faClipboardCheck} aria-hidden="true" />
          {t("taskProcessingQueue.title")} <span>{metrics.failed}</span>
        </button>

      </div>

      {selectionMode ? <div className="task-selection-bar" role="region" aria-label={t("taskBatch.selectionActions")}>
        <strong>{t("taskBatch.selected", { count: selectedTaskIds.length })}</strong>
        {hiddenSelectedCount ? <span>{t(serverPage ? "taskList.hiddenSelection" : "taskBatch.hidden", { count: hiddenSelectedCount })}</span> : null}
        <button type="button" className="ghost-button" disabled={batchPending || visibleTasks.length === 0} onClick={() => onSelectVisibleTasks(visibleTasks.map((task) => task.id))}>{t(serverPage ? "taskList.selectPage" : "taskBatch.selectVisible", { count: visibleTasks.length })}</button>
        <button type="button" className="ghost-button" disabled={batchPending || selectionLoading || retryableIds.length === 0} onClick={() => onBatchRetry(retryableIds)}><FontAwesomeIcon icon={faArrowsRotate} /> {t("taskBatch.retry", { count: retryableIds.length })}</button>
        <button type="button" className="ghost-button task-ops__button--danger" disabled={batchPending || selectedTaskIds.length === 0} onClick={() => onBatchDelete(selectedTaskIds)}><FontAwesomeIcon icon={faTrashCan} /> {t("taskBatch.delete", { count: selectedTaskIds.length })}</button>
        <button type="button" className="ghost-button" disabled={batchPending || selectedTaskIds.length === 0} onClick={onClearTaskSelection} aria-label={t("taskBatch.clear")}><FontAwesomeIcon icon={faXmark} /> {t("taskBatch.clear")}</button>
      </div> : null}

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
