import { SettingsSwitch } from "./SettingsSwitch";
import { SettingsInfoTooltip } from "./SettingsInfoTooltip";
import DownloaderSwitch from "./DownloaderSwitch";
import QBTransferSettings from "./QBTransferSettings";
import { SettingsSubpanel } from "./SettingsPanels";
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery } from "urql";
import { DownloaderSettingsDocumentDocument, SetDownloaderSpeedLimitDocumentDocument, UpdateDownloaderPolicyDocumentDocument, CapacityCleanupOrder, type DownloaderSettingsDocumentQuery } from "../../graphql/generated/graphql";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsCard, SettingsSaveStatus, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import "../../styles/settings.scss";

type Snapshot = DownloaderSettingsDocumentQuery["downloader"];
interface PolicyDraft { enabled: boolean; afterIngest: boolean; capacityEnabled: boolean; minFreeGb: string; capacityOrder: CapacityCleanupOrder; deleteFiles: boolean }
export default function DownloaderSettingsTab() {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refresh] = useQuery({ query: DownloaderSettingsDocumentDocument, requestPolicy: "network-only" });
  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.downloader")} />;
  if (error && !data) return <><SettingsSubpanel id="connection"><ServiceConnectionSettings service="qbittorrent" /></SettingsSubpanel><SettingsSubpanel id={["cleanup","speed"]}><SettingsError title={t("settings.tabs.downloader")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} /></SettingsSubpanel></>;
  if (!data) return null;
  return <div className="drawer-stack"><SettingsSubpanel id="connection"><ServiceConnectionSettings service="qbittorrent" /></SettingsSubpanel><SettingsSubpanel id="cleanup"><CleanupSettings snapshot={data.downloader} onSaved={() => refresh({ requestPolicy: "network-only" })} /></SettingsSubpanel><SettingsSubpanel id="speed"><SettingsCard title={t("downloader.speedLimits")}><p className="settings-note">{t("downloader.limitHelp")}</p><div className="settings-grid"><SpeedLimit snapshot={data.downloader} upload={false} /><SpeedLimit snapshot={data.downloader} upload /></div></SettingsCard><QBTransferSettings /></SettingsSubpanel></div>;
}
function CleanupSettings({ snapshot, onSaved }: { snapshot: Snapshot; onSaved: () => void }) {
  const { t } = useTranslation();
  const initial = useMemo<PolicyDraft>(() => {
    const p = snapshot.policy;
    return { enabled: p.enabled, afterIngest: p.afterIngest, capacityEnabled: p.capacityEnabled, minFreeGb: String(p.minFreeGb), capacityOrder: p.capacityOrder, deleteFiles: p.deleteFiles };
  }, [snapshot.policy]);
  const [,save] = useMutation(UpdateDownloaderPolicyDocumentDocument);
  const autosave = useSettingsAutosave("downloader:policy",initial,value=>({
    enabled:value.enabled, afterIngest:value.afterIngest, capacityEnabled:value.capacityEnabled,
    minFreeGb:value.minFreeGb.trim()?Number(value.minFreeGb):NaN, capacityOrder:value.capacityOrder, deleteFiles:value.deleteFiles
  }),{persist:async patch=>{const result=await save({input:patch});if(result.error)throw result.error;},onSaved:()=>onSaved()});
  const {value:draft,setValue:setDraft}=autosave;
  const patch = (value: Partial<PolicyDraft>) => setDraft(current => ({ ...current, ...value }));
  return <SettingsCard title={t("downloader.policy")} actions={<><SettingsInfoTooltip info={t("downloader.policyScope")} /><SettingsSwitch label={t("downloader.enablePolicy")} checked={draft.enabled} onChange={enabled=>patch({enabled,...(enabled&&!draft.afterIngest&&!draft.capacityEnabled?{afterIngest:true}:{})})} /></>}><form className="settings-form downloader-cleanup" onSubmit={event => event.preventDefault()}>
    {draft.enabled ? <>
      <label className="settings-field"><FieldLabel text={t("downloader.cleanupTrigger")} info={t(draft.capacityEnabled ? "downloader.capacityHint" : "downloader.afterIngestHint")} /><select aria-label={t("downloader.cleanupTrigger")} value={draft.capacityEnabled ? "CAPACITY" : "AFTER_INGEST"} onChange={event=>{
        const capacityEnabled=event.target.value==="CAPACITY";
        patch({afterIngest:!capacityEnabled,capacityEnabled,...(capacityEnabled?{deleteFiles:true}:{})});
      }}><option value="AFTER_INGEST">{t("downloader.cleanupAfterIngest")}</option><option value="CAPACITY">{t("downloader.cleanupCapacity")}</option></select></label>
      {draft.capacityEnabled ? <div className="settings-grid">
        <label className="settings-field"><FieldLabel text={t("downloader.minFreeGb")} /><input type="number" min={0.1} max={100000000} step="0.1" required value={draft.minFreeGb} onChange={event=>patch({minFreeGb:event.target.value})}/></label>
        <label className="settings-field"><FieldLabel text={t("downloader.capacityOrder")} info={t("downloader.capacityOrderHint")} /><select aria-label={t("downloader.capacityOrder")} value={draft.capacityOrder} onChange={event=>patch({capacityOrder:event.target.value as CapacityCleanupOrder})}>{Object.values(CapacityCleanupOrder).map(order=><option key={order} value={order}>{t(`downloader.cleanupOrders.${order}`)}</option>)}</select></label>
      </div> : null}
      <DownloaderSwitch label={t("downloader.policyDeleteFiles")} info={t("downloader.copyGuard")} checked={draft.deleteFiles} disabled={draft.capacityEnabled} onChange={deleteFiles=>patch({deleteFiles})}/>
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
