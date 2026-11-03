// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useIndexerEvents } from "./useIndexerEvents";
import { IndexerEventKind } from "../graphql/generated/graphql";

const state = vi.hoisted(() => ({
  event: { indexerEvents: { streamId: "one", sequence: 0, kind: "READY" } },
  connection: { status: "connected", generation: 1 },
  restart: vi.fn()
}));
vi.mock("urql", () => ({ useSubscription: () => [{ data: state.event }, state.restart] }));
vi.mock("../graphql/client", () => ({
  getGraphQLConnectionSnapshot: () => state.connection,
  subscribeGraphQLConnection: () => () => {}
}));

describe("indexer invalidation recovery", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    state.restart.mockClear();
    state.event = { indexerEvents: { streamId: "one", sequence: 0, kind: "READY" } };
    state.connection = { status: "connected", generation: 1 };
  });
  afterEach(() => { cleanup(); vi.useRealTimers(); });
  it("coalesces bursts without waiting for a continuously busy stream to stop", () => {
    const refresh = vi.fn(); const hook = renderHook(() => useIndexerEvents(refresh));
    act(() => vi.advanceTimersByTime(100));
    state.event = { indexerEvents: { streamId: "one", sequence: 12, kind: "SITE" } }; hook.rerender();
    act(() => vi.advanceTimersByTime(100));
    expect(refresh).toHaveBeenCalledTimes(1);
    state.event = { indexerEvents: { streamId: "one", sequence: 10, kind: "SITE" } }; hook.rerender();
    act(() => vi.advanceTimersByTime(200));
    expect(refresh).toHaveBeenCalledTimes(1);
  });
  it("rereads after a stream reset or a transport reconnect even with the same sequence", () => {
    const refresh = vi.fn(); const hook = renderHook(() => useIndexerEvents(refresh));
    act(() => vi.advanceTimersByTime(200));
    state.event = { indexerEvents: { streamId: "two", sequence: 0, kind: "READY" } }; hook.rerender();
    act(() => vi.advanceTimersByTime(200));
    state.connection = { status: "connected", generation: 2 }; hook.rerender();
    act(() => vi.advanceTimersByTime(200));
    expect(refresh).toHaveBeenCalledTimes(3);
  });
  it("cancels pending work on leaving the page", () => {
    const refresh = vi.fn(); const hook = renderHook(() => useIndexerEvents(refresh));
    hook.unmount(); act(() => vi.advanceTimersByTime(1000));
    expect(refresh).not.toHaveBeenCalled();
  });
  it("keeps unrelated cache notifications out of the source summary refresh", () => {
    const refresh = vi.fn(); const hook = renderHook(() => useIndexerEvents(refresh, [IndexerEventKind.Site]));
    act(() => vi.advanceTimersByTime(200));
    state.event = { indexerEvents: { streamId: "one", sequence: 1, kind: "CACHE" } }; hook.rerender();
    act(() => vi.advanceTimersByTime(200));
    expect(refresh).toHaveBeenCalledTimes(1);
    state.event = { indexerEvents: { streamId: "one", sequence: 2, kind: "SITE" } }; hook.rerender();
    act(() => vi.advanceTimersByTime(200));
    expect(refresh).toHaveBeenCalledTimes(2);
  });
});
