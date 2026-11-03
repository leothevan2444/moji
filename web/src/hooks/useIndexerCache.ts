import { useCallback, useRef, useState } from "react";
import { useMutation, useQuery } from "urql";
import { useIndexerEvents } from "./useIndexerEvents";
import { useFragment } from "../graphql/generated/fragment-masking";
import { ClearIndexerSearchCacheDocument, IndexerCacheStatusDocument, IndexerCacheStatusFieldsFragmentDoc, IndexerEventKind } from "../graphql/generated/graphql";
import i18n from "../i18n/i18n";

export function useIndexerCache() {
  const [{ data, error, fetching }, requery] = useQuery({ query: IndexerCacheStatusDocument, requestPolicy: "network-only" });
  const [{ fetching: clearing }, clearMutation] = useMutation(ClearIndexerSearchCacheDocument);
  const [actionError, setActionError] = useState<unknown>(null);
  const lock = useRef(false);
  const read = useCallback(() => requery({ requestPolicy: "network-only" }), [requery]);
  const events = useIndexerEvents(read, [IndexerEventKind.Cache, IndexerEventKind.Configuration]);
  const refresh = () => { read(); if (events.error) events.retry(); };
  const clear = async (instanceId: string) => {
    if (lock.current) return false;
    lock.current = true;
    setActionError(null);
    try {
      const result = await clearMutation({ instanceId: instanceId || null });
      if (result.error) throw result.error;
      if (!result.data?.clearIndexerSearchCache) throw new Error(i18n.t("settings.autosave.noAcknowledgement"));
      read(); return true;
    } catch (failure) { setActionError(failure); return false; }
    finally { lock.current = false; }
  };
  const snapshot = data ? useFragment(IndexerCacheStatusFieldsFragmentDoc, data.indexerCacheStatus) : null;
  return { snapshot, error: actionError ?? error ?? events.error, fetching, clearing, refresh, clear };
}
