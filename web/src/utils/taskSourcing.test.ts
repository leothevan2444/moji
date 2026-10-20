import { describe, expect, it } from "vitest";
import i18n from "../i18n/i18n";
import { resources } from "../i18n/resources";
import { taskFailureSummary, taskPresentation, type DashboardTask } from "./taskUtils";

describe("sourcing validation failures", () => {
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
