import { TaskCard } from "../components/tasks";
import {
  IngestCard,
  ServiceCard,
  buildJackettConfig,
  buildQBittorrentConfig
} from "../components/home";
import { isScanPending, isStatus, type DashboardTask } from "../utils";
import type { SettingsTab } from "../types";

// Stash 卡片的 KV 列表——只显示 URL（与服务可达性直接相关）。
// ingest 模式/路径由新的 IngestCard 承载，避免数据源分散。
function buildStashConfig(runtime: NonNullable<DashboardDocumentQuery["settings"]>) {
  return [{ key: "URL", value: runtime.stash.url }];
}

// Jackett 与 qBittorrent 的 blocker 与 ingest 无关，保留在 HomePage。
function blockersFor(
  service: "jackett" | "qbittorrent"
): string[] {
  switch (service) {
    case "jackett":
      return ["任务搜索无索引源", "演员更新无上游数据"];
    case "qbittorrent":
      return ["任务无法启动下载", "下载完成后无客户端落地"];
  }
}
import type { DashboardDocumentQuery } from "../graphql/generated/graphql";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type RuntimeStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]>;

interface HomePageProps {
  tasks: DashboardTask[];
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeStatus | null;
  pendingTaskScanId: string | null;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onOpenSettings: (tab: SettingsTab) => void;
}

export function HomePage({
  tasks,
  runtimeSettings,
  runtimeStatus,
  pendingTaskScanId,
  onOpenTask,
  onScanTask,
  onOpenSettings
}: HomePageProps) {
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
                diagnostics={
                  runtimeStatus.stash.configured && !runtimeStatus.stash.ready && runtimeStatus.stashStats?.lastError
                    ? "已配置但最近未联通，请检查 Stash 服务状态。"
                    : !runtimeStatus.stash.configured
                      ? "Stash 必须先配置，否则入库与演员更新流程无法展开。"
                      : !runtimeStatus.stash.ready
                        ? "Stash 已配置，等待首次探测或最近状态已过期。"
                      : undefined
                }
                cta={
                  !runtimeStatus.stash.configured
                    ? { kind: "open-settings", label: "去配置", tab: "连接" }
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
                  runtimeStatus.jackett.configured && !runtimeStatus.jackett.ready && runtimeStatus.jackettStats?.lastError
                    ? "已配置但最近未联通，请检查 Jackett 服务状态。"
                    : !runtimeStatus.jackett.configured
                      ? "Jackett 必须先配置，否则任务搜索与演员更新没有上游数据。"
                      : !runtimeStatus.jackett.ready
                        ? "Jackett 已配置，等待首次探测或最近状态已过期。"
                      : undefined
                }
                cta={
                  !runtimeStatus.jackett.configured
                    ? { kind: "open-settings", label: "去配置", tab: "连接" }
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
                  runtimeStatus.qbittorrent.configured &&
                  !runtimeStatus.qbittorrent.ready &&
                  runtimeStatus.qbittorrentStats?.lastError
                    ? "已配置但最近未联通，请检查 qBittorrent 服务状态。"
                    : !runtimeStatus.qbittorrent.configured
                      ? "qBittorrent 必须先配置，否则下载与落地流程无法展开。"
                      : !runtimeStatus.qbittorrent.ready
                        ? "qBittorrent 已配置，等待首次探测或最近状态已过期。"
                      : undefined
                }
                cta={
                  !runtimeStatus.qbittorrent.configured
                    ? { kind: "open-settings", label: "去配置", tab: "连接" }
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
            <p className="section-kicker">入库策略</p>
            <h2>入库</h2>
          </div>
        </div>
        <IngestCard
          ingest={runtimeSettings?.ingest ?? null}
          ingestStatus={runtimeStatus?.ingest ?? null}
          onOpenSettings={onOpenSettings}
        />
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
