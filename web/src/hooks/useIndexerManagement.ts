import i18n from "../i18n/i18n";
import { createContext, useContext, useCallback, useEffect, useRef, useState } from "react";
import { useClient, useQuery, useSubscription } from "urql";
import { useFragment } from "../graphql/generated/fragment-masking";
import {
  AddBuiltinIndexerDocument, UpdateBuiltinIndexerDocument, RemoveBuiltinIndexerDocument,
  SetIndexerEnabledDocument, ReorderIndexersDocument, RefreshJackettIndexersDocument,
  IndexerInstancesDocument, IndexerInstancesFieldsFragmentDoc, IndexerInstanceFieldsFragmentDoc,
  IndexerSettingFieldsFragmentDoc, IndexerDefinitionStatusDocument, IndexerDefinitionEventsDocument,
  IndexerObservationFieldsFragmentDoc,
  IndexerDefinitionStatusFieldsFragmentDoc, IndexerDefinitionJobFieldsFragmentDoc,
  CheckIndexerDefinitionUpdateDocument, InstallIndexerDefinitionsDocument, UpdateIndexerDefinitionsDocument,
  RollbackIndexerDefinitionsDocument, RepairIndexerDefinitionsDocument, CancelIndexerDefinitionUpdateDocument,
  type BuiltinIndexerInput, type BuiltinIndexerPatchInput, type IndexerInstanceFieldsFragment,
  type IndexerSettingFieldsFragment, type IndexerDefinitionStatusFieldsFragment, type IndexerObservationFieldsFragment
} from "../graphql/generated/graphql";
import type { FragmentType } from "../graphql/generated/fragment-masking";

export type SiteSetting = IndexerSettingFieldsFragment;
export type IndexerSite = Omit<IndexerInstanceFieldsFragment, "settings" | "health"> & { settings: SiteSetting[]; health: { search: IndexerObservationFieldsFragment | null; fetch: IndexerObservationFieldsFragment | null } };
export function indexerSite(value: FragmentType<typeof IndexerInstanceFieldsFragmentDoc>): IndexerSite {
  const site = useFragment(IndexerInstanceFieldsFragmentDoc, value);
  return { ...site, settings: site.settings.map(setting => useFragment(IndexerSettingFieldsFragmentDoc, setting)),health:{search:site.health.search?useFragment(IndexerObservationFieldsFragmentDoc,site.health.search):null,fetch:site.health.fetch?useFragment(IndexerObservationFieldsFragmentDoc,site.health.fetch):null} };
}

