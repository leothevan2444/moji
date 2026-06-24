import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faCopy,
  faDownload,
  faRotate,
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
  type DashboardTask
} from "../../utils";

interface TaskDetailProps {
  task: DashboardTask;
  pendingScan: boolean;
  onCopy: (value: string, successMessage: string) => void | Promise<void>;
  onSyncAll: () => void;
  onScanTask: (taskId: string) => void;
  onScanAll: () => void;
}

export function TaskDetail({
  task,
  pendingScan,
  onCopy,
  onSyncAll,
  onScanTask,
  onScanAll
}: TaskDetailProps) {
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
            <dt>创建时间</dt>
            <dd>{formatDateTime(task.createdAt)}</dd>
          </div>
          <div>
            <dt>最近更新</dt>
            <dd>{formatDateTime(task.updatedAt)}</dd>
          </div>
          <div>
            <dt>完成时间</dt>
            <dd>{formatDateTime(task.completedAt)}</dd>
          </div>
          <div>
            <dt>当前进度</dt>
            <dd>{taskProgressPercent(task)}%</dd>
          </div>
        </dl>
      </article>

      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>生命周期</h3>
          </div>
        </div>
        <TaskTimeline steps={presentation.lifecycle} />
      </article>

      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>任务数据</h3>
          </div>
        </div>
        <dl className="settings-grid">
          <div>
            <dt className={isMagnetLink(task.query) ? "task-inline-label" : undefined}>
              <span>{isMagnetLink(task.query) ? "磁力链接" : "查询文本"}</span>
              {isMagnetLink(task.query) && task.query ? (
                <button
                  type="button"
                  className="task-icon-button"
                  onClick={() => void onCopy(task.query || "", "磁力链接已复制")}
                  aria-label="复制磁力链接"
                  title="复制磁力链接"
                >
                  <FontAwesomeIcon icon={faCopy} />
                </button>
              ) : null}
            </dt>
            {isMagnetLink(task.query) && task.query ? (
              <dd>
                <span className="task-inline-value task-inline-value--truncate" title={task.query}>
                  {task.query}
                </span>
              </dd>
            ) : (
              <dd title={task.query || undefined}>{task.query || "—"}</dd>
            )}
          </div>
          <div>
            <dt>任务来源</dt>
            <dd>{taskSourceLabel(task.source)}</dd>
          </div>
          <div>
            <dt>保存路径</dt>
            <dd>{task.savePath || "—"}</dd>
          </div>
          <div>
            <dt>分类</dt>
            <dd>{task.category || "—"}</dd>
          </div>
          <div>
            <dt>标签</dt>
            <dd>{task.tags || "—"}</dd>
          </div>
          <div>
            <dt>保存内容</dt>
            <dd>{task.contentPath || "—"}</dd>
          </div>
          <div>
            <dt className={isCopyableTaskValue(task.torrentName) ? "task-inline-label" : undefined}>
              <span>Torrent 名称</span>
              {isCopyableTaskValue(task.torrentName) ? (
                <button
                  type="button"
                  className="task-icon-button"
                  onClick={() => void onCopy(task.torrentName || "", "Torrent 名称已复制")}
                  aria-label="复制 Torrent 名称"
                  title="复制 Torrent 名称"
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
              <span>Torrent Hash</span>
              {isCopyableTaskValue(task.torrentHash) ? (
                <button
                  type="button"
                  className="task-icon-button"
                  onClick={() => void onCopy(task.torrentHash || "", "Torrent Hash 已复制")}
                  aria-label="复制 Torrent Hash"
                  title="复制 Torrent Hash"
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
            <dd>{task.qbittorrentState || "待同步"}</dd>
          </div>
          <div>
            <dt>进度</dt>
            <dd>{Math.round(task.progress * 100)}%</dd>
          </div>
          <div>
            <dt>Stash job</dt>
            <dd>{task.stashJobId || "—"}</dd>
          </div>
          <div>
            <dt>入库方式</dt>
            <dd>{task.stashMode || "—"}</dd>
          </div>
          <div>
            <dt>源路径</dt>
            <dd>{task.stashSourcePath || "—"}</dd>
          </div>
          <div>
            <dt>交付动作</dt>
            <dd>{task.stashTransferAction || "—"}</dd>
          </div>
          <div>
            <dt>交付目标</dt>
            <dd>{task.stashTransferPath || "—"}</dd>
          </div>
          <div>
            <dt>交付状态</dt>
            <dd>{task.stashTransferStatus || "未开始"}</dd>
          </div>
          <div>
            <dt>交付错误</dt>
            <dd>{task.stashTransferError || "—"}</dd>
          </div>
          <div>
            <dt>扫描路径</dt>
            <dd>{task.stashScanPath || "—"}</dd>
          </div>
          <div>
            <dt>扫描状态</dt>
            <dd>{task.stashScanStatus || "未开始"}</dd>
          </div>
          <div>
            <dt>扫描提示</dt>
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
            <h3>操作</h3>
          </div>
        </div>
        <div className="task-ops">
          <button type="button" className="ghost-button task-ops__button" onClick={onSyncAll}>
            <FontAwesomeIcon icon={faRotate} />
            <span>同步全部任务进度</span>
          </button>
          {canTriggerTaskStashScan(task) ? (
            <button
              type="button"
              className="ghost-button task-ops__button"
              onClick={() => onScanTask(task.id)}
              disabled={pendingScan}
            >
              <FontAwesomeIcon icon={faWandMagicSparkles} />
              <span>{pendingScan ? "正在触发当前任务扫描" : "触发当前任务扫描"}</span>
            </button>
          ) : null}
          <button type="button" className="ghost-button task-ops__button" onClick={onScanAll}>
            <FontAwesomeIcon icon={faDownload} />
            <span>触发待入库任务扫描</span>
          </button>
        </div>
      </article>
    </>
  );
}
