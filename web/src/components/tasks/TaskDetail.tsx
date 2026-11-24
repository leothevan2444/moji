import { DownloadManager } from "../downloader/DownloadManager";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCopy } from "@fortawesome/free-solid-svg-icons/faCopy";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons/faTrashCan";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { TaskFiles } from "./TaskFiles";
import { TaskSummary } from "./TaskSummary";
import { taskDiagnostics } from "../../utils/taskDiagnostics";
import { TaskPrimaryAction } from "./TaskPrimaryAction";
import {
  canRestartTaskSourcing, taskPrimaryAction, taskSummary, type DashboardTask
} from "../../utils";

interface TaskDetailProps {
  task: DashboardTask;
  pendingRetry: boolean;
  pendingDelete: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onResolveTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onDeleteTask: (taskId: string) => void;
  onRestartTask?: (taskId: string) => void;
  pendingRestart?: boolean;
}

export function TaskDetail({ task, pendingRetry, pendingDelete,
  onCopy, onResolveTask, onRetryTask, onDeleteTask, onRestartTask, pendingRestart
}: TaskDetailProps) {
  const { t } = useTranslation();
  const filesRef = useRef<HTMLDivElement>(null);
  const files = <div ref={filesRef} tabIndex={-1} className="task-detail-files">
    <TaskFiles key={task.id} task={task} onCopy={onCopy}
      torrentControls={task.torrentHash?.trim() ? <DownloadManager key={task.id} taskId={task.id} task={task} onCopy={onCopy} embedded /> : undefined} />
  </div>;

  return <>
    <TaskSummary key={task.id} task={task} onCopy={onCopy} action={taskPrimaryAction(task) ? <TaskPrimaryAction task={task} pendingRetry={pendingRetry}
          onResolve={onResolveTask} onRetry={onRetryTask} onFiles={() => {
            filesRef.current?.scrollIntoView?.({ behavior: "smooth", block: "nearest" });
            (filesRef.current?.querySelector<HTMLInputElement>('input[type="checkbox"]:not(:disabled)') ?? filesRef.current)?.focus();
          }} /> : undefined} />

    {files}
    <article className="drawer-card task-detail-operations">
      <div className="drawer-card__head"><h3>{t("taskDetail.actions")}</h3></div>
      <div className="task-detail-actions" role="group" aria-label={t("taskDetail.actions")}>
        {onRestartTask && canRestartTaskSourcing(task) ? <button type="button" className="ghost-button"
          disabled={pendingRestart || pendingDelete || pendingRetry} aria-busy={Boolean(pendingRestart)}
          onClick={() => onRestartTask(task.id)}>{t(pendingRestart ? "taskRestart.pending" : "taskRestart.action")}</button> : null}
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
