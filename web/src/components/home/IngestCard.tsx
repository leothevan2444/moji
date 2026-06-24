import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons";
import { INGEST_BLOCKERS, deliveryModeGuide, deliveryModeLabel, transferActionLabel } from "../../utils";
import type { SettingsTab } from "../../types";
import type { DashboardDocumentQuery } from "../../graphql/generated/graphql";

type Settings = NonNullable<DashboardDocumentQuery["settings"]>;
type IngestSettings = NonNullable<Settings["ingest"]>;
type IngestStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]["ingest"]>;

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
        { key: "目标媒体库", value: ingest.stashLibraryPath || "—" }
      ];
    case "TRANSFER":
      return [
        { key: "入库方式", value: deliveryModeLabel(ingest.deliveryMode) },
        { key: "目标媒体库", value: ingest.stashLibraryPath || "—" },
        { key: "交付动作", value: transferActionLabel(ingest.transfer.action) },
        { key: "Moji 下载区", value: ingest.transfer.mojiSourceRoot || "—" },
        { key: "Moji 媒体库", value: ingest.transfer.mojiTargetRoot || "—" }
      ];
    default:
      return [{ key: "入库方式", value: "未选择" }];
  }
}

function missingFields(ingest: IngestSettings): string[] {
  switch (ingest.deliveryMode) {
    case "PATH_MAP":
      return [!ingest.stashLibraryPath && "目标媒体库"].filter(Boolean) as string[];
    case "TRANSFER":
      return [
        !ingest.stashLibraryPath && "目标媒体库",
        !ingest.transfer.action && "交付动作",
        !ingest.transfer.mojiSourceRoot && "Moji 下载区",
        !ingest.transfer.mojiTargetRoot && "Moji 媒体库"
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

  // 三状态分支：
  // 1. 未配置：mode 未选或 ingest=null
  // 2. 已配置未启用：mode 选了但字段不齐
  // 3. 已启用：所有字段齐全
  let tone: "tone-neutral" | "tone-warn" | "tone-success";
  let label: string;
  let ctaLabel: string | null = null;
  let ctaTab: SettingsTab = "入库";

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
            : "工作方式已选择，但字段未填完整。"}
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
