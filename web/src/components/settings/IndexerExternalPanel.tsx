import { useTranslation } from "react-i18next";
import { useIndexerManagement } from "../../hooks/useIndexerManagement";
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import { SettingsSwitch } from "./SettingsSwitch";
import { SettingsCard } from "./SettingsTabCommon";
import { describeQueryError } from "../../services/queryError";

export function IndexerExternalPanel() {
  const { t } = useTranslation(); const management = useIndexerManagement();
  return <div className="drawer-stack"><SettingsCard title={t("indexers.external")}>
    <label className="switch-row"><span>{t("indexers.useExternal")}</span><SettingsSwitch checked={management.snapshot?.jackettEnabled ?? false} disabled={management.busy || !management.snapshot} label={t("indexers.useExternal")} onChange={enabled => void management.setEnabled("jackett", enabled)} /></label>
    {management.error ? <p className="settings-feedback tone-danger" role="alert">{describeQueryError(management.error)} <button onClick={management.refresh}>{t("common.retry")}</button></p> : null}
  </SettingsCard><ServiceConnectionSettings service="jackett" onSaved={management.refresh} /></div>;
}
