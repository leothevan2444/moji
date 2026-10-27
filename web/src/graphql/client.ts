import { createClient as createWSClient } from "graphql-ws";
import { createClient, fetchExchange, subscriptionExchange } from "urql";
import { pipe, map } from "wonka";
import type { Exchange } from "urql";
import { recordTaskEvent, publishTaskBatch, protectTaskSnapshot, type TaskRealtimeEvent } from "./taskRealtime";
import type { ExecutionResult } from "@urql/core";
import { graphcacheExchange } from "./cache";
import { createTaskEventSequenceObserver, notifyTaskEventGap } from "./taskEventSequence";

export type GraphQLConnectionStatus = "idle" | "connecting" | "connected" | "disconnected" | "error";

export interface GraphQLConnectionSnapshot {
  status: GraphQLConnectionStatus;
  generation: number;
  error: unknown;
}

let connectionSnapshot: GraphQLConnectionSnapshot = { status: "idle", generation: 0, error: null };
const connectionListeners = new Set<() => void>();

function updateConnection(patch: Partial<GraphQLConnectionSnapshot>) {
  connectionSnapshot = { ...connectionSnapshot, ...patch };
  connectionListeners.forEach((listener) => listener());
}

export function subscribeGraphQLConnection(listener: () => void) {
  connectionListeners.add(listener);
  return () => connectionListeners.delete(listener);
}

export function getGraphQLConnectionSnapshot() {
  return connectionSnapshot;
}

export function getGraphQLWebSocketUrl(location?: Pick<Location, "protocol" | "host">): string {
  const currentLocation = location ?? (typeof window !== "undefined" ? window.location : null);
  if (!currentLocation) throw new Error("GraphQL WebSocket URL requires a browser location");
  const protocol = currentLocation.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${currentLocation.host}/graphql`;
}

const wsClient = createWSClient({
  url: async () => getGraphQLWebSocketUrl(),
  lazy: true,
  retryAttempts: Infinity,
  shouldRetry: () => true,
  on: {
    connecting: () => updateConnection({ status: "connecting", error: null }),
    connected: () => updateConnection({ status: "connected", generation: connectionSnapshot.generation + 1, error: null }),
    closed: () => updateConnection({ status: "disconnected" }),
    error: (error) => updateConnection({ status: "error", error })
  }
});

const realtimeExchange = subscriptionExchange({
  forwardSubscription(request) {
    if (!request.query) throw new Error("GraphQL subscription query is missing");
    const input = { ...request, query: request.query };
    return {
      subscribe(sink) {
        const observeSequence = createTaskEventSequenceObserver(notifyTaskEventGap);
        let timer: ReturnType<typeof setTimeout> | undefined;
        const pending = new Map<string, any>();
        const flush = () => {
          timer = undefined;
          const events: TaskRealtimeEvent[] = [];
          for (const payload of [...pending.values()].sort((a, b) => a.data.taskEvents.sequence - b.data.taskEvents.sequence)) { sink.next(payload); events.push(payload.data.taskEvents); }
          pending.clear();
          if (events.length) publishTaskBatch(events);
        };
        const dispose = wsClient.subscribe<Record<string, unknown>, ExecutionResult["extensions"]>(input, {
          ...sink,
          next(payload) {
            const taskEvent = payload.data?.taskEvents as TaskRealtimeEvent | undefined;
            if (typeof taskEvent?.sequence === "number") {
              const decision = observeSequence(taskEvent.sequence, connectionSnapshot.generation, taskEvent.type === "READY");
              if (!decision.accept && taskEvent.type !== "READY") return;
              if (taskEvent.type === "READY") { if (timer) clearTimeout(timer); timer = undefined; pending.clear(); }
              recordTaskEvent(taskEvent);
              if (taskEvent.type !== "READY") {
                // Coalesce repeated updates for one task, but retain membership changes.
                const before = pending.get(taskEvent.taskId)?.data?.taskEvents;
                const next = { ...payload, data: { ...payload.data, taskEvents: { ...taskEvent, listChanged: taskEvent.listChanged || before?.listChanged } } };
                pending.set(taskEvent.taskId, next);
                if (!timer) timer = setTimeout(flush, 100);
                return;
              }
            }
            sink.next(payload);
          }
        });
        return { unsubscribe() { if (timer) clearTimeout(timer); pending.clear(); dispose(); } };
      }
    };
  }
});

const taskSnapshotExchange: Exchange = ({ forward }) => operations => pipe(forward(operations), map(result =>
  result.data ? { ...result, data: protectTaskSnapshot(result.data) } : result
));

export const client = createClient({
  url: "/graphql",
  preferGetMethod: false,
  exchanges: [graphcacheExchange, taskSnapshotExchange, realtimeExchange, fetchExchange]
});

// Settings always read from the server, including lists and status queries.
// Keep subscriptions available without reading or writing the shared cache.
export const settingsClient = createClient({
  url: "/graphql",
  preferGetMethod: false,
  requestPolicy: "network-only",
  exchanges: [realtimeExchange, fetchExchange]
});
