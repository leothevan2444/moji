import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons/faTrashCan";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { TaskFiles } from "./TaskFiles";
import { TaskValue } from "./TaskValue";
import { TaskTimeline } from "./TaskTimeline";
import { TaskPrimaryAction } from "./TaskPrimaryAction";
import {
  formatDateTime, needsTaskFileSelection, taskFailureSummary,
  taskPresentation, taskSourceLabel, taskSummary, transferActionLabel, type DashboardTask
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
  const informationFields: [string, string | null | undefined][] = [
    [t("taskDetail.code"), task.code],
    [t("taskDetail.source"), taskSourceLabel(task.source)],
    [t("taskDetail.savePath"), task.savePath],
    [t("taskDetail.category"), task.category],
    [t("taskDetail.tags"), task.tags],
    [t("taskDetail.contentPath"), task.contentPath],
    ["qBittorrent", task.qbittorrentState],
    [t("taskDetail.sourcePath"), task.mojiSourcePath],
    [t("taskDetail.action"), task.transferAction ? transferActionLabel(task.transferAction) : null],
    [t("taskDetail.targetPath"), task.mojiTransferPath],
    [t("taskDetail.error"), task.transferError],
    [t("taskDetail.scanPath"), task.stashScanPath],
    ["Stash job", task.stashScanJobId],
    [t("taskModel.ingest.generateJob"), task.stashGenerateJobId],
    [t("taskModel.ingest.identifyJob"), task.stashIdentifyJobId],
    [t("taskModel.ingest.scenes"), task.stashSceneIds?.join(", ")],
    [t("taskDetail.scanHint"), task.stashScanHint],
    [t("taskExperience.errorCode"), task.stageErrorCode],
    [t("taskExperience.errorMessage"), task.stageErrorMessage],
    [t("taskExperience.scanError"), task.stashScanError]
  ];

  return <>
    <article className="drawer-card task-detail-overview">
      <div className="drawer-card__head">
        <div className="task-detail-heading"><h3>{taskSummary(task)}</h3>{presentation.metaLine ? <p>{presentation.metaLine}</p> : null}</div>
        <div className="profile-card__icons">
          <span className="status-chip tone-neutral">{taskSourceLabel(task.source)}</span>
          <span className={`status-chip ${presentation.tone}`}>{presentation.label}</span>
        </div>
      </div>
      <div className={`task-detail-hero${blocked ? ` task-detail-hero--blocked ${failure.tone}` : ""}`}>
        <div className="task-detail-hero__copy">
          <strong>{blocked ? failure.title : presentation.summary}</strong>
          <p>{blocked ? failure.detail : presentation.detail}</p>
        </div>
      </div>
      {presentation.showDownloadProgress ? <div className="task-detail-progress">
        <span>{presentation.progressLabel}</span>
        <div className="progress-shell progress-shell--detail" role="progressbar" aria-label={t("taskModel.states.downloading")}
          aria-valuemin={0} aria-valuemax={100} aria-valuenow={presentation.progressPercent}>
          <div className="progress-fill" style={{ width: `${presentation.progressPercent}%` }} />
        </div>
      </div> : null}
      <dl className="settings-grid task-detail-dates">
        <div><dt>{t("taskDetail.created")}</dt><dd>{formatDateTime(task.createdAt)}</dd></div>
        <div><dt>{t("taskDetail.updated")}</dt><dd>{formatDateTime(task.updatedAt)}</dd></div>
        {task.downloadCompletedAt ? <div><dt>{t("taskExperience.downloadCompletedAt")}</dt><dd>{formatDateTime(task.downloadCompletedAt)}</dd></div> : null}
      </dl>
    </article>

    {selectFiles ? files : null}
    <article className="drawer-card">
      <h3>{t("taskDetail.lifecycle")}</h3>
      <TaskTimeline key={task.id} steps={presentation.lifecycle} />
    </article>
    {(task.mojiTransferPath || task.stashSceneIds?.length || task.stage === "COMPLETED") ? <article className="drawer-card">
      <h3>{t("taskExperience.result")}</h3>
      <dl className="settings-grid">
        <TaskValue label={t("taskDetail.targetPath")} value={task.mojiTransferPath} onCopy={onCopy} />
        <TaskValue label={t("taskModel.ingest.scenes")} value={task.stashSceneIds?.join(", ")} onCopy={onCopy} />
      </dl>
    </article> : null}
    {!selectFiles ? files : null}
    <article className="drawer-card task-information" aria-label={t("taskDetail.data")}>
      <div className="drawer-card__head"><h3>{t("taskDetail.data")}</h3></div>
      <dl className="settings-grid">
        {informationFields.map(([label, value], index) => <TaskValue key={index} label={label} value={value} onCopy={onCopy} />)}
      </dl>
    </article>
    <article className="drawer-card task-detail-operations">
      <div className="drawer-card__head"><h3>{t("taskDetail.actions")}</h3></div>
      <div className="task-detail-actions" role="group" aria-label={t("taskDetail.actions")}>
        <TaskPrimaryAction task={task} pendingScan={pendingScan} pendingRetry={pendingRetry}
          onResolve={onResolveTask} onRetry={onRetryTask} onScan={onScanTask}
          onFiles={() => {
            filesRef.current?.scrollIntoView?.({ behavior: "smooth", block: "nearest" });
            (filesRef.current?.querySelector<HTMLInputElement>('input[type="checkbox"]:not(:disabled)') ?? filesRef.current)?.focus();
          }} />
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
