import { useEffect, useRef, useState } from "react";
import { useClient, useSubscription } from "urql";
import { IndexerCheckActivityDocument, TestIndexerDocument, type TestIndexerInput } from "../graphql/generated/graphql";
import { describeQueryError } from "../services/queryError";

export type ConnectionResult = { state: "pending" | "queued" | "running" | "passed" | "failed"; errorCode?: string; error?: string };
type ScopedResult = ConnectionResult & { version: string };
type Request = { id: string; requestId: string; version: string; controller: AbortController; settled: Promise<void>; finish: () => void };
type Target = Pick<TestIndexerInput, "instanceId" | "definitionId" | "draft">;

// getRandomValues also works when a self-hosted Moji uses plain HTTP.
function checkID() { return Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, "0")).join(""); }

export function useIndexerConnections(revision: string, scopeKey = revision, versions?: Record<string, string>) {
  const client = useClient();
  const [clientId] = useState(checkID);
  const current = useRef({ revision, scopeKey, versions });
  current.current = { revision, scopeKey, versions };
  const requests = useRef(new Map<string, Request>());
  const group = useRef<object | null>(null);
  const mounted = useRef(true);
  const [allRunning, setAllRunning] = useState(false);
  const [results, setResults] = useState<Record<string, ScopedResult>>({});
  const versionOf = (id: string) => current.current.versions ? current.current.versions[id] : current.current.scopeKey;
  const isCurrent = (request: Request) => mounted.current && !request.controller.signal.aborted && requests.current.get(request.id) === request && versionOf(request.id) === request.version;
  const cancel = (request: Request) => {
    request.controller.abort();
    if (requests.current.get(request.id) === request) requests.current.delete(request.id);
    request.finish();
  };
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; for (const request of requests.current.values()) cancel(request); };
  }, []);
  // Draft contents (including credentials) and opaque versions stay in memory.
  const identity = JSON.stringify({ scopeKey, versions });
  useEffect(() => {
    for (const request of requests.current.values()) if (versionOf(request.id) !== request.version) cancel(request);
    setResults(previous => Object.fromEntries(Object.entries(previous).filter(([id, value]) => value.version === versionOf(id))));
  }, [identity]);

  const pending = [...requests.current.values()].some(isCurrent);
  const [{ data }] = useSubscription({ query: IndexerCheckActivityDocument, variables: { clientId }, pause: !pending });
  useEffect(() => {
    if (!data) return;
    setResults(previous => {
      let next = previous;
      for (const activity of data.indexerCheckActivity) {
        const request = [...requests.current.values()].find(value => value.requestId === activity.requestId);
        if (!request || !isCurrent(request)) continue;
        const state = activity.state === "QUEUED" ? "queued" : "running";
        if (next[request.id]?.state === state) continue;
        if (next === previous) next = { ...previous };
        next[request.id] = { version: request.version, state };
      }
      return next;
    });
  }, [data]);

  const start = (id: string, target?: Target): Promise<void> => {
    const version = versionOf(id);
    if (!version || !current.current.revision) return Promise.resolve();
    const existing = requests.current.get(id);
    if (existing && isCurrent(existing)) return existing.settled;
    if (existing) cancel(existing);
    let finish!: () => void;
    const settled = new Promise<void>(resolve => { finish = resolve; });
    const request: Request = { id, requestId: checkID(), version, controller: new AbortController(), settled, finish };
    requests.current.set(id, request);
    const update = (result: ConnectionResult) => {
      if (isCurrent(request)) setResults(previous => ({ ...previous, [id]: { ...result, version } }));
    };
    update({ state: "pending" });
    void (async () => {
      try {
        const response = await client.mutation(TestIndexerDocument, {
          input: { instanceId: id, expectedRevision: current.current.revision, expectedConnectionRevision: current.current.versions ? version : undefined, ...target, clientId, requestId: request.requestId, connectionOnly: true, search: false, download: false }
        }, { fetchOptions: { signal: request.controller.signal } }).toPromise();
        if (response.error) throw response.error;
        const report = response.data?.testIndexer;
        const failure = report?.stages.find(stage => stage.state === "FAILED");
        const passed = report?.stages.some(stage => stage.name === "CONNECTION" && stage.state === "PASSED");
        update(passed && !failure ? { state: "passed" } : { state: "failed", errorCode: failure?.errorCode });
      } catch (error) { update({ state: "failed", error: describeQueryError(error) }); }
      finally {
        if (requests.current.get(id) === request) requests.current.delete(id);
        request.finish();
      }
    })();
    return settled;
  };
  const run = async (ids: string[], all = false, target?: Target) => {
    if (all && group.current) return;
    const batch = all ? {} : null;
    if (batch) { group.current = batch; setAllRunning(true); }
    try { await Promise.all([...new Set(ids)].map(id => start(id, target))); }
    finally {
      if (batch && group.current === batch) {
        group.current = null;
        if (mounted.current) setAllRunning(false);
      }
    }
  };
  return {
    results: Object.fromEntries(Object.entries(results).filter(([id, result]) => result.version === versionOf(id))) as Record<string, ConnectionResult>,
    busy: pending, all: allRunning ? { state: "running" as const } : undefined,
    isTesting: (id: string) => { const request = requests.current.get(id); return Boolean(request && isCurrent(request)); },
    run
  };
}
