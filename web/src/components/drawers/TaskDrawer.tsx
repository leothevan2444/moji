import { TaskDetail } from "../tasks";
import type { DashboardTask } from "../../utils";
import { useTranslation } from "react-i18next";

interface TaskDrawerProps {
  task: DashboardTask | null;
  pendingScan: boolean;
  pendingRetry: boolean;
  pendingDelete: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onSyncAll: () => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onScanAll: () => void;
  onDeleteTask: (taskId: string) => void;
}

export function TaskDrawer({
  task,
  pendingScan,
  pendingRetry,
  pendingDelete,
  onCopy,
  onSyncAll,
  onScanTask,
  onRetryTask,
  onScanAll,
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
          onSyncAll={onSyncAll}
          onScanTask={onScanTask}
          onRetryTask={onRetryTask}
          onScanAll={onScanAll}
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
