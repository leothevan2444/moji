import { useMemo, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useMutation, useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { DownloaderSettingsDocumentDocument, SetDownloaderSpeedLimitDocumentDocument, UpdateDownloaderPolicyDocumentDocument, type DownloaderSettingsDocumentQuery } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { useSettingsDraft } from "./SettingsDraftStore";
import { FieldLabel, SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import "../../styles/settings.scss";

type Snapshot = DownloaderSettingsDocumentQuery["downloader"];
type Trigger = "RATIO" | "SEED_TIME" | "ANY" | "ALL" | "AFTER_INGEST";
interface PolicyDraft { enabled: boolean; trigger: Trigger; ratio: string; hours: string; deleteFiles: boolean }
export default function DownloaderSettingsTab() {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refresh] = useQuery({ query: DownloaderSettingsDocumentDocument, requestPolicy: "network-only" });
  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.downloader")} />;
  if (error && !data) return <SettingsError title={t("settings.tabs.downloader")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;
  if (!data) return null;
  return <div className="drawer-stack"><CleanupSettings snapshot={data.downloader} onSaved={() => refresh({ requestPolicy: "network-only" })} /><SettingsCard title={t("downloader.speedLimits")}><p className="settings-note">{t("downloader.limitHelp")}</p><div className="settings-grid"><SpeedLimit snapshot={data.downloader} upload={false} /><SpeedLimit snapshot={data.downloader} upload /></div></SettingsCard></div>;
}
function CleanupSettings({ snapshot, onSaved }: { snapshot: Snapshot; onSaved: () => void }) {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const initial = useMemo<PolicyDraft>(() => {
    const p = snapshot.policy;
    const trigger: Trigger = p.afterIngest ? "AFTER_INGEST" : p.ratio > 0 && p.seedHours > 0 ? p.matchAll ? "ALL" : "ANY" : p.seedHours > 0 ? "SEED_TIME" : "RATIO";
    return { enabled: p.enabled, trigger, ratio: String(p.ratio || 2), hours: String(p.seedHours || 72), deleteFiles: p.deleteFiles };
  }, [snapshot.policy]);
  const [draft, setDraft, , markSaved] = useSettingsDraft("downloader:policy", initial);
  const [{ fetching: saving }, save] = useMutation(UpdateDownloaderPolicyDocumentDocument);
  const ratioEnabled = ["RATIO", "ANY", "ALL"].includes(draft.trigger);
  const timeEnabled = ["SEED_TIME", "ANY", "ALL"].includes(draft.trigger);
  const patch = (value: Partial<PolicyDraft>) => setDraft(current => ({ ...current, ...value }));
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const result = await save({ input: { enabled: draft.enabled, afterIngest: draft.trigger === "AFTER_INGEST", ratio: ratioEnabled ? Number(draft.ratio) : 0, seedHours: timeEnabled ? Number(draft.hours) : 0, matchAll: draft.trigger === "ALL", deleteFiles: draft.deleteFiles } });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    markSaved(draft); pushToast("tone-success", t("downloader.saved")); onSaved();
  };
  return <SettingsCard title={t("downloader.policy")}><form className="settings-form downloader-cleanup" onSubmit={event => void submit(event)}>
    <label className="switch-row"><span className="switch-row__label">{t("downloader.enablePolicy")}</span><span className="switch" role="switch" aria-checked={draft.enabled}><input type="checkbox" aria-label={t("downloader.enablePolicy")} checked={draft.enabled} disabled={saving} onChange={event => patch({ enabled: event.target.checked })} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></span></label>
    {draft.enabled ? <>
    <label className="settings-field"><FieldLabel text={t("downloader.trigger")} info={draft.trigger === "AFTER_INGEST" ? t("downloader.afterIngestHint") : undefined} /><select value={draft.trigger} onChange={event => patch({ trigger: event.target.value as Trigger })}>{(["RATIO", "SEED_TIME", "ANY", "ALL", "AFTER_INGEST"] as const).map(mode => <option key={mode} value={mode}>{t(`downloader.triggers.${mode}`)}</option>)}</select></label>
    <div className="settings-grid">
      {ratioEnabled ? <label className="settings-field"><FieldLabel text={t("downloader.ratioLimit")} /><input type="number" min={0.1} max={9999} step="0.1" required value={draft.ratio} onChange={event => patch({ ratio: event.target.value })} /></label> : null}
      {timeEnabled ? <label className="settings-field"><FieldLabel text={t("downloader.seedHoursLimit")} /><input type="number" min={0.1} max={87600} step="0.1" required value={draft.hours} onChange={event => patch({ hours: event.target.value })} /></label> : null}
    </div>
    <div className="switch-row"><span className="switch-row__label"><FieldLabel text={t("downloader.policyDeleteFiles")} info={t("downloader.copyGuard")} /></span><label className="switch" role="switch" aria-checked={draft.deleteFiles}><input type="checkbox" aria-label={t("downloader.policyDeleteFiles")} checked={draft.deleteFiles} disabled={saving} onChange={event => patch({ deleteFiles: event.target.checked })} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></label></div>
    </> : null}
    <div className="settings-actions"><button type="submit" disabled={saving}>{t(saving ? "settings.saving" : "downloader.savePolicy")}</button></div>
  </form></SettingsCard>;
}
function SpeedLimit({ snapshot, upload }: { snapshot: Snapshot; upload: boolean }) {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const initial = String(Math.round((upload ? snapshot.uploadLimit : snapshot.downloadLimit) / 1024));
  const [limit, setLimit, , markSaved] = useSettingsDraft(upload ? "downloader:upload" : "downloader:download", initial);
  const [{ fetching: saving }, save] = useMutation(SetDownloaderSpeedLimitDocumentDocument);
  return <form className="settings-form" onSubmit={event => {
    event.preventDefault();
    void save({ upload, kibPerSecond: Number(limit) }).then(result => {
      if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
      markSaved(limit); pushToast("tone-success", t("downloader.commandSent"));
    });
  }}>
    <label className="settings-field"><FieldLabel text={t(upload ? "downloader.uploadLimit" : "downloader.downloadLimit")} /><input type="number" min={0} max={10485760} step={1} required value={limit} onChange={event => setLimit(event.target.value)} /></label>
    <div className="settings-actions"><button type="submit" disabled={saving || snapshot.stale || !snapshot.storageAvailable}>{t(saving ? "settings.saving" : "downloader.saveLimit")}</button></div>
  </form>;
}
