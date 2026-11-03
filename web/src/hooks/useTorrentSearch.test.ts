// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useTorrentSearch } from "./useTorrentSearch";
const state = vi.hoisted(() => ({ options: { pause: false }, requery: vi.fn(), read: { data: undefined, fetching: false, error: undefined }, start: vi.fn(), cancel: vi.fn(), queue: vi.fn(), retry: vi.fn() }));
vi.mock("urql", () => ({
  useQuery: (options: { pause: boolean }) => { state.options = options; return [state.read, state.requery]; },
  useMutation: (document: { definitions: Array<{ kind: string; name?: { value: string } }> }) => {
    const name = document.definitions.find((item) => item.kind === "OperationDefinition")?.name?.value;
    return [{ fetching: false }, name === "StartTorrentSearch" ? state.start : name === "CancelTorrentSearch" ? state.cancel : state.queue];
  }
}));
vi.mock("./useTorrentSearchEvents", () => ({ useTorrentSearchEvents: () => ({ error: null, retry: state.retry }) }));
beforeEach(() => { state.start.mockReset(); state.cancel.mockReset(); state.queue.mockReset(); state.requery.mockClear(); state.retry.mockClear(); });
afterEach(cleanup);
it("mount and refresh never implicitly admit another search", () => {
  const hook = renderHook(() => useTorrentSearch(null)); expect(state.options.pause).toBe(true); act(() => hook.result.current.refresh()); expect(state.start).not.toHaveBeenCalled(); expect(state.requery).not.toHaveBeenCalled();
});
it("explicit admission returns the durable ID and prevents duplicate pending starts", async () => {
  let finish!: (value: { data: { startTorrentSearch: { id: string } } }) => void;
  state.start.mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
  const hook = renderHook(() => useTorrentSearch(null)); let pending!: Promise<string | null>;
  act(() => { pending = hook.result.current.start({ keywords: "ABCD-123" }); });
  expect(await hook.result.current.start({ keywords: "ABCD-123" })).toBeNull();
  await act(async () => { finish({ data: { startTorrentSearch: { id: "accepted" } } }); expect(await pending).toBe("accepted"); }); expect(state.start).toHaveBeenCalledTimes(1);
});
it("queue binds the current run on the server and cancellation rereads persisted state", async () => {
  state.queue.mockResolvedValue({ data: { queueTorrentCandidate: { id: "task" } } }); state.cancel.mockResolvedValue({ data: { cancelTorrentSearch: { id: "a" } } });
  const hook = renderHook(() => useTorrentSearch("a"));
  await act(async () => { const task = await hook.result.current.queue({ code: "ABCD-123", candidateId: "candidate", paused: false }); expect(task?.id).toBe("task"); });
  expect(state.queue).toHaveBeenCalledWith({ input: { code: "ABCD-123", candidateId: "candidate", searchId: "a", paused: false } });
  await act(async () => { expect(await hook.result.current.cancel()).toBe(true); }); expect(state.requery).toHaveBeenCalledWith({ requestPolicy: "network-only" });
});
it("late failures cannot replace errors after navigating to another run", async () => {
  let finish!: (value: { error: Error }) => void; state.cancel.mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
  const hook = renderHook(({ id }) => useTorrentSearch(id), { initialProps: { id: "a" } }); let pending!: Promise<boolean>;
  act(() => { pending = hook.result.current.cancel(); }); hook.rerender({ id: "b" });
  await act(async () => { finish({ error: new Error("old run") }); expect(await pending).toBe(false); }); expect(hook.result.current.error).toBeNull(); expect(state.requery).not.toHaveBeenCalled();
});
