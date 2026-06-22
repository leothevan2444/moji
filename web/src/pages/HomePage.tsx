import { TaskCard } from "../components/tasks";
import {
  ServiceCard,
  buildJackettConfig,
  buildQBittorrentConfig,
  buildStashConfig,
  blockersFor
} from "../components/home";
import { isScanPending, isStatus, type DashboardTask } from "../utils";
import { useDashboardRefresh } from "../hooks";
import type { SettingsTab } from "../types";
import type { DashboardDocumentQuery } from "../graphql/generated/graphql";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type RuntimeStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]>;

interface HomePageProps {
  tasks: DashboardTask[];
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeStatus | null;
  pendingTaskScanId: string | null;
  onRefresh: () => void;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onOpenSettings: (tab: SettingsTab) => void;
}

export function HomePage({
  tasks,
  runtimeSettings,
  runtimeStatus,
  pendingTaskScanId,
  onRefresh,
  onOpenTask,
  onScanTask,
  onOpenSettings
}: HomePageProps) {
  useDashboardRefresh(onRefresh, 30000);

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
          {runtimeSettings && runtimeStatus ? (
            <>
              <ServiceCard
                service="stash"
                title="Stash"
                status={runtimeStatus.stash}
                config={buildStashConfig(runtimeSettings)}
                stats={runtimeStatus.stashStats}
                okAt={runtimeStatus.stashStats?.okAt ?? null}
                lastError={runtimeStatus.stashStats?.lastError ?? null}
                blockers={
                  !runtimeStatus.stash.configured
                    ? blockersFor("stash")
                    : undefined
                }
                diagnostics={
                  !runtimeStatus.stash.configured
                    ? "Stash 必须先配置，否则入库与订阅流程无法展开。"
                    : !runtimeStatus.stash.enabled
                      ? "已填写连接信息，但运行时尚未启用。"
                      : undefined
                }
                cta={
                  !runtimeStatus.stash.configured
                    ? { kind: "open-settings", label: "去配置", tab: "入库" }
                    : !runtimeStatus.stash.enabled
                      ? { kind: "open-settings", label: "去启用", tab: "入库" }
                      : null
                }
                onOpenSettings={onOpenSettings}
              />
              <ServiceCard
                service="jackett"
                title="Jackett"
                status={runtimeStatus.jackett}
                config={buildJackettConfig(runtimeSettings)}
                stats={{
                  ...(runtimeStatus.jackettStats ?? {
                    indexerCount: 0,
                    configuredIndexerCount: 0,
                    lastIndexerLatencyMs: 0,
                    okAt: new Date(0).toISOString()
                  }),
                  hasIndexerSearch: true
                }}
                okAt={runtimeStatus.jackettStats?.okAt ?? null}
                lastError={runtimeStatus.jackettStats?.lastError ?? null}
                blockers={
                  !runtimeStatus.jackett.configured
                    ? blockersFor("jackett")
                    : undefined
                }
                diagnostics={
                  !runtimeStatus.jackett.configured
                    ? "Jackett 必须先配置，否则任务搜索与订阅没有上游数据。"
                    : !runtimeStatus.jackett.enabled
                      ? "已填写连接信息，但运行时尚未启用。"
                      : undefined
                }
                cta={
                  !runtimeStatus.jackett.configured
                    ? { kind: "open-settings", label: "去配置", tab: "连接" }
                    : !runtimeStatus.jackett.enabled
                      ? { kind: "open-settings", label: "去启用", tab: "连接" }
                      : null
                }
                onOpenSettings={onOpenSettings}
              />
              <ServiceCard
                service="qbittorrent"
                title="qBittorrent"
                status={runtimeStatus.qbittorrent}
                config={buildQBittorrentConfig(runtimeSettings)}
                stats={
                  runtimeStatus.qbittorrentStats
                    ? { ...runtimeStatus.qbittorrentStats, hasIndexerSearch: false as const }
                    : null
                }
                okAt={runtimeStatus.qbittorrentStats?.okAt ?? null}
                lastError={runtimeStatus.qbittorrentStats?.lastError ?? null}
                blockers={
                  !runtimeStatus.qbittorrent.configured
                    ? blockersFor("qbittorrent")
                    : undefined
                }
                diagnostics={
                  !runtimeStatus.qbittorrent.configured
                    ? "qBittorrent 必须先配置，否则下载与落地流程无法展开。"
                    : !runtimeStatus.qbittorrent.enabled
                      ? "已填写连接信息，但运行时尚未连接。"
                      : undefined
                }
                cta={
                  !runtimeStatus.qbittorrent.configured
                    ? { kind: "open-settings", label: "去配置", tab: "连接" }
                    : !runtimeStatus.qbittorrent.enabled
                      ? { kind: "open-settings", label: "去启用", tab: "连接" }
                      : null
                }
                onOpenSettings={onOpenSettings}
              />
            </>
          ) : null}
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