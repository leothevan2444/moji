// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskStage, TaskStageStatus } from "../../graphql/generated/graphql";
import type { DashboardTask } from "../../utils";
import { TaskProcessingQueueDrawer } from "./TaskProcessingQueueDrawer";

const mocks = vi.hoisted(() => ({ details: new Map<string, any>(), retry: vi.fn(), source: vi.fn(), files: vi.fn(), refresh: vi.fn() }));
vi.mock("urql", () => ({
  useQuery: ({ variables, query }: any) => query.definitions[0].name.value === "TaskDetailDocument"
    ? [{ data: { task: mocks.details.get(variables.id) }, fetching: false }, mocks.refresh]
    : [{ data: { blockedTaskTorrentCandidates: [] }, fetching: false }, mocks.refresh],
  useMutation: (query: any) => [{ fetching: false }, query.definitions[0].name.value === "RetryTaskDocument" ? mocks.retry
    : query.definitions[0].name.value === "ResolveBlockedSourcingTaskDocument" ? mocks.source : mocks.files]
}));

const task = (id: string, extra: Partial<DashboardTask> = {}) => ({
  id, code: id, source: "MANUAL", stage: TaskStage.Scanning, stageStatus: TaskStageStatus.Blocked,
  createdAt: "2026-10-01T00:00:00Z", updatedAt: "2026-10-01T00:00:00Z", progress: 1, ...extra
}) as DashboardTask;
const callbacks = () => ({ onClose: vi.fn(), onCopy: vi.fn(), onNotify: vi.fn() });
function setup(tasks: DashboardTask[]) {
  tasks.forEach(task => mocks.details.set(task.id, task));
  const props = callbacks();
  return { ...render(<TaskProcessingQueueDrawer tasks={tasks} {...props} />), props };
}
beforeEach(async () => {
  vi.clearAllMocks(); mocks.details.clear(); await i18n.changeLanguage("en");
  mocks.retry.mockImplementation(async ({ id }) => ({ data: { retryTask: task(id, { stageStatus: TaskStageStatus.Running }) } }));
  mocks.source.mockImplementation(async ({ id }) => ({ data: { resolveBlockedSourcingTask: task(id, { stage: TaskStage.Downloading, stageStatus: TaskStageStatus.Running }) } }));
  mocks.files.mockImplementation(async ({ id }) => ({ data: { resolveTaskFiles: task(id, { stage: TaskStage.Downloading, stageStatus: TaskStageStatus.Running }) } }));
});
afterEach(cleanup);

