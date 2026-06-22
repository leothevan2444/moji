import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons";
import { formatBytesRate, formatDateTime, formatRelative } from "../../utils";
import type { SettingsTab } from "../../types";
import type {
  DashboardDocumentQuery,
  JackettStats,
  QBittorrentStats,
  StashStats
} from "../../graphql/generated/graphql";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type RuntimeStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]>;

interface ServiceCardProps {
  service: "stash" | "jackett" | "qbittorrent";
  title: string;
  status: { configured: boolean; enabled: boolean };
  config: Array<{ key: string; value: string }>;
  blockers?: string[];
  diagnostics?: string;
  cta:
    | { kind: "open-settings"; label: string; tab: SettingsTab }
    | { kind: "enable"; label: string; onEnable: () => void }
    | null;
  okAt: string | null;
  lastError: string | null;
  stats:
    | (StashStats & {
        // optional extras computed for display
        hasIndexerSearch?: false;
      })
    | (JackettStats & { hasIndexerSearch: true })
    | (QBittorrentStats & { hasIndexerSearch: false })
    | null;
  onOpenSettings: (tab: SettingsTab) => void;
}

function toneFor(
  status: { configured: boolean; enabled: boolean },
  okAt: string | null,
  lastError: string | null,
  now: number
): { tone: "tone-success" | "tone-warn" | "tone-neutral" | "tone-danger"; label: string } {
  if (!status.configured) {
    return { tone: "tone-neutral", label: "未配置" };
  }
  if (!status.enabled) {
    return { tone: "tone-warn", label: "已配置未启用" };
  }
  if (lastError) {
    return { tone: "tone-danger", label: "运行异常" };
  }
  if (okAt) {
    const thenMs = new Date(okAt).getTime();
    if (!Number.isNaN(thenMs) && now - thenMs > 5 * 60 * 1000) {
      // 5 min stale → downgrade visual confidence
      return { tone: "tone-warn", label: "数据陈旧" };
    }
  }
  return { tone: "tone-success", label: "已启用" };
}

export function ServiceCard({
  title,
  status,
  config,
  blockers,
  diagnostics,
  cta,
  okAt,
  lastError,
  stats,
  onOpenSettings
}: ServiceCardProps) {
  const { tone, label } = toneFor(status, okAt, lastError, Date.now());
  const okRel = formatRelative(okAt);

  return (
    <article className="service-card service-card--detailed">
      <div className="service-card__head">
        <div>
          <h3>{title}</h3>
        </div>
        <span className={`status-chip ${tone}`}>{label}</span>
      </div>

      {config.length > 0 ? (
        <dl className="service-card__meta">
          {config.map((row) => (
            <div key={row.key} className="service-card__meta-row">
              <dt>{row.key}</dt>
              <dd>{row.value || "—"}</dd>
            </div>
          ))}
        </dl>
      ) : null}

      {stats ? (
        <div className="service-card__stats">{renderStats(stats)}</div>
      ) : null}

      {diagnostics ? <p className="service-card__diagnostics">{diagnostics}</p> : null}

      {lastError ? (
        <p className="service-card__error" role="alert">
          {lastError}
        </p>
      ) : null}

      {blockers && blockers.length > 0 ? (
        <ul className="service-card__blockers">
          {blockers.map((b) => (
            <li key={b}>{b}</li>
          ))}
        </ul>
      ) : null}

      <div className="service-card__foot">
        <div className="service-card__refresh" aria-live="polite">
          <span className={`service-card__refresh-dot ${tone === "tone-danger" ? "is-stale" : ""}`} />
          {okAt ? (
            <span>
              {formatDateTime(okAt)}
              {okRel ? ` · ${okRel}` : ""}
            </span>
          ) : (
            <span>暂无数据</span>
          )}
        </div>
        {cta ? (
          <div className="service-card__cta">
            <button
              type="button"
              className="ghost-button"
              onClick={() => {
                if (cta.kind === "open-settings") onOpenSettings(cta.tab);
                else cta.onEnable();
              }}
            >
              <FontAwesomeIcon icon={faArrowUpRightFromSquare} aria-hidden="true" />
              <span>{cta.label}</span>
            </button>
          </div>
        ) : null}
      </div>
    </article>
  );
}

function renderStats(
  stats:
    | (StashStats & { hasIndexerSearch?: false })
    | (JackettStats & { hasIndexerSearch: true })
    | (QBittorrentStats & { hasIndexerSearch?: false })
) {
  if ("pendingMojiScanCount" in stats) {
    return (
      <>
        {stats.version ? (
          <span>
            Stash <strong>v{stats.version}</strong>
            {stats.sceneCount != null ? ` · ${stats.sceneCount.toLocaleString()} 部影片` : ""}
          </span>
        ) : (
          <span>Stash 尚未回报数据</span>
        )}
        <span>Moji 待扫任务 <strong>{stats.pendingMojiScanCount}</strong> 项</span>
      </>
    );
  }
  if ("hasIndexerSearch" in stats && stats.hasIndexerSearch) {
    const idx = stats as JackettStats & { hasIndexerSearch: true };
    return (
      <>
        <span>
          索引器 <strong>{idx.configuredIndexerCount}</strong> / {idx.indexerCount} 已配置
        </span>
        <span>
          上次搜索最慢 <strong>{idx.lastIndexerLatencyMs} ms</strong>
          {idx.lastIndexerSearchAt ? `（${formatRelative(idx.lastIndexerSearchAt)}）` : ""}
        </span>
        {idx.lastIndexerError ? (
          <span className="service-card__error-inline">{idx.lastIndexerError}</span>
        ) : null}
      </>
    );
  }
  const q = stats as QBittorrentStats & { hasIndexerSearch?: false };
  return (
    <>
      <span>
        下载 <strong>{formatBytesRate(q.downloadSpeed)}</strong> · 上传{" "}
        <strong>{formatBytesRate(q.uploadSpeed)}</strong>
      </span>
      <span>
        活跃任务 <strong>{q.activeTorrentCount}</strong> · 连接 {q.connectionStatus}
      </span>
    </>
  );
}

export function buildStashConfig(runtime: RuntimeSettings): Array<{ key: string; value: string }> {
  const s = runtime.stash;
  const i = runtime.ingest;
  return [
    { key: "URL", value: s.url },
    { key: "模式", value: i.mode },
    { key: "库路径", value: i.libraryPath }
  ];
}

export function buildJackettConfig(runtime: RuntimeSettings): Array<{ key: string; value: string }> {
  const j = runtime.jackett;
  return [{ key: "URL", value: j.url }];
}

export function buildQBittorrentConfig(runtime: RuntimeSettings): Array<{ key: string; value: string }> {
  const q = runtime.qbittorrent;
  return [
    { key: "URL", value: q.url },
    { key: "用户", value: q.username },
    { key: "保存路径", value: q.defaultSavePath }
  ];
}

export function blockersFor(
  service: "stash" | "jackett" | "qbittorrent"
): string[] {
  switch (service) {
    case "jackett":
      return ["任务搜索无索引源", "订阅扫描无上游数据"];
    case "qbittorrent":
      return ["任务无法启动下载", "下载完成后无客户端落地"];
    case "stash":
      return ["任务完成后无法入库", "订阅扫描无目标库", "Stash-Box 元数据无法获取"];
  }
}
