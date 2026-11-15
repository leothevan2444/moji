import { expect } from "@playwright/test";
import { test } from "./fixtures";

const details = {
  phase: "FINISHED", catalogStatus: "SUCCEEDED", catalogErrorCode: "",
  catalogSource: "https://metadata.example/graphql", catalogMode: "poll", catalogStopReason: "max_pages_reached",
  catalogPages: 2, catalogPagesWithResults: 2, catalogScenes: 80, catalogReleases: 80,
  newCount: 0, knownCount: 80, upcomingCount: 0, inLibraryCount: 0, queuedCount: 0, reviewCount: 0, skippedCount: 0, creationFailedCount: 0,
  retryTotal: 2, retryAttempted: 2, retrySucceeded: 1, retryFailed: 1, retryDeferred: 0, omittedResults: 0,
  results: [
    { key: "film-one", code: "300MIUM-1437", title: "Synthetic film with a long title for diagnostic detail wrapping", phase: "HISTORICAL_RETRY", outcome: "FAILED", reasonCode:"solo_behavior_download", errorCode: "UPSTREAM_TEMPORARY", error: "metadata request timed out", taskId: null, nextRetryAt: "2026-01-15T12:05:00Z" },
    { key: "film-two", code: "T38-043", title: "Synthetic accepted film", phase: "HISTORICAL_RETRY", outcome: "DOWNLOADED", reasonCode:"solo_behavior_download", errorCode: "", error: "", taskId: "accepted-task", nextRetryAt: null },
  ],
};
test.use({ graphqlOverrides: {
  subscriptionCheck: ({ id }: { id: string }) => ({ id, trigger: "MANUAL", scope: "SINGLE", status: "SUCCEEDED", startedAt: "2026-01-15T11:59:00Z", finishedAt: "2026-01-15T12:00:00Z", error: null,
    totalCount: 1, completedCount: 1, succeededCount: 1, failedCount: 0, skippedCount: 0,
    items: [{ performerId: "1032", name: "Synthetic actor", status: "SUCCEEDED", startedAt: "2026-01-15T11:59:00Z", finishedAt: "2026-01-15T12:00:00Z", error: null, details: id === "legacy-check" ? null : details }] }),
} });

test("check diagnostics separate catalog success and failed historical retries", async ({ page }, info) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/?check=diagnostic-check");
  const dialog = page.getByRole("dialog", { name: "Release checks", exact: true });
  await expect(dialog.getByRole("heading", { name: "Current catalog check · Check completed" })).toBeVisible();
  await expect(dialog).toContainText("Read 2 pages and 80 catalog entries");
  await expect(dialog).toContainText("Historical failures 2 · Retried 2 · Handled 1 · Retry failures 1 · Deferred 0");
  await dialog.getByText("View original error", { exact: true }).click();
  await expect(dialog.getByText("metadata request timed out", { exact: true })).toBeVisible();
  await expect(dialog.getByRole("link", { name: "Resolve this film", exact: true })).toHaveAttribute("href", /performer=1032.*status=ALL.*q=300MIUM-1437/);
  await expect(dialog.getByRole("link", { name: "View task", exact: true })).toHaveAttribute("href", "/tasks/accepted-task");
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.screenshot({ path: info.outputPath("check-diagnostics.png") });
  await page.reload();
  await expect(dialog).toContainText("Historical failures 2");
});

test("older check records do not invent missing observations", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/?check=legacy-check");
  const dialog = page.getByRole("dialog", { name: "Release checks", exact: true });
  await expect(dialog).toContainText("Detailed observations were not captured for this check.");
  await expect(dialog).not.toContainText("Read 0 pages");
});
