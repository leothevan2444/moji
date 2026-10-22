import { TaskGroupSection } from "../components/tasks";
import {
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
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowsRotate } from "@fortawesome/free-solid-svg-icons/faArrowsRotate";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons/faTrashCan";
import { faXmark } from "@fortawesome/free-solid-svg-icons/faXmark";
import { faListCheck } from "@fortawesome/free-solid-svg-icons/faListCheck";

interface TasksPageProps {
  tasks: DashboardTask[];
  taskSearch: string;
  taskStatus: TaskStatusFilter;
  taskSort: TaskSortKey;
  taskGroupOpen: Record<TaskGroupKey, boolean>;
  pendingTaskRetryId: string | null;
  pendingTaskDeleteId: string | null;
  selectedTaskIds: string[];
  batchPending: boolean;
  refreshing: boolean;
  autoSyncEnabled: boolean;
  autoSyncIntervalSeconds: number;
  lastRefreshedAt: Date | null;
  onSearchChange: (value: string) => void;
  onStatusChange: (status: TaskStatusFilter) => void;
  onSortChange: (sort: TaskSortKey) => void;
  onToggleGroup: (group: TaskGroupKey) => void;
  onRefresh: () => void;
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
  tasks,
  taskSearch,
  taskStatus,
  taskSort,
  taskGroupOpen,
  pendingTaskRetryId,
  pendingTaskDeleteId,
  selectedTaskIds,
  batchPending,
  refreshing,
  autoSyncEnabled,
  autoSyncIntervalSeconds,
  lastRefreshedAt,
  onSearchChange,
  onStatusChange,
  onSortChange,
  onToggleGroup,
  onRefresh,
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
  const { t } = useTranslation();
  const deferredTaskSearch = useDeferredValue(taskSearch.trim().toLowerCase());
  const [selectionMode, setSelectionMode] = useState(false);

  const visibleTasks = useMemo(() => {
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

    return [...next].sort(sorters[taskSort]);
  }, [deferredTaskSearch, taskSort, taskStatus, tasks]);

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
    const counts = { active: 0, waiting: 0, completed: 0, failed: 0 };
    for (const task of tasks) {
      const group = taskGroup(task);
      counts[group === "attention" ? "failed" : group]++;
    }
    return counts;
  }, [tasks]);
  const selectedTasks = tasks.filter((task) => selectedTaskIds.includes(task.id));
  const { retryIds: retryableIds } = taskBatchEligibility(selectedTasks);
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

      </div>

      {selectionMode ? <div className="task-selection-bar" role="region" aria-label={t("taskBatch.selectionActions")}>
        <strong>{t("taskBatch.selected", { count: selectedTaskIds.length })}</strong>
        {hiddenSelectedCount ? <span>{t("taskBatch.hidden", { count: hiddenSelectedCount })}</span> : null}
        <button type="button" className="ghost-button" disabled={batchPending || visibleTasks.length === 0} onClick={() => onSelectVisibleTasks(visibleTasks.map((task) => task.id))}>{t("taskBatch.selectVisible", { count: visibleTasks.length })}</button>
        <button type="button" className="ghost-button" disabled={batchPending || retryableIds.length === 0} onClick={() => onBatchRetry(retryableIds)}><FontAwesomeIcon icon={faArrowsRotate} /> {t("taskBatch.retry", { count: retryableIds.length })}</button>
        <button type="button" className="ghost-button task-ops__button--danger" disabled={batchPending || selectedTaskIds.length === 0} onClick={() => onBatchDelete(selectedTaskIds)}><FontAwesomeIcon icon={faTrashCan} /> {t("taskBatch.delete", { count: selectedTaskIds.length })}</button>
        <button type="button" className="ghost-button" disabled={batchPending || selectedTaskIds.length === 0} onClick={onClearTaskSelection} aria-label={t("taskBatch.clear")}><FontAwesomeIcon icon={faXmark} /> {t("taskBatch.clear")}</button>
      </div> : null}

      <p className="task-sync-meta">
        {t(autoSyncEnabled ? "taskBatch.autoSyncOn" : "taskBatch.autoSyncOff", { seconds: autoSyncIntervalSeconds })}
        {lastRefreshedAt ? ` · ${t("taskBatch.updatedAt", { time: lastRefreshedAt.toLocaleTimeString() })}` : ""}
      </p>

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

    </section>
  );
}

export type { TaskStatusFilter, TaskSortKey };
