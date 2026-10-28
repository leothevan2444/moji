// @vitest-environment jsdom
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
const { query } = vi.hoisted(() => ({ query: vi.fn() }));
vi.mock("../graphql/client", () => ({ settingsClient: { query } }));
import { useInterfaceSettings } from "./useInterfaceSettings";
afterEach(() => { cleanup();vi.clearAllMocks(); });
it("loads the saved mode and prevents an older read undoing a successful save", async () => {
  let finish!: (value: unknown) => void;
  query.mockReturnValue({ toPromise: () => new Promise((resolve) => { finish = resolve; }) });
  const { result } = renderHook(() => useInterfaceSettings());
  act(() => result.current.updateDeveloperMode(true));
  await act(async () => finish({ data: { settings: { system: { developerMode: false } } } }));
  expect(result.current.developerMode).toBe(true);
  query.mockReturnValue({ toPromise: async () => ({ data: { settings: { system: { developerMode: false } } } }) });
  act(() => window.dispatchEvent(new Event("focus")));
  await waitFor(() => expect(result.current.developerMode).toBe(false));
});

it("loads instance language and fences old reads after a successful language save", async () => {
  query.mockReturnValue({ toPromise: async () => ({data:{settings:{system:{developerMode:false,language:"en"}}}}) });
  const { result } = renderHook(() => useInterfaceSettings());
  await waitFor(() => expect(result.current.language).toBe("en"));
  let finish!: (value: unknown) => void;
  query.mockReturnValue({toPromise:()=>new Promise(resolve=>{finish=resolve;})});
  act(()=>window.dispatchEvent(new Event("focus")));
  act(()=>result.current.updateLanguage("zh-CN"));
  await act(async()=>finish({data:{settings:{system:{developerMode:false,language:"en"}}}}));
  expect(result.current.language).toBe("zh-CN");
});
