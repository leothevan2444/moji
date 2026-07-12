import { TaskCard } from "./TaskCard";
import type { DashboardTask, TaskGroupKey } from "../../utils";
import { useTranslation } from "react-i18next";

interface TaskGroupSectionProps {
  group: TaskGroupKey;
  tone: string;
  description: string;
  tasks: DashboardTask[];
  open: boolean;
  pendingTaskScanId: string | null;
  pendingTaskRetryId: string | null;
  pendingTaskDeleteId: string | null;
  onToggle: (group: TaskGroupKey) => void;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onResolveTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
  onScanAll: () => void;
}

export function TaskGroupSection({
  group,
  tone,
  description,
  tasks,
  open,
  pendingTaskScanId,
  pendingTaskRetryId,
  pendingTaskDeleteId,
  onToggle,
  onOpenTask,
  onScanTask,
  onRetryTask,
  onResolveTask,
  onDeleteTask,
  onScanAll
}: TaskGroupSectionProps) {
  const { t } = useTranslation();
  return (
    <section className="task-group-section">
      <div className="task-group-section__head">
        <div>
          <h3>{t(`tasks.groups.${group}.label`)}</h3>
          <p>{description}</p>
        </div>
        <div className="task-group-section__actions">
          {group === "ingestPending" ? (
            <button type="button" className="ghost-button" onClick={onScanAll}>
              {t("tasks.actions.scanGroup")}
            </button>
          ) : null}
          <span className={`status-chip ${tone}`}>{t("common.items", { count: tasks.length })}</span>
          <button
            type="button"
            className="ghost-button"
            onClick={() => onToggle(group)}
          >
            {open ? t("common.collapse") : t("common.expand")}
          </button>
        </div>
      </div>

      {open ? (
        <div className="task-grid">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              pendingScanId={pendingTaskScanId}
              pendingRetryId={pendingTaskRetryId}
              pendingDeleteId={pendingTaskDeleteId}
              onOpen={onOpenTask}
              onScan={onScanTask}
              onRetry={onRetryTask}
              onResolve={onResolveTask}
              onDelete={onDeleteTask}
            />
          ))}
        </div>
      ) : null}
    </section>
  );
}
