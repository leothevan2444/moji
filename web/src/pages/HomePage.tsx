import { TaskCard } from "../components/tasks";
import {
  IngestCard,
  ServiceCard,
  buildJackettConfig,
  buildQBittorrentConfig
} from "../components/home";
import { isScanPending, isStatus, type DashboardTask } from "../utils";
import type { SettingsTab } from "../types";
import { useTranslation } from "react-i18next";

// Stash 卡片的 KV 列表——只显示 URL（与服务可达性直接相关）。
// ingest 模式/路径由新的 IngestCard 承载，避免数据源分散。
function buildStashConfig(runtime: NonNullable<HomePageDocumentQuery["settings"]>) {
  return [{ key: "URL", value: runtime.stash.url }];
}

// Jackett 与 qBittorrent 的 blocker 与 ingest 无关，保留在 HomePage。
function blockerKeysFor(service: "jackett" | "qbittorrent"): string[] {
  switch (service) {
    case "jackett":
      return ["jackettSearch", "jackettPerformers"];
    case "qbittorrent":
      return ["qbDownload", "qbLanding"];
  }
}
import type { HomePageDocumentQuery } from "../graphql/generated/graphql";

type RuntimeSettings = NonNullable<HomePageDocumentQuery["settings"]>;
type RuntimeStatus = NonNullable<HomePageDocumentQuery["settingsStatus"]>;

interface HomePageProps {
  tasks: DashboardTask[];
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeStatus | null;
  pendingTaskScanId: string | null;
  pendingTaskRetryId: string | null;
  onOpenTask: (taskId: string) => void;
  onScanTask: (taskId: string) => void;
  onRetryTask: (taskId: string) => void;
  onResolveTask: (taskId: string) => void;
  onOpenSettings: (tab: SettingsTab) => void;
}

export function HomePage({
  tasks,
  runtimeSettings,
  runtimeStatus,
  pendingTaskScanId,
  pendingTaskRetryId,
  onOpenTask,
  onScanTask,
  onRetryTask,
  onResolveTask,
  onOpenSettings
}: HomePageProps) {
  const { t } = useTranslation();
  const blockersFor = (service: "jackett" | "qbittorrent") => blockerKeysFor(service).map((key) => t(`home.blockers.${key}`));
  const todoTasks = tasks.filter((task) => isStatus(task, "failed") || isScanPending(task)).slice(0, 4);
  const hasTodos = tasks.some((task) => isStatus(task, "failed") || isScanPending(task));

  return (
    <>
      <section className="section-band section-band--hero">
        <div className="band-head">
          <div>
            <h2>{t("home.services")}</h2>
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
                    ? t("home.diagnostics.stashError")
                    : !runtimeStatus.stash.configured
                      ? t("home.diagnostics.stashMissing")
                      : !runtimeStatus.stash.ready
                        ? t("home.diagnostics.stashPending")
                      : undefined
                }
                cta={
                  !runtimeStatus.stash.configured
                    ? { kind: "open-settings", label: t("home.configure"), tab: "connections" }
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
                    ? t("home.diagnostics.jackettError")
                    : !runtimeStatus.jackett.configured
                      ? t("home.diagnostics.jackettMissing")
                      : !runtimeStatus.jackett.ready
                        ? t("home.diagnostics.jackettPending")
                      : undefined
                }
                cta={
                  !runtimeStatus.jackett.configured
                    ? { kind: "open-settings", label: t("home.configure"), tab: "connections" }
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
                    ? t("home.diagnostics.qbError")
                    : !runtimeStatus.qbittorrent.configured
                      ? t("home.diagnostics.qbMissing")
                      : !runtimeStatus.qbittorrent.ready
                        ? t("home.diagnostics.qbPending")
                      : undefined
                }
                cta={
                  !runtimeStatus.qbittorrent.configured
                    ? { kind: "open-settings", label: t("home.configure"), tab: "connections" }
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
            <h2>{t("home.ingestPolicy")}</h2>
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
            <h2>{t("home.todos")}</h2>
          </div>
          <p className="band-note">{t("home.todosNote")}</p>
        </div>

        <div className="card-grid">
          {todoTasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              compact
              pendingScanId={pendingTaskScanId}
              pendingRetryId={pendingTaskRetryId}
              onOpen={onOpenTask}
              onScan={onScanTask}
              onRetry={onRetryTask}
              onResolve={onResolveTask}
            />
          ))}
          {!hasTodos ? (
            <article className="empty-card">
              <h3>{t("home.noTodos")}</h3>
              <p>{t("home.noTodosDetail")}</p>
            </article>
          ) : null}
        </div>
      </section>
    </>
  );
}
