import { useCallback, useEffect, useRef, useSyncExternalStore } from "react";
import { useSubscription } from "urql";
import { getGraphQLConnectionSnapshot, subscribeGraphQLConnection } from "../graphql/client";
import { IndexerEventsDocument, IndexerEventKind } from "../graphql/generated/graphql";

// Notifications invalidate a bounded current-state query. Sequence gaps and a
// new stream therefore recover with the same read as an ordinary change.
export function useIndexerEvents(onRefresh: () => void, kinds?: readonly IndexerEventKind[]) {
  const [{ data, error }, restart] = useSubscription({ query: IndexerEventsDocument });
  const connection = useSyncExternalStore(subscribeGraphQLConnection, getGraphQLConnectionSnapshot, getGraphQLConnectionSnapshot);
  const callback = useRef(onRefresh);
  const accepted = useRef(kinds); accepted.current = kinds;
  callback.current = onRefresh;
  const timer = useRef<number | null>(null);
  const cursor = useRef<{ streamId: string; sequence: number } | null>(null);
  const generation = useRef<number | null>(null);
  const schedule = useCallback(() => {
    if (timer.current !== null) return;
    timer.current = window.setTimeout(() => { timer.current = null; callback.current(); }, 200);
  }, []);
  useEffect(() => () => { if (timer.current !== null) window.clearTimeout(timer.current); }, []);
  useEffect(() => {
    const event = data?.indexerEvents;
    if (!event) return;
    if (cursor.current?.streamId === event.streamId && cursor.current.sequence >= event.sequence) return;
    cursor.current = { streamId: event.streamId, sequence: event.sequence };
    if (event.kind !== IndexerEventKind.Ready && accepted.current && !accepted.current.includes(event.kind)) return;
    schedule();
  }, [data, schedule]);
  useEffect(() => {
    if (connection.status !== "connected") return;
    if (generation.current !== null && generation.current !== connection.generation) {
      cursor.current = null;
      schedule();
    }
    generation.current = connection.generation;
  }, [connection.status, connection.generation, schedule]);
  const retry = useCallback(() => restart({ pause: false }), [restart]);
  return { schedule, error, retry };
}
