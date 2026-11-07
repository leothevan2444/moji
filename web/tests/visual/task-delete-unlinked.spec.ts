import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

const tasks = homeOverview.attentionTasks.slice(0, 2);
test.use({ graphqlOverrides: { tasks, task: ({ id }: { id: string }) => tasks.find(task => task.id === id) } });

for (const locale of ["en", "zh-CN"]) for (const entry of ["home", "tasks", "batch"]) {
  test(`${locale} ${entry} explains why Stash deletion is unavailable`, async ({ page }, info) => {
    await page.addInitScript(locale => localStorage.setItem("moji:locale", locale), locale);
    await page.emulateMedia({ colorScheme: locale === "en" ? "light" : "dark" });
    if (entry === "batch") {
      await page.goto("/tasks");
      await page.getByRole("button", { name: locale === "en" ? "Enter multi-select mode" : "进入多选模式", exact: true }).click();
      await page.getByRole("button", { name: locale === "en" ? "Select this page (2)" : "选择本页 2 项", exact: true }).click();
      await page.getByRole("button", { name: locale === "en" ? "Delete (2)" : "删除（2）", exact: true }).click();
    } else {
      await page.goto(entry === "home" ? `/?task=${tasks[0].id}` : `/tasks/${tasks[0].id}`);
      await page.getByRole("dialog").locator(".task-detail-actions__delete").click();
    }
    const checkbox = page.locator(".confirm-delete").getByRole("checkbox");
    await expect(checkbox).toBeDisabled();
    await expect(checkbox).not.toBeChecked();
    const reason = locale === "en" ? "No Stash scene linked yet" : "尚未关联 Stash 影片";
    await expect(page.locator(".confirm-delete__unlinked")).toHaveAttribute("title", reason);
    await page.locator(".confirm-delete__unlinked").hover();
    await expect(page.getByRole("tooltip")).toHaveCount(0);
    await page.locator(".confirm-delete__unlinked").focus();
    await page.screenshot({ path: info.outputPath("unlinked-tooltip.png") });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await expect(page.locator(".confirm-delete")).toBeVisible();
  });
}
