import { TaskCard } from "./TaskCard";
import type { DashboardTask, TaskGroupKey } from "../../utils";

interface TaskGroupSectionProps {
  group: TaskGroupKey;
  tone: string;
  description: string;
  tasks: DashboardTask[];
  open: boolean;
  triggeringScan: boolean;
  onToggle: (group: TaskGroupKey) => void;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onScanAll: () => void;
}

export function TaskGroupSection({
  group,
  tone,
  description,
  tasks,
  open,
  triggeringScan,
  onToggle,
  onOpenTask,
  onScanTask,
  onScanAll
}: TaskGroupSectionProps) {
  return (
    <section className="task-group-section">
      <div className="task-group-section__head">
        <div>
          <h3>{group}</h3>
          <p>{description}</p>
        </div>
        <div className="task-group-section__actions">
          {group === "待入库" ? (
            <button type="button" className="ghost-button" onClick={onScanAll}>
              全部触发扫描
            </button>
          ) : null}
          <span className={`status-chip ${tone}`}>{tasks.length} 项</span>
          <button
            type="button"
            className="ghost-button"
            onClick={() => onToggle(group)}
          >
            {open ? "收起" : "展开"}
          </button>
        </div>
      </div>

      {open ? (
        <div className="task-grid">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              triggeringScan={triggeringScan}
              onOpen={onOpenTask}
              onScan={onScanTask}
            />
          ))}
        </div>
      ) : null}
    </section>
  );
}
