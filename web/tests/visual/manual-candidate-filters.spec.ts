import { expect } from "@playwright/test";
import { homeOverview, searchCandidate, test, torrentSearchSnapshot } from "./fixtures";

const task = homeOverview.attentionTasks[1];
const candidates = Array.from({ length: 45 }, (_, i) => ({ ...searchCandidate, id: `candidate-${i}`, instanceId: i % 2 ? "b" : "a", instanceName: i % 2 ? "Beta" : "Alpha", title: `MOJI-102 release ${i}${i === 33 ? " UniqueTarget" : ""}` }));
test.use({ graphqlOverrides: {
  task,
  torrentSearch: ({ id, offset = 0, limit = 10, instanceId = "", candidateQuery = "" }: { id: string; offset: number; limit: number; instanceId?: string; candidateQuery?: string }) => {
    const matching = candidates.filter(item => candidateQuery.toLowerCase().trim().split(/\s+/).every(word => item.title.toLowerCase().includes(word)));
    const filtered = matching.filter(item => !instanceId || item.instanceId === instanceId);
    return { ...torrentSearchSnapshot, id, offset, limit, taskId: task.id, state: "PARTIAL", errorCode: "INDEXER_SEARCH_INCOMPLETE", filterInstanceId: instanceId || null, candidateQuery: candidateQuery.trim(), total: filtered.length, candidates: filtered.slice(offset, offset + limit), sites: ["a", "b"].map((id, index) => ({ ...torrentSearchSnapshot.sites[0], instanceId: id, name: index ? "Beta" : "Alpha", candidateCount: matching.filter(item => item.instanceId === id).length })) };
  }
} });

test("site and title filters cover the complete run and reset numbered pagination", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto(`/tasks/${task.id}/resolve`);
  const panel = page.locator(".sourcing-resolution__results");
  await expect(panel.getByRole("status")).toHaveText("Search completed");
  await expect(panel.getByRole("option", { name: "Alpha (23)" })).toHaveCount(1);
  await panel.getByRole("button", { name: "Page 5", exact: true }).click();
  await expect(panel.locator(".sourcing-candidate")).toHaveCount(5);
  await panel.getByRole("combobox", { name: "Site", exact: true }).selectOption("b");
  await expect(panel.locator(".candidate-pagination [aria-current=page]")).toHaveText("1");
  await expect(panel.locator(".sourcing-candidate")).toHaveCount(10);
  await panel.getByLabel("Filter candidates", { exact: true }).fill("uniquetarget");
  await expect(panel.locator(".sourcing-candidate")).toHaveCount(1);
  await expect(panel.locator(".sourcing-candidate")).toContainText("release 33 UniqueTarget");
  await expect(panel.getByRole("option", { name: "Beta (1)" })).toHaveCount(1);
  const item = panel.locator(".sourcing-candidate");
  await item.getByRole("button", { name: "Get download link" }).click();
  await expect(item.getByRole("link", { name: "Download torrent" })).toBeVisible();
  const details = await item.getByRole("link", { name: "Indexer details" }).boundingBox();
  const download = await item.getByRole("link", { name: "Download torrent" }).boundingBox();
  expect(details && download && Math.abs(details.y - download.y) < 4).toBe(true);
  await expect(panel).toHaveScreenshot("filtered-candidate.png");
  await page.reload();
  await expect(panel.getByLabel("Filter candidates", { exact: true })).toHaveValue("uniquetarget");
  await expect(panel.locator(".sourcing-candidate")).toHaveCount(1);
  await panel.getByLabel("Filter candidates", { exact: true }).fill("");
  await panel.getByRole("combobox", { name: "Site", exact: true }).selectOption("");
  await expect(panel.getByRole("option", { name: "All sites (45)" })).toHaveCount(1);
  await expect(panel.getByRole("button", { name: "Page 5", exact: true })).toBeVisible();
  expect(errors).toEqual([]);
});
