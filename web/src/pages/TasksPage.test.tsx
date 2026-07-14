// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../i18n/i18n";
import type { DashboardTask } from "../utils";
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
    onRefresh: vi.fn(), onSync: vi.fn(), onOpenTask: vi.fn(), onScanTask: vi.fn(), onRetryTask: vi.fn(),
    onResolveTask: vi.fn(), onDeleteTask: vi.fn(), onToggleTaskSelection: vi.fn(), onSelectVisibleTasks: vi.fn(),
    onClearTaskSelection: vi.fn(), onBatchRetry: vi.fn(), onBatchIngest: vi.fn(), onBatchDelete: vi.fn()
  };
  render(<TasksPage
    tasks={[task]}
    metrics={{ active: 1, completed: 0, pendingScans: 0, failed: 0 }}
    taskSearch=""
    taskStatus="all"
    taskSort="createdAt"
    taskGroupOpen={{ attention: true, active: true, ingestPending: true, completed: true }}
    pendingTaskScanId={null}
    pendingTaskRetryId={null}
    pendingTaskDeleteId={null}
    selectedTaskIds={[]}
    batchPending={false}
    refreshing={false}
    syncing={false}
    autoSyncEnabled={false}
    autoSyncIntervalSeconds={0}
    lastRefreshedAt={null}
    {...callbacks}
    {...overrides}
  />);
  return callbacks;
}

describe("TasksPage task controls", () => {
  beforeEach(async () => i18n.changeLanguage("en"));
  afterEach(cleanup);

  it("switches card activation from details to selection in multi-select mode", async () => {
    const user = userEvent.setup();
    const callbacks = renderPage();
    expect(screen.queryByRole("button", { name: "More actions for TASK-1" })).toBeNull();
    expect(screen.getByRole("button", { name: "Delete task: TASK-1" })).toBeTruthy();
    await user.click(screen.getByRole("button", { name: /TASK-1.*Open details/i }));
    expect(callbacks.onOpenTask).toHaveBeenCalledWith("task-1");

    await user.click(screen.getByRole("button", { name: "Enter multi-select mode" }));
    expect(screen.getByRole("region", { name: "Selected task actions" }).textContent).toContain("0 selected");
    expect(screen.queryByRole("checkbox")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Select task: TASK-1" }));
    expect(callbacks.onToggleTaskSelection).toHaveBeenCalledWith("task-1");
    expect(callbacks.onOpenTask).toHaveBeenCalledTimes(1);
  });

  it("opens more actions in a dialog and keeps refresh visibly bordered", async () => {
    const user = userEvent.setup();
    const callbacks = renderPage();
    expect(screen.getByText("Auto sync disabled").classList.contains("task-sync-meta")).toBe(true);
    expect(screen.getByRole("button", { name: "Refresh" }).classList.contains("task-icon-button--bordered")).toBe(true);

    await user.click(screen.getByRole("button", { name: "More task actions" }));
    expect(screen.getByRole("dialog", { name: "More task actions" })).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Sync qBittorrent now" }));
    expect(callbacks.onSync).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
