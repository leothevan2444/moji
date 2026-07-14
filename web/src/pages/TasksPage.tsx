import { TaskGroupSection } from "../components/tasks";
import {
  isScanPending,
  isStatus,
  isTaskActive,
  taskGroup,
  taskGroupTone,
  taskBatchEligibility,
  taskSummary,
  type DashboardTask,
  type TaskGroupKey
} from "../utils";
import { useDeferredValue, useEffect, useMemo, useState } from "react";
import type { TaskSortKey, TaskStatusFilter } from "../types";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowsRotate } from "@fortawesome/free-solid-svg-icons/faArrowsRotate";
import { faEllipsis } from "@fortawesome/free-solid-svg-icons/faEllipsis";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons/faTrashCan";
import { faXmark } from "@fortawesome/free-solid-svg-icons/faXmark";
import { faBoxArchive } from "@fortawesome/free-solid-svg-icons/faBoxArchive";
import { faListCheck } from "@fortawesome/free-solid-svg-icons/faListCheck";

interface TasksPageProps {
  tasks: DashboardTask[];
  metrics: {
    active: number;
    completed: number;
    pendingScans: number;
    failed: number;
  };
  taskSearch: string;
  taskStatus: TaskStatusFilter;
  taskSort: TaskSortKey;
  taskGroupOpen: Record<TaskGroupKey, boolean>;
  pendingTaskScanId: string | null;
  pendingTaskRetryId: string | null;
  pendingTaskDeleteId: string | null;
  selectedTaskIds: string[];
  batchPending: boolean;
  refreshing: boolean;
  syncing: boolean;
  autoSyncEnabled: boolean;
  autoSyncIntervalSeconds: number;
  lastRefreshedAt: Date | null;
  onSearchChange: (value: string) => void;
  onStatusChange: (status: TaskStatusFilter) => void;
  onSortChange: (sort: TaskSortKey) => void;
  onToggleGroup: (group: TaskGroupKey) => void;
  onRefresh: () => void;
  onSync: () => void;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onResolveTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
  onToggleTaskSelection: (taskId: string) => void;
  onSelectVisibleTasks: (taskIds: string[]) => void;
  onClearTaskSelection: () => void;
  onBatchRetry: (taskIds: string[]) => void;
  onBatchIngest: (taskIds: string[]) => void;
  onBatchDelete: (taskIds: string[]) => void;
}

