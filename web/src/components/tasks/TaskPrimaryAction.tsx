import { useTranslation } from "react-i18next";
import { taskPrimaryAction, taskSummary, isWaitingForResource, type DashboardTask } from "../../utils";

interface TaskPrimaryActionProps {
  task: DashboardTask;
  pendingScan?: boolean;
  pendingRetry?: boolean;
  onResolve?: (id: string) => void;
  onFiles: (id: string) => void;
  onRetry?: (id: string) => void;
  onScan?: (id: string) => void;
}

export function TaskPrimaryAction({ task, pendingScan, pendingRetry, onResolve, onFiles, onRetry, onScan }: TaskPrimaryActionProps) {
  const { t } = useTranslation();
  const action = taskPrimaryAction(task);
  if (!action) return null;
  const handler = { resolve: onResolve, files: onFiles, retry: onRetry, ingest: onScan }[action];
  if (!handler) return null;
  const pending = action === "retry" ? pendingRetry : action === "ingest" ? pendingScan : false;
  const label = isWaitingForResource(task) ? t("resourceTracking.checkNow") : t(`taskExperience.actions.${action}`);
  return <>
    {isWaitingForResource(task) && onResolve ? <button type="button" className="ghost-button task-primary-action"
      disabled={pendingRetry} aria-label={`${t("taskExperience.actions.resolve")}: ${taskSummary(task)}`}
      onClick={event => { event.stopPropagation(); onResolve(task.id); }}>
      {t("taskExperience.actions.resolve")}
    </button> : null}
    <button
    type="button"
    className="ghost-button task-primary-action"
    disabled={pending}
    aria-busy={Boolean(pending)}
    aria-label={`${label}: ${taskSummary(task)}`}
    onClick={event => { event.stopPropagation(); handler(task.id); }}
  >{pending ? t("taskExperience.actions.pending") : label}</button>
  </>;
}
