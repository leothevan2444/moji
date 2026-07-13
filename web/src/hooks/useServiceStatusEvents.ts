import { useCallback, useEffect, useRef, useSyncExternalStore } from "react";
import { useSubscription } from "urql";
import {
  getGraphQLConnectionSnapshot,
  subscribeGraphQLConnection
} from "../graphql/client";
import { ServiceStatusEventsDocument } from "../graphql/generated/graphql";

export interface ServiceStatusSequenceDecision {
  accept: boolean;
  gap: boolean;
  nextSequence: number | null;
}

export function evaluateServiceStatusSequence(previous: number | null, next: number): ServiceStatusSequenceDecision {
  if (previous === null) return { accept: true, gap: false, nextSequence: next };
  if (next <= previous) return { accept: false, gap: false, nextSequence: previous };
  return { accept: true, gap: next !== previous + 1, nextSequence: next };
}

interface UseServiceStatusEventsOptions {
  onRefresh: () => void | Promise<unknown>;
  delay?: number;
}

export function useServiceStatusEvents({ onRefresh, delay = 200 }: UseServiceStatusEventsOptions) {
  const [{ data, error, fetching }] = useSubscription({ query: ServiceStatusEventsDocument });
  const connection = useSyncExternalStore(
    subscribeGraphQLConnection,
    getGraphQLConnectionSnapshot,
    getGraphQLConnectionSnapshot
  );
  const refreshRef = useRef(onRefresh);
  const timer = useRef<number | null>(null);
  const lastSequence = useRef<number | null>(null);
  const connectionGeneration = useRef<number | null>(null);

  useEffect(() => {
    refreshRef.current = onRefresh;
  }, [onRefresh]);

  const scheduleRefresh = useCallback(() => {
    if (timer.current !== null) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      timer.current = null;
      void refreshRef.current();
    }, delay);
  }, [delay]);

  useEffect(() => () => {
    if (timer.current !== null) window.clearTimeout(timer.current);
  }, []);

  useEffect(() => {
    if (connection.status !== "connected") return;
    if (connectionGeneration.current === null) {
      connectionGeneration.current = connection.generation;
      return;
    }
    if (connection.generation !== connectionGeneration.current) {
      connectionGeneration.current = connection.generation;
      lastSequence.current = null;
      if (import.meta.env.DEV) console.info("GraphQL service status subscription recovered; refreshing snapshot");
      scheduleRefresh();
    }
  }, [connection.generation, connection.status, scheduleRefresh]);

  useEffect(() => {
    const event = data?.serviceStatusEvents;
    if (!event) return;
    const decision = evaluateServiceStatusSequence(lastSequence.current, event.sequence);
    if (!decision.accept) return;
    lastSequence.current = decision.nextSequence;
    if (decision.gap && import.meta.env.DEV) {
      console.warn("GraphQL service status subscription sequence gap; refreshing snapshot", event.sequence);
    }
    scheduleRefresh();
  }, [data, scheduleRefresh]);

  useEffect(() => {
    if (error && import.meta.env.DEV) console.error("GraphQL service status subscription error", error);
  }, [error]);

  return {
    event: data?.serviceStatusEvents ?? null,
    error,
    fetching,
    connectionStatus: connection.status,
    lastSequence: lastSequence.current
  };
}
