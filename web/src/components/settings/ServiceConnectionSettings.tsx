import { SettingsGroup } from "./SettingsPanels";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery } from "urql";
import {
  ServiceConnectionSettingsDocument,
  TestServiceConnectionDocument, SettingsConnectionService, ServiceConnectionTestCode,

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
  const [{fetching: testing}, testConnection] = useMutation(TestServiceConnectionDocument);
  const [testResult, setTestResult] = useState<{revision:number;code:string} | null>(null);
  const initial = useMemo<ServiceConnectionDraft>(() => ({
    stash: { url: data?.settings.stash.url ?? "", apiKey: data?.settings.stash.apiKey ?? "" },
    jackett: { url: data?.settings.jackett.url ?? "", apiKey: data?.settings.jackett.apiKey ?? "", password: data?.settings.jackett.password ?? "" },
    qbittorrent: { url: data?.settings.qbittorrent.url ?? "", username: data?.settings.qbittorrent.username ?? "", password: data?.settings.qbittorrent.password ?? "", defaultSavePath: data?.settings.qbittorrent.defaultSavePath ?? "", category: data?.settings.qbittorrent.category ?? "", tags: data?.settings.qbittorrent.tags ?? "" }
  }), [data]);
  const autosave = useSettingsAutosave(`${service}:connection`, initial, value => {
    const fields = Object.fromEntries(Object.entries(value[service]).map(([key, item]) => [({apiKey:"api_key",defaultSavePath:"default_save_path"} as Record<string,string>)[key] ?? key, item.trim()]));
    return service === "stash" ? { stash: { connection: fields } } : service === "jackett" ? { indexers: { jackett: fields } } : { downloader: { qbittorrent: fields } };
  });
  const {value:form,setValue:setForm} = autosave;
  const draft = `${service}:${JSON.stringify(form[service])}`;
  const currentDraft = useRef({draft,revision:0});
  if (currentDraft.current.draft !== draft) currentDraft.current = {draft,revision:currentDraft.current.revision + 1};
  const runTest = async () => {
    const sent = currentDraft.current.revision;
    setTestResult(null);
    const input = service === "stash" ? {service:SettingsConnectionService.Stash,url:form.stash.url,apiKey:form.stash.apiKey}
      : service === "jackett" ? {service:SettingsConnectionService.Jackett,url:form.jackett.url,apiKey:form.jackett.apiKey}
      : {service:SettingsConnectionService.Qbittorrent,url:form.qbittorrent.url,username:form.qbittorrent.username,password:form.qbittorrent.password};
    const result = await testConnection({input});
    if (currentDraft.current.revision === sent) setTestResult({revision:sent,code:result.error ? "FAILED" : result.data?.testServiceConnection.code ?? "FAILED"});
  };

  useEffect(() => {
    if (!testResult) return;
    const timeout = window.setTimeout(() => setTestResult(null), 5000);
    return () => window.clearTimeout(timeout);
  }, [testResult]);

  const serviceLabel = service === "stash" ? "Stash" : service === "jackett" ? "Jackett" : "qBittorrent";
  if (fetching && !data) return <SettingsLoading title={serviceLabel} />;
  if (error && !data) return <SettingsError title={serviceLabel} error={error} onRetry={() => refreshSettings({ requestPolicy: "network-only" })} />;

  const serviceForm = (service: "stash" | "jackett" | "qbittorrent") => {
    return <form className="settings-form" onSubmit={event => event.preventDefault()}>
      <div className="settings-group__fields">
      <label className="settings-field"><span>{t("serviceConnection.url")}</span><input value={form[service].url} onChange={(event) => setForm((current) => ({ ...current, [service]: { ...current[service], url: event.target.value } }))} /></label>
      {service === "qbittorrent" ? <label className="settings-field"><span>{t("serviceConnection.username")}</span><input value={form.qbittorrent.username} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, username: event.target.value } }))} /></label> : null}
      <label className="settings-field"><span>{service === "qbittorrent" ? t("serviceConnection.password") : t("serviceConnection.apiKey")}</span><SecretInput value={service === "stash" ? form.stash.apiKey : service === "jackett" ? form.jackett.apiKey : form.qbittorrent.password} onChange={(value) => setForm((current) => service === "stash" ? { ...current, stash: { ...current.stash, apiKey: value } } : service === "jackett" ? { ...current, jackett: { ...current.jackett, apiKey: value } } : { ...current, qbittorrent: { ...current.qbittorrent, password: value } })} /></label>
      {service === "jackett" ? <label className="settings-field"><span>{t("serviceConnection.dashboardPassword")}</span><SecretInput value={form.jackett.password} onChange={(value) => setForm((current) => ({ ...current, jackett: { ...current.jackett, password: value } }))} /></label> : null}
      <div className="settings-connection-test" aria-busy={testing}>
        <button type="button" disabled={testing || !form[service].url.trim()} onClick={() => void runTest()}>{t(testing ? "serviceConnection.testing" : "serviceConnection.test")}</button>
        {!testing && testResult?.revision === currentDraft.current.revision ? <p className={`settings-connection-test__result ${testResult.code === ServiceConnectionTestCode.Ok ? "is-success" : "is-failure"}`} role={testResult.code === ServiceConnectionTestCode.Ok ? "status" : "alert"}>{t(`serviceConnection.results.${testResult.code}`)}</p> : null}
      </div>
      </div>
      {service === "qbittorrent" ? <SettingsGroup title={t("settingsLayout.groups.downloads")} description={t("settingsLayout.groups.downloadsHelp")}><label className="settings-field"><span>{t("serviceConnection.defaultSavePath")}</span><input value={form.qbittorrent.defaultSavePath} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, defaultSavePath: event.target.value } }))} /></label><label className="settings-field"><span>{t("serviceConnection.defaultCategory")}</span><input value={form.qbittorrent.category} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, category: event.target.value } }))} /></label><label className="settings-field"><span>{t("serviceConnection.defaultTags")}</span><input value={form.qbittorrent.tags} onChange={(event) => setForm((current) => ({ ...current, qbittorrent: { ...current.qbittorrent, tags: event.target.value } }))} /></label></SettingsGroup> : null}
      <SettingsSaveStatus {...autosave} />
    </form>;
  };

  return <SettingsCard title={serviceLabel}>
    {serviceForm(service)}
  </SettingsCard>;
}
