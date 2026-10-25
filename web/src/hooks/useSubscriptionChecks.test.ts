// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ result: {} as Record<string, unknown>, refresh: vi.fn(), start: vi.fn() }));
vi.mock("urql", () => ({ useQuery: () => [mocks.result, mocks.refresh], useMutation: () => [{ fetching: false }, mocks.start] }));
import { useSubscriptionChecks } from "./useSubscriptionChecks";

describe("subscription check snapshots", () => {
  beforeEach(() => { vi.useFakeTimers(); mocks.refresh.mockClear(); mocks.start.mockReset(); mocks.result = { fetching: true }; });
  afterEach(() => { cleanup(); vi.useRealTimers(); });
  it("loads a fresh snapshot on mount and polls faster while a run is active", () => {
    const view = renderHook(() => useSubscriptionChecks(true));
    expect(view.result.current.checksLoading).toBe(true);
    mocks.result = { data: { subscriptionCheckActivity: { activeCount: 0, allRunId: null, performers: [] } } };
    view.rerender();
    act(() => vi.advanceTimersByTime(10000));
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
    mocks.result = { data: { subscriptionCheckActivity: { activeCount: 1, allRunId: "running", performers: [] } } };
    view.rerender();
    act(() => vi.advanceTimersByTime(2000));
    expect(mocks.refresh).toHaveBeenCalledTimes(2);
    act(() => window.dispatchEvent(new Event("focus")));
    expect(mocks.refresh).toHaveBeenCalledTimes(3);
  });
  it("refreshes backend state after submission and returns the authoritative run ID", async () => {
    mocks.start.mockResolvedValue({ data: { startSubscriptionCheck: { id: "check-1" } } });
    const view = renderHook(() => useSubscriptionChecks(true));
    let result;
    await act(async () => { result = await view.result.current.startCheck(["p1"]); });
    expect(mocks.start).toHaveBeenCalledWith({ ids: ["p1"] });
    expect(mocks.refresh).toHaveBeenCalledWith({ requestPolicy: "network-only" });
    expect(result).toEqual({ error: undefined, run: { id: "check-1" } });
  });
});
