// @vitest-environment jsdom

import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LogLevel } from "../graphql/generated/graphql";

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

import { evaluateLogSequence, mergeLogEntries, useLogEvents } from "./useLogEvents";

const entry = (sequence: number, message = `entry ${sequence}`) => ({
  sequence,
  time: `2026-07-13T08:00:${String(sequence).padStart(2, "0")}Z`,
  level: LogLevel.Info,
  message
});

describe("log event sequence handling", () => {
  it("uses the first event as a baseline and accepts consecutive events", () => {
    expect(evaluateLogSequence(null, 10)).toEqual({ accept: true, gap: false, nextSequence: 10 });
    expect(evaluateLogSequence(10, 11)).toEqual({ accept: true, gap: false, nextSequence: 11 });
  });

  it("ignores duplicate or old events and detects gaps", () => {
    expect(evaluateLogSequence(10, 10)).toEqual({ accept: false, gap: false, nextSequence: 10 });
    expect(evaluateLogSequence(10, 9)).toEqual({ accept: false, gap: false, nextSequence: 10 });
    expect(evaluateLogSequence(10, 13)).toEqual({ accept: true, gap: true, nextSequence: 13 });
  });
});

describe("mergeLogEntries", () => {
  it("deduplicates by sequence, keeps newest first, and respects the limit", () => {
    expect(mergeLogEntries([entry(2), entry(1)], [entry(3), entry(2, "duplicate")], 2)).toEqual([
      entry(3),
      entry(2, "duplicate")
    ]);
  });
});

describe("useLogEvents", () => {
  beforeEach(() => {
    mocks.subscriptionResult = { data: undefined, error: undefined, fetching: true };
    mocks.connection = { status: "connected", generation: 1, error: null };
    mocks.listeners.clear();
  });

  it("forwards valid entries and requests a resync on a sequence gap", () => {
    const onEvent = vi.fn();
    const onResync = vi.fn();
    const { rerender } = renderHook(() => useLogEvents({ onEvent, onResync }));

    act(() => {
      mocks.subscriptionResult = {
        data: { logEvents: { sequence: 5, entry: entry(5) } },
        error: undefined,
        fetching: false
      };
      rerender();
    });
    act(() => {
      mocks.subscriptionResult = {
        data: { logEvents: { sequence: 7, entry: entry(7) } },
        error: undefined,
        fetching: false
      };
      rerender();
    });

    expect(onEvent).toHaveBeenCalledTimes(2);
    expect(onResync).toHaveBeenCalledTimes(1);
  });

  it("refreshes the snapshot after a connection generation change", () => {
    const onResync = vi.fn();
    renderHook(() => useLogEvents({ onResync }));
    act(() => {
      mocks.connection = { status: "connected", generation: 2, error: null };
      mocks.listeners.forEach((listener) => listener());
    });
    expect(onResync).toHaveBeenCalledTimes(1);
  });

  it("does not forward events while paused", () => {
    const onEvent = vi.fn();
    mocks.subscriptionResult = {
      data: { logEvents: { sequence: 1, entry: entry(1) } },
      error: undefined,
      fetching: false
    };
    renderHook(() => useLogEvents({ pause: true, onEvent }));
    expect(onEvent).not.toHaveBeenCalled();
  });
});
