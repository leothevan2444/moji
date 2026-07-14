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
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faEllipsis } from "@fortawesome/free-solid-svg-icons/faEllipsis";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons/faTrashCan";
import { Menu } from "../common/Menu";

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

export function TaskCard({
  task,
  compact = false,
  pendingScanId = null,
  pendingRetryId = null,
  pendingDeleteId = null,
  onOpen,
  onScan,
  onRetry,
  onResolve,
  onDelete,
  selectionMode = false,
  selected = false,
  onToggleSelection
}: TaskCardProps) {
  const { t } = useTranslation();
  const presentation = taskPresentation(task);
  const failure = taskFailureSummary(task);
  const state = taskCardState(presentation, failure);
  const isPendingScan = pendingScanId === task.id;
  const isPendingRetry = pendingRetryId === task.id;
  const isPendingDelete = pendingDeleteId === task.id;
  const activateCard = () => {
    if (selectionMode && onToggleSelection) onToggleSelection(task.id);
    else onOpen(task.id);
  };

  return (
    <article
      className={`task-card ${presentation.tone} ${compact ? "task-card--compact" : ""} ${selectionMode ? "is-selection-mode" : ""} ${selected ? "is-selected" : ""}`}
      onClick={activateCard}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          activateCard();
        }
      }}
      role="button"
      tabIndex={0}
      aria-pressed={selectionMode ? selected : undefined}
      aria-label={selectionMode
        ? t(selected ? "taskBatch.deselectTask" : "taskBatch.selectTask", { task: taskSummary(task) })
        : t("taskUi.card.aria", { task: taskSummary(task), status: presentation.label })}
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
          {task.stage === "SOURCING" && task.stageStatus === "BLOCKED" && onResolve ? (
            <button
              type="button"
              className="ghost-button task-card__action-button"
              onClick={(event) => {
                event.stopPropagation();
                onResolve(task.id);
              }}
              onKeyDown={(event) => event.stopPropagation()}
              aria-label={t("taskUi.card.resolveLabel", { task: taskSummary(task) })}
            >
              {t("taskUi.card.resolve")}
            </button>
          ) : task.stageStatus === "BLOCKED" && onRetry ? (
            <button
              type="button"
              className="ghost-button task-card__action-button"
              onClick={(event) => {
                event.stopPropagation();
                onRetry(task.id);
              }}
              onKeyDown={(event) => event.stopPropagation()}
              disabled={isPendingRetry}
              aria-label={t("taskUi.card.retryLabel", { task: taskSummary(task) })}
            >
              {isPendingRetry ? t("taskUi.card.retrying") : t("taskUi.card.retry")}
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
              aria-label={t("taskUi.card.scanLabel", { task: taskSummary(task) })}
            >
              {isPendingScan ? t("taskUi.card.scanning") : t("taskUi.card.scan")}
            </button>
          ) : null}
          {onDelete ? <div onClick={(event) => event.stopPropagation()} onKeyDown={(event) => event.stopPropagation()}>
            <Menu
              triggerLabel={<FontAwesomeIcon icon={faEllipsis} />}
              triggerAriaLabel={t("taskBatch.moreActions", { task: taskSummary(task) })}
              ariaLabel={t("taskBatch.moreActions", { task: taskSummary(task) })}
              items={[{ key: "delete", disabled: isPendingDelete, onSelect: () => onDelete(task.id), label: <><FontAwesomeIcon icon={faTrashCan} /> {isPendingDelete ? t("taskUi.card.deleting") : t("taskUi.card.delete")}</> }]}
            />
          </div> : null}
        </div>
      </div>
    </article>
  );
}
