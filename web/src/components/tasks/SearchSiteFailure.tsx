import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleExclamation } from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";
import { Tooltip } from "../common/Tooltip";

export function SearchSiteFailure({ name, label, code, onNotify }: { name: string; label: string; code: string; onNotify?: (tone: "tone-success" | "tone-danger", message: string) => void }) {
  const { t, i18n } = useTranslation();
  const reason = i18n.exists(`indexerErrors.${code}`) ? t(`indexerErrors.${code}`) : t("indexerErrors.INDEXER_OPERATION_FAILED");
  const copy = async () => {
    try { await navigator.clipboard.writeText(`${name}: ${reason} (${code})`); onNotify?.("tone-success", t("torrentSearch.errorCopied")); }
    catch { onNotify?.("tone-danger", t("toast.copyFailed")); }
  };
  return <Tooltip className="torrent-search-progress__failure" label={`${name} · ${label} · ${t("torrentSearch.copyError")}`} info={`${reason}\n${t("torrentSearch.copyErrorHint")}`} onClick={() => void copy()}>
    <FontAwesomeIcon icon={faCircleExclamation} aria-hidden="true" /> {label}
  </Tooltip>;
}