describe("processing queue", () => {
  it("reviews only attention tasks, oldest first, and advances after retry", async () => {
    setup([task("new", { createdAt: "2026-10-02T00:00:00Z" }), task("old"),
      task("waiting", { stage: TaskStage.Sourcing, stageStatus: TaskStageStatus.Pending, sourcingNextCheckAt: "2026-10-23T00:00:00Z" })]);
    expect(screen.getByRole("heading", { name: "old" })).toBeTruthy();
    expect(screen.getByText("0 handled · 2 remaining")).toBeTruthy();
    expect((screen.getByRole("button", { name: /Previous/ }) as HTMLButtonElement).disabled).toBe(true);
    await userEvent.setup().click(screen.getByRole("button", { name: "Retry current stage" }));
    expect(screen.getByRole("heading", { name: "new" })).toBeTruthy();
    expect(screen.getByText("1 handled · 1 remaining")).toBeTruthy();
    await userEvent.setup().click(screen.getByRole("button", { name: "Retry current stage" }));
    expect(screen.getByRole("heading", { name: "This round is complete" })).toBeTruthy();
    expect(mocks.retry.mock.calls.map(([input]) => input.id)).toEqual(["old", "new"]);
  });

  it("keeps a failed retry on the same task and protects navigation while submitting", async () => {
    let finish: (value: any) => void = () => {};
    mocks.retry.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const { props } = setup([task("one"), task("two")]);
    await userEvent.setup().click(screen.getByRole("button", { name: "Retry current stage" }));
    expect((screen.getByRole("button", { name: /Next/ }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: "Close" }) as HTMLButtonElement).disabled).toBe(true);
    await userEvent.setup().keyboard("{Escape}");
    expect(props.onClose).not.toHaveBeenCalled();
    finish({ error: new Error("still offline") });
    await waitFor(() => expect(screen.getByRole("alert").textContent).toContain("still offline"));
    expect(screen.getByRole("heading", { name: "one" })).toBeTruthy();
    expect((screen.getByRole("button", { name: /Next/ }) as HTMLButtonElement).disabled).toBe(false);
  });

  it("does not count a retry as resolved if the task is still blocked", async () => {
    mocks.retry.mockResolvedValue({ data: { retryTask: task("one") } });
    setup([task("one"), task("two")]);
    await userEvent.setup().click(screen.getByRole("button", { name: "Retry current stage" }));
    expect(screen.getByRole("heading", { name: "one" })).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toContain("still needs attention");
    expect(screen.getByText("0 handled · 2 remaining")).toBeTruthy();
  });

  it("keeps magnet drafts when switching and resolves seeds inside the same panel", async () => {
    setup([task("seed", { stage: TaskStage.Sourcing }), task("two")]);
    fireEvent.change(screen.getByRole("textbox", { name: "Magnet link" }), { target: { value: "magnet:?xt=urn:btih:test" } });
    await userEvent.setup().click(screen.getByRole("button", { name: /Next/ }));
    await userEvent.setup().click(screen.getByRole("button", { name: /Previous/ }));
    expect((screen.getByRole("textbox", { name: "Magnet link" }) as HTMLInputElement).value).toBe("magnet:?xt=urn:btih:test");
    await userEvent.setup().click(screen.getByRole("button", { name: "Use magnet link" }));
    expect(mocks.source).toHaveBeenCalledWith({ id: "seed", input: expect.objectContaining({ torrentUrl: "magnet:?xt=urn:btih:test" }) });
    expect(screen.getByRole("heading", { name: "two" })).toBeTruthy();
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
  });

  it("keeps file drafts when switching and advances after confirmation", async () => {
    setup([task("files", { stage: TaskStage.Downloading, stageErrorCode: "FILE_SELECTION_REQUIRED", files: [
      { index: 0, path: "part-a.mp4", size: 100, kind: "UNKNOWN_VIDEO", selected: false, reason: "IDENTITY_UNCERTAIN", part: 0 },
      { index: 1, path: "part-b.mp4", size: 100, kind: "UNKNOWN_VIDEO", selected: false, reason: "IDENTITY_UNCERTAIN", part: 0 }
    ] }), task("two")]);
    await userEvent.setup().click(screen.getByRole("checkbox", { name: "part-b.mp4" }));
    await userEvent.setup().click(screen.getByRole("button", { name: /Next/ }));
    await userEvent.setup().click(screen.getByRole("button", { name: /Previous/ }));
    expect((screen.getByRole("checkbox", { name: "part-b.mp4" }) as HTMLInputElement).checked).toBe(true);
    await userEvent.setup().click(screen.getByRole("button", { name: "Confirm and continue download" }));
    expect(mocks.files).toHaveBeenCalledWith({ id: "files", videoIndices: [1] });
    expect(screen.getByRole("heading", { name: "two" })).toBeTruthy();
  });

  it("keeps the same task when selecting a seed reveals a file-confirmation step", async () => {
    const seed = task("seed", { stage: TaskStage.Sourcing });
    const review = task("seed", { stage: TaskStage.Downloading, stageErrorCode: "FILE_SELECTION_REQUIRED", files: [
      { index: 0, path: "movie.mp4", size: 100, kind: "UNKNOWN_VIDEO", selected: false, reason: "IDENTITY_UNCERTAIN", part: 0 }
    ] });
    mocks.source.mockImplementation(async () => {
      mocks.details.set("seed", review);
      return { data: { resolveBlockedSourcingTask: review } };
    });
    setup([seed, task("two")]);
    fireEvent.change(screen.getByRole("textbox", { name: "Magnet link" }), { target: { value: "magnet:?xt=urn:btih:test" } });
    await userEvent.setup().click(screen.getByRole("button", { name: "Use magnet link" }));
    expect(await screen.findByRole("checkbox", { name: "movie.mp4" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "seed" })).toBeTruthy();
    expect(screen.getByText("0 handled · 2 remaining")).toBeTruthy();
    await userEvent.setup().click(screen.getByRole("checkbox", { name: "movie.mp4" }));
    await userEvent.setup().click(screen.getByRole("button", { name: "Confirm and continue download" }));
    expect(screen.getByRole("heading", { name: "two" })).toBeTruthy();
  });

  it("removes externally handled or deleted tasks without adding new arrivals to the current round", async () => {
    const one = task("one"), two = task("two");
    const { rerender, props } = setup([one, two]);
    const newArrival = task("three");
    rerender(<TaskProcessingQueueDrawer tasks={[{ ...one, stageStatus: TaskStageStatus.Running }, two, newArrival]} {...props} />);
    expect(await screen.findByRole("heading", { name: "two" })).toBeTruthy();
    expect(screen.getByText("1 handled · 1 remaining")).toBeTruthy();
    rerender(<TaskProcessingQueueDrawer tasks={[newArrival]} {...props} />);
    expect(await screen.findByRole("heading", { name: "This round is complete" })).toBeTruthy();
  });
});
