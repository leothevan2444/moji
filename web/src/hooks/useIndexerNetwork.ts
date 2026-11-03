import { useCallback, useRef, useState } from "react";
import { useClient, useQuery } from "urql";
import { useFragment } from "../graphql/generated/fragment-masking";
import { AddIndexerNetworkProfileDocument, IndexerEventKind, IndexerNetworkSettingsDocument, IndexerNetworkSettingsFieldsFragmentDoc, RemoveIndexerNetworkProfileDocument, UpdateIndexerNetworkProfileDocument, type IndexerNetworkProfilePatchInput } from "../graphql/generated/graphql";
import { useIndexerEvents } from "./useIndexerEvents";
import i18n from "../i18n/i18n";

export function useIndexerNetwork(onChanged: () => void) {
  const client = useClient();
  const [query, requery] = useQuery({ query: IndexerNetworkSettingsDocument, requestPolicy: "network-only" });
  const [busy, setBusy] = useState(false); const [error, setError] = useState<unknown>(null); const lock = useRef(false);
  const read = useCallback(() => requery({ requestPolicy: "network-only" }), [requery]);
  const events = useIndexerEvents(read, [IndexerEventKind.Configuration]);
  const refresh = () => { read(); if (events.error) events.retry(); };
  const run = async (operation: () => Promise<boolean>) => {
    if (lock.current) return false;
    lock.current = true; setBusy(true); setError(null);
    try { if (!await operation()) throw new Error(i18n.t("settings.autosave.noAcknowledgement")); read(); onChanged(); return true; }
    catch (failure) { setError(failure); return false; }
    finally { lock.current = false; setBusy(false); }
  };
  return {
    snapshot: query.data ? useFragment(IndexerNetworkSettingsFieldsFragmentDoc, query.data.indexerNetworkSettings) : null,
    busy, error: error ?? query.error ?? events.error, refresh, resetError: () => setError(null),
    save: (id: string, expectedRevision: string, create: boolean, input: IndexerNetworkProfilePatchInput) => run(async () => {
      if (create) {
        const result = await client.mutation(AddIndexerNetworkProfileDocument, { id, expectedRevision, input }).toPromise();
        if (result.error) throw result.error; return Boolean(result.data?.addIndexerNetworkProfile);
      }
      const result = await client.mutation(UpdateIndexerNetworkProfileDocument, { id, expectedRevision, input }).toPromise();
      if (result.error) throw result.error; return Boolean(result.data?.updateIndexerNetworkProfile);
    }),
    remove: (id: string, expectedRevision: string) => run(async () => {
      const result = await client.mutation(RemoveIndexerNetworkProfileDocument, { id, expectedRevision }).toPromise();
      if (result.error) throw result.error; return Boolean(result.data?.removeIndexerNetworkProfile);
    })
  };
}
