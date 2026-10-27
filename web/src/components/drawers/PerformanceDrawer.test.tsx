// @vitest-environment jsdom
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { restart, subscribe, state } = vi.hoisted(() => ({ restart: vi.fn(), subscribe: vi.fn(), state: { current: {} as Record<string, unknown>, connection: { status: "connected" } } }));
vi.mock("urql", () => ({ useSubscription: (options: unknown) => { subscribe(options); return [state.current, restart]; } }));
vi.mock("../../graphql/client", () => ({ getGraphQLConnectionSnapshot: () => state.connection, subscribeGraphQLConnection: () => () => {} }));
vi.mock("react-i18next", async (importOriginal) => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: (key: string) => key, i18n: { resolvedLanguage: "en" } }) }));
import PerformanceDrawer from "./PerformanceDrawer";

beforeEach(() => {
  vi.useFakeTimers();restart.mockClear();subscribe.mockClear();state.connection = { status: "connected" };
  vi.spyOn(document, "visibilityState", "get").mockReturnValue("visible");
  const pool = { open: 2, inUse: 1, idle: 1, limit: 4, waitCount: 7, waitMilliseconds: 18 };
  state.current = { fetching: false, data: { databasePerformance: { startedAt: new Date().toISOString(), sampledAt: new Date().toISOString(), intervalSeconds: 5, lastError: "", databases: [{ role: "core", databaseBytes: 4096, walBytes: 1024, freeBytes: 0, writeAttempts: 20, writeErrors: 0, busyRetries: 3, queueRejected: 0, queueCanceled: 0, queued: 1, queueCapacity: 64, writesPerSecond: 2, averageWaitMilliseconds: 4, averageExecutionMilliseconds: 8, totalWaitMilliseconds: 40, totalExecutionMilliseconds: 80, reader: pool, writer: { ...pool, limit: 1 } }] } } };
});
afterEach(() => { cleanup();vi.useRealTimers();vi.restoreAllMocks(); });

describe("database performance drawer", () => {
  it("uses a subscription without polling and pauses when hidden", () => {
    const interval = vi.spyOn(window, "setInterval");
    const view = render(<PerformanceDrawer />);
    expect(screen.getByText("performance.roles.core")).toBeTruthy();
    const options = subscribe.mock.calls.at(-1)?.[0];
    expect(options.query.definitions[0].operation).toBe("subscription");
    expect(options.pause).toBe(false);
    expect(screen.queryByText("performance.sampledAt")).toBeNull();
    act(() => { vi.advanceTimersByTime(20000); });
    expect(restart).not.toHaveBeenCalled();
    expect(interval).not.toHaveBeenCalled();
    vi.spyOn(document, "visibilityState", "get").mockReturnValue("hidden");
    act(() => { document.dispatchEvent(new Event("visibilitychange")); });
    expect(subscribe.mock.calls.at(-1)?.[0].pause).toBe(true);
    vi.spyOn(document, "visibilityState", "get").mockReturnValue("visible");
    act(() => { document.dispatchEvent(new Event("visibilitychange")); });
    expect(subscribe.mock.calls.at(-1)?.[0].pause).toBe(false);
    view.unmount();
    expect(vi.getTimerCount()).toBe(0);
  });
  it("renders pushed values and retains them while disconnected", () => {
    const view = render(<PerformanceDrawer />);
    const data = state.current.data as { databasePerformance: { databases: Array<{ writesPerSecond: number }> } };
    data.databasePerformance.databases[0].writesPerSecond = 77;
    view.rerender(<PerformanceDrawer />);
    expect(screen.getByText("77")).toBeTruthy();
    state.connection = { status: "disconnected" };
    view.rerender(<PerformanceDrawer />);
    expect(screen.getByText("performance.stale")).toBeTruthy();
    expect(screen.getByText("77")).toBeTruthy();
  });
  it("retains last metrics and marks failed or stale samples", () => {
    const view = render(<PerformanceDrawer />);
    expect(screen.queryByRole("status")).toBeNull();
    act(() => { vi.advanceTimersByTime(20000); });
    expect(screen.getByText("performance.stale")).toBeTruthy();
    const data = state.current.data as { databasePerformance: { lastError: string } };
    data.databasePerformance.lastError = "sample failed";
    view.rerender(<PerformanceDrawer />);
    expect(screen.getByRole("alert").textContent).toContain("sample failed");
    expect(screen.getByText("performance.roles.core")).toBeTruthy();
  });
});
