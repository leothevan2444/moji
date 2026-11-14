// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../i18n/i18n";
import { HomePage, type HomePageProps } from "./HomePage";
import { TaskDeletePolicy } from "../graphql/generated/graphql";
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

  it.each(["jackett", "indexers"])("restores saved %s resource mode as unified indexers", async stored => {
    sessionStorage.setItem("moji:home-source", stored);
    const props = setup();
    const user = userEvent.setup();
    expect((screen.getByRole("combobox") as HTMLSelectElement).value).toBe("indexers");
    await user.type(screen.getByRole("textbox", { name: "Search movies or resources" }), "DEMO-101");
    await user.click(screen.getByRole("button", { name: "Search" }));
    expect(props.onSearch).toHaveBeenCalledWith("DEMO-101", "indexers");
  });

  it("preserves search drafts and sends the selected source without silently creating a task", async () => {
    const props = setup();
    const user = userEvent.setup();
    await user.type(screen.getByRole("textbox", { name: "Search movies or resources" }), "  DEMO-101  ");
    await user.selectOptions(screen.getByRole("combobox"), "indexers");
    await user.click(screen.getByRole("button", { name: "Search" }));
    expect(props.onSearch).toHaveBeenCalledWith("DEMO-101", "indexers");
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
    const status = { stash: { configured: true, ready: false }, qbittorrent: { configured: true, ready: true }, ingest: { configured: true }, stashStats: {lastError:"connection refused", pendingMojiScanCount:0} } satisfies NonNullable<HomePageProps["runtimeStatus"]>;
    setup({ runtimeStatus: status });
    expect(screen.getByText(/Stash scanning, identification and ingest verification are affected/)).toBeTruthy();
    expect(within(screen.getByRole("region", { name: "Stash" })).getByRole("link", { name: "Stash settings" }).getAttribute("href")).toBe("/settings/stash?panel=connection");
    expect(screen.queryByText("Finish setup to start processing")).toBeNull();
  });

  it("offers configured milestones in first-run setup and never infers readiness from configuration", () => {
    setup({ runtimeStatus: { stash: { configured: true, ready: false }, qbittorrent: { configured: false, ready: false }, ingest: { configured: false }, stashStats: {lastError:"connection refused", pendingMojiScanCount:0} } satisfies NonNullable<HomePageProps["runtimeStatus"]> });
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

describe("Home release checks", () => {
  const run = {
    id: "check-home", trigger: "SCHEDULED", scope: "ALL", status: "RUNNING", startedAt: "2026-01-15T11:55:00Z", finishedAt: null,
    totalCount: 10, completedCount: 4, succeededCount: 3, failedCount: 1, skippedCount: 0,
    currentPerformers: [{ performerId: "p1", name: "Alpha" }]
  } as NonNullable<HomePageProps["checks"]>["activeRuns"][number];
  const checks = { enabled: true, nextRunAt: "2026-01-15T18:00:00Z", activeCount: 1, latestScheduled: run, latestFinishedScheduled: null, statistics: { runCount: 7, succeededCount: 81, failedCount: 5, skippedCount: 2 }, activeRuns: [run], recentRuns: [] };

  it("shows schedule and true progress and opens a run through Home's detail action", async () => {
    const onOpenCheck = vi.fn();
    setup({ checks, onOpenCheck });
    const region = screen.getByRole("region", { name: /Release checks/ });
    expect(within(region).queryByText("Latest scheduled run")).toBeNull();
    expect(within(region).getByText("Next scheduled run")).toBeTruthy();
    expect(within(region).getByText("Checked 4 / 10 performers")).toBeTruthy();
    expect(within(region).getByText("Checking: Alpha")).toBeTruthy();
    expect(within(region).getByRole("progressbar").getAttribute("aria-valuenow")).toBe("4");
    expect(within(region).queryByText("Run ID")).toBeNull();
    await userEvent.setup().click(within(region).getByRole("button", { name: /View Scheduled check/ }));
    expect(onOpenCheck).toHaveBeenCalledWith("check-home");
  });

  it.each(["SUCCEEDED", "FAILED", "PARTIAL", "INTERRUPTED"])("retains the %s scheduled result while a new run is active", status => {
    const finished = { ...run, id: "older-scheduled", status, finishedAt: "2026-01-15T11:50:00Z" } as typeof run;
    setup({ checks: { ...checks, latestFinishedScheduled: finished } });
    const history = within(screen.getByRole("region", { name: /Release checks/ })).getByRole("button", { name: /^History/ });
    expect(history.classList.contains(status === "SUCCEEDED" ? "home-success" : "home-warning")).toBe(true);
    expect(history.getAttribute("aria-label")).toContain(i18n.t(`subscriptionChecks.states.${status}`));
    expect(history.title).toContain("Latest scheduled result");
  });

  it("uses a neutral history button when no finished scheduled result exists", () => {
    setup({ checks });
    const history = within(screen.getByRole("region", { name: /Release checks/ })).getByRole("button", { name: /^History/ });
    expect(history.classList.contains("home-success")).toBe(false);
    expect(history.classList.contains("home-warning")).toBe(false);
    expect(history.title).toBe("No finished scheduled checks recorded");
  });

  it("isolates failed check reads from task lists and retries only that module", async () => {
    const onRefreshChecks = vi.fn();
    setup({ checksError: "Check snapshot unavailable", onRefreshChecks });
    const region = screen.getByRole("region", { name: /Release checks/ });
    expect(within(region).getByText("Check snapshot unavailable")).toBeTruthy();
    expect(screen.getByText("DEMO-101")).toBeTruthy();
    await userEvent.setup().click(within(region).getByRole("button", { name: "Retry" }));
    expect(onRefreshChecks).toHaveBeenCalledOnce();
  });

  it("shows the latest finished scheduled result on History and renders backend statistics", () => {
    const finished = { ...run, status: "PARTIAL", failedCount: 1 } as typeof run;
    setup({ checks: { ...checks, activeCount: 0, activeRuns: [], latestScheduled: run, latestFinishedScheduled: finished, recentRuns: [finished] } });
    const region = screen.getByRole("region", { name: /Release checks/ });
    expect(within(region).getByText("Idle")).toBeTruthy();
    expect(region.querySelector("details")).toBeNull();
    expect(within(region).queryByLabelText("Recent runs")).toBeNull();
    const history = within(region).getByRole("button", { name: /^History/ });
    expect(history.title).toContain("Partially failed");
    expect(history.classList.contains("home-warning")).toBe(true);
    const statistics = region.querySelector(".home-check-statistics")!;
    expect(statistics.textContent).toContain("Succeeded81");
    expect(statistics.textContent).toContain("Issues5");
    expect(statistics.textContent).toContain("Skipped2");
    expect(statistics.getAttribute("title")).toContain("7 recent retained finished runs");
    expect(within(region).queryByText("Succeeded 3 · Failed 1 · Skipped 0")).toBeNull();
    expect(within(region).getByRole("button", {name:/^History/})).toBeTruthy();
  });
});

it("keeps the large subscription total without subscription issue information", () => {
  setup({ subscription: { total: 128, attention: 12, unchecked: 3, lastCheckedAt: "2026-01-15T12:00:00Z", attentionPerformerIds: ["actor/1", "actor2"] } });
  const region = screen.getByRole("region", { name: /My subscriptions/ });
  expect(region.querySelector(".home-big-number")?.textContent).toContain("128");
  expect(within(region).getByRole("link", { name: "Manage performers & subscriptions" }).getAttribute("href")).toBe("/performers");
  expect(region.querySelector("details")).toBeNull();
  expect(within(region).queryByText(/subscription checks need attention/)).toBeNull();
  expect(within(region).queryByRole("link", { name: /Review subscription issue/ })).toBeNull();
});


describe("Home Stash overview", () => {
  const status = { stash: { configured: true, ready: true }, qbittorrent: { configured: true, ready: true }, ingest: { configured: true }, stashStats: { version: "0.28.0", sceneCount: 1234, pendingMojiScanCount: 2, okAt: "2026-11-14T00:00:00Z", lastError: null } } satisfies NonNullable<HomePageProps["runtimeStatus"]>;
  const settings = { system: { taskDeletePolicy: TaskDeletePolicy.KeepOnly }, stash: { url: "http://stash.invalid" } } satisfies NonNullable<HomePageProps["runtimeSettings"]>;

  it("shows only Stash metrics with connection settings and a safe external link", () => {
    setup({ runtimeStatus: status, runtimeSettings: settings });
    const region = screen.getByRole("region", { name: "Stash" });
    expect(within(region).getByText("Connected")).toBeTruthy();
    expect(within(region).getByText("0.28.0")).toBeTruthy();
    expect(within(region).getByText("1,234")).toBeTruthy();
    expect(within(region).getByText("2")).toBeTruthy();
    expect(within(region).getByRole("link", { name: "Stash settings" }).getAttribute("href")).toBe("/settings/stash?panel=connection");
    const open = within(region).getByRole("link", { name: "Open Stash" });
    expect(open.getAttribute("href")).toBe("http://stash.invalid/");
    expect(open.getAttribute("rel")).toBe("noopener noreferrer");
    expect(region.textContent).not.toMatch(/Task processing|Delivery|qBittorrent/);
    expect(region.querySelector("details, .home-dot")).toBeNull();
  });

  it("does not include qB or delivery failures in Stash status", () => {
    setup({ runtimeStatus: { ...status, qbittorrent: { configured: true, ready: false }, ingest: { configured: false } }, runtimeSettings: settings });
    const region = screen.getByRole("region", { name: "Stash" });
    expect(within(region).getByText("Connected")).toBeTruthy();
    expect(region.textContent).not.toMatch(/Delivery|qBittorrent/);
  });

  it("keeps retained metrics but marks a failed refresh as unknown and retries only Stash status", async () => {
    const props = setup({ runtimeStatus: status, runtimeSettings: settings, statusError: "Status read failed" });
    const region = screen.getByRole("region", { name: "Stash" });
    expect(within(region).getByText("Status unknown")).toBeTruthy();
    expect(within(region).queryByText("Connected")).toBeNull();
    expect(within(region).getByText("1,234")).toBeTruthy();
    await userEvent.setup().click(within(region).getByRole("button", { name: "Retry" }));
    expect(props.onRefreshStatus).toHaveBeenCalledOnce();
    expect(props.onRefresh).not.toHaveBeenCalled();
    expect(screen.getByText("DEMO-101")).toBeTruthy();
  });

  it("shows unknown measurements before the first successful observation instead of zeroes", () => {
    setup({ runtimeStatus: { ...status, stash: { configured: true, ready: false }, stashStats: { version: null, sceneCount: null, pendingMojiScanCount: 0, okAt: null, lastError: null } } });
    const region = screen.getByRole("region", { name: "Stash" });
    expect(within(region).getByText("Status unknown")).toBeTruthy();
    expect(within(region).getAllByText("—")).toHaveLength(3);
    expect(within(region).queryByText("0")).toBeNull();
    expect(within(region).queryByText("Unavailable")).toBeNull();
  });

  it("refreshes only Stash status and disables its refresh button while a read is pending", async () => {
    const props = setup({ runtimeStatus: status, statusFetching: true });
    const region = screen.getByRole("region", { name: "Stash" });
    const refresh = within(region).getByRole("button", { name: "Refresh Stash status" }) as HTMLButtonElement;
    expect(refresh.disabled).toBe(true);
    props.rerender(<MemoryRouter><HomePage {...props} statusFetching={false} /></MemoryRouter>);
    await userEvent.setup().click(refresh);
    expect(props.onRefreshStatus).toHaveBeenCalledOnce();
    expect(props.onRefresh).not.toHaveBeenCalled();
  });

  it("opens the Stash UI from a GraphQL endpoint while retaining a reverse proxy path", () => {
    setup({ runtimeStatus: status, runtimeSettings: { ...settings, stash: { url: "http://stash.invalid/library/graphql" } } });
    const region = screen.getByRole("region", { name: "Stash" });
    expect(within(region).getByRole("link", { name: "Open Stash" }).getAttribute("href")).toBe("http://stash.invalid/library/");
  });

  it("does not create an external link from an unsafe or absent URL", () => {
    setup({ runtimeStatus: status, runtimeSettings: { ...settings, stash: { url: "javascript:alert(1)" } } });
    expect(within(screen.getByRole("region", { name: "Stash" })).queryByRole("link", { name: "Open Stash" })).toBeNull();
  });
});
