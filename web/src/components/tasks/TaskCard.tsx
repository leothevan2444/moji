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
  pendingRetryId?: string | null;
  pendingDeleteId?: string | null;
  onOpen: (taskId: string) => void;
  onScan?: (taskId: string) => void;
  onRetry?: (taskId: string) => void;
  onDelete?: (taskId: string) => void;
}

export function TaskCard({
  task,
  compact = false,
  pendingScanId = null,
  pendingRetryId = null,
  pendingDeleteId = null,
  onOpen,
  onScan,
  onRetry,
  onDelete
}: TaskCardProps) {
  const presentation = taskPresentation(task);
  const failure = taskFailureSummary(task);
  const state = taskCardState(presentation, failure);
  const isPendingScan = pendingScanId === task.id;
  const isPendingRetry = pendingRetryId === task.id;
  const isPendingDelete = pendingDeleteId === task.id;

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

        <div className="task-card__right">
          {task.stageStatus === "BLOCKED" && onRetry ? (
            <button
              type="button"
              className="ghost-button task-card__action-button"
              onClick={(event) => {
                event.stopPropagation();
                onRetry(task.id);
              }}
              onKeyDown={(event) => event.stopPropagation()}
              disabled={isPendingRetry}
              aria-label={`重试任务：${taskSummary(task)}`}
            >
              {isPendingRetry ? "重试中..." : "重试"}
            </button>
          ) : null}
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
          {onDelete ? (
            <button
              type="button"
              className="ghost-button task-card__action-button task-card__action-button--danger"
              onClick={(event) => {
                event.stopPropagation();
                onDelete(task.id);
              }}
              disabled={isPendingDelete}
              aria-label={`删除任务：${taskSummary(task)}`}
            >
              {isPendingDelete ? "删除中..." : "删除"}
            </button>
          ) : null}
        </div>
      </div>
    </article>
  );
}
