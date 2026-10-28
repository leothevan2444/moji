import ServiceConnectionSettings from "./ServiceConnectionSettings";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery } from "urql";
import { DownloaderSettingsDocumentDocument, SetDownloaderSpeedLimitDocumentDocument, UpdateDownloaderPolicyDocumentDocument, type DownloaderSettingsDocumentQuery } from "../../graphql/generated/graphql";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsCard, SettingsSaveStatus, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import "../../styles/settings.scss";

type Snapshot = DownloaderSettingsDocumentQuery["downloader"];
type Trigger = "RATIO" | "SEED_TIME" | "ANY" | "ALL" | "AFTER_INGEST";
interface PolicyDraft { enabled: boolean; trigger: Trigger; ratio: string; hours: string; deleteFiles: boolean }
export default function DownloaderSettingsTab() {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refresh] = useQuery({ query: DownloaderSettingsDocumentDocument, requestPolicy: "network-only" });
  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.downloader")} />;
  if (error && !data) return <><ServiceConnectionSettings service="qbittorrent" /><SettingsError title={t("settings.tabs.downloader")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} /></>;
  if (!data) return null;
  return <div className="drawer-stack"><ServiceConnectionSettings service="qbittorrent" /><CleanupSettings snapshot={data.downloader} onSaved={() => refresh({ requestPolicy: "network-only" })} /><SettingsCard title={t("downloader.speedLimits")}><p className="settings-note">{t("downloader.limitHelp")}</p><div className="settings-grid"><SpeedLimit snapshot={data.downloader} upload={false} /><SpeedLimit snapshot={data.downloader} upload /></div></SettingsCard></div>;
}
function CleanupSettings({ snapshot, onSaved }: { snapshot: Snapshot; onSaved: () => void }) {
  const { t } = useTranslation();
  const initial = useMemo<PolicyDraft>(() => {
    const p = snapshot.policy;
    const trigger: Trigger = p.afterIngest ? "AFTER_INGEST" : p.ratio > 0 && p.seedHours > 0 ? p.matchAll ? "ALL" : "ANY" : p.seedHours > 0 ? "SEED_TIME" : "RATIO";
    return { enabled: p.enabled, trigger, ratio: String(p.ratio), hours: String(p.seedHours), deleteFiles: p.deleteFiles };
  }, [snapshot.policy]);
  const [,save] = useMutation(UpdateDownloaderPolicyDocumentDocument);
  const autosave = useSettingsAutosave("downloader:policy",initial,value=>({
    enabled:value.enabled,afterIngest:value.trigger==="AFTER_INGEST",ratio:["RATIO","ANY","ALL"].includes(value.trigger)?(value.ratio.trim()?Number(value.ratio):NaN):0,
    seedHours:["SEED_TIME","ANY","ALL"].includes(value.trigger)?(value.hours.trim()?Number(value.hours):NaN):0,matchAll:value.trigger==="ALL",deleteFiles:value.deleteFiles
  }),{persist:async patch=>{const result=await save({input:patch});if(result.error)throw result.error;},onSaved:()=>onSaved()});
  const {value:draft,setValue:setDraft}=autosave;
  const ratioEnabled = ["RATIO", "ANY", "ALL"].includes(draft.trigger);
  const timeEnabled = ["SEED_TIME", "ANY", "ALL"].includes(draft.trigger);
  const patch = (value: Partial<PolicyDraft>) => setDraft(current => ({ ...current, ...value }));
  return <SettingsCard title={t("downloader.policy")}><form className="settings-form downloader-cleanup" onSubmit={event => event.preventDefault()}>
    <label className="switch-row"><span className="switch-row__label">{t("downloader.enablePolicy")}</span><span className="switch" role="switch" aria-checked={draft.enabled}><input type="checkbox" aria-label={t("downloader.enablePolicy")} checked={draft.enabled} onChange={event => patch({ enabled: event.target.checked, ...(event.target.checked ? { ratio: Number(draft.ratio) > 0 ? draft.ratio : "2", hours: Number(draft.hours) > 0 ? draft.hours : "72" } : {}) })} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></span></label>
    {draft.enabled ? <>
    <label className="settings-field"><FieldLabel text={t("downloader.trigger")} info={draft.trigger === "AFTER_INGEST" ? t("downloader.afterIngestHint") : undefined} /><select value={draft.trigger} onChange={event => patch({ trigger: event.target.value as Trigger })}>{(["RATIO", "SEED_TIME", "ANY", "ALL", "AFTER_INGEST"] as const).map(mode => <option key={mode} value={mode}>{t(`downloader.triggers.${mode}`)}</option>)}</select></label>
    <div className="settings-grid">
      {ratioEnabled ? <label className="settings-field"><FieldLabel text={t("downloader.ratioLimit")} /><input type="number" min={0.1} max={9999} step="0.1" required value={draft.ratio} onChange={event => patch({ ratio: event.target.value })} /></label> : null}
      {timeEnabled ? <label className="settings-field"><FieldLabel text={t("downloader.seedHoursLimit")} /><input type="number" min={0.1} max={87600} step="0.1" required value={draft.hours} onChange={event => patch({ hours: event.target.value })} /></label> : null}
    </div>
    <div className="switch-row"><span className="switch-row__label"><FieldLabel text={t("downloader.policyDeleteFiles")} info={t("downloader.copyGuard")} /></span><label className="switch" role="switch" aria-checked={draft.deleteFiles}><input type="checkbox" aria-label={t("downloader.policyDeleteFiles")} checked={draft.deleteFiles} onChange={event => patch({ deleteFiles: event.target.checked })} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></label></div>
    </> : null}
    <SettingsSaveStatus {...autosave} />
  </form></SettingsCard>;
}
function SpeedLimit({ snapshot, upload }: { snapshot: Snapshot; upload: boolean }) {
  const { t } = useTranslation();
  const initial = String(Math.round((upload ? snapshot.uploadLimit : snapshot.downloadLimit) / 1024));
  const [, save] = useMutation(SetDownloaderSpeedLimitDocumentDocument);
  const autosave = useSettingsAutosave(upload ? "downloader:upload" : "downloader:download",initial,value=>({kibPerSecond:value.trim()?Number(value):NaN}),{
    persist:async patch=>{
      if(snapshot.stale || !snapshot.storageAvailable)throw new Error(t("downloader.unavailable"));
      const result=await save({upload,kibPerSecond:patch.kibPerSecond as number});if(result.error)throw result.error;
    }
  });
  const {value:limit,setValue:setLimit}=autosave;
  return <form className="settings-form" onSubmit={event=>event.preventDefault()}>
    <label className="settings-field"><FieldLabel text={t(upload ? "downloader.uploadLimit" : "downloader.downloadLimit")} /><input type="number" min={0} max={10485760} step={1} required value={limit} onChange={event => setLimit(event.target.value)} /></label>
    <SettingsSaveStatus {...autosave} />
  </form>;
}
