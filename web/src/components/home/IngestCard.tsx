import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons";
import { INGEST_BLOCKERS, deliveryModeGuide, deliveryModeLabel, transferActionLabel } from "../../utils";
import type { SettingsTab } from "../../types";
import type { HomePageDocumentQuery } from "../../graphql/generated/graphql";

type Settings = NonNullable<HomePageDocumentQuery["settings"]>;
type IngestSettings = NonNullable<Settings["ingest"]>;
type IngestStatus = NonNullable<HomePageDocumentQuery["settingsStatus"]["ingest"]>;

interface IngestCardProps {
  ingest: IngestSettings | null;
  ingestStatus: IngestStatus | null;
  onOpenSettings: (tab: SettingsTab) => void;
}

function ingestConfigRows(ingest: IngestSettings): Array<{ key: string; value: string }> {
  switch (ingest.deliveryMode) {
    case "PATH_MAP":
      return [
        { key: "入库方式", value: deliveryModeLabel(ingest.deliveryMode) },
        { key: "qB 下载根", value: ingest.downloads.qbRoot || "—" },
        { key: "Stash 媒体库根", value: ingest.library.stashRoot || "—" }
      ];
    case "TRANSFER":
      return [
        { key: "入库方式", value: deliveryModeLabel(ingest.deliveryMode) },
        { key: "交付动作", value: transferActionLabel(ingest.transfer.action) },
        { key: "qB 下载根", value: ingest.downloads.qbRoot || "—" },
        { key: "Moji 下载根", value: ingest.downloads.mojiRoot || "—" },
        { key: "Moji 媒体库根", value: ingest.library.mojiRoot || "—" },
        { key: "Stash 媒体库根", value: ingest.library.stashRoot || "—" }
      ];
    default:
      return [{ key: "入库方式", value: "未选择" }];
  }
}

function missingFields(ingest: IngestSettings): string[] {
  switch (ingest.deliveryMode) {
    case "PATH_MAP":
      return [
        !ingest.downloads.qbRoot && "qB 下载根",
        !ingest.library.stashRoot && "Stash 媒体库根"
      ].filter(Boolean) as string[];
    case "TRANSFER":
      return [
        !ingest.transfer.action && "交付动作",
        !ingest.downloads.qbRoot && "qB 下载根",
        !ingest.downloads.mojiRoot && "Moji 下载根",
        !ingest.library.mojiRoot && "Moji 媒体库根",
        !ingest.library.stashRoot && "Stash 媒体库根"
      ].filter(Boolean) as string[];
    default:
      return [];
  }
}

export function IngestCard({ ingest, ingestStatus, onOpenSettings }: IngestCardProps) {
  const configured = ingestStatus?.configured ?? false;
  const mode = ingest?.deliveryMode ?? "";
  const hasMode = Boolean(mode);
  const guide = deliveryModeGuide(mode);

  let tone: "tone-neutral" | "tone-warn" | "tone-success";
  let label: string;
  let ctaLabel: string | null = null;
  const ctaTab: SettingsTab = "入库";

  if (!ingest || !hasMode) {
    tone = "tone-neutral";
    label = "未配置";
    ctaLabel = "去配置";
  } else if (!configured) {
    tone = "tone-warn";
    label = "已配置未启用";
    ctaLabel = "去调整";
  } else {
    tone = "tone-success";
    label = "已启用";
  }

  const showBlockers = !ingest || !hasMode;

  return (
    <article className="ingest-card service-card service-card--detailed">
      <div className="service-card__head">
        <div>
          <h3>入库</h3>
        </div>
        <span className={`status-chip ${tone}`}>{label}</span>
      </div>

      {ingest && hasMode ? (
        <dl className="service-card__meta">
          {ingestConfigRows(ingest).map((row) => (
            <div key={row.key} className="service-card__meta-row">
              <dt>{row.key}</dt>
              <dd>{row.value}</dd>
            </div>
          ))}
        </dl>
      ) : null}

      {ingest && hasMode && !configured ? (
        <p className="service-card__diagnostics">
          {missingFields(ingest).length > 0
            ? `缺少: ${missingFields(ingest).join("、")}`
            : "工作方式已选择，但路径映射未填完整。"}
        </p>
      ) : null}

      <section className={`ingest-card__mode-guide ${guide.tone}`}>
        <strong className="ingest-card__mode-guide__title">{guide.title}</strong>
        <span className="ingest-card__mode-guide__summary">{guide.summary}</span>
        {guide.caution ? (
          <p className="ingest-card__mode-guide__caution">{guide.caution}</p>
        ) : null}
      </section>

      {showBlockers ? (
        <ul className="service-card__blockers">
          {INGEST_BLOCKERS.map((b) => (
            <li key={b}>{b}</li>
          ))}
        </ul>
      ) : null}

      <div className="service-card__foot">
        <div className="service-card__refresh" aria-live="polite">
          <span className={`service-card__refresh-dot ${tone === "tone-success" ? "" : "is-stale"}`} />
          <span>入库方式: {deliveryModeLabel(ingest?.deliveryMode || "")}</span>
        </div>
        {ctaLabel ? (
          <div className="service-card__cta">
            <button
              type="button"
              className="ghost-button"
              onClick={() => onOpenSettings(ctaTab)}
            >
              <FontAwesomeIcon icon={faArrowUpRightFromSquare} aria-hidden="true" />
              <span>{ctaLabel}</span>
            </button>
          </div>
        ) : null}
      </div>
    </article>
  );
}
