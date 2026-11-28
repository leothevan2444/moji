// @vitest-environment jsdom

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../i18n/i18n";
import type { DashboardTask } from "../utils";
import { TaskStage, TaskStageStatus } from "../graphql/generated/graphql";
import { TasksPage } from "./TasksPage";

const task = {
  id: "task-1",
  source: "MANUAL",
  code: "TASK-1",
  stage: "DOWNLOADING",
  stageStatus: "RUNNING",
  stageLabel: "Downloading",
  stageStatusLabel: "Running",
  progress: 0.25,
  createdAt: "2026-07-14T00:00:00Z",
  updatedAt: "2026-07-14T00:00:00Z"
} as DashboardTask;

function renderPage(overrides: Partial<React.ComponentProps<typeof TasksPage>> = {}) {
  const callbacks = {
    onSearchChange: vi.fn(), onStatusChange: vi.fn(), onSortChange: vi.fn(), onToggleGroup: vi.fn(),
    onRefresh: vi.fn(), onOpenQueue: vi.fn(), onOpenTask: vi.fn(), onRetryTask: vi.fn(),
    onResolveTask: vi.fn(), onDeleteTask: vi.fn(), onToggleTaskSelection: vi.fn(), onSelectVisibleTasks: vi.fn(),
    onClearTaskSelection: vi.fn(), onBatchRetry: vi.fn(), onBatchDelete: vi.fn()
  };
  render(<TasksPage
    tasks={[task]}
    taskSearch=""
    taskStatus="all"
    taskSort="createdAt"
    taskGroupOpen={{ attention: true, active: true, waiting: true, completed: true }}
    pendingTaskRetryId={null}
    pendingTaskDeleteId={null}
    selectedTaskIds={[]}
    batchPending={false}
    refreshing={false}
    lastRefreshedAt={null}
    {...callbacks}
    {...overrides}
  />);
  return callbacks;
}

