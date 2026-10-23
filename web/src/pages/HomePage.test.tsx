// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../i18n/i18n";
import { HomePage, type HomePageProps } from "./HomePage";
import type { DashboardTask } from "../utils";

const baseTask = { id: "task-1", source: "MANUAL", code: "DEMO-101", stage: "DOWNLOADING", stageStatus: "RUNNING", progress: .62, files: [], stashSceneIds: [], createdAt: "2026-10-22T12:00:00Z", updatedAt: "2026-10-22T12:00:00Z" } as unknown as DashboardTask;
const overview = { total: 8, attention: 0, active: 1, waiting: 4, completedLast24Hours: 3, attentionTasks: [], activeTasks: [baseTask], recentTasks: [] };
function setup(overrides: Partial<HomePageProps> = {}) {
  const props: HomePageProps = { overview, runtimeSettings: null, runtimeStatus: null, subscription: null, overviewError: null, statusError: null, subscriptionError: null, fetching: false, connectionStatus: "connected", pendingTaskRetryId: null, onRefresh: vi.fn(), onRefreshStatus: vi.fn(), onRefreshSubscription: vi.fn(), onSearch: vi.fn(), onOpenTask: vi.fn(), onConfirmFiles: vi.fn(), onRetryTask: vi.fn(), onResolveTask: vi.fn(), onOpenQueue: vi.fn(), onOpenSettings: vi.fn(), ...overrides };
  const view = render(<MemoryRouter><HomePage {...props} /></MemoryRouter>);
  return { ...props, ...view };
}
beforeEach(async () => { sessionStorage.clear(); await i18n.changeLanguage("en"); });
afterEach(cleanup);

describe("Home workspace", () => {
  it.each(["idle", "connecting", "connected"] as const)("keeps normal %s connection details out of the page", connectionStatus => {
    setup({ connectionStatus });
    expect(screen.queryByText(/Connection lost|Live connection|Updated|Connecting/)).toBeNull();
    expect(screen.queryByRole("button", { name: "Refresh" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
  });

  it("keeps an interruption notice during reconnect and removes it after recovery", async () => {
    const props = setup({ connectionStatus: "disconnected" });
    await userEvent.setup().click(screen.getByRole("button", { name: "Retry" }));
    expect(props.onRefresh).toHaveBeenCalledOnce();
    props.rerender(<MemoryRouter><HomePage {...props} connectionStatus="connecting" /></MemoryRouter>);
    expect(screen.getByText("Connection lost. Reconnecting automatically.")).toBeTruthy();
    props.rerender(<MemoryRouter><HomePage {...props} connectionStatus="connected" /></MemoryRouter>);
    expect(screen.queryByText("Connection lost. Reconnecting automatically.")).toBeNull();
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
  });

  it("routes each metric to its exact task category including the completion window", () => {
    setup();
    const region = screen.getByLabelText("Task overview");
    const links = within(region).getAllByRole("link");
    expect(links.map(link => link.getAttribute("href"))).toEqual(["/tasks?status=failed", "/tasks?status=active", "/tasks?status=waiting", "/tasks?status=completed&completedWithin=24h"]);
    expect(within(screen.getByRole("region", { name: "Needs attention" })).getByText(/Nothing needs/)).toBeTruthy();
    expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("62");
  });

  it("preserves search drafts and sends the selected source without silently creating a task", async () => {
    const props = setup();
    const user = userEvent.setup();
    await user.type(screen.getByRole("textbox", { name: "Search movies or resources" }), "  DEMO-101  ");
    await user.selectOptions(screen.getByRole("combobox"), "jackett");
    await user.click(screen.getByRole("button", { name: "Search" }));
    expect(props.onSearch).toHaveBeenCalledWith("DEMO-101", "jackett");
    expect(sessionStorage.getItem("moji:home-search")).toBe("  DEMO-101  ");
    expect(props.onRetryTask).not.toHaveBeenCalled();
  });

  it("gives file confirmation and source selection their own direct actions", async () => {
    const file = { ...baseTask, stageStatus: "BLOCKED", stageErrorCode: "FILE_SELECTION_REQUIRED" } as DashboardTask;
    const source = { ...baseTask, id: "task-2", code: "DEMO-102", stage: "SOURCING", stageStatus: "BLOCKED" } as DashboardTask;
    const props = setup({ overview: { ...overview, attention: 2, attentionTasks: [file, source] } });
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Select movie files: DEMO-101" }));
    expect(props.onConfirmFiles).toHaveBeenCalledWith("task-1");
    await user.click(screen.getByRole("button", { name: "Select seed manually: DEMO-102" }));
    expect(props.onResolveTask).toHaveBeenCalledWith("task-2");
    await user.click(screen.getByRole("button", { name: "Review queue" }));
    expect(props.onOpenQueue).toHaveBeenCalledOnce();
  });

  it("keeps other modules usable when subscription health fails", () => {
    setup({ subscriptionError: "Subscription snapshot unavailable" });
    expect(screen.getByText("Subscription snapshot unavailable")).toBeTruthy();
    expect(screen.getByText("DEMO-101")).toBeTruthy();
    expect(screen.getByLabelText("Task overview")).toBeTruthy();
  });

  it("does not announce zero work or a successful connection while loading", () => {
    setup({ overview: null, connectionStatus: "disconnected" });
    expect(screen.queryByText(/Nothing needs your attention/)).toBeNull();
    expect(screen.getByText("Connection lost. Reconnecting automatically.")).toBeTruthy();
    expect(within(screen.getByLabelText("Task overview")).getAllByText("—")).toHaveLength(4);
  });

  it("distinguishes a missing configuration from a configured but unavailable dependency", async () => {
    const status = { stash: { configured: true, ready: false }, jackett: { configured: true, ready: true }, qbittorrent: { configured: true, ready: true }, ingest: { configured: true } } as HomePageProps["runtimeStatus"];
    const props = setup({ runtimeStatus: status });
    expect(screen.getByText(/Stash is unavailable/)).toBeTruthy();
    await userEvent.setup().click(screen.getByRole("button", { name: "Check connections" }));
    expect(props.onOpenSettings).toHaveBeenCalledWith("connections");
    expect(screen.queryByText("Finish setup to start processing")).toBeNull();
  });

  it("offers configured milestones in first-run setup and never infers readiness from configuration", () => {
    setup({ runtimeStatus: { stash: { configured: true, ready: false }, jackett: { configured: false, ready: false }, qbittorrent: { configured: false, ready: false }, ingest: { configured: false } } as HomePageProps["runtimeStatus"] });
    expect(screen.getByRole("region", { name: "Finish setup to start processing" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Stash · Configured" })).toBeTruthy();
    expect(screen.getByText("Unavailable")).toBeTruthy();
  });

  it("keeps the slash shortcut out of open task dialogs", () => {
    setup({ interactionOpen: true });
    fireEvent.keyDown(window, { key: "/" });
    expect(document.activeElement).not.toBe(screen.getByRole("textbox"));
  });
});
