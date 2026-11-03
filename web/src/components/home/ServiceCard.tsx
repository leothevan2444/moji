import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons/faArrowUpRightFromSquare";
import { formatDateTime, formatRelative } from "../../utils";
import type { SettingsTab } from "../../types";
import type { StashStats } from "../../graphql/generated/graphql";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

interface ServiceCardProps {
  title: string;
  status: { configured: boolean; ready: boolean };
  config: Array<{ key: string; value: string }>;
  blockers?: string[];
  diagnostics?: string;
  cta:
    | { kind: "open-settings"; label: string; tab: SettingsTab }
    | { kind: "enable"; label: string; onEnable: () => void }
    | null;
  okAt: string | null;
  lastError: string | null;
  stats: StashStats | null;
  onOpenSettings: (tab: SettingsTab) => void;
}

function toneFor(
  status: { configured: boolean; ready: boolean },
  okAt: string | null,
  lastError: string | null,
  now: number
): { tone: "tone-success" | "tone-warn" | "tone-neutral" | "tone-danger"; labelKey: string } {
  if (!status.configured) {
    return { tone: "tone-neutral", labelKey: "home.status.unconfigured" };
  }
  if (status.ready) {
    if (okAt) {
      const thenMs = new Date(okAt).getTime();
      if (!Number.isNaN(thenMs) && now - thenMs > 5 * 60 * 1000) {
        return { tone: "tone-warn", labelKey: "home.status.stale" };
      }
    }
    return { tone: "tone-success", labelKey: "home.status.enabled" };
  }
  if (lastError) {
    return { tone: "tone-danger", labelKey: "home.status.error" };
  }
  if (!okAt) {
    return { tone: "tone-warn", labelKey: "home.status.pending" };
  }
  return { tone: "tone-warn", labelKey: "home.status.stale" };
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
  const { t } = useTranslation();
  const { tone, labelKey } = toneFor(status, okAt, lastError, Date.now());
  const okRel = formatRelative(okAt);

  return (
    <article className="service-card service-card--detailed">
      <div className="service-card__head">
        <div>
          <h3>{title}</h3>
        </div>
        <span className={`status-chip ${tone}`}>{t(labelKey)}</span>
      </div>

      {config.length > 0 ? (
        <dl className="service-card__meta">
          {config.map((row) => (
            <div key={row.key} className="service-card__meta-row">
              <dt>{row.key.startsWith("home.") ? t(row.key) : row.key}</dt>
              <dd>{row.value || "—"}</dd>
            </div>
          ))}
        </dl>
      ) : null}

      {stats ? (
        <div className="service-card__stats">{renderStats(stats, t)}</div>
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
            <span>{t("home.noData")}</span>
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

function renderStats(stats: StashStats, t: TFunction) {
  return <>
    {stats.version ? <span>Stash <strong>v{stats.version}</strong>{stats.sceneCount != null ? ` · ${t("home.stats.scenes", {count:stats.sceneCount})}` : ""}</span> : <span>{t("home.stats.stashMissing")}</span>}
    <span>{t("home.stats.pendingScans", {count:stats.pendingMojiScanCount})}</span>
  </>;
}
