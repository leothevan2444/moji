// @vitest-environment jsdom
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskStage, TaskStageStatus } from "../../graphql/generated/graphql";
import { taskLifecycle, taskFailureSummary, type DashboardTask } from "../../utils";
import { TaskCard } from "./TaskCard";
import { TaskDetail } from "./TaskDetail";

vi.mock("urql", () => ({ useQuery: () => [{ fetching: false }, vi.fn()], useMutation: () => [{ fetching: false }, vi.fn()] }));
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
    if (stage === TaskStage.Completed) {
      const overview = screen.getByRole("article", { name: "Task overview" });
      expect(within(overview).queryByText("Download and ingest completed.")).toBeNull();
      expect(overview.querySelector(".task-detail-hero")).toBeNull();
    }
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
    const actions = screen.getAllByRole("button", { name: `${label}: DEMO-001` });
    expect(actions).toHaveLength(2);
    await user.click(actions[0]);
    if (handler === "files") {
      expect(open).toHaveBeenCalledWith(task.id);
      const fileDetails = screen.getByRole("article", { name: "Torrent information" });
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

  it("moves file locations to their sections and removes the task information section", async () => {
    const user = userEvent.setup();
    const callbacks = props();
    const path = "/stash/library/long-directory/movie.mkv";
    const download = "/qb/downloads/movie.mkv";
    const { container } = render(<TaskDetail task={{ ...task, stashScanPath: path, contentPath: download, savePath: "/qb/downloads", tags: "favorites", mojiTransferPath: "/moji/library/movie.mkv" }} {...callbacks} />);
    expect(screen.queryByRole("article", { name: "Task information" })).toBeNull();
    const information = screen.getByRole("article", { name: "Stash movie information" });
    const torrent = screen.getByRole("article", { name: "Torrent information" });
    expect(information.querySelector("details")).toBeNull();
    expect(within(information).getByText(path).getAttribute("title")).toBe(path);
    expect(within(information).getByText(path).classList.contains("task-inline-value--truncate")).toBe(true);
    expect(within(torrent).getByText(download)).toBeTruthy();
    expect(within(torrent).getByText("favorites")).toBeTruthy();
    expect(within(torrent).getByText("Example movie")).toBeTruthy();
    expect(screen.queryByText("/moji/library/movie.mkv")).toBeNull();
    await user.click(within(information).getByRole("button", { name: "Copy Library location" }));
    expect(callbacks.onCopy).toHaveBeenCalledWith(path, "Library location copied");
    await user.click(within(torrent).getByRole("button", { name: "Copy Download location" }));
    expect(callbacks.onCopy).toHaveBeenCalledWith(download, "Download location copied");
    expect(screen.queryByRole("button", { name: "Sync all task progress" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Scan pending-ingest tasks" })).toBeNull();
    const actions = screen.getByRole("group", { name: "Actions" });
    expect(container.lastElementChild).toBe(actions.closest("article"));
    expect(actions.querySelector("details, summary")).toBeNull();
    await user.click(within(actions).getByRole("button", { name: "Delete task: DEMO-001" }));
    expect(callbacks.onDeleteTask).toHaveBeenCalledWith(task.id);
  });

  it("copies operational diagnostics without adding persistent technical fields", async () => {
    const callbacks = props();
    const current = { ...task, stage: TaskStage.Scanning, stageStatus: TaskStageStatus.Blocked, stashIngestPhase: "GENERATE", stashGenerateJobId: "generate-1", stageErrorCode: "STASH_GENERATE_FAILED", stashScanError: "raw failure", mojiSourcePath: "/moji/downloads/movie", torrentUrl: "https://example.test/download?token=secret" };
    render(<TaskDetail task={current} {...callbacks} />);
    expect(screen.queryByText("generate-1")).toBeNull();
    expect(screen.queryByText("raw failure")).toBeNull();
    await userEvent.setup().click(screen.getByRole("button", { name: "Copy diagnostic information" }));
    const [value, message] = callbacks.onCopy.mock.calls[0];
    expect(JSON.parse(value)).toMatchObject({ id: task.id, stashGenerateJobId: "generate-1", stashScanError: "raw failure", mojiSourcePath: "/moji/downloads/movie" });
    expect(value).not.toContain("token=secret");
    expect(message).toBe("Diagnostic information copied");
    const currentStep = screen.getByText("Generate").closest("li")!;
    expect(currentStep.getAttribute("aria-current")).toBe("step");
    expect(currentStep.querySelector("p")?.textContent).toContain("then retry");
  });

  it("shows transfer details only in the active transfer substep", async () => {
    const callbacks = props();
    const current = { ...task, stage: TaskStage.Transferring, transferAction: "COPY", mojiTransferPath: "/moji/library/movie.mkv" };
    const { rerender } = render(<TaskDetail task={current} {...callbacks} />);
    const step = screen.getByText("Transfer").closest("li")!;
    expect(within(step).getByText("Copy")).toBeTruthy();
    expect(within(step).getByText(current.mojiTransferPath).getAttribute("title")).toBe(current.mojiTransferPath);
    await userEvent.setup().click(within(step).getByRole("button", { name: "Copy Destination" }));
    expect(callbacks.onCopy).toHaveBeenCalledWith(current.mojiTransferPath, "Destination copied");
    rerender(<TaskDetail task={{ ...current, stage: TaskStage.Completed, stageStatus: TaskStageStatus.Done }} {...callbacks} />);
    expect(screen.queryByText(current.mojiTransferPath)).toBeNull();
  });

  it("keeps all three stages visible and embeds compact substeps in the active ingest stage", () => {
    render(<TaskDetail task={{ ...task, stage: TaskStage.Scanning, stashIngestPhase: "IDENTIFY", progress: 1 }} {...props()} />);
    const overview = screen.getByRole("article", { name: "Task overview" });
    expect(within(overview).getByRole("list", { name: "Lifecycle" }).children).toHaveLength(3);
    expect(within(overview).getByRole("region", { name: "Ingest progress" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Lifecycle" })).toBeNull();
    expect(screen.getByRole("region", { name: "Ingest progress" })).toBeTruthy();
    expect(within(screen.getByRole("list", { name: "Ingest steps" })).getAllByRole("listitem")).toHaveLength(6);
    expect(screen.getByText("Identify").closest("li")?.getAttribute("aria-current")).toBe("step");
    expect(screen.queryByText("Expand ingest steps")).toBeNull();
  });

  it("places download progress, stage details, and key dates inside the overview", () => {
    const { container } = render(<TaskDetail task={task} {...props()} />);
    const overview = screen.getByRole("article", { name: "Task overview" });
    expect(container.firstElementChild).toBe(overview);
    expect(within(overview).getByRole("heading", { name: "Task overview" })).toBeTruthy();
    expect(within(overview).queryByRole("heading", { name: "DEMO-001" })).toBeNull();
    const progress = within(overview).getByRole("progressbar", { name: "Downloading" });
    expect(progress.closest("li")?.getAttribute("aria-current")).toBe("step");
    expect(progress.getAttribute("aria-valuenow")).toBe("46");
    expect(within(overview).getByText("Created")).toBeTruthy();
    expect(within(overview).getByText("Last updated")).toBeTruthy();
    expect(within(overview).getAllByText("Download progress 46%")).toHaveLength(1);
    expect(screen.queryByRole("heading", { name: "Lifecycle" })).toBeNull();
  });

  it("never fabricates milestone timestamps from updatedAt", () => {
    const current = { ...task, stage: TaskStage.Completed, sourcingCompletedAt: "2026-10-17T02:10:00Z", downloadCompletedAt: "2026-10-17T02:30:00Z", stashScanStartedAt: "2026-10-17T02:35:00Z", ingestCompletedAt: "2026-10-17T03:00:00Z" };
    const steps = taskLifecycle(current, taskFailureSummary(current));
    expect(steps.filter(step => step.time).map(step => [step.key, step.time])).toEqual([
      ["sourcing", current.sourcingCompletedAt], ["downloading", current.downloadCompletedAt], ["scanning", current.stashScanStartedAt], ["completed", current.ingestCompletedAt]
    ]);
  });
});