describe("TasksPage task controls", () => {
  beforeEach(async () => i18n.changeLanguage("en"));
  afterEach(cleanup);

  it("matches the homepage rolling completion count using verified completion time", () => {
    const now = Date.now();
    const completed = (id: string, hours: number, stage = TaskStage.Completed) => ({ ...task, id, code: id, stage, stageStatus: TaskStageStatus.Done, ingestCompletedAt: new Date(now - hours * 3600000).toISOString() });
    renderPage({ taskStatus: "completed", completedWithin24Hours: true, tasks: [completed("RECENT", 1), completed("OLD", 25), { ...completed("NO-MILESTONE", 1), ingestCompletedAt: null }, completed("FUTURE", -1), completed("DOWNLOAD-ONLY", 1, TaskStage.PendingIngest)] });
    expect(screen.getByText("RECENT")).toBeTruthy();
    for (const label of ["OLD", "NO-MILESTONE", "FUTURE", "DOWNLOAD-ONLY"]) expect(screen.queryByText(label)).toBeNull();
    expect(screen.getByText(/Completed in the last 24 hours/)).toBeTruthy();
  });

  it("switches card activation from details to selection in multi-select mode", async () => {
    const user = userEvent.setup();
    const callbacks = renderPage();
    expect(screen.queryByLabelText("More actions for TASK-1")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Delete task: TASK-1" }));
    expect(callbacks.onDeleteTask).toHaveBeenCalledWith("task-1");
    expect(callbacks.onOpenTask).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: /TASK-1.*Open details/i }));
    expect(callbacks.onOpenTask).toHaveBeenCalledWith("task-1");

    await user.click(screen.getByRole("button", { name: "Enter multi-select mode" }));
    expect(screen.getByRole("region", { name: "Selected task actions" }).textContent).toContain("0 selected");
    expect(screen.getByRole("checkbox", { name: "Select task: TASK-1" })).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Select task: TASK-1" }));
    expect(callbacks.onToggleTaskSelection).toHaveBeenCalledWith("task-1");
    expect(callbacks.onOpenTask).toHaveBeenCalledTimes(1);
  });

  it("offers refresh without a manual task-processing button", () => {
    renderPage();
    expect(screen.queryByText("Check tasks automatically every 60 seconds")).toBeNull();
    expect(screen.queryByText(/processing is paused/)).toBeNull();
    expect(screen.getByRole("button", { name: "Refresh" }).classList.contains("workspace-toolbar__action")).toBe(true);
    expect(screen.queryByRole("button", { name: "Sync now" })).toBeNull();
    expect(screen.queryByRole("button", { name: "More task actions" })).toBeNull();
    expect(screen.queryByRole("button", { name: /Process.*ingest/ })).toBeNull();
  });

  it("shows pending ingest with other active tasks without a manual ingest action", async () => {
    renderPage({ taskStatus: "running", tasks: [{ ...task, stage: TaskStage.PendingIngest, stageStatus: TaskStageStatus.Pending }] });
    expect(screen.getByRole("button", { name: /TASK-1.*Open details/i })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "In progress" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Process ingest/ })).toBeNull();
    await userEvent.setup().click(screen.getByRole("button", { name: "Enter multi-select mode" }));
    expect(screen.getByRole("region", { name: "Selected task actions" }).textContent).not.toContain("Ingest");
  });

  it("keeps waiting resources out of in-progress results and offers a dedicated category", () => {
    renderPage({ taskStatus: "running", tasks: [{ ...task, stage: TaskStage.Sourcing, stageStatus: TaskStageStatus.Pending, sourcingNextCheckAt: "2026-10-23T00:00:00Z" }] });
    expect(screen.queryByRole("button", { name: /TASK-1.*Open details/i })).toBeNull();
    const filters = within(screen.getByRole("toolbar", { name: "Filter tasks by status" }));
    expect(filters.getByRole("button", { name: "Waiting for resources 1" })).toBeTruthy();
    expect(filters.getByRole("button", { name: "In progress 0" }).getAttribute("aria-pressed")).toBe("true");
    expect(screen.queryByRole("combobox", { name: "Task category" })).toBeNull();
  });

  it("opens a queue for all attention tasks even when the list uses another category", async () => {
    const callbacks = renderPage({ taskStatus: "completed", tasks: [{ ...task, stageStatus: TaskStageStatus.Blocked }] });
    await userEvent.setup().click(screen.getByRole("button", { name: "Open task workbench (1 tasks)" }));
    expect(callbacks.onOpenQueue).toHaveBeenCalledTimes(1);
  });

  it("finds tasks by torrent name while displaying the code as their title", () => {
    renderPage({ taskSearch: "Release title", tasks: [{ ...task, torrentName: "Release title 1080p" }] });
    expect(screen.getByRole("button", { name: /TASK-1.*Open details/i })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Release title 1080p" })).toBeNull();
  });

  it("keeps counted filters independent of the selected category and the global workbench", async () => {
    const callbacks = renderPage({ taskStatus: "completed", taskSearch: "match", tasks: [
      {...task, id:"active", code:"MATCH-1"},
      {...task, id:"done", code:"MATCH-2", stage:TaskStage.Completed, stageStatus:TaskStageStatus.Done},
      {...task, id:"attention", code:"OTHER", stageStatus:TaskStageStatus.Blocked},
    ] });
    const filters = within(screen.getByRole("toolbar", {name:"Filter tasks by status"}));
    expect(filters.getByRole("button", {name:"All tasks 2"})).toBeTruthy();
    expect(filters.getByRole("button", {name:"In progress 1"})).toBeTruthy();
    const selected = filters.getByRole("button", {name:"Completed 1"});
    expect(selected.getAttribute("aria-pressed")).toBe("true");
    await userEvent.setup().click(selected);
    expect(callbacks.onStatusChange).not.toHaveBeenCalled();
    await userEvent.setup().click(filters.getByRole("button", {name:"Needs attention 0"}));
    expect(callbacks.onStatusChange).toHaveBeenCalledWith("failed");
    expect(screen.getByRole("button", {name:"Open task workbench (1 tasks)"})).toBeTruthy();
  });

  it("uses server counts across all pages instead of counting the visible cards", () => {
    renderPage({serverPage:{page:2,pageSize:24,totalPages:5,totalCount:100,
      stats:{total:100,active:100,completed:0,attention:2,waiting:0},
      filterCounts:{total:100,active:98,completed:0,attention:2,waiting:0}}});
    const filters = within(screen.getByRole("toolbar", {name:"Filter tasks by status"}));
    expect(filters.getByRole("button", {name:"All tasks 100"})).toBeTruthy();
    expect(filters.getByRole("button", {name:"In progress 98"})).toBeTruthy();
  });

  it("shows unknown counts until the first server response rather than fake zeroes", () => {
    renderPage({tasks:[],serverPage:null,refreshing:true});
    const group=within(screen.getByRole("toolbar",{name:"Filter tasks by status"}));
    expect(group.getByRole("button",{name:"All tasks —"})).toBeTruthy();
    expect(group.queryByRole("button",{name:"All tasks 0"})).toBeNull();
  });

});
