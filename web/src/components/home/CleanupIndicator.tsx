import { useNavigate } from "react-router";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faBroom } from "@fortawesome/free-solid-svg-icons";
import { Tooltip } from "../common/Tooltip";
import type { DownloaderSummaryDocumentQuery } from "../../graphql/generated/graphql";

type Status = DownloaderSummaryDocumentQuery["downloader"]["cleanup"];

export function CleanupIndicator({ status, unavailable }: { status: Status; unavailable: boolean }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  if (!status.enabled && status.phase === "IDLE") return null;

  const phase = unavailable ? "WARNING" : status.phase;
  const reason = unavailable ? "STATUS_UNKNOWN" : status.reason;
  const label = t(`downloader.cleanupUi.phases.${phase}`);
  const detail = reason ? t(`downloader.cleanupUi.reasons.${reason}`) : t("downloader.cleanupUi.enabled");

  return (
    <Tooltip
      className={`home-cleanup home-cleanup--${phase.toLowerCase()}`}
      label={t("downloader.cleanupUi.label", { state: label })}
      info={detail}
      onClick={() => navigate("/settings/downloader?panel=cleanup")}
    >
      <FontAwesomeIcon icon={faBroom} aria-hidden="true" />
      {phase !== "IDLE" ? <span role="status">{label}</span> : null}
    </Tooltip>
  );
}