export function useIndexerManagementState() {
  const client = useClient();
  const [query, refresh] = useQuery({ query: IndexerInstancesDocument, requestPolicy: "network-only" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const lock = useRef(false);
  const snapshot = query.data ? useFragment(IndexerInstancesFieldsFragmentDoc, query.data.indexerInstances) : null;
  const sites = snapshot?.items.map(indexerSite) ?? [];
  const run = useCallback(async (action: () => Promise<void>) => {
    if (lock.current) return false;
    lock.current = true; setBusy(true); setError(null);
    try { await action(); refresh({ requestPolicy: "network-only" }); return true; }
    catch (failure) { setError(failure); return false; }
    finally { lock.current = false; setBusy(false); }
  }, [refresh]);
  const save = (id: string | null, expectedRevision: string, input: BuiltinIndexerInput | BuiltinIndexerPatchInput) => run(async () => {
    if (id) {
      const result = await client.mutation(UpdateBuiltinIndexerDocument, { id, expectedRevision, input }).toPromise();
      if (result.error) throw result.error;
      if (!result.data?.updateBuiltinIndexer.instance) throw new Error(i18n.t("settings.autosave.noAcknowledgement"));
    } else {
      if (!("definitionId" in input)) throw new Error(i18n.t("settings.autosave.noAcknowledgement"));
      const result = await client.mutation(AddBuiltinIndexerDocument, { expectedRevision, input }).toPromise();
      if (result.error) throw result.error;
      if (!result.data?.addBuiltinIndexer.instance) throw new Error(i18n.t("settings.autosave.noAcknowledgement"));
    }
  });
  return {
    snapshot, sites, busy, error: error ?? query.error, loading: query.fetching && !snapshot,
    refresh: () => refresh({ requestPolicy: "network-only" }), save,
    setEnabled: (id: string, enabled: boolean) => run(async () => { const result = await client.mutation(SetIndexerEnabledDocument, { id, enabled, expectedRevision: snapshot?.revision ?? "" }).toPromise(); if (result.error) throw result.error; }),
    remove: (id: string) => run(async () => { const result = await client.mutation(RemoveBuiltinIndexerDocument, { id, expectedRevision: snapshot?.revision ?? "" }).toPromise(); if (result.error) throw result.error; }),
    reorder: (ids: string[]) => run(async () => { const result = await client.mutation(ReorderIndexersDocument, { ids, expectedRevision: snapshot?.revision ?? "" }).toPromise(); if (result.error) throw result.error; }),
    refreshJackett: () => run(async () => { const result = await client.mutation(RefreshJackettIndexersDocument, {}).toPromise(); if (result.error) throw result.error; })
  };
}

type LibraryStatus = IndexerDefinitionStatusFieldsFragment;
export interface LibraryHistory { current: LibraryStatus | null; retired: ReadonlySet<string> }
export function acceptLibraryStatus(history: LibraryHistory, incoming: LibraryStatus | null): LibraryHistory {
  if (!incoming || history.retired.has(incoming.streamId)) return history;
  if (history.current && history.current.streamId !== incoming.streamId) {
    const retired = new Set(history.retired); retired.add(history.current.streamId);
    if (retired.size > 32) retired.delete(retired.values().next().value!);
    return { current: incoming, retired };
  }
  const current = newerLibraryStatus(history.current, incoming);
  return current === history.current ? history : { ...history, current };
}
// Core revision orders checks/activations, then job start/revision orders worker
// progress. A late initial response cannot replace a newer subscription value.
export function newerLibraryStatus(a: LibraryStatus | null, b: LibraryStatus | null): LibraryStatus | null {
  if (!a) return b; if (!b) return a;
  if (a.revision !== b.revision) return a.revision > b.revision ? a : b;
  const aj = a.job ? useFragment(IndexerDefinitionJobFieldsFragmentDoc, a.job) : null;
  const bj = b.job ? useFragment(IndexerDefinitionJobFieldsFragmentDoc, b.job) : null;
  if (!aj) return b; if (!bj) return a;
  if (aj.id === bj.id) return aj.revision > bj.revision ? a : b;
  const stamp = (value: string) => value.replace(/(?:\.(\d+))?Z$/, (_, fraction: string | undefined) => `.${(fraction ?? "").padEnd(9, "0")}Z`);
  return stamp(aj.startedAt) > stamp(bj.startedAt) ? a : b;
}

export function useIndexerLibraryState() {
  const client = useClient();
  const [query, refresh] = useQuery({ query: IndexerDefinitionStatusDocument, requestPolicy: "network-only" });
  const [events] = useSubscription({ query: IndexerDefinitionEventsDocument });
  const [history, setHistory] = useState<LibraryHistory>({ current: null, retired: new Set() });
  useEffect(() => {
    const snapshot = query.data ? useFragment(IndexerDefinitionStatusFieldsFragmentDoc, query.data.indexerDefinitionStatus) : null;
    const event = events.data ? useFragment(IndexerDefinitionStatusFieldsFragmentDoc, events.data.indexerDefinitionEvents) : null;
    setHistory(current => acceptLibraryStatus(acceptLibraryStatus(current, snapshot), event));
  }, [query.data, events.data]);
  const status = history.current;
  const job = status?.job ? useFragment(IndexerDefinitionJobFieldsFragmentDoc, status.job) : null;
  const active = Boolean(job && ["QUEUED", "DOWNLOADING", "VALIDATING", "CHECKING_INSTANCES", "ACTIVATING"].includes(job.state));
  const [busy, setBusy] = useState(false); const [error, setError] = useState<unknown>(null); const lock = useRef(false);
  useEffect(() => { if (events.error) refresh({ requestPolicy: "network-only" }); }, [events.error, refresh]);
  const run = async (kind: "CHECK" | "INSTALL" | "UPDATE" | "ROLLBACK" | "REPAIR" | "CANCEL", reviewed?: { activeRevision: string; previousRevision: string }) => {
    if (lock.current || !status) return;
    lock.current = true; setBusy(true); setError(null);
    try {
      if (kind === "CHECK") {
        const result = await client.mutation(CheckIndexerDefinitionUpdateDocument, {}).toPromise(); if (result.error) throw result.error;
      } else if (kind === "CANCEL") {
        if (!job) return;
        const result = await client.mutation(CancelIndexerDefinitionUpdateDocument, { id: job.id }).toPromise(); if (result.error) throw result.error;
      } else {
        let availableRevision = status.availableRevision;
        if (kind === "INSTALL" || kind === "UPDATE") {
          const checked = await client.mutation(CheckIndexerDefinitionUpdateDocument, {}).toPromise(); if (checked.error) throw checked.error;
          if (!checked.data) throw new Error(i18n.t("settings.autosave.noAcknowledgement"));
          availableRevision = useFragment(IndexerDefinitionStatusFieldsFragmentDoc, checked.data.checkIndexerDefinitionUpdate).availableRevision;
          if (!availableRevision) throw new Error(i18n.t("indexerErrors.DEFINITION_SOURCE_UNAVAILABLE"));
          if (kind === "UPDATE" && availableRevision === status.activeRevision) return;
        }
        const variables = { input: { expectedActiveRevision: reviewed?.activeRevision ?? status.activeRevision, targetRevision: kind === "ROLLBACK" ? reviewed?.previousRevision ?? status.previousRevision : kind === "REPAIR" ? status.activeRevision : availableRevision } };
        const result = kind === "INSTALL" ? await client.mutation(InstallIndexerDefinitionsDocument, variables).toPromise()
          : kind === "UPDATE" ? await client.mutation(UpdateIndexerDefinitionsDocument, variables).toPromise()
          : kind === "ROLLBACK" ? await client.mutation(RollbackIndexerDefinitionsDocument, variables).toPromise()
          : await client.mutation(RepairIndexerDefinitionsDocument, variables).toPromise();
        if (result.error) throw result.error;
      }
    } catch (failure) { setError(failure); }
    finally { lock.current = false; setBusy(false); refresh({ requestPolicy: "network-only" }); }
  };
  return { status, job, active, busy, error: error ?? query.error ?? events.error, loading: !status && (query.fetching || Boolean(query.data || events.data)), refresh: () => refresh({ requestPolicy: "network-only" }), run };
}

export const IndexerManagementContext = createContext<ReturnType<typeof useIndexerManagementState> | null>(null);
export const IndexerLibraryContext = createContext<ReturnType<typeof useIndexerLibraryState> | null>(null);
export function useIndexerManagement() { const value = useContext(IndexerManagementContext); if (!value) throw new Error("Missing indexer management owner"); return value; }
export function useIndexerLibrary() { const value = useContext(IndexerLibraryContext); if (!value) throw new Error("Missing definition library owner"); return value; }
