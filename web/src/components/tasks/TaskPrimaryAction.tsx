import { useTranslation } from "react-i18next";
import { taskPrimaryAction, taskSummary, isWaitingForResource, type DashboardTask } from "../../utils";

interface TaskPrimaryActionProps {
  task: DashboardTask;
  compact?: boolean;
  pendingRetry?: boolean;
  onResolve?: (id: string) => void;
  onFiles: (id: string) => void;
  onRetry?: (id: string) => void;
}

export function TaskPrimaryAction({ task, compact = false, pendingRetry, onResolve, onFiles, onRetry }: TaskPrimaryActionProps) {
  const { t } = useTranslation();
  const action = taskPrimaryAction(task);
  if (!action) return null;
  const handler = { resolve: onResolve, files: onFiles, retry: onRetry }[action];
  if (!handler) return null;
  const pending = action === "retry" ? pendingRetry : false;
  const label = isWaitingForResource(task) ? t("resourceTracking.checkNow") : t(`taskExperience.actions.${action}`);
  const visibleLabel = compact ? t(`taskCardActions.${isWaitingForResource(task) ? "check" : action}`) : label;
  return <>
    {isWaitingForResource(task) && onResolve ? <button type="button" className="ghost-button task-primary-action task-primary-action--secondary"
      title={compact ? t("taskActionHints.resolve") : undefined}
      disabled={pendingRetry} aria-label={`${t("taskExperience.actions.resolve")}: ${taskSummary(task)}`}
      onClick={event => { event.stopPropagation(); onResolve(task.id); }}>
      {t(compact ? "taskCardActions.resolve" : "taskExperience.actions.resolve")}
    </button> : null}
    <button
    type="button"
    className="ghost-button task-primary-action"
    title={compact ? t(`taskActionHints.${isWaitingForResource(task) ? "check" : action}`) : undefined}
    disabled={pending}
    aria-busy={Boolean(pending)}
    aria-label={`${label}: ${taskSummary(task)}`}
    onClick={event => { event.stopPropagation(); handler(task.id); }}
  >{pending ? t(compact ? "taskCardActions.pending" : "taskExperience.actions.pending") : visibleLabel}</button>
  </>;
}
