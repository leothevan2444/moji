import { SettingsSubpanel, useSettingsPanel } from "./SettingsPanels";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useMutation, useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { ClearImageCacheDocumentDocument, SystemSettingsTabDocument } from "../../graphql/generated/graphql";
import { SUPPORTED_LOCALES, isLocalePreference, type LocalePreference } from "../../i18n/locales";
import { describeQueryError } from "../../services/queryError";
import { formatBytes, formatDateTime } from "../../utils";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsCard, SettingsSaveStatus, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import "../../styles/settings-system.scss";

interface SystemDraft { releaseTimezone: string; developerMode: boolean; language: LocalePreference; enabled: boolean; maxSizeMb: string; retentionDays: string }

export default function SystemSettingsTab() {
  const { t } = useTranslation();
  const panel = useSettingsPanel();
  const { pushToast, onDeveloperModeChange, onLanguageChange, localeLoadError, retryLocale } = useOutletContext<AppOutletContext>();
  const [{ data, fetching, error }, refresh] = useQuery({ query: SystemSettingsTabDocument });
  const [{ fetching: clearing }, clearImageCache] = useMutation(ClearImageCacheDocumentDocument);

  const [confirming, setConfirming] = useState(false);

  const initial = useMemo<SystemDraft>(() => ({ releaseTimezone: data?.settings.system.releaseTimezone ?? "auto", developerMode: data?.settings.system.developerMode ?? false, language: isLocalePreference(data?.settings.system.language) ? data.settings.system.language : "auto", enabled: data?.settings.system.imageCache.enabled ?? true, maxSizeMb: String(data?.settings.system.imageCache.maxSizeMb ?? 1024), retentionDays: String(data?.settings.system.imageCache.retentionDays ?? 30) }), [data]);
  const autosave = useSettingsAutosave("system", initial, value => ({system:{
    release_timezone:value.releaseTimezone,language:value.language,developer_mode:value.developerMode,
    image_cache:{enabled:value.enabled,max_size_mb:value.maxSizeMb.trim() ? Number(value.maxSizeMb) : NaN,retention_days:value.retentionDays.trim() ? Number(value.retentionDays) : NaN}
  }}), {onSaved:(value, patch)=>{
    const system = patch.system as Record<string,unknown>;
    if ("developer_mode" in system) onDeveloperModeChange?.(value.developerMode);
    if ("language" in system) onLanguageChange?.(value.language);
  }});
  const {value:form,setValue:setForm}=autosave;

  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.system")} />;
  if (error && !data) return <SettingsError title={t("settings.tabs.system")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;
  const image = data?.settingsStatus.imageCache;

  const clear = async () => {
    const released = Number(image?.usedBytes ?? 0);
    const result = await clearImageCache({});
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    setConfirming(false); pushToast("tone-success", released ? t("systemSettings.clearedBytes", { size: formatBytes(released) }) : t("systemSettings.cleared"));
    refresh();
  };

  return <SettingsCard title={t(panel ? `settingsLayout.panels.${panel}` : "settings.tabs.system")}><form className="settings-form" onSubmit={event => event.preventDefault()}>
    <SettingsSubpanel id="general">
    <label className="settings-field"><FieldLabel text={t("common.language")} /><select aria-label={t("common.language")} aria-describedby="system-language-help" value={form.language} onChange={event => setForm(current => ({...current,language:event.target.value as LocalePreference}))}><option value="auto">{t("common.auto")}</option>{Object.entries(SUPPORTED_LOCALES).map(([value,meta]) => <option key={value} value={value}>{meta.label}</option>)}</select><small id="system-language-help">{t("systemSettings.languageHelp")}</small></label>
    <label className="settings-field"><FieldLabel text={t("upcoming.timezoneLabel")} /><input aria-label={t("upcoming.timezoneLabel")} list="release-timezones" value={form.releaseTimezone} onChange={event => setForm(current => ({...current,releaseTimezone:event.target.value}))} /><datalist id="release-timezones"><option value="auto" /><option value="UTC" /><option value="Asia/Shanghai" /><option value="Asia/Tokyo" /><option value="America/New_York" /><option value="Europe/London" /></datalist><small>{t("upcoming.timezoneHelp", { timezone: data?.settings.system.effectiveReleaseTimezone ?? "" })}</small></label>
    {localeLoadError ? <p className="settings-feedback tone-danger" role="alert">{t("localeUi.loadFailed")} <button type="button" onClick={retryLocale}>{t("localeUi.retry")}</button></p> : null}
    <div className="settings-field"><label className="switch-row image-cache-switch developer-mode-switch"><FieldLabel text={t("systemSettings.developerMode")} info={t("systemSettings.developerModeInfo")} /><span className="switch"><input role="switch" type="checkbox" aria-checked={form.developerMode} checked={form.developerMode} aria-label={t("systemSettings.developerMode")} onChange={(event) => setForm((current) => ({ ...current, developerMode: event.target.checked }))} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></span></label></div>

    </SettingsSubpanel>
    <SettingsSubpanel id="imageCache">
    <div className="settings-field"><label className="switch-row image-cache-switch"><FieldLabel text={t("systemSettings.enableCache")} info={t("systemSettings.cacheInfo")} /><span className="switch" role="switch" aria-checked={form.enabled}><input type="checkbox" checked={form.enabled} aria-label={t("systemSettings.enableCache")} onChange={(event) => setForm((current) => ({ ...current, enabled: event.target.checked }))} /><span className="switch__track" aria-hidden="true" /><span className="switch__thumb" aria-hidden="true" /></span></label></div>
    <div className="image-cache-config">
      <label className="settings-field"><FieldLabel text={t("systemSettings.maxSize")} info={t("systemSettings.maxSizeInfo")} /><input type="number" min="64" max="20480" value={form.maxSizeMb} onChange={(event) => setForm((current) => ({ ...current, maxSizeMb: event.target.value }))} /></label>
      <label className="settings-field"><FieldLabel text={t("systemSettings.retention")} info={t("systemSettings.retentionInfo")} /><input type="number" min="1" max="365" value={form.retentionDays} onChange={(event) => setForm((current) => ({ ...current, retentionDays: event.target.value }))} /></label>
    </div>
    <div className="image-cache-management"><div><div className="settings-meta"><span>{t("systemSettings.usage", { size: formatBytes(Number(image?.usedBytes ?? 0)) })}</span><span>{t("systemSettings.images", { count: image?.entryCount ?? 0 })}</span><span>{t("systemSettings.cleanup", { time: formatDateTime(image?.lastCleanupAt) })}</span></div></div><button type="button" className="image-cache-management__clear" disabled={clearing || !image?.entryCount} onClick={() => setConfirming(true)}>{clearing ? t("systemSettings.clearing") : t("systemSettings.clear")}</button></div>
    {confirming ? <div className="image-cache-confirm" role="alertdialog"><div><strong>{t("systemSettings.clearTitle")}</strong><p>{t("systemSettings.clearDescription", { count: image?.entryCount ?? 0, size: formatBytes(Number(image?.usedBytes ?? 0)) })}</p></div><div className="image-cache-confirm__actions"><button type="button" className="ghost-button" onClick={() => setConfirming(false)}>{t("systemSettings.cancel")}</button><button type="button" className="image-cache-confirm__submit" onClick={() => void clear()}>{t("systemSettings.confirm")}</button></div></div> : null}
    {image?.lastError ? <p className="settings-feedback tone-danger">{image.lastError}</p> : null}
    </SettingsSubpanel>
    <SettingsSaveStatus {...autosave} />
  </form></SettingsCard>;
}
