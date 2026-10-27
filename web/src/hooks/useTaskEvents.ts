import { useEffect, useRef, useSyncExternalStore } from "react";
import { useSubscription } from "urql";
import {
  getGraphQLConnectionSnapshot,
  subscribeGraphQLConnection
} from "../graphql/client";
import {
  TaskEventsDocument,
  type TaskEventsSubscription
} from "../graphql/generated/graphql";

import { getTaskEventGapGeneration, subscribeTaskEventGaps } from "../graphql/taskEventSequence";

export type TaskEvent = TaskEventsSubscription["taskEvents"];

interface UseTaskEventsOptions {
  onEvent?: (event: TaskEvent) => void;
  onSequenceGap?: () => void;
  onReconnect?: () => void;
}

export { evaluateTaskEventSequence } from "../graphql/taskEventSequence";

export function useTaskEvents(options: UseTaskEventsOptions = {}) {
  const [{ data, error, fetching }] = useSubscription({ query: TaskEventsDocument });
  const connection = useSyncExternalStore(
    subscribeGraphQLConnection,
    getGraphQLConnectionSnapshot,
    getGraphQLConnectionSnapshot
  );
  const callbacks = useRef(options);
  const gapGeneration = useSyncExternalStore(subscribeTaskEventGaps, getTaskEventGapGeneration, getTaskEventGapGeneration);
  const observedGapGeneration = useRef(gapGeneration);
  const connectionGeneration = useRef<number | null>(null);

  useEffect(() => {
    callbacks.current = options;
  }, [options]);

  useEffect(() => {
    if (connection.status !== "connected") return;
    if (connectionGeneration.current === null) {
      connectionGeneration.current = connection.generation;
      return;
    }
    if (connection.generation !== connectionGeneration.current) {
      connectionGeneration.current = connection.generation;
      if (import.meta.env.DEV) console.info("GraphQL task subscription recovered; refreshing snapshots");
      callbacks.current.onReconnect?.();
    }
  }, [connection.generation, connection.status]);

  useEffect(() => {
    const event = data?.taskEvents;
    if (!event) return;
    callbacks.current.onEvent?.(event);
  }, [data]);

  useEffect(() => {
    if (observedGapGeneration.current === gapGeneration) return;
    observedGapGeneration.current = gapGeneration;
    if (import.meta.env.DEV) console.warn("GraphQL task transport sequence gap; refreshing snapshots");
    callbacks.current.onSequenceGap?.();
  }, [gapGeneration]);

  useEffect(() => {
    if (error && import.meta.env.DEV) console.error("GraphQL task subscription error", error);
  }, [error]);

  return {
    event: data?.taskEvents ?? null,
    error,
    fetching,
    connectionStatus: connection.status,
    lastSequence: data?.taskEvents.sequence ?? null
  };
}
