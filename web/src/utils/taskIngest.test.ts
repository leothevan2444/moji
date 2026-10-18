import { beforeEach, describe, expect, it } from "vitest";
import i18n from "../i18n/i18n";
import { taskPresentation, type DashboardTask } from "./taskUtils";

const task = (phase: string, status = "RUNNING") => ({
  id: "ingest", source: "MANUAL", code: "ABCD-123", stage: "SCANNING", stageStatus: status,
  progress: 1, stashIngestPhase: phase, stashScanJobId: "scan-1", stashGenerateJobId: "generate-1", stashIdentifyJobId: "identify-1",
}) as DashboardTask;

describe("Stash ingest task presentation", () => {
  beforeEach(async () => { await i18n.changeLanguage("en"); });

  it("shows generation as current and leaves identification unfinished", () => {
    const presentation = taskPresentation(task("GENERATE"));
    expect(presentation.label).toBe("Stash generating");
    expect(presentation.detail).toBe("Generating previews and playback assets");
    expect(presentation.detail).not.toContain("generate-1");
    expect(presentation.lifecycle.find(step => step.key === "scanning")?.state).toBe("done");
    expect(presentation.lifecycle.find(step => step.key === "generating")?.state).toBe("current");
    expect(presentation.lifecycle.find(step => step.key === "identifying")?.state).toBe("upcoming");
    expect(presentation.lifecycle.find(step => step.key === "completed")?.state).toBe("upcoming");
  });

  it("describes identification without showing internal job IDs", () => {
    const presentation = taskPresentation(task("IDENTIFY"));
    expect(presentation.label).toBe("Stash identifying");
    expect(presentation.detail).toBe("Identifying movie information");
    expect(presentation.detail).not.toContain("identify-1");
    expect(presentation.lifecycle.find(step => step.key === "identifying")?.state).toBe("current");
  });

  it("keeps incomplete metadata blocked with a localized action", () => {
    const presentation = taskPresentation({ ...task("IDENTIFY", "BLOCKED"), stageErrorCode: "STASH_INGEST_INCOMPLETE", stashScanError: "diagnostic failure" });
    expect(presentation.phase).toBe("failed");
    expect(presentation.detail).toContain("Review Identify sources");
    expect(presentation.lifecycle.find(step => step.key === "identifying")?.state).toBe("error");
    expect(presentation.lifecycle.find(step => step.key === "completed")?.state).toBe("upcoming");
  });
});

it.each([
  ["STASH_SCENE_MISSING", "No Stash scenes"],
  ["STASH_SCENE_CHANGED", "Automatic recovery stopped"],
  ["STASH_IDENTIFY_SOURCES_MISSING", "no saved Identify sources"],
])("explains %s without exposing backend diagnostics", async (code, expected) => {
  await i18n.changeLanguage("en");
  const presentation = taskPresentation({ ...task("IDENTIFY", "BLOCKED"), stageErrorCode: code, stashScanError: "raw server diagnostic" });
  expect(presentation.phase).toBe("failed");
  expect(presentation.detail).toContain(expected);
  expect(presentation.detail).not.toContain("raw server diagnostic");
});
