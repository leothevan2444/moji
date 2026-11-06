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
  test(`${locale} ${theme} compact progress and bounded site details`, async ({ page, context }) => {
    const errors: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    await page.addInitScript(({ locale, theme }) => { localStorage.setItem("moji:locale", locale); localStorage.setItem("moji:theme", theme); }, { locale, theme });
    await page.goto(`/tasks/${task.id}/resolve?__locale=${locale}`);
    const progress = page.locator(".torrent-search-progress");
    const summary = progress.locator(".torrent-search-progress__details > summary");
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
    expect(await sites.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
    await expect(progress).toHaveScreenshot(`${locale}-${theme}-expanded.png`);
    const failedSite = sites.locator("li").nth(3);
    await expect(failedSite.locator("details")).toHaveCount(0);
    expect(await sites.locator("li").evaluateAll(items => items.every(item => item.getBoundingClientRect().height === 54))).toBe(true);
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);
    const failure = failedSite.getByRole("button");
    await failure.hover();
    const tooltip = page.getByRole("tooltip");
    await expect(tooltip).toContainText(locale === "en" ? "The indexer operation failed" : "索引器操作失败");
    const bounds = await tooltip.boundingBox();
    expect(bounds && bounds.x >= 0 && bounds.x + bounds.width <= page.viewportSize()!.width).toBe(true);
    await expect(tooltip).toHaveScreenshot(`${locale}-${theme}-error-tooltip.png`);
    const originalTip = await tooltip.textContent();
    await failure.click();
    await expect(page.locator(".toast-card").filter({ hasText: locale === "en" ? "Error details copied" : "错误信息已复制" }).first()).toBeVisible();
    await expect(tooltip).toHaveText(originalTip!);
    expect(await page.evaluate(() => navigator.clipboard.readText())).toContain("INDEXER_OPERATION_FAILED");
    await page.mouse.move(0, 0);
    await expect(tooltip).toHaveCount(0);
    await failure.hover();
    await expect(tooltip).toHaveText(originalTip!);
    const otherFailure = sites.locator("li").nth(10).getByRole("button");
    await otherFailure.click();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toContain("Example site 11:");
    await failure.hover();
    await expect(tooltip).toHaveText(originalTip!);
    expect((await failedSite.boundingBox())!.height).toBe(54);
    await summary.click();
    await expect(sites).toBeHidden();
    expect(errors).toEqual([]);
  });
}
