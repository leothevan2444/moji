import { describe, expect, it } from "vitest";
import { acceptLibraryStatus, newerLibraryStatus, type LibraryHistory } from "./useIndexerManagement";
import { makeFragmentData } from "../graphql/generated/fragment-masking";
import { IndexerDefinitionJobFieldsFragmentDoc, IndexerDefinitionJobKind, IndexerDefinitionJobState, type IndexerDefinitionStatusFieldsFragment } from "../graphql/generated/graphql";

function status(revision: number, jobRevision: number, streamId = "one", state = IndexerDefinitionJobState.Downloading): IndexerDefinitionStatusFieldsFragment {
  return { streamId, revision, activeRevision: "a", previousRevision: "", availableRevision: "b", available: true, checkErrorCode: "", errorCode: "", installed: [], job: makeFragmentData({ id: "job", kind: IndexerDefinitionJobKind.Update, state, targetRevision: "b", expectedActiveRevision: "a", downloadedBytes: 0, errorCode: "", correlationId: "correlation", startedAt: "2026-11-02T00:00:00Z", revision: jobRevision }, IndexerDefinitionJobFieldsFragmentDoc) };
}
describe("definition library snapshot acceptance", () => {
  it("does not replace completed durable progress with a late initial response", () => {
    const complete = status(4, 8, "one", IndexerDefinitionJobState.Completed);
    expect(newerLibraryStatus(complete, status(4, 2))).toBe(complete);
  });
  it("accepts a newer core revision", () => {
    const next = status(5, 1); expect(newerLibraryStatus(status(4, 8), next)).toBe(next);
  });
  it("accepts a restarted service even when its restored database revision is lower", () => {
    const original = status(50, 8); const restored = status(2, 1, "restarted");
    const first: LibraryHistory = { current: original, retired: new Set() };
    const next = acceptLibraryStatus(first, restored); expect(next.current).toBe(restored);
    expect(acceptLibraryStatus(next, original)).toBe(next);
    expect(first.retired.size).toBe(0);
  });
  it("keeps retirement history bounded", () => {
    let history: LibraryHistory = { current: null, retired: new Set() };
    for (let i = 0; i < 80; i++) history = acceptLibraryStatus(history, status(1, 1, `process-${i}`));
    expect(history.retired.size).toBe(32); expect(history.current?.streamId).toBe("process-79");
  });
});
