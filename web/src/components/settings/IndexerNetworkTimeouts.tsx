import { useState } from "react";
import { useClient } from "urql";
import { useTranslation } from "react-i18next";
import { useFragment } from "../../graphql/generated/fragment-masking";
import { IndexerNetworkSettingsDocument, IndexerNetworkSettingsFieldsFragmentDoc, UpdateIndexerNetworkTimeoutsDocument, type IndexerNetworkSettingsFieldsFragment, type IndexerNetworkTimeoutPatchInput } from "../../graphql/generated/graphql";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { SettingsSaveStatus } from "./SettingsTabCommon";
import { Drawer } from "../layout/Drawer";
import { describeQueryError } from "../../services/queryError";
import type { SettingsObject } from "./settingsAutosave";
const draft = (value: IndexerNetworkSettingsFieldsFragment) => ({ requestTimeoutSeconds: String(value.requestTimeoutSeconds), siteTimeoutSeconds: String(value.siteTimeoutSeconds), challengeTimeoutSeconds: String(value.challengeTimeoutSeconds) });
const numeric = (value: string) => value.trim() ? Number(value) : NaN;
const encode = (value: ReturnType<typeof draft>) => ({ requestTimeoutSeconds: numeric(value.requestTimeoutSeconds), siteTimeoutSeconds: numeric(value.siteTimeoutSeconds), challengeTimeoutSeconds: numeric(value.challengeTimeoutSeconds) });
export function IndexerNetworkTimeouts({ snapshot, onSaved }: { snapshot: IndexerNetworkSettingsFieldsFragment; onSaved: () => void }) {
  const { t } = useTranslation(); const client = useClient();
  const [reloadOpen, setReloadOpen] = useState(false); const [busy, setBusy] = useState(false); const [error, setError] = useState<unknown>(null);
  const persist = async (patch: SettingsObject, revision?: string) => {
    if (!revision) throw new Error(t("settings.autosave.noAcknowledgement"));
    const input: IndexerNetworkTimeoutPatchInput = {};
    for (const key of ["requestTimeoutSeconds", "siteTimeoutSeconds", "challengeTimeoutSeconds"] as const) if (key in patch) { const value = patch[key]; if (typeof value !== "number" || !Number.isInteger(value)) throw new Error(t("settings.autosave.invalid")); input[key] = value; }
    const result = await client.mutation(UpdateIndexerNetworkTimeoutsDocument, { expectedRevision: revision, input }).toPromise();
    if (result.error) throw result.error;
    if (!result.data?.updateIndexerNetworkTimeouts) throw new Error(t("settings.autosave.noAcknowledgement"));
    return useFragment(IndexerNetworkSettingsFieldsFragmentDoc, result.data.updateIndexerNetworkTimeouts).revision;
  };
  const form = useSettingsAutosave("indexers:network-timeouts", draft(snapshot), encode, { persist, sourceRevision: snapshot.revision, onSaved });
  const reload = async () => {
    setBusy(true); setError(null);
    try {
      const result = await client.query(IndexerNetworkSettingsDocument, {}, { requestPolicy: "network-only" }).toPromise();
      if (result.error) throw result.error;
      if (!result.data?.indexerNetworkSettings) throw new Error(t("settings.autosave.noAcknowledgement"));
      const current = useFragment(IndexerNetworkSettingsFieldsFragmentDoc, result.data.indexerNetworkSettings);
      if (form.discard(draft(current), current.revision)) setReloadOpen(false);
    } catch (failure) { setError(failure); } finally { setBusy(false); }
  };
  return <>
    <div className="settings-form indexer-cache-settings">{([ ["requestTimeoutSeconds", "requestTimeout", 1, 120], ["siteTimeoutSeconds", "siteTimeout", 10, 600], ["challengeTimeoutSeconds", "challengeTimeout", 10, 300] ] as const).map(([key, label, min, max]) => <label className="settings-field" key={key}><span>{t(`indexerNetwork.${label}`)}</span><input type="number" min={min} max={max} step={1} value={form.value[key]} onChange={event => { const text = event.target.value; form.setValue(value => ({ ...value, [key]: text })); }} /></label>)}</div>
    <SettingsSaveStatus state={form.state} error={form.error} retry={form.retry} />
    {form.state === "error" ? <button onClick={() => { setError(null); setReloadOpen(true); }}>{t("indexerCache.reload")}</button> : null}
    {reloadOpen ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexerCache.reload")} closeDisabled={busy} onClose={() => { if (!busy) setReloadOpen(false); }}><p>{t("indexerCache.discard")}</p>{error ? <p role="alert">{describeQueryError(error)}</p> : null}<div className="indexer-actions"><button disabled={busy || form.state === "saving"} onClick={() => void reload()}>{t("indexerCache.reload")}</button><button disabled={busy} onClick={() => setReloadOpen(false)}>{t("common.close")}</button></div></Drawer> : null}
  </>;
}
