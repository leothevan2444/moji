import { TaskDetail } from "../tasks";
import type { DashboardTask } from "../../utils";

interface TaskDrawerProps {
  task: DashboardTask | null;
  pendingScan: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onSyncAll: () => void;
  onScanTask: (taskId: string) => void;
  onScanAll: () => void;
}

export function TaskDrawer({ task, pendingScan, onCopy, onSyncAll, onScanTask, onScanAll }: TaskDrawerProps) {
  return (
    <div className="drawer-stack">
      {task ? (
        <TaskDetail
          task={task}
          pendingScan={pendingScan}
          onCopy={onCopy}
          onSyncAll={onSyncAll}
          onScanTask={onScanTask}
          onScanAll={onScanAll}
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
