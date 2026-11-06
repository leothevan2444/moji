import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark, faSpinner, faCrosshairs, faClock } from "@fortawesome/free-solid-svg-icons";
import type { ConnectionResult } from "../../hooks/useIndexerConnections";

export function IndexerConnectionButton({ result, all = false, jackett = false, disabled, onClick }: {
  result?: ConnectionResult; all?: boolean; jackett?: boolean; disabled: boolean; onClick: () => void;
}) {
  const { t, i18n } = useTranslation();
  const state = all && result?.state !== "running" ? undefined : result?.state;
  const label = t(state === "queued" ? "indexers.connectionQueued" : state === "pending" ? "indexers.connectionPending" : state === "running" ? "indexers.connectionRunning" : state === "passed" ? "indexers.connectionPassed" : state === "failed" ? "indexers.connectionFailed" : all ? "indexers.testAll" : "indexers.test");
  const scope = t(jackett ? "indexers.jackettConnectionScope" : "indexers.connectionScope");
  const detail = !all && result?.errorCode ? (i18n.exists(`indexerErrors.${result.errorCode}`) ? t(`indexerErrors.${result.errorCode}`) : t("indexers.connectionFailed")) : all ? undefined : result?.error;
  return <button type="button" className={`ghost-button indexer-test${state ? ` is-${state}` : ""}`} disabled={disabled} onClick={onClick} title={[detail, scope].filter(Boolean).join(" · ")} aria-busy={state === "running"}>
    <FontAwesomeIcon icon={state === "queued" || state === "pending" ? faClock : state === "running" ? faSpinner : state === "passed" ? faCheck : state === "failed" ? faXmark : faCrosshairs} spin={state === "running"} aria-hidden="true" /><span aria-live="polite">{label}</span>
  </button>;
}
