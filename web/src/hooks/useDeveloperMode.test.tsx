// @vitest-environment jsdom
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
const { query } = vi.hoisted(() => ({ query: vi.fn() }));
vi.mock("../graphql/client", () => ({ settingsClient: { query } }));
import { useDeveloperMode } from "./useDeveloperMode";
afterEach(() => { cleanup();vi.clearAllMocks(); });
it("loads the saved mode and prevents an older read undoing a successful save", async () => {
  let finish!: (value: unknown) => void;
  query.mockReturnValue({ toPromise: () => new Promise((resolve) => { finish = resolve; }) });
  const { result } = renderHook(() => useDeveloperMode());
  act(() => result.current.update(true));
  await act(async () => finish({ data: { settings: { system: { developerMode: false } } } }));
  expect(result.current.enabled).toBe(true);
  query.mockReturnValue({ toPromise: async () => ({ data: { settings: { system: { developerMode: false } } } }) });
  act(() => window.dispatchEvent(new Event("focus")));
  await waitFor(() => expect(result.current.enabled).toBe(false));
});
