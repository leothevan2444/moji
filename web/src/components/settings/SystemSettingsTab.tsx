import { useMemo, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useMutation, useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { ClearImageCacheDocumentDocument, SystemSettingsTabDocument, TaskDeletePolicy, UpdateSystemSettingsDocumentDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { formatBytes, formatDateTime } from "../../utils";
import { useSettingsDraft } from "./SettingsDraftStore";
import { SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";

interface SystemDraft { taskDeletePolicy: TaskDeletePolicy; enabled: boolean; maxSizeMb: string; retentionDays: string }

export default function SystemSettingsTab() {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [{ data, fetching, error }, refresh] = useQuery({ query: SystemSettingsTabDocument, requestPolicy: "cache-first" });
  const [{ fetching: saving }, updateSystem] = useMutation(UpdateSystemSettingsDocumentDocument);
  const [{ fetching: clearing }, clearImageCache] = useMutation(ClearImageCacheDocumentDocument);
  const [confirming, setConfirming] = useState(false);
  const initial = useMemo<SystemDraft>(() => ({ taskDeletePolicy: data?.settings.system.taskDeletePolicy ?? TaskDeletePolicy.KeepOnly, enabled: data?.settings.system.imageCache.enabled ?? true, maxSizeMb: String(data?.settings.system.imageCache.maxSizeMb ?? 1024), retentionDays: String(data?.settings.system.imageCache.retentionDays ?? 30) }), [data]);
  const [form, setForm, , markSaved] = useSettingsDraft("system", initial);

  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.system")} />;
  if (error && !data) return <SettingsError title={t("settings.tabs.system")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;
  const image = data?.settingsStatus.imageCache;

  const save = async (event: FormEvent) => {
    event.preventDefault();
    const result = await updateSystem({ input: { taskDeletePolicy: form.taskDeletePolicy, imageCache: { enabled: form.enabled, maxSizeMb: Number(form.maxSizeMb), retentionDays: Number(form.retentionDays) } } });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    markSaved(form); pushToast("tone-success", t("systemUi.saved"));
  };
  const clear = async () => {
    const released = Number(image?.usedBytes ?? 0);
    const result = await clearImageCache({});
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    setConfirming(false); pushToast("tone-success", released ? t("systemUi.clearedBytes", { size: formatBytes(released) }) : t("systemUi.cleared"));
  };

  return <SettingsCard title={t("settings.tabs.system")}><form className="settings-form" onSubmit={(event) => void save(event)}>
    <label className="settings-field"><span>{t("systemUi.deletePolicy")}</span><select value={form.taskDeletePolicy} onChange={(event) => setForm((current) => ({ ...current, taskDeletePolicy: event.target.value as TaskDeletePolicy }))}><option value={TaskDeletePolicy.KeepOnly}>{t("systemUi.keep")}</option><option value={TaskDeletePolicy.RemoveTorrent}>{t("systemUi.removeTorrent")}</option><option value={TaskDeletePolicy.RemoveTorrentAndFiles}>{t("systemUi.removeFiles")}</option></select></label>
    <label className="settings-field settings-field--switch"><span>{t("systemUi.cache")}</span><input type="checkbox" checked={form.enabled} onChange={(event) => setForm((current) => ({ ...current, enabled: event.target.checked }))} /></label>
    <label className="settings-field"><span>{t("systemUi.maxSize")}</span><input type="number" value={form.maxSizeMb} onChange={(event) => setForm((current) => ({ ...current, maxSizeMb: event.target.value }))} /></label>
    <label className="settings-field"><span>{t("systemUi.retention")}</span><input type="number" value={form.retentionDays} onChange={(event) => setForm((current) => ({ ...current, retentionDays: event.target.value }))} /></label>
    <div className="image-cache-management"><div><div className="settings-meta"><span>{t("systemUi.usage", { size: formatBytes(Number(image?.usedBytes ?? 0)) })}</span><span>{t("systemUi.images", { count: image?.entryCount ?? 0 })}</span><span>{t("systemUi.cleanup", { time: formatDateTime(image?.lastCleanupAt) })}</span></div><p className="image-cache-management__hint">{t("systemUi.clearHint")}</p></div><button type="button" className="image-cache-management__clear" disabled={clearing || !image?.entryCount} onClick={() => setConfirming(true)}>{clearing ? t("systemUi.clearing") : image?.entryCount ? t("systemUi.clear") : t("systemUi.noCache")}</button></div>
    {confirming ? <div className="image-cache-confirm" role="alertdialog"><div><strong>{t("systemUi.clearTitle")}</strong><p>{t("systemUi.clearDescription", { count: image?.entryCount ?? 0, size: formatBytes(Number(image?.usedBytes ?? 0)) })}</p></div><div className="image-cache-confirm__actions"><button type="button" className="ghost-button" onClick={() => setConfirming(false)}>{t("systemUi.cancel")}</button><button type="button" className="image-cache-confirm__submit" onClick={() => void clear()}>{t("systemUi.confirm")}</button></div></div> : null}
    {image?.lastError ? <p className="settings-feedback tone-danger">{image.lastError}</p> : null}
    <div className="settings-actions"><button type="submit" disabled={saving}>{saving ? t("settings.saving") : t("systemUi.save")}</button></div>
  </form></SettingsCard>;
}
