import { expect } from "@playwright/test";
import { test } from "./fixtures";

const actors = ["Alpha", "Beta"].map((name, index) => ({ id: `p${index+1}`, name, aliasList: [], favorite: false, sceneCount: 0, imagePath: null, subscribed: true }));
const states = actors.map(performer => ({ performer, lastCheckedAt: null, lastError: null, pendingReleaseCount: 0, reviewReleaseCount: 0, processedReleaseCount: 0, recentReleases: [] }));
function makeRun(id: string, trigger = "MANUAL", ids = ["p1"]) {
  return { id, trigger, scope: ids.length === 1 ? "SINGLE" : "ALL", status: "RUNNING", startedAt: "2026-01-15T11:55:00Z", finishedAt: null as string | null, error: null,
    totalCount: ids.length, completedCount: 0, succeededCount: 0, failedCount: 0, skippedCount: 0,
    items: ids.map(performerId => ({ performerId, name: actors.find(actor => actor.id === performerId)!.name, status: "RUNNING", error: null as string | null, startedAt: "2026-01-15T11:55:00Z", finishedAt: null as string | null })) };
}
function summarize(run: ReturnType<typeof makeRun>) {
  const { items, ...summary } = run;
  return { ...summary, currentPerformers: items.filter(item => item.status === "RUNNING").slice(0, 3) };
}
let active: ReturnType<typeof makeRun> | null = null;
let recent: ReturnType<typeof makeRun>[] = [];
let starts = 0;
let detailReads = 0;
test.beforeEach(async ({ page }) => {
  active = null; recent = []; starts = 0; detailReads = 0;
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
});
test.use({ graphqlOverrides: {
  performerWorkspace: { performers: { items: actors, page: 1, pageSize: 24, totalCount: 2, totalPages: 1, hasPrevPage: false, hasNextPage: false }, subscribedPerformers: states },
  homeSubscriptionChecks: () => {
    const scheduled = active?.trigger === "SCHEDULED" ? active : recent.find(run => run.trigger === "SCHEDULED");
    return { enabled: true, nextRunAt: "2026-01-15T18:00:00Z", activeCount: active ? 1 : 0, latestScheduled: scheduled ? summarize(scheduled) : null, activeRuns: active ? [summarize(active)] : [], recentRuns: recent.slice(0, 3).map(summarize) };
  },
  subscriptionCheckActivity: ({ ids }: { ids: string[] }) => ({ activeCount: active ? 1 : 0, allRunId: active?.scope === "ALL" ? active.id : null, performers: active?.items.filter(item => ids.includes(item.performerId)).map(item => ({ performerId: item.performerId, runId: active!.id, status: item.status })) ?? [] }),
  subscriptionCheck: ({ id }: { id: string }) => { detailReads++; return active?.id === id ? active : recent.find(run => run.id === id) ?? null; },
  startSubscriptionCheck: ({ ids }: { ids?: string[] }) => { starts++; active ??= makeRun("check-manual-persisted", "MANUAL", ids ?? actors.map(actor => actor.id)); return active; }
} });

test("manual check survives actor reload and is tracked in Home's drawer", async ({ page }) => {
  await page.goto("/performers");
  const check = page.getByRole("button", { name: "Check Alpha for updates" });
  await expect(check).toBeEnabled();
  await check.click();
  await expect(check).toBeDisabled();
  await expect(page.getByRole("region", { name: "Release checks" })).toHaveCount(0);
  await page.reload();
  await expect(check).toBeDisabled();
  await expect(page.getByRole("button", { name: "Check Beta for updates" })).toBeEnabled();
  expect(starts).toBe(1);
  expect(detailReads).toBe(0);
  await page.getByRole("link", { name: "Checking", exact: true }).click();
  await expect(page).toHaveURL(/\?check=check-manual-persisted$/);
  const dialog = page.getByRole("dialog", { name: "Release checks" });
  await expect(dialog.getByText("check-manual-persisted", { exact: true })).toBeVisible();
  await page.reload();
  await expect(dialog.getByText("check-manual-persisted", { exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(page).toHaveURL("/");
  const home = page.getByRole("region", { name: "Release checks" });
  await expect(home.getByRole("progressbar", { name: "Check progress" })).toHaveAttribute("aria-valuenow", "0");
  await expect(home).toHaveScreenshot("checks-running.png");

  const finished = active!;
  finished.status = "SUCCEEDED"; finished.completedCount = 1; finished.succeededCount = 1; finished.finishedAt = "2026-01-15T12:00:00Z";
  finished.items[0].status = "SUCCEEDED"; finished.items[0].finishedAt = finished.finishedAt;
  recent = [finished]; active = null;
  await expect(home.getByText("Idle", { exact: true })).toBeVisible();
  await home.getByText("Recent runs (1)", { exact: true }).click();
  await expect(home.getByText("Completed", { exact: true })).toBeVisible();
  await page.goto("/performers");
  await expect(check).toBeEnabled();
});

test("Home shows bounded scheduled progress and loads failures only on demand", async ({ page }) => {
  active = makeRun("check-scheduled-12345678-1234-1234-1234-123456789012", "SCHEDULED", ["p1", "p2"]);
  active.completedCount = 1; active.failedCount = 1;
  active.items[0].status = "FAILED";
  active.items[0].error = "The release source could not be reached. ".repeat(8);
  await page.goto("/");
  const home = page.getByRole("region", { name: "Release checks" });
  await expect(home.getByText("Checked 1 / 2 performers", { exact: false })).toBeVisible();
  await expect(home.getByText("Checking: Beta", { exact: true })).toBeVisible();
  expect(detailReads).toBe(0);
  const open = home.getByRole("button", { name: /View Scheduled check/ });
  await open.click();
  const dialog = page.getByRole("dialog", { name: "Release checks" });
  await expect(dialog.getByText(active.items[0].error, { exact: true })).toBeVisible();
  await page.keyboard.press("/");
  await expect(page.getByRole("textbox", { name: "Search movies or resources" })).not.toBeFocused();
  await page.keyboard.press("Escape");
  await expect(open).toBeFocused();
  await page.setViewportSize({ width: 320, height: 900 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await expect(home).toHaveScreenshot("checks-scheduled-narrow.png");
  await open.click();
  await expect(dialog).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.keyboard.press("Escape");
  await page.goto("/performers");
  await expect(page.getByRole("button", { name: "Check Alpha for updates" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Check Beta for updates" })).toBeDisabled();
  await expect(page.getByRole("region", { name: "Release checks" })).toHaveCount(0);
});
