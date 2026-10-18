// @vitest-environment jsdom
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskStage, TaskStageStatus } from "../../graphql/generated/graphql";
import { taskLifecycle, taskFailureSummary, type DashboardTask } from "../../utils";
import { TaskCard } from "./TaskCard";
import { TaskDetail } from "./TaskDetail";

vi.mock("urql", () => ({ useMutation: () => [{ fetching: false }, vi.fn()] }));
const task = {
  id: "task-1", source: "MANUAL", code: "DEMO-001", torrentName: "Example movie",
  stage: "DOWNLOADING", stageStatus: "RUNNING", progress: 0.46,
  createdAt: "2026-10-17T02:00:00Z", updatedAt: "2026-10-17T03:00:00Z",
  files: [{ index: 1, path: "movie/part-1.mp4", kind: "MAIN", part: 1, selected: true, reason: "TARGET_CODE", size: 100 }]
} as DashboardTask;
const props = () => ({ pendingScan: false, pendingRetry: false, pendingDelete: false,
  onCopy: vi.fn(), onResolveTask: vi.fn(), onScanTask: vi.fn(), onRetryTask: vi.fn(), onDeleteTask: vi.fn() });

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);

describe("task experience", () => {
  it.each([TaskStage.PendingIngest, TaskStage.Transferring, TaskStage.Scanning, TaskStage.Completed] as const)("does not show download percentages as overall progress during %s", stage => {
    const current = { ...task, stage, progress: 1 };
    const { container } = render(<><TaskCard task={current} onOpen={vi.fn()} /><TaskDetail task={current} {...props()} /></>);
    expect(screen.queryAllByRole("progressbar")).toHaveLength(0);
    expect(container.textContent).not.toContain("100%");
  });

  it("labels download progress and reports its numeric value accessibly", () => {
    render(<TaskCard task={task} onOpen={vi.fn()} />);
    expect(screen.getByText("Download progress 46%")).toBeTruthy();
    expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("46");
  });

  it.each([
    [TaskStage.Sourcing, undefined, "Choose or add a resource", "onResolveTask"],
    [TaskStage.Downloading, "FILE_SELECTION_REQUIRED", "Confirm movie files", "files"],
    [TaskStage.Transferring, "MEDIA_PREPARATION_FAILED", "Retry current stage", "onRetryTask"]
  ] as const)("uses the same next action in cards and details for %s / %s", async (stage, stageErrorCode, label, handler) => {
    const user = userEvent.setup();
    const current = { ...task, stage, stageErrorCode, stageStatus: TaskStageStatus.Blocked };
    const callbacks = props();
    const open = vi.fn();
    render(<>
      <TaskCard task={current} onOpen={open} onResolve={callbacks.onResolveTask} onRetry={callbacks.onRetryTask} />
      <TaskDetail task={current} {...callbacks} />
    </>);
    const actions = screen.getAllByRole("button", { name: `${label}: Example movie` });
    expect(actions).toHaveLength(2);
    await user.click(actions[0]);
    if (handler === "files") {
      expect(open).toHaveBeenCalledWith(task.id);
      const fileDetails = screen.getByRole("article", { name: "File list" });
      expect(fileDetails.querySelector("summary")).toBeNull();
      await user.click(actions[1]);
      expect(document.activeElement).toBe(within(fileDetails).getByRole("checkbox"));
    } else {
      expect(callbacks[handler]).toHaveBeenCalledWith(task.id);
      callbacks[handler].mockClear();
      await user.click(actions[1]);
      expect(callbacks[handler]).toHaveBeenCalledWith(task.id);
      expect(open).not.toHaveBeenCalled();
    }
  });

  it("shows task information directly with hover titles and preserves path copying", async () => {
    const user = userEvent.setup();
    const callbacks = props();
    const path = "/library/long-directory/movie.mkv";
    render(<TaskDetail task={{ ...task, mojiTransferPath: path }} {...callbacks} />);
    const information = screen.getByRole("article", { name: "Task information" });
    expect(information.querySelector("details")).toBeNull();
    expect(within(information).getByText(path).getAttribute("title")).toBe(path);
    expect(within(information).getByText(path).classList.contains("task-inline-value--truncate")).toBe(true);
    expect(screen.getByRole("article", { name: "File list" }).querySelector("details")).toBeNull();
    await user.click(within(information).getByRole("button", { name: "Copy Moji delivery target" }));
    expect(callbacks.onCopy).toHaveBeenCalledWith(path, "Moji delivery target copied");
    expect(screen.queryByRole("button", { name: "Sync all task progress" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Scan pending-ingest tasks" })).toBeNull();
  });

  it("summarizes the lifecycle in four steps and expands ingest stages", async () => {
    const user = userEvent.setup();
    render(<TaskDetail task={{ ...task, stage: TaskStage.Scanning, stashIngestPhase: "IDENTIFY", progress: 1 }} {...props()} />);
    expect(within(screen.getAllByRole("list")[0]).getAllByRole("listitem")).toHaveLength(4);
    expect(screen.getByText("Expand ingest steps").closest("details")?.open).toBe(false);
    expect(screen.getByText("Current stage: Identify")).toBeTruthy();
    await user.click(screen.getByText("Expand ingest steps"));
    expect(screen.getByText("Expand ingest steps").closest("details")?.open).toBe(true);
    expect(within(screen.getAllByRole("list")[1]).getAllByRole("listitem")).toHaveLength(6);
    expect(screen.getAllByText("Identify")[0].closest("li")?.getAttribute("aria-current")).toBe("step");
  });

  it("never fabricates milestone timestamps from updatedAt", () => {
    const current = { ...task, stage: TaskStage.Completed, downloadCompletedAt: "2026-10-17T02:30:00Z", stashScanStartedAt: "2026-10-17T02:35:00Z" };
    const steps = taskLifecycle(current, taskFailureSummary(current));
    expect(steps.filter(step => step.time).map(step => [step.key, step.time])).toEqual([
      ["sourcing", current.createdAt], ["pending_ingest", current.downloadCompletedAt], ["scanning", current.stashScanStartedAt]
    ]);
  });
});
