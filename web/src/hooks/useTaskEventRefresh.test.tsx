// @vitest-environment jsdom

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { TaskEvent } from "./useTaskEvents";
import { useTaskEventRefresh } from "./useTaskEventRefresh";
import { TaskEventType } from "../graphql/generated/graphql";

function event(taskId: string): TaskEvent {
  return {
    __typename: "TaskEvent",
    sequence: 1,
    type: TaskEventType.Updated,
    taskId,
    task: null,
    dashboardStats: {
      __typename: "DashboardStats",
      total: 1,
      active: 1,
      completed: 0,
      downloading: 1,
      pendingScans: 0,
      failed: 0
    }
  };
}

describe("useTaskEventRefresh", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("coalesces rapid events into one overview refresh", () => {
    const refreshOverview = vi.fn();
    const refreshDetail = vi.fn();
    const { result } = renderHook(() => useTaskEventRefresh({
      taskId: undefined,
      refreshOverview,
      refreshDetail
    }));

    act(() => {
      result.current.onEvent(event("task-1"));
      result.current.onEvent(event("task-2"));
      result.current.onEvent(event("task-3"));
      vi.advanceTimersByTime(200);
    });

    expect(refreshOverview).toHaveBeenCalledTimes(1);
    expect(refreshDetail).not.toHaveBeenCalled();
  });

  it("refreshes detail only for the currently open task", () => {
    const refreshOverview = vi.fn();
    const refreshDetail = vi.fn();
    const { result } = renderHook(() => useTaskEventRefresh({
      taskId: "task-1",
      refreshOverview,
      refreshDetail
    }));

    act(() => {
      result.current.onEvent(event("task-2"));
      vi.advanceTimersByTime(200);
    });
    expect(refreshOverview).toHaveBeenCalledTimes(1);
    expect(refreshDetail).not.toHaveBeenCalled();

    act(() => {
      result.current.onEvent(event("task-1"));
      vi.advanceTimersByTime(200);
    });
    expect(refreshOverview).toHaveBeenCalledTimes(2);
    expect(refreshDetail).toHaveBeenCalledTimes(1);
  });

  it("refreshes overview and open detail for full calibration", () => {
    const refreshOverview = vi.fn();
    const refreshDetail = vi.fn();
    const { result } = renderHook(() => useTaskEventRefresh({ taskId: "task-1", refreshOverview, refreshDetail }));
    act(() => {
      result.current.onFullRefresh();
      vi.advanceTimersByTime(200);
    });
    expect(refreshOverview).toHaveBeenCalledTimes(1);
    expect(refreshDetail).toHaveBeenCalledTimes(1);
  });

  it("clears a pending timer when unmounted", () => {
    const refreshOverview = vi.fn();
    const { result, unmount } = renderHook(() => useTaskEventRefresh({
      taskId: undefined,
      refreshOverview,
      refreshDetail: vi.fn()
    }));
    act(() => result.current.onEvent(event("task-1")));
    unmount();
    act(() => vi.advanceTimersByTime(200));
    expect(refreshOverview).not.toHaveBeenCalled();
  });
});
