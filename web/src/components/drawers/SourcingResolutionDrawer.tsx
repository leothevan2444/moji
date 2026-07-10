import { BlockedSourcingResolution } from "../tasks/BlockedSourcingResolution";
import type { DashboardTask } from "../../utils";

interface SourcingResolutionDrawerProps {
  task: DashboardTask | null;
  onResolved: () => void | Promise<void>;
}

export function SourcingResolutionDrawer({ task, onResolved }: SourcingResolutionDrawerProps) {
  if (!task || task.stage !== "SOURCING" || task.stageStatus !== "BLOCKED") {
    return (
      <article className="drawer-card">
        <h3>任务当前无法人工处理</h3>
        <p>任务可能已经离开选种受阻阶段，请关闭后刷新任务列表。</p>
      </article>
    );
  }

  return (
    <div className="drawer-stack">
      <BlockedSourcingResolution task={task} onResolved={onResolved} />
    </div>
  );
}
