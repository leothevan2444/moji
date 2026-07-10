import { TaskGroupSection } from "../components/tasks";
import {
  isScanPending,
  isStatus,
  isTaskActive,
  taskGroup,
  taskGroupDescription,
  taskGroupTone,
  taskSummary,
  type DashboardTask,
  type TaskGroupKey
} from "../utils";
import { useDeferredValue, useMemo } from "react";
import type { TaskSortKey, TaskStatusFilter } from "../types";

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
  onRetryBlockedTasks,
  onDeleteTask
}: TasksPageProps) {
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

    if (taskStatus === "运行中") {
      next = next.filter(isTaskActive);
    } else if (taskStatus === "完成") {
      next = next.filter((task) => isStatus(task, "completed"));
    } else if (taskStatus === "失败") {
      next = next.filter((task) => task.stageStatus === "BLOCKED");
    } else if (taskStatus === "待扫描") {
      next = next.filter(isScanPending);
    }

    const sorters: Record<TaskSortKey, (a: DashboardTask, b: DashboardTask) => number> = {
      最新: (a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
      更新时间: (a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt),
      进度: (a, b) => b.progress - a.progress
    };

    return [...next].sort(sorters[taskSort]);
  }, [deferredTaskSearch, taskSort, taskStatus, tasks]);

  const taskGroups = useMemo(() => {
    const order: TaskGroupKey[] = ["需处理", "运行中", "待入库", "已完成"];
    return order.map((group) => ({
      group,
      tone: taskGroupTone(group),
      description: taskGroupDescription(group),
      tasks: visibleTasks.filter((task) => taskGroup(task) === group)
    }));
  }, [visibleTasks]);

  return (
    <section className="section-band">
      <div className="band-head">
        <div>
          <h2>工作台</h2>
        </div>
        <p className="band-note">
          活跃 {metrics.active} · 完成 {metrics.completed} · 待扫 {metrics.pendingScans} · 失败 {metrics.failed}
        </p>
      </div>

      <div className="toolbar-inline">
        <input
          value={taskSearch}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="搜索任务、番号、tracker、状态"
        />
        <select value={taskStatus} onChange={(event) => onStatusChange(event.target.value as TaskStatusFilter)}>
          <option value="全部">全部</option>
          <option value="运行中">运行中</option>
          <option value="完成">完成</option>
          <option value="失败">失败</option>
          <option value="待扫描">待扫描</option>
        </select>
        <select value={taskSort} onChange={(event) => onSortChange(event.target.value as TaskSortKey)}>
          <option value="最新">最新</option>
          <option value="更新时间">更新时间</option>
          <option value="进度">进度</option>
        </select>
        <button type="button" className="ghost-button" onClick={onRefresh}>
          刷新
        </button>
        <button type="button" className="ghost-button" onClick={onSync}>
          同步进度
        </button>
        <button type="button" className="ghost-button" onClick={onScanAll}>
          触发扫描
        </button>
        <button
          type="button"
          className="ghost-button"
          onClick={onRetryBlockedTasks}
          disabled={retryingBlockedTasks || metrics.failed === 0}
        >
          {retryingBlockedTasks ? "正在重试受阻任务..." : `重试受阻任务${metrics.failed > 0 ? ` (${metrics.failed})` : ""}`}
        </button>
      </div>

      {!visibleTasks.length ? (
        <div className="task-grid">
          <article className="empty-card empty-card--wide">
            <h3>没有匹配的任务</h3>
            <p>换个过滤条件，或者先去发现区创建任务。</p>
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
            onDeleteTask={onDeleteTask}
            onScanAll={onScanAll}
          />
        ))}
    </section>
  );
}

export type { TaskStatusFilter, TaskSortKey };
