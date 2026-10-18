import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCopy } from "@fortawesome/free-solid-svg-icons/faCopy";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons/faTrashCan";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { TaskStashScenes } from "./TaskStashScenes";
import { TaskFiles } from "./TaskFiles";
import { taskDiagnostics } from "../../utils/taskDiagnostics";
import { TaskTimeline } from "./TaskTimeline";
import { TaskPrimaryAction } from "./TaskPrimaryAction";
import {
  formatDateTime, needsTaskFileSelection, taskFailureSummary,
  taskPresentation, taskSourceLabel, taskSummary, type DashboardTask
} from "../../utils";

interface TaskDetailProps {
  task: DashboardTask;
  pendingScan: boolean;
  pendingRetry: boolean;
  pendingDelete: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onResolveTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
}

export function TaskDetail({ task, pendingScan, pendingRetry, pendingDelete,
  onCopy, onResolveTask, onScanTask, onRetryTask, onDeleteTask
}: TaskDetailProps) {
  const { t } = useTranslation();
  const presentation = taskPresentation(task);
  const failure = taskFailureSummary(task);
  const blocked = task.stageStatus === "BLOCKED";
  const selectFiles = needsTaskFileSelection(task);
  const filesRef = useRef<HTMLDivElement>(null);
  const files = <div ref={filesRef} tabIndex={-1} className="task-detail-files">
    <TaskFiles key={task.id} task={task} onCopy={onCopy} />
  </div>;

  return <>
    <article className="drawer-card task-detail-overview" aria-label={t("taskDetail.overview")}>
      <div className="drawer-card__head">
        <div className="task-detail-heading"><h3>{t("taskDetail.overview")}</h3>{presentation.metaLine ? <p>{presentation.metaLine}</p> : null}</div>
        <div className="profile-card__icons">
          <span className="status-chip tone-neutral">{taskSourceLabel(task.source)}</span>
          <span className={`status-chip ${presentation.tone}`}>{presentation.label}</span>
        </div>
      </div>
      {blocked ? <div className={`task-detail-hero task-detail-hero--blocked ${failure.tone}`}>
        <div className="task-detail-hero__copy"><strong>{failure.title}</strong></div>
      </div> : null}
      <TaskTimeline key={task.id} steps={presentation.lifecycle} transfer={{ action: task.transferAction, target: task.mojiTransferPath }} onCopy={onCopy}
        downloadProgress={presentation.showDownloadProgress ? { percent: presentation.progressPercent, label: presentation.progressLabel } : undefined} />
      <dl className="settings-grid task-detail-dates">
        <div><dt>{t("taskDetail.created")}</dt><dd>{formatDateTime(task.createdAt)}</dd></div>
        <div><dt>{t("taskDetail.updated")}</dt><dd>{formatDateTime(task.updatedAt)}</dd></div>
      </dl>
    </article>

    {selectFiles ? files : null}
    <TaskStashScenes key={`${task.id}:${task.stage}:${task.stageStatus}:${task.stashIngestPhase}:${task.stashSceneIds?.join(",")}`} task={task} onCopy={onCopy} />
    {!selectFiles ? files : null}
    <article className="drawer-card task-detail-operations">
      <div className="drawer-card__head"><h3>{t("taskDetail.actions")}</h3></div>
      <div className="task-detail-actions" role="group" aria-label={t("taskDetail.actions")}>
        <TaskPrimaryAction task={task} pendingScan={pendingScan} pendingRetry={pendingRetry}
          onResolve={onResolveTask} onRetry={onRetryTask} onScan={onScanTask}
          onFiles={() => {
            filesRef.current?.scrollIntoView?.({ behavior: "smooth", block: "nearest" });
            (filesRef.current?.querySelector<HTMLInputElement>('input[type="checkbox"]:not(:disabled)') ?? filesRef.current)?.focus();
          }} />
        <button type="button" className="ghost-button task-detail-actions__diagnostics"
          onClick={() => void onCopy(taskDiagnostics(task), t("taskInfo.diagnosticsCopied"))}>
          <FontAwesomeIcon icon={faCopy} /><span>{t("taskInfo.copyDiagnostics")}</span>
        </button>
        <button type="button" className="ghost-button task-ops__button--danger task-detail-actions__delete"
          disabled={pendingDelete} aria-busy={pendingDelete}
          aria-label={t("taskUi.card.deleteLabel", { task: taskSummary(task) })}
          onClick={() => onDeleteTask(task.id)}>
          <FontAwesomeIcon icon={faTrashCan} />
          <span>{t(pendingDelete ? "taskDetail.deleting" : "taskDetail.delete")}</span>
        </button>
      </div>
    </article>
  </>;
}
