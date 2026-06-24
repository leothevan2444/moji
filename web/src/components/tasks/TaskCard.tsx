import {
  canTriggerTaskStashScan,
  formatDateTime,
  taskCardState,
  taskFailureSummary,
  taskPresentation,
  taskSourceLabel,
  taskSummary,
  type DashboardTask
} from "../../utils";

interface TaskCardProps {
  task: DashboardTask;
  compact?: boolean;
  pendingScanId?: string | null;
  onOpen: (taskId: string) => void;
  onScan?: (taskId: string) => void;
}

export function TaskCard({ task, compact = false, pendingScanId = null, onOpen, onScan }: TaskCardProps) {
  const presentation = taskPresentation(task);
  const failure = taskFailureSummary(task);
  const state = taskCardState(presentation, failure);
  const isPendingScan = pendingScanId === task.id;

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
        <div className="profile-card__icons">
          <span className="status-chip tone-neutral">{taskSourceLabel(task.source)}</span>
          <span className={`status-chip ${presentation.tone}`}>{presentation.label}</span>
        </div>
      </div>

      <div className="task-card__status">
        {state === "error" ? (
          <div className={`task-status-error ${failure.tone}`}>
            <strong>{failure.title}</strong>
            <span>{failure.detail}</span>
          </div>
        ) : state === "progress" ? (
          <div className="task-status-progress">
            <div className="task-status-progress__copy">
              <strong>{presentation.summary}</strong>
              <span>{presentation.progressLabel}</span>
            </div>
            <div className="progress-shell">
              <div className="progress-fill" style={{ width: `${presentation.progressPercent}%` }} />
            </div>
          </div>
        ) : state === "completed" ? (
          <div className="task-status-completed">
            {presentation.summary}
          </div>
        ) : (
          <div className={`task-status-pending ${failure.tone}`}>
            <strong>{failure.title}</strong>
            <span>{failure.detail}</span>
          </div>
        )}
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
            disabled={isPendingScan}
            aria-label={`重扫任务：${taskSummary(task)}`}
          >
            {isPendingScan ? "扫描中..." : "重扫"}
          </button>
        ) : null}
      </div>
    </article>
  );
}
