import { useEffect, useRef, useState } from "react";
import { useClient } from "urql";
import { TestIndexerDocument, type TestIndexerInput } from "../graphql/generated/graphql";
import { describeQueryError } from "../services/queryError";

export type ConnectionResult = { state: "running" | "passed" | "failed"; errorCode?: string; error?: string };
export function useIndexerConnections(revision: string, scopeKey = revision) {
  const client = useClient();
  const current = useRef(scopeKey);
  current.current = scopeKey;
  const requests = useRef(new Set<AbortController>());
  const lock = useRef(false);
  const [snapshot, setSnapshot] = useState<{ revision: string; results: Record<string, ConnectionResult>; busy: boolean; all?: ConnectionResult }>({ revision: scopeKey, results: {}, busy: false });
  useEffect(() => {
    lock.current = false;
    return () => { for (const request of requests.current) request.abort(); requests.current.clear(); };
  }, [scopeKey]);
  const run = async (ids: string[], all = false, target?: Pick<TestIndexerInput, "instanceId" | "definitionId" | "draft">) => {
    if (lock.current || !revision || !ids.length) return;
    lock.current = true;
    const scope = scopeKey;
    const valid = () => current.current === scope && !cancelled;
    let cancelled = false;
    const batch = new AbortController();
    requests.current.add(batch);
    const update = (change: (value: typeof snapshot) => typeof snapshot) => {
      if (valid() && !batch.signal.aborted) setSnapshot(value => change(value.revision === scope ? value : { revision: scope, results: {}, busy: false }));
    };
    update(value => ({ ...value, busy: true, all: all ? { state: "running" } : undefined }));
    let next = 0, failed = 0;
    const worker = async () => {
      while (next < ids.length && valid() && !batch.signal.aborted) {
        const id = ids[next++];
        update(value => ({ ...value, results: { ...value.results, [id]: { state: "running" } } }));
        let outcome: ConnectionResult;
        try {
          const result = await client.mutation(TestIndexerDocument, { input: { instanceId: id, expectedRevision: revision, ...target, connectionOnly: true, search: false, download: false } }, { fetchOptions: { signal: batch.signal } }).toPromise();
          if (result.error) throw result.error;
          const report = result.data?.testIndexer;
          const failure = report?.stages.find(stage => stage.state === "FAILED");
          const passed = report?.stages.some(stage => stage.name === "CONNECTION" && stage.state === "PASSED");
          outcome = passed && !failure ? { state: "passed" } : { state: "failed", errorCode: failure?.errorCode };
        } catch (error) { outcome = { state: "failed", error: describeQueryError(error) }; }
        if (outcome.state === "failed") failed++;
        update(value => ({ ...value, results: { ...value.results, [id]: outcome } }));
      }
    };
    try { await Promise.all(Array.from({ length: Math.min(2, ids.length) }, worker)); }
    finally {
      update(value => ({ ...value, busy: false, all: all ? { state: failed ? "failed" : "passed" } : undefined }));
      cancelled = true;
      requests.current.delete(batch);
      if (current.current === scope) lock.current = false;
    }
  };
  const state: { results: Record<string, ConnectionResult>; busy: boolean; all?: ConnectionResult } = snapshot.revision === scopeKey ? snapshot : { results: {}, busy: false, all: undefined };
  return { ...state, run };
}
