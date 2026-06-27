import { TaskDetail } from "../tasks";
import type { DashboardTask } from "../../utils";

interface TaskDrawerProps {
  task: DashboardTask | null;
  pendingScan: boolean;
  pendingDelete: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onSyncAll: () => void;
  onScanTask: (taskId: string) => void;
  onScanAll: () => void;
  onDeleteTask: (taskId: string) => void;
}

export function TaskDrawer({
  task,
  pendingScan,
  pendingDelete,
  onCopy,
  onSyncAll,
  onScanTask,
  onScanAll,
  onDeleteTask
}: TaskDrawerProps) {
  return (
    <div className="drawer-stack">
      {task ? (
        <TaskDetail
          task={task}
          pendingScan={pendingScan}
          pendingDelete={pendingDelete}
          onCopy={onCopy}
          onSyncAll={onSyncAll}
          onScanTask={onScanTask}
          onScanAll={onScanAll}
          onDeleteTask={onDeleteTask}
        />
      ) : (
        <article className="drawer-card">
          <h3>还没有选中任务</h3>
          <p>点击任务卡片后，这里会显示详细信息和操作。</p>
        </article>
      )}
    </div>
  );
}
