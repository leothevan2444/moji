import { TaskCard } from "./TaskCard";
import type { DashboardTask, TaskGroupKey } from "../../utils";
import { useTranslation } from "react-i18next";

interface TaskGroupSectionProps {
  group: TaskGroupKey;
  tone: string;
  description: string;
  tasks: DashboardTask[];
  open: boolean;
  pendingTaskRetryId: string | null;
  pendingTaskDeleteId: string | null;
  onToggle: (group: TaskGroupKey) => void;
  onOpenTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onResolveTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
  selectionMode: boolean;
  selectedTaskIds: string[];
  onToggleTaskSelection: (taskId: string) => void;
}

export function TaskGroupSection({
  group,
  tone,
  description,
  tasks,
  open,
  pendingTaskRetryId,
  pendingTaskDeleteId,
  onToggle,
  onOpenTask,
  onRetryTask,
  onResolveTask,
  onDeleteTask,
  selectionMode,
  selectedTaskIds,
  onToggleTaskSelection
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
              pendingRetryId={pendingTaskRetryId === task.id ? task.id : null}
              pendingDeleteId={pendingTaskDeleteId === task.id ? task.id : null}
              onOpen={onOpenTask}
              onRetry={onRetryTask}
              onResolve={onResolveTask}
              onDelete={onDeleteTask}
              selectionMode={selectionMode}
              selected={selectedTaskIds.includes(task.id)}
              onToggleSelection={onToggleTaskSelection}
            />
          ))}
        </div>
      ) : null}
    </section>
  );
}
