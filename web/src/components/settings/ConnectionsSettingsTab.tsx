import { useMemo, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useMutation, useQuery } from "urql";
import { useServiceStatusEvents } from "../../hooks/useServiceStatusEvents";
import { describeQueryError } from "../../services/queryError";
import { serviceStatus } from "../../utils";
import {
  ConnectionsSettingsDocument,
  ConnectionsStatusDocument,
  UpdateJackettSettingsDocumentDocument,
  UpdateQBittorrentSettingsDocumentDocument,
  UpdateStashSettingsDocumentDocument
} from "../../graphql/generated/graphql";
import type { AppOutletContext } from "../../app/AppLayout";
import { useSettingsDraft } from "./SettingsDraftStore";
import { SecretInput, SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";

interface ConnectionsDraft {
  stash: { url: string; apiKey: string };
  jackett: { url: string; apiKey: string; password: string };
  qbittorrent: { url: string; username: string; password: string; defaultSavePath: string; category: string; tags: string };
}

export default function ConnectionsSettingsTab() {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [{ data, fetching, error }, refreshSettings] = useQuery({ query: ConnectionsSettingsDocument, requestPolicy: "cache-first" });
  const [{ data: statusData, error: statusError }, refreshStatus] = useQuery({ query: ConnectionsStatusDocument, requestPolicy: "cache-and-network" });
  useServiceStatusEvents({ onRefresh: () => refreshStatus({ requestPolicy: "network-only" }) });
  const [, updateStash] = useMutation(UpdateStashSettingsDocumentDocument);
  const [, updateJackett] = useMutation(UpdateJackettSettingsDocumentDocument);
  const [, updateQBittorrent] = useMutation(UpdateQBittorrentSettingsDocumentDocument);
  const initial = useMemo<ConnectionsDraft>(() => ({
    stash: { url: data?.settings.stash.url ?? "", apiKey: data?.settings.stash.apiKey ?? "" },
    jackett: { url: data?.settings.jackett.url ?? "", apiKey: data?.settings.jackett.apiKey ?? "", password: data?.settings.jackett.password ?? "" },
    qbittorrent: { url: data?.settings.qbittorrent.url ?? "", username: data?.settings.qbittorrent.username ?? "", password: data?.settings.qbittorrent.password ?? "", defaultSavePath: data?.settings.qbittorrent.defaultSavePath ?? "", category: data?.settings.qbittorrent.category ?? "", tags: data?.settings.qbittorrent.tags ?? "" }
  }), [data]);
  const [form, setForm, , markSaved] = useSettingsDraft("connections", initial);

  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.connections")} />;
  if (error && !data) return <SettingsError title={t("settings.tabs.connections")} error={error} onRetry={() => refreshSettings({ requestPolicy: "network-only" })} />;

  const status = statusData?.settingsStatus;
  const statusChip = (service: "stash" | "jackett" | "qbittorrent") => {
    const current = status?.[service];
    const stats = status?.[`${service}Stats` as "stashStats"] as { lastError?: string | null; okAt?: string | null } | undefined;
    return serviceStatus(current?.configured ?? false, current?.ready ?? false, stats?.lastError ?? null, stats?.okAt ?? null);
  };
  const save = async (service: "stash" | "jackett" | "qbittorrent", event: FormEvent) => {
    event.preventDefault();
    const result = service === "stash"
      ? await updateStash({ input: { url: form.stash.url.trim(), apiKey: form.stash.apiKey.trim() } })
      : service === "jackett"
        ? await updateJackett({ input: { url: form.jackett.url.trim(), apiKey: form.jackett.apiKey.trim(), password: form.jackett.password.trim() } })
        : await updateQBittorrent({ input: { ...form.qbittorrent, url: form.qbittorrent.url.trim(), username: form.qbittorrent.username.trim(), password: form.qbittorrent.password.trim(), defaultSavePath: form.qbittorrent.defaultSavePath.trim(), category: form.qbittorrent.category.trim(), tags: form.qbittorrent.tags.trim() } });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    markSaved(form);
    pushToast("tone-success", t("settings.connections.saved", { service: service === "stash" ? "Stash" : service === "jackett" ? "Jackett" : "qBittorrent" }));
  };

  const serviceForm = (service: "stash" | "jackett" | "qbittorrent", label: string) => {
    const chip = statusChip(service);
    return <form className="settings-form" onSubmit={(event) => void save(service, event)}>
      <div className="settings-meta"><span>{label}</span><span className={`status-chip ${chip.tone}`}>{t(chip.labelKey)}</span></div>
      <label className="settings-field"><span>{label} URL</span><input value={form[service].url} onChange={(event) => setForm((current) => ({ ...current, [service]: { ...current[service], url: event.target.value } }))} /></label>
      {service === "qbittorrent" ? <label className="settings-field"><span>{t("settings.connections.username")}</span><input value={form.qbittorrent.username} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, username: event.target.value } }))} /></label> : null}
      <label className="settings-field"><span>API key{service === "qbittorrent" ? ` / ${t("settings.connections.password")}` : ""}</span><SecretInput value={service === "stash" ? form.stash.apiKey : service === "jackett" ? form.jackett.apiKey : form.qbittorrent.password} onChange={(value) => setForm((current) => service === "stash" ? { ...current, stash: { ...current.stash, apiKey: value } } : service === "jackett" ? { ...current, jackett: { ...current.jackett, apiKey: value } } : { ...current, qbittorrent: { ...current.qbittorrent, password: value } })} /></label>
      {service === "jackett" ? <label className="settings-field"><span>{t("settings.connections.dashboardPassword")}</span><SecretInput value={form.jackett.password} onChange={(value) => setForm((current) => ({ ...current, jackett: { ...current.jackett, password: value } }))} placeholder={t("settings.connections.dashboardPasswordPlaceholder")} /></label> : null}
      {service === "qbittorrent" ? <><label className="settings-field"><span>{t("settings.connections.defaultSavePath")}</span><input value={form.qbittorrent.defaultSavePath} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, defaultSavePath: event.target.value } }))} /></label><label className="settings-field"><span>{t("settings.connections.defaultCategory")}</span><input value={form.qbittorrent.category} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, category: event.target.value } }))} /></label><label className="settings-field"><span>{t("settings.connections.defaultTags")}</span><input value={form.qbittorrent.tags} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, tags: event.target.value } }))} /></label></> : null}
      <div className="settings-actions"><button type="submit">{t("settings.connections.save", { service: label })}</button></div>
    </form>;
  };

  return <SettingsCard title={t("settings.tabs.connections")}>
    {statusError ? <p className="settings-feedback tone-danger">{describeQueryError(statusError)}</p> : null}
    {serviceForm("stash", "Stash")}{serviceForm("jackett", "Jackett")}{serviceForm("qbittorrent", "qBittorrent")}
  </SettingsCard>;
}
