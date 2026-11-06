// @vitest-environment jsdom
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TestIndexerMutationVariables } from "../graphql/generated/graphql";
import { useIndexerConnections } from "./useIndexerConnections";

const mock = vi.hoisted(() => ({ execute: vi.fn() }));
vi.mock("urql", () => ({
  useClient: () => ({
    mutation: (_document: unknown, variables: TestIndexerMutationVariables, options: { fetchOptions: { signal: AbortSignal } }) => ({
      toPromise: () => mock.execute(variables, options.fetchOptions.signal)
    })
  }),
  useSubscription: () => [{ data: undefined }]
}));
const passed = { data: { testIndexer: { stages: [{ name: "CONNECTION", state: "PASSED", errorCode: "" }] } } };
beforeEach(() => { mock.execute.mockReset(); });
afterEach(cleanup);

it("keeps an unchanged account running and sends its connection revision across unrelated edits", async () => {
  let finish!: (response: unknown) => void;
  mock.execute.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
  const view = renderHook(({ revision }) => useIndexerConnections(revision, "sites", { one: "same-account" }), { initialProps: { revision: "old-config" } });
  act(() => { void view.result.current.run(["one"]); });
  view.rerender({ revision: "new-config" });
  expect((mock.execute.mock.calls[0][1] as AbortSignal).aborted).toBe(false);
  await act(async () => finish(passed));
  await waitFor(() => expect(view.result.current.results.one.state).toBe("passed"));
  expect(mock.execute).toHaveBeenCalledTimes(1);
  expect(mock.execute.mock.calls[0][0].input.expectedConnectionRevision).toBe("same-account");
});

it("a cancelled request cannot replace a newer test when an account returns to its old configuration", async () => {
  const finish: ((response: unknown) => void)[] = [];
  mock.execute.mockImplementation(() => new Promise(resolve => finish.push(resolve)));
  const view = renderHook(({ version }) => useIndexerConnections("config", "sites", { one: version }), { initialProps: { version: "a" } });
  act(() => { void view.result.current.run(["one"]); });
  const oldSignal = mock.execute.mock.calls[0][1] as AbortSignal;
  view.rerender({ version: "b" });
  expect(oldSignal.aborted).toBe(true);
  view.rerender({ version: "a" });
  act(() => { void view.result.current.run(["one"]); });
  await act(async () => finish[0](passed));
  expect(view.result.current.results.one.state).toBe("pending");
  await act(async () => finish[1]({ data: { testIndexer: { stages: [{ name: "CONNECTION", state: "FAILED", errorCode: "INDEXER_TIMEOUT" }] } } }));
  await waitFor(() => expect(view.result.current.results.one.state).toBe("failed"));
});
