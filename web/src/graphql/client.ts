import { createClient as createWSClient } from "graphql-ws";
import { createClient, fetchExchange, subscriptionExchange } from "urql";
import { graphcacheExchange } from "./cache";

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

export const client = createClient({
  url: "/graphql",
  preferGetMethod: false,
  exchanges: [
    graphcacheExchange,
    subscriptionExchange({
      forwardSubscription(request) {
        if (!request.query) throw new Error("GraphQL subscription query is missing");
        const input = { ...request, query: request.query };
        return {
          subscribe(sink) {
            const dispose = wsClient.subscribe(input, sink);
            return { unsubscribe: dispose };
          }
        };
      }
    }),
    fetchExchange
  ]
});
