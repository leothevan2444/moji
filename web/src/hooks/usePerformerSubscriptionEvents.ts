import { useCallback, useEffect, useRef, useSyncExternalStore } from "react";
import { useSubscription } from "urql";
import {
  getGraphQLConnectionSnapshot,
  subscribeGraphQLConnection
} from "../graphql/client";
import { PerformerSubscriptionEventsDocument } from "../graphql/generated/graphql";

export interface PerformerSubscriptionSequenceDecision {
  accept: boolean;
  gap: boolean;
  nextSequence: number | null;
}

export function evaluatePerformerSubscriptionSequence(previous: number | null, next: number): PerformerSubscriptionSequenceDecision {
  if (previous === null) return { accept: true, gap: false, nextSequence: next };
  if (next <= previous) return { accept: false, gap: false, nextSequence: previous };
  return { accept: true, gap: next !== previous + 1, nextSequence: next };
}

interface Options {
  enabled: boolean;
  onRefresh: () => void | Promise<unknown>;
  delay?: number;
}

export function usePerformerSubscriptionEvents({ enabled, onRefresh, delay = 200 }: Options) {
  const [{ data, error, fetching }] = useSubscription({ query: PerformerSubscriptionEventsDocument, pause: !enabled });
  const connection = useSyncExternalStore(
    subscribeGraphQLConnection,
    getGraphQLConnectionSnapshot,
    getGraphQLConnectionSnapshot
  );
  const refreshRef = useRef(onRefresh);
  const timer = useRef<number | null>(null);
  const lastSequence = useRef<number | null>(null);
  const connectionGeneration = useRef<number | null>(null);
  const recoveringFromError = useRef(false);

  useEffect(() => {
    refreshRef.current = onRefresh;
  }, [onRefresh]);

  const scheduleRefresh = useCallback((reason: "reconnect" | "sequence-gap" | "error-recovery") => {
    if (timer.current !== null) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      timer.current = null;
      if (import.meta.env.DEV) console.info("GraphQL performer subscription snapshot recalibrating", reason);
      void refreshRef.current();
    }, delay);
  }, [delay]);

  useEffect(() => () => {
    if (timer.current !== null) window.clearTimeout(timer.current);
  }, []);

  useEffect(() => {
    if (!enabled) {
      lastSequence.current = null;
      connectionGeneration.current = null;
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
      scheduleRefresh("reconnect");
    }
  }, [connection.generation, connection.status, enabled, scheduleRefresh]);

  useEffect(() => {
    if (!enabled) return;
    const event = data?.performerSubscriptionEvents;
    if (!event) return;
    if (recoveringFromError.current) {
      recoveringFromError.current = false;
      lastSequence.current = null;
      scheduleRefresh("error-recovery");
    }
    const decision = evaluatePerformerSubscriptionSequence(lastSequence.current, event.sequence);
    if (!decision.accept) return;
    lastSequence.current = decision.nextSequence;
    if (decision.gap) {
      if (import.meta.env.DEV) console.warn("GraphQL performer subscription sequence gap", event.sequence);
      scheduleRefresh("sequence-gap");
    }
  }, [data, enabled, scheduleRefresh]);

  useEffect(() => {
    if (!error) return;
    recoveringFromError.current = true;
    if (import.meta.env.DEV) console.error("GraphQL performer subscription error", error);
  }, [error]);

  return {
    event: data?.performerSubscriptionEvents ?? null,
    error,
    fetching,
    connectionStatus: connection.status,
    lastSequence: lastSequence.current
  };
}
