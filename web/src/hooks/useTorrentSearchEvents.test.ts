// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useTorrentSearchEvents } from "./useTorrentSearchEvents";
const state = vi.hoisted(() => ({ event: { torrentSearchEvents: { streamId: "one", sequence: 1, searchId: "a" } }, connection: { status: "connected", generation: 1 }, restart: vi.fn() }));
vi.mock("urql", () => ({ useSubscription: () => [{ data: state.event }, state.restart] }));
vi.mock("../graphql/client", () => ({ getGraphQLConnectionSnapshot: () => state.connection, subscribeGraphQLConnection: () => () => {} }));
beforeEach(() => { vi.useFakeTimers(); state.restart.mockClear(); state.event = { torrentSearchEvents: { streamId: "one", sequence: 1, searchId: "a" } }; state.connection = { status: "connected", generation: 1 }; });
afterEach(() => { cleanup(); vi.useRealTimers(); });
it("coalesces run updates and rejects backwards notifications", () => {
  const read = vi.fn(); const hook = renderHook(() => useTorrentSearchEvents("a", read));
  act(() => vi.advanceTimersByTime(100)); state.event = { torrentSearchEvents: { streamId: "one", sequence: 10, searchId: "a" } }; hook.rerender();
  act(() => vi.advanceTimersByTime(100)); expect(read).toHaveBeenCalledTimes(1);
  state.event = { torrentSearchEvents: { streamId: "one", sequence: 3, searchId: "a" } }; hook.rerender(); act(() => vi.advanceTimersByTime(200)); expect(read).toHaveBeenCalledTimes(1);
});
it("changing runs drops the old timer and ignores old run notifications", () => {
  const read = vi.fn(); const hook = renderHook(({ id }) => useTorrentSearchEvents(id, read), { initialProps: { id: "a" } });
  hook.rerender({ id: "b" }); act(() => vi.advanceTimersByTime(1000)); expect(read).not.toHaveBeenCalled();
  state.event = { torrentSearchEvents: { streamId: "one", sequence: 2, searchId: "b" } }; hook.rerender({ id: "b" }); act(() => vi.advanceTimersByTime(200)); expect(read).toHaveBeenCalledTimes(1);
  hook.unmount(); act(() => vi.advanceTimersByTime(1000)); expect(read).toHaveBeenCalledTimes(1);
});
it("recovers after stream reset and transport reconnect with bounded reads", () => {
  const read = vi.fn(); const hook = renderHook(() => useTorrentSearchEvents("a", read)); act(() => vi.advanceTimersByTime(200));
  state.event = { torrentSearchEvents: { streamId: "two", sequence: 0, searchId: "a" } }; hook.rerender(); act(() => vi.advanceTimersByTime(200));
  state.connection = { status: "connected", generation: 2 }; hook.rerender(); act(() => vi.advanceTimersByTime(200)); expect(read).toHaveBeenCalledTimes(3);
  act(() => hook.result.current.retry()); expect(state.restart).toHaveBeenCalledTimes(1); act(() => vi.advanceTimersByTime(200)); expect(read).toHaveBeenCalledTimes(4);
});
