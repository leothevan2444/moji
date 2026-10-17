import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons/faArrowUpRightFromSquare";
import type { SettingsTab } from "../../types";
import type { HomePageDocumentQuery, HomeServiceStatusQuery } from "../../graphql/generated/graphql";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

type Settings = NonNullable<HomePageDocumentQuery["settings"]>;
type IngestSettings = NonNullable<Settings["ingest"]>;
type IngestStatus = NonNullable<HomeServiceStatusQuery["settingsStatus"]["ingest"]>;

interface IngestCardProps {
  ingest: IngestSettings | null;
  ingestStatus: IngestStatus | null;
  onOpenSettings: (tab: SettingsTab) => void;
}

function actionLabel(action: string, t: TFunction) {
  if (action === "COPY") return t("home.ingest.copy");
  if (action === "MOVE") return t("home.ingest.move");
  return action || t("home.ingest.none");
}

function ingestConfigRows(ingest: IngestSettings, t: TFunction): Array<{ key: string; value: string }> {
  return [
    { key: t("home.config.action"), value: actionLabel(ingest.transfer.action, t) },
    { key: t("home.config.qbRoot"), value: ingest.downloads.qbRoot || "—" },
    { key: t("home.config.mojiDownloadRoot"), value: ingest.downloads.mojiRoot || "—" },
    { key: t("home.config.mojiLibraryRoot"), value: ingest.library.mojiRoot || "—" },
    { key: t("home.config.stashRoot"), value: ingest.library.stashRoot || "—" }
  ];
}

function missingFields(ingest: IngestSettings, t: TFunction): string[] {
  return [
    !["COPY", "MOVE"].includes(ingest.transfer.action) && t("home.config.action"),
    !ingest.downloads.qbRoot.trim() && t("home.config.qbRoot"),
    !ingest.downloads.mojiRoot.trim() && t("home.config.mojiDownloadRoot"),
    !ingest.library.mojiRoot.trim() && t("home.config.mojiLibraryRoot"),
    !ingest.library.stashRoot.trim() && t("home.config.stashRoot")
  ].filter(Boolean) as string[];
}

export function IngestCard({ ingest, ingestStatus, onOpenSettings }: IngestCardProps) {
  const { t } = useTranslation();
  const configured = ingestStatus?.configured ?? false;
  const guide = { tone: "tone-info", title: t("home.ingest.transfer"), summary: t("home.ingest.guideTransfer"), caution: t("home.ingest.cautionTransfer") };

  let tone: "tone-neutral" | "tone-warn" | "tone-success";
  let label: string;
  let ctaLabel: string | null = null;
  const ctaTab: SettingsTab = "ingest";

  if (!ingest) {
    tone = "tone-neutral";
    label = t("home.status.unconfigured");
    ctaLabel = t("home.configure");
  } else if (!configured) {
    tone = "tone-warn";
    label = t("home.status.configuredDisabled");
    ctaLabel = t("home.adjust");
  } else {
    tone = "tone-success";
    label = t("home.status.enabled");
  }

  const showBlockers = !ingest;

  return (
    <article className="ingest-card service-card service-card--detailed">
      <div className="service-card__head">
        <div>
          <h3>{t("home.ingest.title")}</h3>
        </div>
        <span className={`status-chip ${tone}`}>{label}</span>
      </div>

      {ingest ? (
        <dl className="service-card__meta">
          {ingestConfigRows(ingest, t).map((row) => (
            <div key={row.key} className="service-card__meta-row">
              <dt>{row.key}</dt>
              <dd>{row.value}</dd>
            </div>
          ))}
        </dl>
      ) : null}

      {ingest && !configured ? (
        <p className="service-card__diagnostics">
          {missingFields(ingest, t).length > 0
            ? t("home.ingest.missing", { fields: missingFields(ingest, t).join(", ") })
            : t("home.ingest.incomplete")}
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
          {(["blockerQb", "blockerStash", "blockerScan"] as const).map((key) => (
            <li key={key}>{t(`home.ingest.${key}`)}</li>
          ))}
        </ul>
      ) : null}

      <div className="service-card__foot">
        <div className="service-card__refresh" aria-live="polite">
          <span className={`service-card__refresh-dot ${tone === "tone-success" ? "" : "is-stale"}`} />
          <span>{t("home.ingest.transfer")}</span>
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
