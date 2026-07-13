import { useMemo, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useMutation, useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { IngestSettingsTabDocument, UpdateIngestSettingsDocumentDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { useSettingsDraft } from "./SettingsDraftStore";
import { FieldLabel, SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";

interface IngestDraft { deliveryMode: string; qbRoot: string; mojiDownloadsRoot: string; mojiLibraryRoot: string; stashRoot: string; transferAction: string }

export default function IngestSettingsTab() {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [{ data, fetching, error }, refresh] = useQuery({ query: IngestSettingsTabDocument, requestPolicy: "cache-first" });
  const [{ fetching: saving }, updateIngest] = useMutation(UpdateIngestSettingsDocumentDocument);
  const [pendingDefault, setPendingDefault] = useState<string | null>(null);
  const initial = useMemo<IngestDraft>(() => ({
    deliveryMode: data?.settings.ingest.deliveryMode ?? "PATH_MAP",
    qbRoot: data?.settings.ingest.downloads.qbRoot ?? "",
    mojiDownloadsRoot: data?.settings.ingest.downloads.mojiRoot ?? "",
    mojiLibraryRoot: data?.settings.ingest.library.mojiRoot ?? "",
    stashRoot: data?.settings.ingest.library.stashRoot ?? "",
    transferAction: data?.settings.ingest.transfer.action ?? "COPY"
  }), [data]);
  const [form, setForm, , markSaved] = useSettingsDraft("ingest", initial);

  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.ingest")} />;
  if (error && !data) return <SettingsError title={t("settings.tabs.ingest")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;

  const submit = async (qbRoot: string) => {
    const result = await updateIngest({ input: { deliveryMode: form.deliveryMode, downloads: { qbRoot, mojiRoot: form.mojiDownloadsRoot.trim() }, library: { mojiRoot: form.mojiLibraryRoot.trim(), stashRoot: form.stashRoot.trim() }, transfer: { action: form.transferAction } } });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    const saved = { ...form, qbRoot };
    markSaved(saved);
    setPendingDefault(null);
    pushToast("tone-success", t("settings.ingest.saved"));
  };
  const save = (event: FormEvent) => {
    event.preventDefault();
    const qbRoot = form.qbRoot.trim();
    const fallback = data?.settings.qbittorrent.defaultSavePath.trim() ?? "";
    if (!qbRoot && fallback) setPendingDefault(fallback); else void submit(qbRoot);
  };
  const libraries = data?.settingsStatus.stashLibraries ?? [];

  return <SettingsCard title={t("settings.tabs.ingest")}>
    <form className="settings-form" onSubmit={save}>
      <label className="settings-field"><FieldLabel text={t("settings.ingest.mode")} info={form.deliveryMode === "PATH_MAP" ? t("settings.ingest.pathMapInfo") : t("settings.ingest.transferInfo")} /><select value={form.deliveryMode} onChange={(event) => setForm((current) => ({ ...current, deliveryMode: event.target.value }))}><option value="PATH_MAP">{t("settings.ingest.pathMap")}</option><option value="TRANSFER">{t("settings.ingest.transfer")}</option></select></label>
      <label className="settings-field"><FieldLabel text={t("settings.ingest.qbRoot")} info={t("settings.ingest.qbRootInfo")} /><input value={form.qbRoot} onChange={(event) => setForm((current) => ({ ...current, qbRoot: event.target.value }))} placeholder="/downloads" /></label>
      <label className="settings-field"><FieldLabel text={t("settings.ingest.stashRoot")} info={t("settings.ingest.stashInfo")} /><select value={form.stashRoot} onChange={(event) => setForm((current) => ({ ...current, stashRoot: event.target.value }))}><option value="">{libraries.length ? t("settings.ingest.selectStashRoot") : t("settings.ingest.noStashRoot")}</option>{form.stashRoot && !libraries.some((item) => item.path === form.stashRoot) ? <option value={form.stashRoot}>{form.stashRoot}</option> : null}{libraries.map((item) => <option key={item.path} value={item.path}>{item.path}</option>)}</select></label>
      {data?.settingsStatus.stashLibrariesLoadError ? <p className="service-card__error" role="alert">{data.settingsStatus.stashLibrariesLoadError}</p> : null}
      {form.deliveryMode === "TRANSFER" ? <><label className="settings-field"><FieldLabel text={t("settings.ingest.action")} info={t("settings.ingest.actionInfo")} /><select value={form.transferAction} onChange={(event) => setForm((current) => ({ ...current, transferAction: event.target.value }))}><option value="COPY">{t("settings.ingest.copy")}</option><option value="MOVE">{t("settings.ingest.move")}</option><option value="SYMLINK">{t("settings.ingest.symlink")}</option></select></label><label className="settings-field"><FieldLabel text={t("settings.ingest.mojiDownloadRoot")} info={t("settings.ingest.mojiDownloadInfo")} /><input value={form.mojiDownloadsRoot} onChange={(event) => setForm((current) => ({ ...current, mojiDownloadsRoot: event.target.value }))} /></label><label className="settings-field"><FieldLabel text={t("settings.ingest.mojiLibraryRoot")} info={t("settings.ingest.mojiLibraryInfo")} /><input value={form.mojiLibraryRoot} onChange={(event) => setForm((current) => ({ ...current, mojiLibraryRoot: event.target.value }))} /></label></> : null}
      <div className="settings-actions"><button type="submit" disabled={saving}>{saving ? t("settings.saving") : t("settings.ingest.save")}</button></div>
    </form>
    {pendingDefault ? <div className="image-cache-confirm" role="alertdialog"><div><strong>{t("settings.ingest.initTitle")}</strong><p>{t("settings.ingest.initQuestion", { path: pendingDefault })}</p></div><div className="image-cache-confirm__actions"><button type="button" onClick={() => void submit(pendingDefault)} disabled={saving}>{t("settings.ingest.useAndSave")}</button><button type="button" className="ghost-button" onClick={() => void submit("")} disabled={saving}>{t("settings.ingest.keepEmpty")}</button><button type="button" className="ghost-button" onClick={() => setPendingDefault(null)}>{t("common.close")}</button></div></div> : null}
  </SettingsCard>;
}
