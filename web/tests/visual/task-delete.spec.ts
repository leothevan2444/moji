import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

const task = homeOverview.attentionTasks[0];
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

for (const entry of ["tasks", "home"]) {
  test(`${entry} deletion shows success and closes the task`, async ({ page }, testInfo) => {
    await page.goto(entry === "tasks" ? `/tasks/${task.id}` : `/?task=${task.id}`);
    await page.getByRole("dialog").getByRole("button", { name: /^Delete task:/ }).click();
    await page.getByRole("button", { name: "Confirm deletion", exact: true }).click();
    await expect(page.getByText(/^Deleted task:/)).toBeVisible();
    await expect(page.getByText(/server returned no deleted task record/)).toHaveCount(0);
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page).toHaveURL(entry === "tasks" ? /\/tasks$/ : /\/$/);
    expect(deleted).toBe(true);
    await page.screenshot({ path: testInfo.outputPath("delete-success.png") });
    await page.reload();
    await expect(page.getByRole("button", { name: /^Delete task:/ })).toHaveCount(0);
  });
}
