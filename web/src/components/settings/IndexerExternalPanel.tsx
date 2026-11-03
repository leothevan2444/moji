import { useTranslation } from "react-i18next";
import { useIndexerManagement } from "../../hooks/useIndexerManagement";
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import { SettingsCard } from "./SettingsTabCommon";
import { describeQueryError } from "../../services/queryError";

export function IndexerExternalPanel() {
  const { t } = useTranslation(); const management = useIndexerManagement();
  return <div className="drawer-stack"><SettingsCard title={t("indexers.external")}>
    <label className="settings-toggle"><input type="checkbox" checked={management.snapshot?.jackettEnabled ?? false} disabled={management.busy || !management.snapshot} onChange={event => void management.setEnabled("jackett", event.target.checked)} /><span>{t("indexers.useExternal")}</span></label>
    {management.error ? <p className="settings-feedback tone-danger" role="alert">{describeQueryError(management.error)} <button onClick={management.refresh}>{t("common.retry")}</button></p> : null}
  </SettingsCard><ServiceConnectionSettings service="jackett" onSaved={management.refresh} /></div>;
}
