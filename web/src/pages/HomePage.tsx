import { TaskCard } from "../components/tasks";
import { formatDateTime, isScanPending, isStatus, serviceStatus, type DashboardTask } from "../utils";
import type { DashboardDocumentQuery } from "../graphql/generated/graphql";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type RuntimeSettingsStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]>;

interface HomePageProps {
  tasks: DashboardTask[];
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeSettingsStatus | null;
  lastCheckedAt: string | null | undefined;
  pendingTaskScanId: string | null;
  onRefresh: () => void;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
}

export function HomePage({
  tasks,
  runtimeSettings,
  runtimeStatus,
  lastCheckedAt,
  pendingTaskScanId,
  onRefresh,
  onOpenTask,
  onScanTask
}: HomePageProps) {
  const dependencyCards = runtimeSettings && runtimeStatus
    ? [
        {
          name: "Stash",
          ...serviceStatus(runtimeStatus.stash.configured, runtimeStatus.stash.enabled),
          detail: runtimeStatus.stash.enabled
            ? `入库模式: ${runtimeSettings.stash.mode}`
            : runtimeStatus.stash.configured
              ? "配置已存在，但运行时尚未启用"
              : "缺少 Stash URL 或入库策略"
        },
        {
          name: "Jackett",
          ...serviceStatus(runtimeStatus.jackett.configured, runtimeStatus.jackett.enabled),
          detail: runtimeStatus.jackett.enabled
            ? `索引地址: ${runtimeSettings.jackett.url || "未设置"}`
            : "缺少 URL 或 API key"
        },
        {
          name: "qBittorrent",
          ...serviceStatus(runtimeStatus.qbittorrent.configured, runtimeStatus.qbittorrent.enabled),
          detail: runtimeStatus.qbittorrent.enabled
            ? `默认保存路径: ${runtimeSettings.qbittorrent.defaultSavePath || "未设置"}`
            : runtimeStatus.qbittorrent.configured
              ? "配置完整，但运行时未连接成功"
              : "缺少 URL、用户名或密码"
        }
      ]
    : [];

  const todoTasks = tasks.filter((task) => isStatus(task, "failed") || isScanPending(task)).slice(0, 4);
  const hasTodos = tasks.some((task) => isStatus(task, "failed") || isScanPending(task));

  return (
    <>
      <section className="section-band section-band--hero">
        <div className="band-head">
          <div>
            <p className="section-kicker">依赖状态</p>
            <h2>外部服务</h2>
          </div>
        </div>

        <div className="card-grid card-grid--deps">
          {dependencyCards.map((card) => (
            <article key={card.name} className="service-card">
              <div className="service-card__head">
                <div>
                  <h3>{card.name}</h3>
                </div>
                <span className={`status-chip ${card.tone}`}>
                  {card.label}
                </span>
              </div>
              <p className="service-card__detail">{card.detail}</p>
              <div className="service-card__actions">
                <span>上次检测: {formatDateTime(lastCheckedAt ?? null)}</span>
                <button type="button" className="ghost-button" onClick={onRefresh}>
                  重试
                </button>
              </div>
            </article>
          ))}
        </div>
      </section>

      <section className="section-band">
        <div className="band-head">
          <div>
            <p className="section-kicker">待办</p>
            <h2>需要人工确认的任务</h2>
          </div>
          <p className="band-note">失败项、待扫描项和长时间停滞项都放在这里。</p>
        </div>

        <div className="card-grid">
          {todoTasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              compact
              pendingScanId={pendingTaskScanId}
              onOpen={onOpenTask}
              onScan={onScanTask}
            />
          ))}
          {!hasTodos ? (
            <article className="empty-card">
              <h3>暂无待处理项</h3>
              <p>这里会优先显示失败、待扫和异常任务。</p>
            </article>
          ) : null}
        </div>
      </section>
    </>
  );
}
