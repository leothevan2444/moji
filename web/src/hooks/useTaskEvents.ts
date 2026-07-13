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

export type TaskEvent = TaskEventsSubscription["taskEvents"];

interface UseTaskEventsOptions {
  onEvent?: (event: TaskEvent) => void;
  onSequenceGap?: () => void;
  onReconnect?: () => void;
}

export interface SequenceDecision {
  accept: boolean;
  gap: boolean;
  nextSequence: number | null;
}

export function evaluateTaskEventSequence(previous: number | null, next: number): SequenceDecision {
  if (previous === null) return { accept: true, gap: false, nextSequence: next };
  if (next <= previous) return { accept: false, gap: false, nextSequence: previous };
  return { accept: true, gap: next !== previous + 1, nextSequence: next };
}

export function useTaskEvents(options: UseTaskEventsOptions = {}) {
  const [{ data, error, fetching }] = useSubscription({ query: TaskEventsDocument });
  const connection = useSyncExternalStore(
    subscribeGraphQLConnection,
    getGraphQLConnectionSnapshot,
    getGraphQLConnectionSnapshot
  );
  const callbacks = useRef(options);
  const lastSequence = useRef<number | null>(null);
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
      lastSequence.current = null;
      if (import.meta.env.DEV) console.info("GraphQL task subscription recovered; refreshing snapshots");
      callbacks.current.onReconnect?.();
    }
  }, [connection.generation, connection.status]);

  useEffect(() => {
    const event = data?.taskEvents;
    if (!event) return;
    const decision = evaluateTaskEventSequence(lastSequence.current, event.sequence);
    if (!decision.accept) return;
    lastSequence.current = decision.nextSequence;
    if (decision.gap) {
      if (import.meta.env.DEV) console.warn("GraphQL task subscription sequence gap; refreshing snapshots", event.sequence);
      callbacks.current.onSequenceGap?.();
    }
    callbacks.current.onEvent?.(event);
  }, [data]);

  useEffect(() => {
    if (error && import.meta.env.DEV) console.error("GraphQL task subscription error", error);
  }, [error]);

  return {
    event: data?.taskEvents ?? null,
    error,
    fetching,
    connectionStatus: connection.status,
    lastSequence: lastSequence.current
  };
}
