// @vitest-environment jsdom
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskStage, TaskStageStatus } from "../../graphql/generated/graphql";
import { taskPresentation, type DashboardTask } from "../../utils";
import { TaskTimeline } from "./TaskTimeline";

const task = {
  id: "timeline", source: "MANUAL", code: "DEMO-001", stage: TaskStage.Downloading,
  stageStatus: TaskStageStatus.Running, progress: 0.46,
  createdAt: "2026-10-18T02:00:00Z", updatedAt: "2026-10-18T03:00:00Z"
} as DashboardTask;
const view = (patch: Partial<DashboardTask> = {}) => <TaskTimeline steps={taskPresentation({ ...task, ...patch }).lifecycle} />;

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);

describe("adaptive task lifecycle", () => {
  it.each([TaskStage.Sourcing, TaskStage.Downloading, TaskStage.Completed])("hides ingest substeps outside ingest during %s", stage => {
    render(view({ stage, stageStatus: stage === TaskStage.Completed ? TaskStageStatus.Done : TaskStageStatus.Running }));
    const main = screen.getByRole("list", { name: "Lifecycle" });
    expect(main.children).toHaveLength(3);
    for (const label of ["Source", "Download", "Ingest"]) expect(within(main).getByText(label)).toBeTruthy();
    expect(screen.queryByRole("region", { name: "Ingest progress" })).toBeNull();
    expect(screen.queryByRole("list", { name: "Ingest steps" })).toBeNull();
    expect(main.querySelector("details")).toBeNull();
    expect(within(main).queryByText("Completed")).toBeNull();
  });

  it("places real completion times on their stages and leaves unknown history empty", () => {
    const sourcingCompletedAt = "2026-10-18T02:10:00Z";
    const downloadCompletedAt = "2026-10-18T02:30:00Z";
    const ingestCompletedAt = "2026-10-18T02:55:00Z";
    const { rerender } = render(view({ stage: TaskStage.Completed, stageStatus: TaskStageStatus.Done, sourcingCompletedAt, downloadCompletedAt, ingestCompletedAt }));
    const stages = screen.getByRole("list", { name: "Lifecycle" }).children;
    expect(stages[0].querySelector("time")?.dateTime).toBe(sourcingCompletedAt);
    expect(stages[0].querySelector("time")?.textContent).toContain("Completed at");
    expect(stages[1].querySelector("time")?.dateTime).toBe(downloadCompletedAt);
    expect(stages[2].querySelector("time")?.dateTime).toBe(ingestCompletedAt);
    expect(stages[2].querySelector("time")?.textContent).toContain("Completed at");
    rerender(view({ stage: TaskStage.Completed, stageStatus: TaskStageStatus.Done, downloadCompletedAt }));
    expect(stages[0].querySelector("time")).toBeNull();
    expect(stages[2].querySelector("time")).toBeNull();
    rerender(view({ stage: TaskStage.Scanning, stashIngestPhase: "VERIFY" }));
    expect(stages[2].querySelector("time")).toBeNull();
  });

  it("compresses past and future stages and expands only the current stage", () => {
    render(view());
    const stages = Array.from(screen.getByRole("list", { name: "Lifecycle" }).children);
    expect(stages[0].querySelector("p")).toBeNull();
    expect(stages[0].querySelector("time")).toBeNull();
    expect(stages[1].getAttribute("aria-current")).toBe("step");
    expect(stages[1].querySelector("p")?.textContent).toContain("46%");
    expect(stages[2].querySelector("p")).toBeNull();
    expect(within(stages[0] as HTMLElement).queryByText("Done")).toBeNull();
    expect(within(stages[1] as HTMLElement).getByText("In progress")).toBeTruthy();
    expect(within(stages[2] as HTMLElement).getByText("Upcoming")).toBeTruthy();
  });

  it.each([
    [TaskStage.PendingIngest, undefined, "Pending ingest"],
    [TaskStage.Transferring, undefined, "Transfer"],
    [TaskStage.Scanning, "SCAN", "Scan"],
    [TaskStage.Scanning, "GENERATE", "Generate"],
    [TaskStage.Scanning, "IDENTIFY", "Identify"],
    [TaskStage.Scanning, "VERIFY", "Verify"]
  ])("shows compact progress for %s / %s", (stage, phase, label) => {
    render(view({ stage, stashIngestPhase: phase }));
    const progress = screen.getByRole("region", { name: "Ingest progress" });
    const substeps = within(progress).getAllByRole("listitem");
    expect(substeps).toHaveLength(6);
    expect(substeps.filter(step => step.hasAttribute("aria-current"))).toHaveLength(1);
    const current = within(progress).getByText(label).closest("li")!;
    expect(current.getAttribute("aria-current")).toBe("step");
    expect(current.querySelector("p")).not.toBeNull();
    expect(substeps.filter(step => step !== current).every(step => !step.querySelector("p"))).toBe(true);
    expect(screen.getByRole("list", { name: "Lifecycle" }).children[2].contains(progress)).toBe(true);
    expect(progress.textContent).not.toContain("100%");
    expect(within(progress).queryByText("Done")).toBeNull();
  });

  it("shows a blocked substep without counting it as completed", () => {
    render(view({ stage: TaskStage.Scanning, stashIngestPhase: "GENERATE", stageStatus: TaskStageStatus.Blocked, stageErrorCode: "STASH_GENERATE_FAILED" }));
    const progress = screen.getByRole("region", { name: "Ingest progress" });
    const current = within(progress).getByText("Generate").closest("li")!;
    expect(current.classList.contains("is-error")).toBe(true);
    expect(within(current).getByText("Needs attention")).toBeTruthy();
    expect(within(progress).getByText("3 / 6 steps complete")).toBeTruthy();
    expect(current.querySelector("p")).not.toBeNull();
    expect(within(progress).getByText("Identify").closest("li")?.querySelector("p")).toBeNull();
  });

  it("removes ingest details when the task completes or returns to an earlier stage", () => {
    const { rerender } = render(view({ stage: TaskStage.Scanning, stashIngestPhase: "IDENTIFY" }));
    expect(screen.getByRole("region", { name: "Ingest progress" })).toBeTruthy();
    rerender(view({ stage: TaskStage.Completed, stageStatus: TaskStageStatus.Done }));
    expect(screen.queryByRole("region", { name: "Ingest progress" })).toBeNull();
    const completedStages = Array.from(screen.getByRole("list", { name: "Lifecycle" }).children);
    expect(completedStages).toHaveLength(3);
    expect(completedStages.every(stage => stage.classList.contains("is-done"))).toBe(true);
    expect(screen.getByRole("list", { name: "Lifecycle" }).querySelectorAll("p")).toHaveLength(0);
    rerender(view());
    expect(screen.queryByRole("list", { name: "Ingest steps" })).toBeNull();
    expect(screen.getByRole("list", { name: "Lifecycle" }).querySelectorAll("p")).toHaveLength(1);
  });
});