export function TasksPage({
  tasks,
  metrics,
  taskSearch,
  taskStatus,
  taskSort,
  taskGroupOpen,
  pendingTaskScanId,
  pendingTaskRetryId,
  pendingTaskDeleteId,
  selectedTaskIds,
  batchPending,
  refreshing,
  syncing,
  autoSyncEnabled,
  autoSyncIntervalSeconds,
  lastRefreshedAt,
  onSearchChange,
  onStatusChange,
  onSortChange,
  onToggleGroup,
  onRefresh,
  onSync,
  onOpenTask,
  onScanTask,
  onRetryTask,
  onResolveTask,
  onDeleteTask,
  onToggleTaskSelection,
  onSelectVisibleTasks,
  onClearTaskSelection,
  onBatchRetry,
  onBatchIngest,
  onBatchDelete
}: TasksPageProps) {
  const { t } = useTranslation();
  const deferredTaskSearch = useDeferredValue(taskSearch.trim().toLowerCase());
  const [moreActionsOpen, setMoreActionsOpen] = useState(false);
  const [selectionMode, setSelectionMode] = useState(false);

  useEffect(() => {
    if (!moreActionsOpen) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setMoreActionsOpen(false);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [moreActionsOpen]);

  const visibleTasks = useMemo(() => {
    const search = deferredTaskSearch;
    let next = tasks.filter((task) => {
      if (!search) return true;
      const haystack = [
        taskSummary(task),
        task.stage,
        task.stageStatus,
        task.stageLabel,
        task.stageStatusLabel,
        task.qbittorrentState,
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

    if (taskStatus === "running") {
      next = next.filter(isTaskActive);
    } else if (taskStatus === "completed") {
      next = next.filter((task) => isStatus(task, "completed"));
    } else if (taskStatus === "failed") {
      next = next.filter((task) => task.stageStatus === "BLOCKED");
    } else if (taskStatus === "scanPending") {
      next = next.filter(isScanPending);
    }

    const sorters: Record<TaskSortKey, (a: DashboardTask, b: DashboardTask) => number> = {
      createdAt: (a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
      updatedAt: (a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt),
      progress: (a, b) => b.progress - a.progress
    };

    return [...next].sort(sorters[taskSort]);
  }, [deferredTaskSearch, taskSort, taskStatus, tasks]);

  const taskGroups = useMemo(() => {
    const order: TaskGroupKey[] = ["attention", "active", "ingestPending", "completed"];
    return order.map((group) => ({
      group,
      tone: taskGroupTone(group),
      description: t(`tasks.groups.${group}.description`),
      tasks: visibleTasks.filter((task) => taskGroup(task) === group)
    }));
  }, [t, visibleTasks]);
  const selectedTasks = tasks.filter((task) => selectedTaskIds.includes(task.id));
  const { retryIds: retryableIds, ingestIds } = taskBatchEligibility(selectedTasks);
  const hiddenSelectedCount = selectedTaskIds.filter((id) => !visibleTasks.some((task) => task.id === id)).length;

  return (
    <section className="section-band">
      <div className="band-head">
        <div>
          <h2>{t("tasks.title")}</h2>
        </div>
        <p className="band-note">
          {t("tasks.metrics", metrics)}
        </p>
      </div>

      <div className="toolbar-inline">
        <input
          value={taskSearch}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={t("tasks.searchPlaceholder")}
        />
        <select value={taskStatus} onChange={(event) => onStatusChange(event.target.value as TaskStatusFilter)}>
          {(["all", "running", "completed", "failed", "scanPending"] as const).map((value) => <option key={value} value={value}>{t(`tasks.filters.${value}`)}</option>)}
        </select>
        <select value={taskSort} onChange={(event) => onSortChange(event.target.value as TaskSortKey)}>
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
        <button
          type="button"
          className="ghost-button task-icon-button task-icon-button--bordered"
          onClick={() => setMoreActionsOpen(true)}
          aria-haspopup="dialog"
          aria-expanded={moreActionsOpen}
          aria-label={t("taskBatch.more")}
          title={t("taskBatch.more")}
        >
          <FontAwesomeIcon icon={faEllipsis} />
        </button>
      </div>

      {selectionMode ? <div className="task-selection-bar" role="region" aria-label={t("taskBatch.selectionActions")}>
        <strong>{t("taskBatch.selected", { count: selectedTaskIds.length })}</strong>
        {hiddenSelectedCount ? <span>{t("taskBatch.hidden", { count: hiddenSelectedCount })}</span> : null}
        <button type="button" className="ghost-button" disabled={batchPending || visibleTasks.length === 0} onClick={() => onSelectVisibleTasks(visibleTasks.map((task) => task.id))}>{t("taskBatch.selectVisible", { count: visibleTasks.length })}</button>
        <button type="button" className="ghost-button" disabled={batchPending || retryableIds.length === 0} onClick={() => onBatchRetry(retryableIds)}><FontAwesomeIcon icon={faArrowsRotate} /> {t("taskBatch.retry", { count: retryableIds.length })}</button>
        <button type="button" className="ghost-button" disabled={batchPending || ingestIds.length === 0} onClick={() => onBatchIngest(ingestIds)}><FontAwesomeIcon icon={faBoxArchive} /> {t("taskBatch.ingest", { count: ingestIds.length })}</button>
        <button type="button" className="ghost-button task-ops__button--danger" disabled={batchPending || selectedTaskIds.length === 0} onClick={() => onBatchDelete(selectedTaskIds)}><FontAwesomeIcon icon={faTrashCan} /> {t("taskBatch.delete", { count: selectedTaskIds.length })}</button>
        <button type="button" className="ghost-button" disabled={batchPending || selectedTaskIds.length === 0} onClick={onClearTaskSelection} aria-label={t("taskBatch.clear")}><FontAwesomeIcon icon={faXmark} /> {t("taskBatch.clear")}</button>
      </div> : null}

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
            pendingTaskScanId={pendingTaskScanId}
            pendingTaskRetryId={pendingTaskRetryId}
            pendingTaskDeleteId={pendingTaskDeleteId}
            onToggle={onToggleGroup}
            onOpenTask={onOpenTask}
            onScanTask={onScanTask}
            onRetryTask={onRetryTask}
            onResolveTask={onResolveTask}
            onDeleteTask={onDeleteTask}
            onProcessIngest={onBatchIngest}
            batchPending={batchPending}
            selectionMode={selectionMode}
            selectedTaskIds={selectedTaskIds}
            onToggleTaskSelection={onToggleTaskSelection}
          />
        ))}

      <p className="task-sync-meta">
        {t(autoSyncEnabled ? "taskBatch.autoSyncOn" : "taskBatch.autoSyncOff", { seconds: autoSyncIntervalSeconds })}
        {lastRefreshedAt ? ` · ${t("taskBatch.updatedAt", { time: lastRefreshedAt.toLocaleTimeString() })}` : ""}
      </p>

      {moreActionsOpen ? (
        <div className="task-actions-dialog__scrim" onClick={() => setMoreActionsOpen(false)}>
          <section
            className="task-actions-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="task-actions-dialog-title"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="task-actions-dialog__head">
              <h3 id="task-actions-dialog-title">{t("taskBatch.more")}</h3>
              <button type="button" className="ghost-button" onClick={() => setMoreActionsOpen(false)} aria-label={t("common.close")} autoFocus>
                <FontAwesomeIcon icon={faXmark} />
              </button>
            </div>
            <div className="task-actions-dialog__body">
              <button
                type="button"
                className="ghost-button task-actions-dialog__action"
                disabled={syncing}
                onClick={() => {
                  onSync();
                  setMoreActionsOpen(false);
                }}
              >
                <FontAwesomeIcon icon={faArrowsRotate} />
                <span>{syncing ? t("taskBatch.syncing") : t("taskBatch.syncNow")}</span>
              </button>
            </div>
          </section>
        </div>
      ) : null}
    </section>
  );
}

export type { TaskStatusFilter, TaskSortKey };
