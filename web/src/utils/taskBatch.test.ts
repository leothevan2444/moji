import { describe, expect, it } from "vitest";
import { mergeTaskSelection, taskBatchEligibility, type DashboardTask } from "./taskUtils";

const task = (id: string, stage: string, stageStatus: string) => ({ id, stage, stageStatus }) as DashboardTask;

describe("task batch helpers", () => {
  it("classifies retry and ingest eligibility", () => {
    const result = taskBatchEligibility([
      task("retry", "SOURCING", "BLOCKED"),
      task("ingest", "PENDING_INGEST", "PENDING"),
      task("running", "SCANNING", "RUNNING"),
      task("blocked-scan", "SCANNING", "BLOCKED")
    ]);
    expect(result.retryIds).toEqual(["retry", "blocked-scan"]);
    expect(result.ingestIds).toEqual(["ingest", "blocked-scan"]);
  });

  it("deduplicates selection and enforces the limit", () => {
    expect(mergeTaskSelection(["a", "b"], ["b", "c", "d"], 3)).toEqual(["a", "b", "c"]);
  });
});
