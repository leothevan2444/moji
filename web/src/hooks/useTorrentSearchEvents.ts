import { useCallback, useEffect, useRef, useSyncExternalStore } from "react";
import { useSubscription } from "urql";
import { getGraphQLConnectionSnapshot, subscribeGraphQLConnection } from "../graphql/client";
import { TorrentSearchEventsDocument } from "../graphql/generated/graphql";

export function useTorrentSearchEvents(id: string | null, onRefresh: () => void, enabled = true) {
  const [{ data, error }, restart] = useSubscription({ query: TorrentSearchEventsDocument, variables: { id: id ?? "" }, pause: !id || !enabled });
  const connection = useSyncExternalStore(subscribeGraphQLConnection, getGraphQLConnectionSnapshot, getGraphQLConnectionSnapshot);
  const callback = useRef(onRefresh); callback.current = onRefresh;
  const currentID = useRef(id); currentID.current = id;
  const timer = useRef<number | null>(null);
  const cursor = useRef<{ streamId: string; sequence: number } | null>(null);
  const generation = useRef<number | null>(null);
  const schedule = useCallback(() => {
    if (!id || timer.current !== null) return;
    timer.current = window.setTimeout(() => { timer.current = null; if (currentID.current === id) callback.current(); }, 200);
  }, [id]);
  useEffect(() => {
    cursor.current = null;
    return () => { if (timer.current !== null) window.clearTimeout(timer.current); timer.current = null; };
  }, [id]);
  useEffect(() => {
    const event = data?.torrentSearchEvents;
    if (!event || event.searchId !== id) return;
    if (cursor.current?.streamId === event.streamId && cursor.current.sequence >= event.sequence) return;
    cursor.current = { streamId: event.streamId, sequence: event.sequence };
    schedule();
  }, [data, id, schedule]);
  useEffect(() => {
    if (connection.status !== "connected") return;
    if (generation.current !== null && generation.current !== connection.generation) { cursor.current = null; schedule(); }
    generation.current = connection.generation;
  }, [connection.status, connection.generation, schedule]);
  const retry = useCallback(() => { if (id && enabled) restart({ pause: false }); schedule(); }, [id, enabled, restart, schedule]);
  return { error, retry };
}
