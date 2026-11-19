import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useMutation, useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { ClearStashBoxDataCacheDocumentDocument, SystemSettingsTabDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { formatBytes, formatDateTime } from "../../utils";
import { ConfirmationDialog } from "../common/ConfirmationDialog";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsCard, SettingsSaveStatus, SettingsError, SettingsLoading } from "./SettingsTabCommon";

export default function StashBoxCacheSettings() {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [{ data, fetching, error }, refresh] = useQuery({ query: SystemSettingsTabDocument });
  const [{ fetching: clearing }, clear] = useMutation(ClearStashBoxDataCacheDocumentDocument);
  const initial = useMemo(() => String(data?.settings.system.stashBoxDataCache.ttlHours ?? 24), [data]);
  const autosave=useSettingsAutosave("stash:cache",initial,value=>({stash:{data_cache:{ttl_hours:value.trim() ? Number(value) : NaN}}}));
  const {value:ttl,setValue:setTtl}=autosave;
  const [confirming, setConfirming] = useState(false);
  const [clearError, setClearError] = useState<string | null>(null);
  if (fetching && !data) return <SettingsLoading title={t("stashBoxCacheUi.title")} />;
  if (error && !data) return <SettingsError title={t("stashBoxCacheUi.title")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;
  const cache = data?.settingsStatus.stashBoxDataCache;
  const clearCache = async () => {
    const result = await clear({});
    if (result.error) { setClearError(describeQueryError(result.error)); return; }
    setConfirming(false); pushToast("tone-success", t("stashBoxCacheUi.cleared")); refresh({ requestPolicy: "network-only" });
  };
  return <SettingsCard title={t("stashBoxCacheUi.title")}>
    <form className="settings-form" onSubmit={event => event.preventDefault()}>
      <label className="settings-field"><FieldLabel text={t("stashBoxCacheUi.ttl")} info={t("stashBoxCacheUi.ttlInfo")} /><input type="number" required min="12" max="360" value={ttl} onChange={event => setTtl(event.target.value)} /></label>
      <SettingsSaveStatus {...autosave} />
    </form>
    <div className="image-cache-management"><div><div className="settings-meta"><span>{t("systemSettings.usage", { size: formatBytes(Number(cache?.usedBytes ?? 0)) })}</span><span>{t("stashBoxCacheUi.scenes", { count: cache?.sceneCount ?? 0 })}</span><span>{t("stashBoxCacheUi.performers", { count: cache?.performerCount ?? 0 })}</span><span>{t("systemSettings.cleanup", { time: formatDateTime(cache?.lastCleanupAt) })}</span></div></div><button type="button" className="image-cache-management__clear" disabled={clearing || (!cache?.sceneCount && !cache?.performerCount)} onClick={() => setConfirming(true)}>{t(clearing ? "systemSettings.clearing" : "stashBoxCacheUi.clear")}</button></div>
    {confirming ? <ConfirmationDialog title={t("stashBoxCacheUi.clearTitle")} confirmLabel={t(clearing ? "systemSettings.clearing" : "systemSettings.confirm")} cancelLabel={t("systemSettings.cancel")} pending={clearing} error={clearError} onConfirm={() => { setClearError(null); void clearCache(); }} onCancel={() => { setConfirming(false); setClearError(null); }}>
      <p>{t("stashBoxCacheUi.clearDescription")}</p>
    </ConfirmationDialog> : null}
    {cache?.lastError ? <p role="alert" className="settings-feedback tone-danger">{cache.lastError}</p> : null}
  </SettingsCard>;
}
