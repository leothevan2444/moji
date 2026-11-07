import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

const task = { ...homeOverview.attentionTasks[0], stashSceneId: "101", stashSceneIds: ["101"] };
let deleted = false;
test.beforeEach(async ({ page }) => {
  deleted = false;
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
});
test.use({ graphqlOverrides: {
  tasks: () => deleted ? [] : [task],
  task: () => deleted ? null : task,
  homeTaskSnapshot: () => ({ ...homeOverview, sequence: 0, streamId: "visual",
    attentionTasks: deleted ? [] : [task], activeTasks: [], recentTasks: [] }),
  deleteTask: () => { deleted = true; return task; }
} });

for (const locale of ["en", "zh-CN"]) for (const entry of ["tasks", "home"]) for (const removeStash of [false, true]) {
  test(`${locale} ${entry} deletion with Stash=${removeStash}`, async ({ page }, testInfo) => {
    const requests: boolean[] = [];
    await page.addInitScript(locale => localStorage.setItem("moji:locale", locale), locale);
    await page.emulateMedia({ colorScheme: locale === "en" ? "light" : "dark" });
    page.on("request", request => {
      if (request.url().endsWith("/graphql") && request.method() === "POST") {
        const body = request.postDataJSON();
        if (body.query.includes("mutation DeleteTaskDocument")) requests.push(body.variables.deleteStashScene);
      }
    });
    await page.goto(entry === "tasks" ? `/tasks/${task.id}` : `/?task=${task.id}`);
    await page.getByRole("dialog").locator(".task-detail-actions__delete").click();
    const choice = page.locator(".confirm-delete").getByRole("checkbox");
    await expect(choice).not.toBeChecked();
    // Cancelling a checked confirmation must never remember the choice.
    await choice.check();
    await page.keyboard.press("Escape");
    await page.getByRole("dialog").locator(".task-detail-actions__delete").click();
    await expect(choice).not.toBeChecked();
    if (removeStash) await choice.check();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({ path: testInfo.outputPath("delete-confirmation.png") });
    await page.locator(".confirm-delete__confirm").click();
    await expect(page.getByText(locale === "en" ? /^Deleted task:/ : /^已删除任务：/)).toBeVisible();
    await expect(page.getByText(/server returned no deleted task record|后端没有返回已删除/)).toHaveCount(0);
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page).toHaveURL(entry === "tasks" ? /\/tasks$/ : /\/$/);
    expect(deleted).toBe(true);
    expect(requests).toEqual([removeStash]);
    await page.screenshot({ path: testInfo.outputPath("delete-success.png") });
    await page.reload();
    await expect(page.locator(".task-detail-actions__delete")).toHaveCount(0);
  });
}
