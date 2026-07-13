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

vi.mock("urql", () => ({ useSubscription: () => [mocks.subscriptionResult] }));
vi.mock("../graphql/client", () => ({
  getGraphQLConnectionSnapshot: () => mocks.connection,
  subscribeGraphQLConnection: (listener: () => void) => {
    mocks.listeners.add(listener);
    return () => mocks.listeners.delete(listener);
  }
}));

import { evaluatePerformerSubscriptionSequence, usePerformerSubscriptionEvents } from "./usePerformerSubscriptionEvents";

describe("evaluatePerformerSubscriptionSequence", () => {
  it("handles baselines, duplicates and gaps", () => {
    expect(evaluatePerformerSubscriptionSequence(null, 4)).toEqual({ accept: true, gap: false, nextSequence: 4 });
    expect(evaluatePerformerSubscriptionSequence(4, 5)).toEqual({ accept: true, gap: false, nextSequence: 5 });
    expect(evaluatePerformerSubscriptionSequence(5, 5)).toEqual({ accept: false, gap: false, nextSequence: 5 });
    expect(evaluatePerformerSubscriptionSequence(5, 8)).toEqual({ accept: true, gap: true, nextSequence: 8 });
  });
});

describe("usePerformerSubscriptionEvents", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.subscriptionResult = { data: undefined, error: undefined, fetching: true };
    mocks.connection = { status: "connected", generation: 1, error: null };
    mocks.listeners.clear();
  });
  afterEach(() => vi.useRealTimers());

  it("calibrates on initial connection and coalesces a sequence gap", () => {
    const onRefresh = vi.fn();
    const { rerender } = renderHook(() => usePerformerSubscriptionEvents({ enabled: true, onRefresh }));
    act(() => {
      mocks.subscriptionResult = { data: { performerSubscriptionEvents: { sequence: 3 } }, error: undefined, fetching: false };
      rerender();
      mocks.subscriptionResult = { data: { performerSubscriptionEvents: { sequence: 6 } }, error: undefined, fetching: false };
      rerender();
      vi.advanceTimersByTime(200);
    });
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("refreshes after reconnect and clears timers on unmount", () => {
    const onRefresh = vi.fn();
    const view = renderHook(() => usePerformerSubscriptionEvents({ enabled: true, onRefresh }));
    act(() => vi.advanceTimersByTime(200));
    expect(onRefresh).toHaveBeenCalledTimes(1);
    act(() => {
      mocks.connection = { status: "connected", generation: 2, error: null };
      mocks.listeners.forEach((listener) => listener());
    });
    view.unmount();
    act(() => vi.advanceTimersByTime(200));
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });
});
