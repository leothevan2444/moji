import { TaskDetail } from "../tasks";
import type { DashboardTask } from "../../utils";
import { useTranslation } from "react-i18next";

interface TaskDrawerProps {
  task: DashboardTask | null;
  pendingRetry: boolean;
  pendingDelete: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onResolveTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
  onRestartTask?: (taskId: string) => void;
  pendingRestart?: boolean;
}

export function TaskDrawer({
  task,
  pendingRetry,
  pendingDelete,
  onCopy,
  onResolveTask,
  onRetryTask,
  onDeleteTask,
  onRestartTask,
  pendingRestart
}: TaskDrawerProps) {
  const { t } = useTranslation();
  return (
    <div className="drawer-stack">
      {task ? (
        <TaskDetail
          task={task}
          pendingRetry={pendingRetry}
          pendingDelete={pendingDelete}
          onCopy={onCopy}
          onResolveTask={onResolveTask}
          onRetryTask={onRetryTask}
          onDeleteTask={onDeleteTask}
          onRestartTask={onRestartTask}
          pendingRestart={pendingRestart}
        />
      ) : (
        <article className="drawer-card">
          <h3>{t("taskUi.noneSelected")}</h3>
          <p>{t("taskUi.noneSelectedDetail")}</p>
        </article>
      )}
    </div>
  );
}
