// @vitest-environment jsdom

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  subscriptionResult: { data: undefined as unknown, error: undefined as unknown, fetching: true },
  connection: { status: "connected", generation: 1, error: null } as {
    status: "connected" | "disconnected" | "error";
    generation: number;
    error: unknown;
  },
  listeners: new Set<() => void>()
}));

vi.mock("urql", () => ({
  useSubscription: () => [mocks.subscriptionResult]
}));

vi.mock("../graphql/client", () => ({
  getGraphQLConnectionSnapshot: () => mocks.connection,
  subscribeGraphQLConnection: (listener: () => void) => {
    mocks.listeners.add(listener);
    return () => mocks.listeners.delete(listener);
  }
}));

import { evaluateServiceStatusSequence, useServiceStatusEvents } from "./useServiceStatusEvents";

describe("evaluateServiceStatusSequence", () => {
  it("establishes the first sequence as a baseline", () => {
    expect(evaluateServiceStatusSequence(null, 10)).toEqual({ accept: true, gap: false, nextSequence: 10 });
  });

  it("accepts consecutive sequences", () => {
    expect(evaluateServiceStatusSequence(10, 11)).toEqual({ accept: true, gap: false, nextSequence: 11 });
  });

  it("ignores duplicate and old sequences", () => {
    expect(evaluateServiceStatusSequence(10, 10)).toEqual({ accept: false, gap: false, nextSequence: 10 });
    expect(evaluateServiceStatusSequence(10, 9)).toEqual({ accept: false, gap: false, nextSequence: 10 });
  });

  it("reports a gap for a newer non-consecutive sequence", () => {
    expect(evaluateServiceStatusSequence(10, 13)).toEqual({ accept: true, gap: true, nextSequence: 13 });
  });
});

describe("useServiceStatusEvents", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.subscriptionResult = { data: undefined, error: undefined, fetching: true };
    mocks.connection = { status: "connected", generation: 1, error: null };
    mocks.listeners.clear();
  });

  afterEach(() => vi.useRealTimers());

  it("coalesces rapid valid events into one refresh", () => {
    const onRefresh = vi.fn();
    const { rerender } = renderHook(() => useServiceStatusEvents({ onRefresh }));

    act(() => {
      mocks.subscriptionResult = {
        data: { serviceStatusEvents: { sequence: 1, services: ["QBITTORRENT"], observedAt: "2026-07-13T08:00:00Z" } },
        error: undefined,
        fetching: false
      };
      rerender();
    });
    act(() => {
      mocks.subscriptionResult = {
        data: { serviceStatusEvents: { sequence: 2, services: ["STASH"], observedAt: "2026-07-13T08:00:01Z" } },
        error: undefined,
        fetching: false
      };
      rerender();
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("refreshes after a connection generation change", () => {
    const onRefresh = vi.fn();
    renderHook(() => useServiceStatusEvents({ onRefresh }));
    act(() => {
      mocks.connection = { status: "connected", generation: 2, error: null };
      mocks.listeners.forEach((listener) => listener());
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("clears a pending refresh when unmounted", () => {
    const onRefresh = vi.fn();
    const { rerender, unmount } = renderHook(() => useServiceStatusEvents({ onRefresh }));
    act(() => {
      mocks.subscriptionResult = {
        data: { serviceStatusEvents: { sequence: 1, services: ["JACKETT"], observedAt: "2026-07-13T08:00:00Z" } },
        error: undefined,
        fetching: false
      };
      rerender();
    });
    unmount();
    act(() => vi.advanceTimersByTime(200));
    expect(onRefresh).not.toHaveBeenCalled();
  });
});
