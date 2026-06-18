import { TaskCard } from "../components/tasks";
import { formatDateTime, isScanPending, isStatus, serviceStatus, type DashboardTask } from "../utils";
import type { DashboardDocumentQuery } from "../graphql/generated/graphql";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;

interface HomePageProps {
  tasks: DashboardTask[];
  runtimeSettings: RuntimeSettings | null;
  lastCheckedAt: string | null | undefined;
  pendingTaskScanId: string | null;
  onRefresh: () => void;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
}

export function HomePage({
  tasks,
  runtimeSettings,
  lastCheckedAt,
  pendingTaskScanId,
  onRefresh,
  onOpenTask,
  onScanTask
}: HomePageProps) {
  const dependencyCards = runtimeSettings
    ? [
        {
          name: "Stash",
          ...serviceStatus(runtimeSettings.stash.configured, runtimeSettings.stash.enabled),
          detail: runtimeSettings.stash.enabled
            ? `媒体库路径: ${runtimeSettings.stash.libraryPath || "未设置"}`
            : runtimeSettings.stash.configured
              ? "配置已存在，但运行时尚未启用"
              : "缺少 Stash URL 或库路径"
        },
        {
          name: "Jackett",
          ...serviceStatus(runtimeSettings.jackett.configured, runtimeSettings.jackett.enabled),
          detail: runtimeSettings.jackett.enabled
            ? `索引地址: ${runtimeSettings.jackett.url || "未设置"}`
            : "缺少 URL 或 API key"
        },
        {
          name: "qBittorrent",
          ...serviceStatus(runtimeSettings.qbittorrent.configured, runtimeSettings.qbittorrent.enabled),
          detail: runtimeSettings.qbittorrent.enabled
            ? `默认保存路径: ${runtimeSettings.qbittorrent.defaultSavePath || "未设置"}`
            : runtimeSettings.qbittorrent.configured
              ? "配置完整，但运行时未连接成功"
              : "缺少 URL、用户名或密码"
        },
        {
          name: "订阅",
          label: runtimeSettings.subscription.pollEnabled ? "已启用" : "未启用",
          tone: runtimeSettings.subscription.pollEnabled ? "tone-success" : "tone-neutral",
          detail: (runtimeSettings.subscription.stashBoxes?.length ?? 0) > 0
            ? `Stash-Box: ${runtimeSettings.subscription.stashBoxes?.length} 个，已选 ${runtimeSettings.subscription.selectedStashBoxEndpoints?.length ?? 0} 个`
            : "Stash 中尚未配置任何 Stash-Box"
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
