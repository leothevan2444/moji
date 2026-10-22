import { describe, expect, it } from "vitest";
import { isTaskActive, taskGroup, taskPrimaryAction, type DashboardTask } from "./taskUtils";

const task = (stage: string, stageStatus = "PENDING", extra = {}) => ({ stage, stageStatus, ...extra }) as DashboardTask;

describe("task categories and automatic ingest", () => {
  it.each(["SOURCING", "DOWNLOADING", "PENDING_INGEST", "TRANSFERRING", "SCANNING"])("keeps %s in progress until it completes or needs attention", stage => {
    expect(taskGroup(task(stage))).toBe("active");
    expect(isTaskActive(task(stage))).toBe(true);
    expect(taskGroup(task(stage, "BLOCKED"))).toBe("attention");
    expect(isTaskActive(task(stage, "BLOCKED"))).toBe(false);
  });

  it("separates resource tracking from work that is progressing", () => {
    const waiting = task("SOURCING", "PENDING", { sourcingNextCheckAt: "2026-10-23T00:00:00Z" });
    expect(taskGroup(waiting)).toBe("waiting");
    expect(isTaskActive(waiting)).toBe(false);
    expect(taskPrimaryAction(waiting)).toBe("retry");
    expect(taskGroup(task("COMPLETED", "SUCCEEDED"))).toBe("completed");
  });

  it("offers retry for blocked ingest and no manual action for pending ingest", () => {
    for (const stage of ["PENDING_INGEST", "TRANSFERRING", "SCANNING"]) {
      expect(taskPrimaryAction(task(stage))).toBeNull();
      expect(taskPrimaryAction(task(stage, "BLOCKED"))).toBe("retry");
    }
  });
});
