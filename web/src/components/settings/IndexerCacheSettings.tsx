import { useState, type ReactNode } from "react";
import { useClient, useMutation } from "urql";
import { useTranslation } from "react-i18next";
import { IndexerCacheStatusDocument, IndexerCacheStatusFieldsFragmentDoc, UpdateIndexerSearchCacheDocument, type IndexerCacheStatusFieldsFragment, type IndexerSearchCachePatchInput } from "../../graphql/generated/graphql";
import { useFragment } from "../../graphql/generated/fragment-masking";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { SettingsCard, SettingsSaveStatus } from "./SettingsTabCommon";
import { SettingsSwitch } from "./SettingsSwitch";
import { Drawer } from "../layout/Drawer";
import { describeQueryError } from "../../services/queryError";
import type { SettingsObject } from "./settingsAutosave";

const draft = (value: IndexerCacheStatusFieldsFragment) => ({ enabled: value.enabled, ttlSeconds: String(value.ttlSeconds), emptyTtlSeconds: String(value.emptyTtlSeconds), maxSizeMb: String(value.maxSizeMb) });
type Draft = ReturnType<typeof draft>;
const number = (value: string) => value.trim() ? Number(value) : NaN;
const encode = (value: Draft) => ({ enabled: value.enabled, ttlSeconds: number(value.ttlSeconds), emptyTtlSeconds: number(value.emptyTtlSeconds), maxSizeMb: number(value.maxSizeMb) });

export function IndexerCacheSettings({ snapshot, onSaved, children }: { snapshot: IndexerCacheStatusFieldsFragment; onSaved: () => void; children:ReactNode }) {
  const { t } = useTranslation(); const client = useClient();
  const [, update] = useMutation(UpdateIndexerSearchCacheDocument);
  const [reloadOpen, setReloadOpen] = useState(false); const [reloadBusy, setReloadBusy] = useState(false); const [reloadError, setReloadError] = useState<unknown>(null);
  const persist = async (patch: SettingsObject, expectedRevision?: string) => {
    if (!expectedRevision) throw new Error(t("settings.autosave.noAcknowledgement"));
    const input: IndexerSearchCachePatchInput = {};
    if ("enabled" in patch) { if (typeof patch.enabled !== "boolean") throw new Error(t("settings.autosave.invalid")); input.enabled = patch.enabled; }
    for (const key of ["ttlSeconds", "emptyTtlSeconds", "maxSizeMb"] as const) if (key in patch) { const value = patch[key]; if (typeof value !== "number" || !Number.isInteger(value)) throw new Error(t("settings.autosave.invalid")); input[key] = value; }
    const result = await update({ expectedRevision, input });
    if (result.error) throw result.error;
    if (!result.data?.updateIndexerSearchCache) throw new Error(t("settings.autosave.noAcknowledgement"));
    return useFragment(IndexerCacheStatusFieldsFragmentDoc, result.data.updateIndexerSearchCache).revision;
  };
  const form = useSettingsAutosave("indexers:cache", draft(snapshot), encode, { persist, sourceRevision: snapshot.revision, onSaved });
  const reload = async () => {
    setReloadBusy(true); setReloadError(null);
    try {
      const result = await client.query(IndexerCacheStatusDocument, {}, { requestPolicy: "network-only" }).toPromise();
      if (result.error) throw result.error;
      if (!result.data?.indexerCacheStatus) throw new Error(t("settings.autosave.noAcknowledgement"));
      const current = useFragment(IndexerCacheStatusFieldsFragmentDoc, result.data.indexerCacheStatus);
      if (form.discard(draft(current), current.revision)) setReloadOpen(false);
    } catch (error) { setReloadError(error); }
    finally { setReloadBusy(false); }
  };
  return <>
   <SettingsCard title={t("indexerCache.title")} actions={<SettingsSwitch checked={form.value.enabled} label={t("indexerCache.useCache")} onChange={enabled=>form.setValue(value=>({...value,enabled}))}/>}>
   <div className="settings-form">
    <div className="settings-form indexer-cache-fields">
      {([ ["ttlSeconds", "ttl", 1, 3600], ["emptyTtlSeconds", "emptyTtl", 1, 300], ["maxSizeMb", "capacity", 1, 256] ] as const).map(([key, label, min, max]) => <label className="settings-field" key={key} htmlFor={`cache-${key}`}><span>{t(`indexerCache.${label}`)}</span><input id={`cache-${key}`} type="number" min={min} max={max} step={1} value={form.value[key]} onChange={event => { const text = event.target.value; form.setValue(value => ({ ...value, [key]: text })); }} /></label>)}
    </div>
    <SettingsSaveStatus state={form.state} error={form.error} retry={form.retry} />
    {form.state === "error" ? <button type="button" onClick={() => { setReloadError(null); setReloadOpen(true); }}>{t("indexerCache.reload")}</button> : null}
    {children}
   </div>
   </SettingsCard>
    {reloadOpen ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexerCache.reload")} onClose={() => { if (!reloadBusy) setReloadOpen(false); }} closeDisabled={reloadBusy}><p>{t("indexerCache.discard")}</p>{reloadError ? <p role="alert">{describeQueryError(reloadError)}</p> : null}<div className="indexer-actions"><button disabled={reloadBusy || form.state === "saving"} onClick={() => void reload()}>{t("indexerCache.reload")}</button><button disabled={reloadBusy} onClick={() => setReloadOpen(false)}>{t("common.close")}</button></div></Drawer> : null}
  </>;
}
