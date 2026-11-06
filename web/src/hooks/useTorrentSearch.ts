import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "urql";
import { useFragment } from "../graphql/generated/fragment-masking";
import {
  TorrentSearchDocument, TorrentSearchFieldsFragmentDoc, TorrentSearchSort, TorrentSearchState,
  StartTorrentSearchDocument, CancelTorrentSearchDocument, QueueTorrentCandidateDocument,
  type StartTorrentSearchInput, type QueueTorrentCandidateInput
} from "../graphql/generated/graphql";
import { useTorrentSearchEvents } from "./useTorrentSearchEvents";
import i18n from "../i18n/i18n";

// Mounting/reconnecting reads an existing run. Only start() admits new work.
export function useTorrentSearch(id: string | null, offset = 0, limit = 100, sortBy = TorrentSearchSort.Relevance, instanceId = "", candidateQuery = "") {
  const [read, requery] = useQuery({ query: TorrentSearchDocument, variables: { id: id ?? "", offset, limit, sortBy, instanceId: instanceId || undefined, candidateQuery: candidateQuery || undefined }, pause: !id, requestPolicy: "network-only" });
  const raw = read.data ? useFragment(TorrentSearchFieldsFragmentDoc, read.data.torrentSearch) : null;
  const snapshot = raw?.id === id && raw.offset === offset && raw.limit === limit && (raw.filterInstanceId ?? "") === instanceId && (raw.candidateQuery ?? "") === candidateQuery.trim() ? raw : null;
  const running = !snapshot || snapshot.state === TorrentSearchState.Queued || snapshot.state === TorrentSearchState.Running;
  const refresh = useCallback(() => { if (id) requery({ requestPolicy: "network-only" }); }, [id, requery]);
  const events = useTorrentSearchEvents(id, refresh, running);
  const [startState, startMutation] = useMutation(StartTorrentSearchDocument);
  const [cancelState, cancelMutation] = useMutation(CancelTorrentSearchDocument);
  const [queueState, queueMutation] = useMutation(QueueTorrentCandidateDocument);
  const [actionError, setActionError] = useState<unknown>(null);
  const currentID = useRef(id); currentID.current = id;
  useEffect(() => setActionError(null), [id]);
  const startLock = useRef(false); const cancelLock = useRef(false); const queueLock = useRef(false);
  const start = async (input: StartTorrentSearchInput) => {
    if (startLock.current) return null;
    startLock.current = true; setActionError(null); const scope = id;
    try { const result = await startMutation({ input }); if (result.error) throw result.error; if (!result.data?.startTorrentSearch.id) throw new Error(i18n.t("errors.unknown")); return currentID.current === scope ? result.data.startTorrentSearch.id : null; }
    catch (failure) { if (currentID.current === scope) setActionError(failure); return null; }
    finally { startLock.current = false; }
  };
  const cancel = async () => {
    if (!id || cancelLock.current) return false;
    cancelLock.current = true; setActionError(null);
    try { const result = await cancelMutation({ id }); if (result.error) throw result.error; if (result.data?.cancelTorrentSearch.id !== id) throw new Error(i18n.t("errors.unknown")); if (currentID.current !== id) return false; refresh(); return true; }
    catch (failure) { if (currentID.current === id) setActionError(failure); return false; }
    finally { cancelLock.current = false; }
  };
  const queue = async (input: Omit<QueueTorrentCandidateInput, "searchId">) => {
    if (!id || queueLock.current) return null;
    queueLock.current = true; setActionError(null);
    try { const result = await queueMutation({ input: { ...input, searchId: id } }); if (result.error) throw result.error; if (!result.data?.queueTorrentCandidate.id) throw new Error(i18n.t("errors.unknown")); return currentID.current === id ? result.data.queueTorrentCandidate : null; }
    catch (failure) { if (currentID.current === id) setActionError(failure); return null; }
    finally { queueLock.current = false; }
  };
  return { snapshot, progressSnapshot: raw?.id === id ? raw : null, fetching: read.fetching, starting: startState.fetching, cancelling: cancelState.fetching, queueing: queueState.fetching,
    error: actionError ?? read.error ?? events.error, start, cancel, queue, refresh: () => { refresh(); events.retry(); }, clearActionError: () => setActionError(null) };
}
