import { expect } from "@playwright/test";
import { homeOverview, test, torrentSearchSnapshot } from "./fixtures";

const task = homeOverview.attentionTasks[1];
const states = ["RUNNING", "QUEUED", "COMPLETED", "FAILED", "INTERRUPTED", "CANCELLED", "SKIPPED"];
test.use({ graphqlOverrides: {
  task: { ...task },
  torrentSearch: ({ id, offset, limit }: { id: string; offset: number; limit: number }) => ({
    ...torrentSearchSnapshot, id, offset, limit, taskId: task.id, state: "RUNNING",
    sites: Array.from({ length: 24 }, (_, i) => ({
      ...torrentSearchSnapshot.sites[0], instanceId: `site-${i}`, name: i === 3 ? "A very long site name without spaces " + "LongSiteName".repeat(7) : `Example site ${i + 1}`,
      kind: i % 2 ? "JACKETT" : "BUILTIN", state: states[i % states.length], results: i % states.length === 2 ? 0 : i,
      cacheHit: i === 2, errorCode: i % states.length === 3 ? "INDEXER_OPERATION_FAILED" : null
    }))
  })
} });

for (const [locale, theme] of [["en", "dark"], ["zh-CN", "light"]] as const) {
  test(`${locale} ${theme} compact progress and bounded site details`, async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    await page.addInitScript(({ locale, theme }) => { localStorage.setItem("moji:locale", locale); localStorage.setItem("moji:theme", theme); }, { locale, theme });
    await page.goto(`/tasks/${task.id}/resolve?__locale=${locale}`);
    const progress = page.locator(".torrent-search-progress");
    const summary = progress.locator("summary");
    const sites = progress.getByRole("list");
    await expect(summary).toContainText(locale === "en" ? "16/24 sites finished" : "已结束 16/24 站");
    await expect(sites).toBeHidden();
    await expect(progress.getByRole("button", { name: locale === "en" ? "Cancel search" : "取消搜索", exact: true })).toBeEnabled();
    const action = await progress.getByRole("button").boundingBox();
    const toggle = await progress.locator(".torrent-search-progress__toggle").boundingBox();
    if (action && toggle) expect(action.y + action.height <= toggle.y || action.x >= toggle.x + toggle.width).toBe(true);
    await expect(progress).toHaveScreenshot(`${locale}-${theme}-summary.png`);
    await summary.focus();
    await page.keyboard.press("Enter");
    await expect(sites).toBeVisible();
    await expect(sites.locator("li")).toHaveCount(24);
    expect(await sites.evaluate(el => el.clientHeight <= 240 && el.scrollHeight > el.clientHeight)).toBe(true);
    expect(await progress.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
    await expect(progress).toHaveScreenshot(`${locale}-${theme}-expanded.png`);
    await summary.click();
    await expect(sites).toBeHidden();
    expect(errors).toEqual([]);
  });
}
