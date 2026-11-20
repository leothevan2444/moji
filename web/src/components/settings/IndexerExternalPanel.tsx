import { useAppTimezone } from "../../hooks/useAppTimezone";
import { formatDateTime } from "../../utils";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { useIndexerManagement } from "../../hooks/useIndexerManagement";
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import { SettingsSwitch } from "./SettingsSwitch";
import { FieldLabel, SettingsCard } from "./SettingsTabCommon";
import { describeQueryError } from "../../services/queryError";

export function IndexerExternalPanel() {
  useAppTimezone();
  const { t } = useTranslation();
  const management = useIndexerManagement();
  return <SettingsCard title={t("indexers.jackettConnection")} actions={
    <label className="indexer-external-toggle"><FieldLabel text={t("indexers.enable")} info={t("indexers.jackettPurpose")} /><SettingsSwitch checked={management.snapshot?.jackettEnabled ?? false} disabled={management.busy || !management.snapshot} label={t("indexers.enable")} onChange={enabled => void management.setEnabled("jackett", enabled)} /></label>
  }>
    {management.error ? <p className="settings-feedback tone-danger" role="alert">{describeQueryError(management.error)} <button onClick={management.refresh}>{t("common.retry")}</button></p> : null}
    {management.snapshot?.catalogErrorCode?<p className="settings-feedback tone-danger" role="status">{t(`indexerErrors.${management.snapshot.catalogErrorCode}`,{defaultValue:t("indexers.catalogFailed")})}</p>:null}
    {management.snapshot?.jackettEnabled ? <p className="settings-note">{management.snapshot.jackettCatalogUpdatedAt?t("indexerRecordUi.catalogUpdated",{time:formatDateTime(management.snapshot.jackettCatalogUpdatedAt)}):t("indexerRecordUi.catalogUnknown")}{management.snapshot?.jackettCatalogStale?` · ${t("indexerRecordUi.catalogStale")}`:""}</p> : null}
    <ServiceConnectionSettings embedded service="jackett" onSaved={management.refresh} />
    <div className="indexer-external-sites"><div><h4>{t("indexers.jackettSites")}</h4><p className="settings-note">{t("indexers.jackettSitesHelp")}</p></div><Link to="/settings/indexers?panel=sites">{t("indexers.manageSites")}</Link></div>
  </SettingsCard>;
}
