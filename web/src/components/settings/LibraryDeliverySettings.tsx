import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "urql";
import { LibraryDeliverySettingsDocument } from "../../graphql/generated/graphql";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsCard, SettingsSaveStatus, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import "../../styles/settings-system.scss";

interface LibraryDeliveryDraft { keepSubtitles: boolean; generateTranscodes: boolean; qbRoot: string; mojiDownloadsRoot: string; mojiLibraryRoot: string; stashRoot: string; transferAction: string }

export default function LibraryDeliverySettings() {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refresh] = useQuery({ query: LibraryDeliverySettingsDocument });
  const initial = useMemo<LibraryDeliveryDraft>(() => ({
    keepSubtitles: data?.settings.delivery.keepSubtitles ?? false,
    generateTranscodes: data?.settings.delivery.generateTranscodes ?? false,
    qbRoot: data?.settings.delivery.downloads.qbRoot ?? "",
    mojiDownloadsRoot: data?.settings.delivery.downloads.mojiRoot ?? "",
    mojiLibraryRoot: data?.settings.delivery.library.mojiRoot ?? "",
    stashRoot: data?.settings.delivery.library.stashRoot ?? "",
    transferAction: data?.settings.delivery.transfer.action ?? "COPY"
  }), [data]);
  const autosave = useSettingsAutosave("stash:delivery", initial, value => ({stash:{ingest:{
    keep_subtitles:value.keepSubtitles, generate_transcodes:value.generateTranscodes,
    downloads:{qb_root:value.qbRoot.trim(),moji_root:value.mojiDownloadsRoot.trim()},
    library:{moji_root:value.mojiLibraryRoot.trim(),stash_root:value.stashRoot.trim()},
    transfer:{action:value.transferAction}
  }}}));
  const {value:form,setValue:setForm}=autosave;

  if (fetching && !data) return <SettingsLoading title={t("libraryDelivery.title")} />;
  if (error && !data) return <SettingsError title={t("libraryDelivery.title")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;

  const libraries = data?.settingsStatus.stashLibraries ?? [];

  return <SettingsCard title={t("libraryDelivery.title")}>
    <form className="settings-form" onSubmit={event => event.preventDefault()}>
      <label className="settings-field"><span className="switch-row__label"><FieldLabel text={t("libraryDelivery.generateTranscodes")} info={t("libraryDelivery.generateTranscodesInfo")} /></span><span className="switch" role="switch" aria-checked={form.generateTranscodes}><input type="checkbox" checked={form.generateTranscodes} aria-label={t("libraryDelivery.generateTranscodes")} onChange={(event) => setForm((current) => ({ ...current, generateTranscodes: event.target.checked }))} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></span></label>
      <label className="settings-field"><span className="switch-row__label"><FieldLabel text={t("mediaFiles.keepSubtitles")} info={t("mediaFiles.keepSubtitlesInfo")} /></span><span className="switch" role="switch" aria-checked={form.keepSubtitles}><input type="checkbox" checked={form.keepSubtitles} aria-label={t("mediaFiles.keepSubtitles")} onChange={(event) => setForm((current) => ({ ...current, keepSubtitles: event.target.checked }))} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></span></label>
      <label className="settings-field"><FieldLabel text={t("libraryDelivery.qbRoot")} info={t("libraryDelivery.qbRootInfo")} /><input value={form.qbRoot} onChange={(event) => setForm((current) => ({ ...current, qbRoot: event.target.value }))} placeholder="/downloads" /></label>
      <label className="settings-field"><FieldLabel text={t("libraryDelivery.stashRoot")} info={t("libraryDelivery.stashInfo")} /><select value={form.stashRoot} onChange={(event) => setForm((current) => ({ ...current, stashRoot: event.target.value }))}><option value="">{libraries.length ? t("libraryDelivery.selectStashRoot") : t("libraryDelivery.noStashRoot")}</option>{form.stashRoot && !libraries.some((item) => item.path === form.stashRoot) ? <option value={form.stashRoot}>{form.stashRoot}</option> : null}{libraries.map((item) => <option key={item.path} value={item.path}>{item.path}</option>)}</select></label>
      {data?.settingsStatus.stashLibrariesLoadError ? <p className="service-card__error" role="alert">{data.settingsStatus.stashLibrariesLoadError}</p> : null}
      <><label className="settings-field"><FieldLabel text={t("libraryDelivery.action")} info={t("libraryDelivery.actionInfo")} /><select value={form.transferAction} onChange={(event) => setForm((current) => ({ ...current, transferAction: event.target.value }))}><option value="COPY">{t("libraryDelivery.copy")}</option><option value="MOVE">{t("libraryDelivery.move")}</option></select></label><label className="settings-field"><FieldLabel text={t("libraryDelivery.mojiDownloadRoot")} info={t("libraryDelivery.mojiDownloadInfo")} /><input value={form.mojiDownloadsRoot} onChange={(event) => setForm((current) => ({ ...current, mojiDownloadsRoot: event.target.value }))} /></label><label className="settings-field"><FieldLabel text={t("libraryDelivery.mojiLibraryRoot")} info={t("libraryDelivery.mojiLibraryInfo")} /><input value={form.mojiLibraryRoot} onChange={(event) => setForm((current) => ({ ...current, mojiLibraryRoot: event.target.value }))} /></label></>
      <SettingsSaveStatus {...autosave} />
    </form>

  </SettingsCard>;
}
