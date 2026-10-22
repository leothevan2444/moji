import { describe, expect, it } from "vitest";
import i18n from "../i18n/i18n";
import { resources } from "../i18n/resources";
import { taskFailureSummary, taskPresentation, taskPrimaryAction, taskBatchEligibility, type DashboardTask } from "./taskUtils";

describe("sourcing validation failures", () => {
  it("keeps a restarted task in manual torrent selection and out of batch retry", async () => {
    await i18n.changeLanguage("en");
    const task = { id: "restarted", source: "SUBSCRIPTION", code: "SNOS-184", stage: "SOURCING", stageStatus: "BLOCKED", progress: 0,
      stageErrorCode: "MANUAL_SOURCE_REQUIRED" } as DashboardTask;
    expect(taskFailureSummary(task).title).toBe("Choose a torrent again");
    expect(taskPrimaryAction(task)).toBe("resolve");
    expect(taskBatchEligibility([task]).retryIds).toEqual([]);
  });
  it("treats scheduled resource absence as normal waiting with an immediate check action", async () => {
    await i18n.changeLanguage("en");
    const task = { id: "tracked", source: "SEARCH", code: "ABCD-123", stage: "SOURCING", stageStatus: "PENDING", progress: 0,
      sourcingLastCheckedAt: "2026-10-20T00:00:00Z", sourcingNextCheckAt: "2026-10-20T06:00:00Z" } as DashboardTask;
    const presentation = taskPresentation(task);
    expect(presentation.label).toBe("Waiting for resources");
    expect(presentation.phase).toBe("queued");
    expect(presentation.tone).toBe("tone-neutral");
    expect(presentation.showDownloadProgress).toBe(false);
    expect(presentation.detail).toContain("keeps searching");
    expect(presentation.lifecycle.find(step => step.key === "sourcing")?.state).not.toBe("error");
    expect(taskPrimaryAction(task)).toBe("retry");
    expect(taskBatchEligibility([task]).retryIds).toEqual([task.id]);
  });
  it.each([
    ["CANDIDATE_CODE_MISMATCH", "Candidate code mismatch", resources["zh-CN"].translation.taskModel.failure.candidateMismatch],
    ["CANDIDATE_IDENTITY_UNVERIFIED", "Candidate code unverified", resources["zh-CN"].translation.taskModel.failure.candidateUnverified],
    ["TORRENT_METADATA_FAILED", "Torrent validation failed", resources["zh-CN"].translation.taskModel.failure.torrentMetadata],
  ])("explains %s in both languages without exposing diagnostics", async (code, en, zh) => {
    const task = {
      id: "blocked", source: "SEARCH", code: "OFJE-657", stage: "SOURCING", stageStatus: "BLOCKED",
      progress: 0, stageErrorCode: code, stageErrorMessage: "raw indexer diagnostics with apikey=secret",
    } as DashboardTask;
    for (const [locale, expected] of [["en", en], ["zh-CN", zh]]) {
      await i18n.changeLanguage(locale);
      const summary = taskFailureSummary(task);
      expect(summary.title).toBe(expected);
      expect(summary.detail).not.toContain("secret");
      const presentation = taskPresentation(task);
      expect(presentation.phase).toBe("failed");
      expect(presentation.detail).toBe(summary.detail);
      expect(presentation.lifecycle.find(step => step.key === "sourcing")?.state).toBe("error");
      expect(presentation.lifecycle.find(step => step.key === "downloading")?.state).toBe("upcoming");
    }
  });
});

it.each([
 ["PENDING", "Waiting for source selection", "upcoming"],
 ["RUNNING", "Selecting source", "current"]
])("labels sourcing %s without presenting it as a download", async (stageStatus,label,state) => {
 await i18n.changeLanguage("en");
 const presentation = taskPresentation({ id:"queued",source:"SEARCH",code:"ABCD-123",stage:"SOURCING",stageStatus,progress:0 } as DashboardTask);
 expect(presentation.label).toBe(label);
 expect(presentation.summary).toBe(label);
 expect(presentation.showDownloadProgress).toBe(false);
 expect(presentation.lifecycle.find(step => step.key === "sourcing")?.state).toBe(state);
});
