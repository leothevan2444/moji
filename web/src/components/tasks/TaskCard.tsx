import {
  formatDateTime, formatRelative, taskFailureSummary, taskPresentation,
  taskSourceLabel, taskSummary, type DashboardTask
} from "../../utils";
import { taskProgressTime } from "../../utils/taskProgressTime";
import { useTranslation } from "react-i18next";
import { TaskPrimaryAction } from "./TaskPrimaryAction";
import { TaskMoreActions } from "./TaskMoreActions";

interface TaskCardProps {
  task: DashboardTask;
  compact?: boolean;
  pendingScanId?: string | null;
  pendingRetryId?: string | null;
  pendingDeleteId?: string | null;
  onOpen: (taskId: string) => void;
  onScan?: (taskId: string) => void;
  onRetry?: (taskId: string) => void;
  onResolve?: (taskId: string) => void;
  onDelete?: (taskId: string) => void;
  selectionMode?: boolean;
  selected?: boolean;
  onToggleSelection?: (taskId: string) => void;
}

export function TaskCard({ task, compact = false, pendingScanId, pendingRetryId, pendingDeleteId,
  onOpen, onScan, onRetry, onResolve, onDelete, selectionMode = false, selected = false, onToggleSelection
}: TaskCardProps) {
  const { t } = useTranslation();
  const presentation = taskPresentation(task);
  const failure = taskFailureSummary(task);
  const progressTime = taskProgressTime(task);
  const cardTime = progressTime ?? task.createdAt;
  const timeLabel = progressTime ? "taskExperience.updated" : "taskExperience.created";
  const blocked = task.stageStatus === "BLOCKED";
  const activateCard = () => selectionMode && onToggleSelection ? onToggleSelection(task.id) : onOpen(task.id);
  const selectionLabel = t(selected ? "taskBatch.deselectTask" : "taskBatch.selectTask", { task: taskSummary(task) });

  return <article
    className={`task-card ${presentation.tone} ${compact ? "task-card--compact" : ""} ${selectionMode ? "is-selection-mode" : ""} ${selected ? "is-selected" : ""}`}
    onClick={event => {
      if (!selectionMode) event.currentTarget.querySelector<HTMLButtonElement>(".task-card__open")?.focus();
      activateCard();
    }}
  >
    <div className="task-card__head">
      {selectionMode ? <label className="task-card__selection" onClick={event => event.stopPropagation()}><input className="task-card__checkbox" type="checkbox" checked={selected}
        aria-label={selectionLabel} onClick={event => event.stopPropagation()} onChange={() => onToggleSelection?.(task.id)} /></label> : null}
      <div className="task-card__identity">
        <h3><button type="button" className="task-card__open"
          data-task-id={task.id}
          aria-label={selectionMode ? selectionLabel : t("taskUi.card.aria", { task: taskSummary(task), status: presentation.label })}
          aria-pressed={selectionMode ? selected : undefined}
        >{taskSummary(task)}</button></h3>
        {presentation.metaLine ? <p>{presentation.metaLine}</p> : null}
      </div>
      <div className="profile-card__icons">
        <span className="status-chip tone-neutral">{taskSourceLabel(task.source)}</span>
        <span className={`status-chip ${presentation.tone}`}>{presentation.label}</span>
      </div>
    </div>
    <div className="task-card__status">
      {blocked ? <div className={`task-status-error ${failure.tone}`}>
        <strong>{failure.title}</strong><span className="task-card__error-summary">{failure.detail}</span>
      </div> : presentation.showDownloadProgress ? <div className="task-status-progress">
        <div className="task-status-progress__copy"><strong>{presentation.progressLabel}</strong></div>
        <div className="progress-shell" role="progressbar" aria-label={t("taskModel.states.downloading")}
          aria-valuemin={0} aria-valuemax={100} aria-valuenow={presentation.progressPercent}>
          <div className="progress-fill" style={{ width: `${presentation.progressPercent}%` }} />
        </div>
      </div> : <div className={presentation.phase === "completed" ? "task-status-completed" : "task-status-pending"}>
        <strong>{presentation.summary}</strong>
      </div>}
    </div>
    <div className="task-card__actions">
      <time dateTime={cardTime} title={t(timeLabel, { time: formatDateTime(cardTime) })}>
        {t(timeLabel, { time: formatRelative(cardTime) ?? formatDateTime(cardTime) })}
      </time>
      {!selectionMode ? <div className="task-card__right">
        <TaskPrimaryAction task={task} pendingScan={pendingScanId === task.id} pendingRetry={pendingRetryId === task.id}
          onResolve={onResolve} onFiles={onOpen} onScan={onScan} onRetry={onRetry} />
        {onDelete ? <TaskMoreActions task={task} onDelete={onDelete} pendingDelete={pendingDeleteId === task.id} /> : null}
      </div> : null}
    </div>
  </article>;
}
