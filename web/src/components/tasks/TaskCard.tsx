import {
  canTriggerTaskStashScan,
  formatDateTime,
  taskFailureSummary,
  taskPresentation,
  taskSummary,
  type DashboardTask
} from "../../utils";

interface TaskCardProps {
  task: DashboardTask;
  compact?: boolean;
  triggeringScan?: boolean;
  onOpen: (taskId: string) => void;
  onScan?: (taskId: string) => void;
}

export function TaskCard({ task, compact = false, triggeringScan = false, onOpen, onScan }: TaskCardProps) {
  const presentation = taskPresentation(task);
  const failure = taskFailureSummary(task);

  // 判断是否需要显示进度条（进行中状态）
  const needsProgress =
    presentation.phase === "downloading" ||
    presentation.phase === "scanRunning";

  // 判断是否有真正的错误（排除正常的下载/扫描状态）
  const hasError = failure.tone === "tone-danger" || failure.tone === "tone-warn";

  return (
    <article
      className={`task-card ${presentation.tone} ${compact ? "task-card--compact" : ""}`}
      onClick={() => onOpen(task.id)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onOpen(task.id);
        }
      }}
      role="button"
      tabIndex={0}
      aria-label={`${taskSummary(task)}，状态：${presentation.label}，点击查看详情`}
    >
      <div className="task-card__head">
        <div>
          <h3>{taskSummary(task)}</h3>
          <p>{presentation.metaLine}</p>
        </div>
        <span className={`status-chip ${presentation.tone}`}>{presentation.label}</span>
      </div>

      <div className="task-card__status">
        {hasError ? (
          <div className={`task-status-error ${failure.tone}`}>
            <strong>{failure.title}</strong>
            <span>{failure.detail}</span>
          </div>
        ) : needsProgress ? (
          <div className="task-status-progress">
            <div className="task-status-progress__copy">
              <strong>{presentation.summary}</strong>
              <span>{presentation.progressLabel}</span>
            </div>
            <div className="progress-shell">
              <div className="progress-fill" style={{ width: `${presentation.progressPercent}%` }} />
            </div>
          </div>
        ) : presentation.phase === "completed" ? (
          <div className="task-status-completed">
            {presentation.summary}
          </div>
        ) : null}
      </div>

      <div className="task-card__actions">
        <span>{formatDateTime(task.updatedAt)}</span>
        {canTriggerTaskStashScan(task) && onScan ? (
          <button
            type="button"
            className="ghost-button task-card__action-button"
            onClick={(event) => {
              event.stopPropagation();
              onScan(task.id);
            }}
            disabled={triggeringScan}
            aria-label={`重扫任务：${taskSummary(task)}`}
          >
            {triggeringScan ? "扫描中..." : "重扫"}
          </button>
        ) : null}
      </div>
    </article>
  );
}
