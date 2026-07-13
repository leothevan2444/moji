import { useEffect, useRef, useSyncExternalStore } from "react";
import { useSubscription } from "urql";
import {
  getGraphQLConnectionSnapshot,
  subscribeGraphQLConnection
} from "../graphql/client";
import {
  LogEventsDocumentDocument,
  type LogEventsDocumentSubscription,
  type LogsDocumentQuery
} from "../graphql/generated/graphql";

export type StreamedLogEntry = LogEventsDocumentSubscription["logEvents"]["entry"];
export type LogEntry = LogsDocumentQuery["logs"][number];

export interface LogSequenceDecision {
  accept: boolean;
  gap: boolean;
  nextSequence: number | null;
}

export function evaluateLogSequence(previous: number | null, next: number): LogSequenceDecision {
  if (previous === null) return { accept: true, gap: false, nextSequence: next };
  if (next <= previous) return { accept: false, gap: false, nextSequence: previous };
  return { accept: true, gap: next !== previous + 1, nextSequence: next };
}

export function mergeLogEntries(
  snapshot: readonly LogEntry[],
  streamed: readonly StreamedLogEntry[],
  limit = 200
): LogEntry[] {
  const entries = new Map<number, LogEntry>();
  for (const entry of [...snapshot, ...streamed]) entries.set(entry.sequence, entry);
  return [...entries.values()]
    .sort((left, right) => right.sequence - left.sequence)
    .slice(0, Math.max(0, limit));
}

interface UseLogEventsOptions {
  pause?: boolean;
  onEvent?: (entry: StreamedLogEntry) => void;
  onResync?: () => void;
}

export function useLogEvents(options: UseLogEventsOptions = {}) {
  const [{ data, error, fetching }] = useSubscription({
    query: LogEventsDocumentDocument,
    pause: options.pause
  });
  const connection = useSyncExternalStore(
    subscribeGraphQLConnection,
    getGraphQLConnectionSnapshot,
    getGraphQLConnectionSnapshot
  );
  const callbacks = useRef(options);
  const lastSequence = useRef<number | null>(null);
  const connectionGeneration = useRef<number | null>(null);
  const recoveringFromError = useRef(false);

  useEffect(() => {
    callbacks.current = options;
  }, [options]);

  useEffect(() => {
    if (options.pause) {
      lastSequence.current = null;
      return;
    }
    if (connection.status !== "connected") return;
    if (connectionGeneration.current === null) {
      connectionGeneration.current = connection.generation;
      return;
    }
    if (connection.generation !== connectionGeneration.current) {
      connectionGeneration.current = connection.generation;
      lastSequence.current = null;
      recoveringFromError.current = false;
      if (import.meta.env.DEV) console.info("GraphQL log subscription recovered; refreshing snapshot");
      callbacks.current.onResync?.();
    }
  }, [connection.generation, connection.status, options.pause]);

  useEffect(() => {
    if (options.pause) return;
    const event = data?.logEvents;
    if (!event) return;
    if (recoveringFromError.current) {
      recoveringFromError.current = false;
      lastSequence.current = null;
      if (import.meta.env.DEV) console.info("GraphQL log subscription resumed after an error; refreshing snapshot");
      callbacks.current.onResync?.();
    }
    const decision = evaluateLogSequence(lastSequence.current, event.sequence);
    if (!decision.accept) return;
    lastSequence.current = decision.nextSequence;
    if (decision.gap) {
      if (import.meta.env.DEV) console.warn("GraphQL log subscription sequence gap; refreshing snapshot", event.sequence);
      callbacks.current.onResync?.();
    }
    callbacks.current.onEvent?.(event.entry);
  }, [data, options.pause]);

  useEffect(() => {
    if (!error) return;
    recoveringFromError.current = true;
    if (import.meta.env.DEV) console.error("GraphQL log subscription error", error);
  }, [error]);

  return {
    event: data?.logEvents ?? null,
    error,
    fetching,
    connectionStatus: connection.status,
    lastSequence: lastSequence.current
  };
}
