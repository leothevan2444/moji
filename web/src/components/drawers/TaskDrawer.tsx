import { TaskDetail } from "../tasks";
import type { DashboardTask } from "../../utils";
import { useTranslation } from "react-i18next";

interface TaskDrawerProps {
  task: DashboardTask | null;
  pendingScan: boolean;
  pendingRetry: boolean;
  pendingDelete: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onResolveTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
}

export function TaskDrawer({
  task,
  pendingScan,
  pendingRetry,
  pendingDelete,
  onCopy,
  onResolveTask,
  onScanTask,
  onRetryTask,
  onDeleteTask
}: TaskDrawerProps) {
  const { t } = useTranslation();
  return (
    <div className="drawer-stack">
      {task ? (
        <TaskDetail
          task={task}
          pendingScan={pendingScan}
          pendingRetry={pendingRetry}
          pendingDelete={pendingDelete}
          onCopy={onCopy}
          onResolveTask={onResolveTask}
          onScanTask={onScanTask}
          onRetryTask={onRetryTask}
          onDeleteTask={onDeleteTask}
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
