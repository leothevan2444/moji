import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faCopy,
  faDownload,
  faRotate,
  faTrashCan,
  faWandMagicSparkles
} from "@fortawesome/free-solid-svg-icons";
import { TaskTimeline } from "./TaskTimeline";
import {
  canTriggerTaskStashScan,
  formatDateTime,
  isCopyableTaskValue,
  isMagnetLink,
  taskFailureSummary,
  taskPresentation,
  taskProgressPercent,
  taskSourceLabel,
  taskSummary,
  deliveryModeLabel,
  transferActionLabel,
  type DashboardTask
} from "../../utils";
import { useTranslation } from "react-i18next";

interface TaskDetailProps {
  task: DashboardTask;
  pendingScan: boolean;
  pendingRetry: boolean;
  pendingDelete: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onSyncAll: () => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onScanAll: () => void;
  onDeleteTask: (taskId: string) => void;
}

export function TaskDetail({
  task,
  pendingScan,
  pendingRetry,
  pendingDelete,
  onCopy,
  onSyncAll,
  onScanTask,
  onRetryTask,
  onScanAll,
  onDeleteTask
}: TaskDetailProps) {
  const { t } = useTranslation();
  const presentation = taskPresentation(task);
  const failure = taskFailureSummary(task);

  return (
    <>
      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>{taskSummary(task)}</h3>
            <p>{presentation.metaLine}</p>
          </div>
          <div className="profile-card__icons">
            <span className="status-chip tone-neutral">{taskSourceLabel(task.source)}</span>
            <span className={`status-chip ${presentation.tone}`}>{presentation.label}</span>
          </div>
        </div>
        <div className="task-detail-hero">
          <div className="task-detail-hero__copy">
            <strong>{presentation.summary}</strong>
            <p>{presentation.detail}</p>
          </div>
          <span className={`task-detail-hero__metric ${presentation.tone}`}>
            {presentation.progressLabel}
          </span>
        </div>
        <div className="progress-shell progress-shell--detail">
          <div className="progress-fill" style={{ width: `${presentation.progressPercent}%` }} />
        </div>
        <dl className="settings-grid">
          <div>
            <dt>{t("taskDetail.created")}</dt>
            <dd>{formatDateTime(task.createdAt)}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.updated")}</dt>
            <dd>{formatDateTime(task.updatedAt)}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.completedAt")}</dt>
            <dd>{formatDateTime(task.downloadCompletedAt)}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.currentProgress")}</dt>
            <dd>{taskProgressPercent(task)}%</dd>
          </div>
        </dl>
      </article>

      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>{t("taskDetail.lifecycle")}</h3>
          </div>
        </div>
        <TaskTimeline steps={presentation.lifecycle} />
      </article>

      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>{t("taskDetail.data")}</h3>
          </div>
        </div>
        <dl className="settings-grid">
          <div>
            <dt className={isMagnetLink(task.torrentUrl) ? "task-inline-label" : undefined}>
              <span>{t(isMagnetLink(task.torrentUrl) ? "taskDetail.magnet" : "taskDetail.taskCode")}</span>
              {isMagnetLink(task.torrentUrl) && task.torrentUrl ? (
                <button
                  type="button"
                  className="task-icon-button"
                  onClick={() => void onCopy(task.torrentUrl || "", t("taskDetail.copiedMagnet"))}
                  aria-label={t("taskDetail.copyMagnet")}
                  title={t("taskDetail.copyMagnet")}
                >
                  <FontAwesomeIcon icon={faCopy} />
                </button>
              ) : null}
            </dt>
            {isMagnetLink(task.torrentUrl) && task.torrentUrl ? (
              <dd>
                <span className="task-inline-value task-inline-value--truncate" title={task.torrentUrl}>
                  {task.torrentUrl}
                </span>
              </dd>
            ) : (
              <dd title={task.code || undefined}>{task.code || "—"}</dd>
            )}
          </div>
          <div>
            <dt>{t("taskDetail.code")}</dt>
            <dd>{task.code || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.source")}</dt>
            <dd>{taskSourceLabel(task.source)}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.savePath")}</dt>
            <dd>{task.savePath || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.category")}</dt>
            <dd>{task.category || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.tags")}</dt>
            <dd>{task.tags || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.contentPath")}</dt>
            <dd>{task.contentPath || "—"}</dd>
          </div>
          <div>
            <dt className={isCopyableTaskValue(task.torrentName) ? "task-inline-label" : undefined}>
              <span>{t("taskDetail.torrentName")}</span>
              {isCopyableTaskValue(task.torrentName) ? (
                <button
                  type="button"
                  className="task-icon-button"
                  onClick={() => void onCopy(task.torrentName || "", t("taskDetail.copiedName"))}
                  aria-label={t("taskDetail.copyName")}
                  title={t("taskDetail.copyName")}
                >
                  <FontAwesomeIcon icon={faCopy} />
                </button>
              ) : null}
            </dt>
            <dd>
              <span className="task-inline-value task-inline-value--truncate" title={task.torrentName || undefined}>
                {task.torrentName || "—"}
              </span>
            </dd>
          </div>
          <div>
            <dt className={isCopyableTaskValue(task.torrentHash) ? "task-inline-label" : undefined}>
              <span>{t("taskDetail.torrentHash")}</span>
              {isCopyableTaskValue(task.torrentHash) ? (
                <button
                  type="button"
                  className="task-icon-button"
                  onClick={() => void onCopy(task.torrentHash || "", t("taskDetail.copiedHash"))}
                  aria-label={t("taskDetail.copyHash")}
                  title={t("taskDetail.copyHash")}
                >
                  <FontAwesomeIcon icon={faCopy} />
                </button>
              ) : null}
            </dt>
            <dd>
              <span className="task-inline-value task-inline-value--truncate" title={task.torrentHash || undefined}>
                {task.torrentHash || "—"}
              </span>
            </dd>
          </div>
          <div>
            <dt>qBittorrent</dt>
            <dd>{task.qbittorrentState || t("taskDetail.waitingSync")}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.progress")}</dt>
            <dd>{Math.round(task.progress * 100)}%</dd>
          </div>
          <div>
            <dt>Stash job</dt>
            <dd>{task.stashScanJobId || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.ingestMode")}</dt>
            <dd>{task.deliveryMode ? deliveryModeLabel(task.deliveryMode) : "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.sourcePath")}</dt>
            <dd>{task.mojiSourcePath || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.action")}</dt>
            <dd>{task.transferAction ? transferActionLabel(task.transferAction) : "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.targetPath")}</dt>
            <dd>{task.mojiTransferPath || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.result")}</dt>
            <dd>{task.mojiTransferPath ? t("taskDetail.resultDone") : t("taskDetail.notTriggered")}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.error")}</dt>
            <dd>{task.transferError || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.scanPath")}</dt>
            <dd>{task.stashScanPath || "—"}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.scanStage")}</dt>
            <dd>{task.stage === "SCANNING" ? t("taskModel.states.scanning") : task.stage === "COMPLETED" ? t("taskModel.states.completed") : t("taskDetail.notStarted")}</dd>
          </div>
          <div>
            <dt>{t("taskDetail.scanHint")}</dt>
            <dd>{task.stashScanHint || "—"}</dd>
          </div>
        </dl>
        <div className={`task-issue ${failure.tone}`}>
          <strong>{failure.title}</strong>
          <span>{failure.detail}</span>
        </div>
      </article>

      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>{t("taskDetail.actions")}</h3>
          </div>
        </div>
        <div className="task-ops">
          {task.stageStatus === "BLOCKED" ? (
            <button
              type="button"
              className="ghost-button task-ops__button"
              onClick={() => onRetryTask(task.id)}
              disabled={pendingRetry}
            >
              <FontAwesomeIcon icon={faRotate} />
              <span>{pendingRetry ? t("taskDetail.retrying") : t("taskDetail.retryBlocked")}</span>
            </button>
          ) : null}
          <button type="button" className="ghost-button task-ops__button" onClick={onSyncAll}>
            <FontAwesomeIcon icon={faRotate} />
            <span>{t("taskDetail.syncAll")}</span>
          </button>
          {canTriggerTaskStashScan(task) ? (
            <button
              type="button"
              className="ghost-button task-ops__button"
              onClick={() => onScanTask(task.id)}
              disabled={pendingScan}
            >
              <FontAwesomeIcon icon={faWandMagicSparkles} />
              <span>{pendingScan ? t("taskDetail.triggering") : t("taskDetail.trigger")}</span>
            </button>
          ) : null}
          <button type="button" className="ghost-button task-ops__button" onClick={onScanAll}>
            <FontAwesomeIcon icon={faDownload} />
            <span>{t("taskDetail.scanPending")}</span>
          </button>
          <button
            type="button"
            className="ghost-button task-ops__button task-ops__button--danger"
            onClick={() => onDeleteTask(task.id)}
            disabled={pendingDelete}
          >
            <FontAwesomeIcon icon={faTrashCan} />
            <span>{pendingDelete ? t("taskDetail.deleting") : t("taskDetail.delete")}</span>
          </button>
        </div>
      </article>
    </>
  );
}
