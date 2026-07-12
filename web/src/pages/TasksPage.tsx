import { TaskGroupSection } from "../components/tasks";
import {
  isScanPending,
  isStatus,
  isTaskActive,
  taskGroup,
  taskGroupTone,
  taskSummary,
  type DashboardTask,
  type TaskGroupKey
} from "../utils";
import { useDeferredValue, useMemo } from "react";
import type { TaskSortKey, TaskStatusFilter } from "../types";
import { useTranslation } from "react-i18next";

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
  retryingBlockedTasks: boolean;
  onSearchChange: (value: string) => void;
  onStatusChange: (status: TaskStatusFilter) => void;
  onSortChange: (sort: TaskSortKey) => void;
  onToggleGroup: (group: TaskGroupKey) => void;
  onRefresh: () => void;
  onSync: () => void;
  onScanAll: () => void;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onResolveTask: (taskId: string) => void;
  onRetryBlockedTasks: () => void;
  onDeleteTask: (taskId: string) => void;
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
  retryingBlockedTasks,
  onSearchChange,
  onStatusChange,
  onSortChange,
  onToggleGroup,
  onRefresh,
  onSync,
  onScanAll,
  onOpenTask,
  onScanTask,
  onRetryTask,
  onResolveTask,
  onRetryBlockedTasks,
  onDeleteTask
}: TasksPageProps) {
  const { t } = useTranslation();
  const deferredTaskSearch = useDeferredValue(taskSearch.trim().toLowerCase());

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
        <button type="button" className="ghost-button" onClick={onRefresh}>
          {t("common.refresh")}
        </button>
        <button type="button" className="ghost-button" onClick={onSync}>
          {t("tasks.actions.sync")}
        </button>
        <button type="button" className="ghost-button" onClick={onScanAll}>
          {t("tasks.actions.scanAll")}
        </button>
        <button
          type="button"
          className="ghost-button"
          onClick={onRetryBlockedTasks}
          disabled={retryingBlockedTasks || metrics.failed === 0}
        >
          {retryingBlockedTasks ? t("tasks.actions.retryingBlocked") : `${t("tasks.actions.retryBlocked")}${metrics.failed > 0 ? ` (${metrics.failed})` : ""}`}
        </button>
      </div>

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
            onScanAll={onScanAll}
          />
        ))}
    </section>
  );
}

export type { TaskStatusFilter, TaskSortKey };
