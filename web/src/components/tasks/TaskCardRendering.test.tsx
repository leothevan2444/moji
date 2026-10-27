// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import * as utils from "../../utils";
import i18n from "../../i18n/i18n";
import { TaskGroupSection } from "./TaskGroupSection";

const first = { id: "one", code: "ABCD-123", source: "MANUAL", stage: "DOWNLOADING", stageStatus: "RUNNING", progress: .2, createdAt: "2026-10-26T12:00:00Z", updatedAt: "2026-10-26T12:00:00Z" } as utils.DashboardTask;
const second = { ...first, id: "two", code: "ABCD-456" };
beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

it("recomputes only the changed card and still updates memoized cards when locale changes", async () => {
  const summarize = vi.spyOn(utils, "taskCardPresentation");
  const props = { group: "active" as const, tone: "tone-info", description: "", open: true, pendingTaskRetryId: null, pendingTaskDeleteId: null,
    onToggle: vi.fn(), onOpenTask: vi.fn(), onRetryTask: vi.fn(), onResolveTask: vi.fn(), onDeleteTask: vi.fn(),
    selectionMode: false, selectedTaskIds: [], onToggleTaskSelection: vi.fn() };
  const { rerender } = render(<TaskGroupSection {...props} tasks={[first, second]} />);
  summarize.mockClear();
  const changed = { ...first, progress: .4 };
  rerender(<TaskGroupSection {...props} tasks={[changed, second]} />);
  expect(summarize).toHaveBeenCalledTimes(1);
  expect(summarize).toHaveBeenCalledWith(changed);
  expect(screen.getAllByRole("progressbar").map(el => el.getAttribute("aria-valuenow"))).toEqual(["40", "20"]);
  summarize.mockClear();
  rerender(<TaskGroupSection {...props} tasks={[changed, second]} pendingTaskRetryId={first.id} />);
  expect(summarize).toHaveBeenCalledTimes(1);
  summarize.mockClear();
  await act(async () => { await i18n.changeLanguage("zh-CN"); });
  expect(summarize).toHaveBeenCalledWith(second);
});

it("ages relative time without task events or recomputing the card", async () => {
  vi.useFakeTimers();
  try {
    vi.setSystemTime(new Date("2026-10-27T12:00:00Z"));
    const summarize = vi.spyOn(utils, "taskCardPresentation");
    const { TaskCard } = await import("./TaskCard");
    const { container, unmount } = render(<TaskCard task={{ ...first, lastProgressAt: "2026-10-27T11:59:30Z" }} onOpen={vi.fn()} />);
    const before = container.querySelector("time")?.textContent;
    summarize.mockClear();
    act(() => { vi.advanceTimersByTime(60_000); });
    expect(container.querySelector("time")?.textContent).not.toBe(before);
    expect(summarize).not.toHaveBeenCalled();
    unmount();
    expect(vi.getTimerCount()).toBe(0);
  } finally { vi.useRealTimers(); }
});
