import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "urql";
import { useServiceStatusEvents } from "../../hooks/useServiceStatusEvents";
import { describeQueryError } from "../../services/queryError";
import { serviceStatus } from "../../utils";
import {
  ServiceConnectionSettingsDocument,
  ServiceConnectionStatusDocument,

} from "../../graphql/generated/graphql";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { SecretInput, SettingsCard, SettingsSaveStatus, SettingsError, SettingsLoading } from "./SettingsTabCommon";

interface ServiceConnectionDraft {
  stash: { url: string; apiKey: string };
  jackett: { url: string; apiKey: string; password: string };
  qbittorrent: { url: string; username: string; password: string; defaultSavePath: string; category: string; tags: string };
}

export default function ServiceConnectionSettings({ service = "stash" }: { service?: "stash" | "jackett" | "qbittorrent" }) {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refreshSettings] = useQuery({ query: ServiceConnectionSettingsDocument });
  const [{ data: statusData, error: statusError }, refreshStatus] = useQuery({ query: ServiceConnectionStatusDocument });
  useServiceStatusEvents({ onRefresh: () => refreshStatus({ requestPolicy: "network-only" }) });
  const initial = useMemo<ServiceConnectionDraft>(() => ({
    stash: { url: data?.settings.stash.url ?? "", apiKey: data?.settings.stash.apiKey ?? "" },
    jackett: { url: data?.settings.jackett.url ?? "", apiKey: data?.settings.jackett.apiKey ?? "", password: data?.settings.jackett.password ?? "" },
    qbittorrent: { url: data?.settings.qbittorrent.url ?? "", username: data?.settings.qbittorrent.username ?? "", password: data?.settings.qbittorrent.password ?? "", defaultSavePath: data?.settings.qbittorrent.defaultSavePath ?? "", category: data?.settings.qbittorrent.category ?? "", tags: data?.settings.qbittorrent.tags ?? "" }
  }), [data]);
  const autosave = useSettingsAutosave(`${service}:connection`, initial, value => {
    const fields = Object.fromEntries(Object.entries(value[service]).map(([key, item]) => [({apiKey:"api_key",defaultSavePath:"default_save_path"} as Record<string,string>)[key] ?? key, item.trim()]));
    return service === "stash" ? { stash: { connection: fields } } : service === "jackett" ? { indexers: { jackett: fields } } : { downloader: { qbittorrent: fields } };
  }, { onSaved: () => refreshStatus({ requestPolicy: "network-only" }) });
  const {value:form,setValue:setForm} = autosave;

  const serviceLabel = service === "stash" ? "Stash" : service === "jackett" ? "Jackett" : "qBittorrent";
  if (fetching && !data) return <SettingsLoading title={serviceLabel} />;
  if (error && !data) return <SettingsError title={serviceLabel} error={error} onRetry={() => refreshSettings({ requestPolicy: "network-only" })} />;

  const status = statusData?.settingsStatus;
  const statusChip = (service: "stash" | "jackett" | "qbittorrent") => {
    const current = status?.[service];
    const stats = status?.[`${service}Stats` as "stashStats"] as { lastError?: string | null; okAt?: string | null } | undefined;
    return serviceStatus(current?.configured ?? false, current?.ready ?? false, stats?.lastError ?? null, stats?.okAt ?? null);
  };
  const serviceForm = (service: "stash" | "jackett" | "qbittorrent", label: string) => {
    const chip = statusChip(service);
    return <form className="settings-form" onSubmit={event => event.preventDefault()}>
      <div className="settings-meta"><span>{label}</span><span className={`status-chip ${chip.tone}`}>{t(chip.labelKey)}</span></div>
      <label className="settings-field"><span>{label} URL</span><input value={form[service].url} onChange={(event) => setForm((current) => ({ ...current, [service]: { ...current[service], url: event.target.value } }))} /></label>
      {service === "qbittorrent" ? <label className="settings-field"><span>{t("serviceConnection.username")}</span><input value={form.qbittorrent.username} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, username: event.target.value } }))} /></label> : null}
      <label className="settings-field"><span>{service === "qbittorrent" ? t("serviceConnection.password") : "API key"}</span><SecretInput value={service === "stash" ? form.stash.apiKey : service === "jackett" ? form.jackett.apiKey : form.qbittorrent.password} onChange={(value) => setForm((current) => service === "stash" ? { ...current, stash: { ...current.stash, apiKey: value } } : service === "jackett" ? { ...current, jackett: { ...current.jackett, apiKey: value } } : { ...current, qbittorrent: { ...current.qbittorrent, password: value } })} /></label>
      {service === "jackett" ? <label className="settings-field"><span>{t("serviceConnection.dashboardPassword")}</span><SecretInput value={form.jackett.password} onChange={(value) => setForm((current) => ({ ...current, jackett: { ...current.jackett, password: value } }))} placeholder={t("serviceConnection.dashboardPasswordPlaceholder")} /></label> : null}
      {service === "qbittorrent" ? <><label className="settings-field"><span>{t("serviceConnection.defaultSavePath")}</span><input value={form.qbittorrent.defaultSavePath} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, defaultSavePath: event.target.value } }))} /></label><label className="settings-field"><span>{t("serviceConnection.defaultCategory")}</span><input value={form.qbittorrent.category} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, category: event.target.value } }))} /></label><label className="settings-field"><span>{t("serviceConnection.defaultTags")}</span><input value={form.qbittorrent.tags} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, tags: event.target.value } }))} /></label></> : null}
      <SettingsSaveStatus {...autosave} />
    </form>;
  };

  return <SettingsCard title={serviceLabel}>
    {statusError ? <p className="settings-feedback tone-danger">{describeQueryError(statusError)}</p> : null}
    {serviceForm(service, service === "stash" ? "Stash" : service === "jackett" ? "Jackett" : "qBittorrent")}
  </SettingsCard>;
}
