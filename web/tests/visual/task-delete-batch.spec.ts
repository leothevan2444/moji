import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

const tasks = homeOverview.attentionTasks.slice(0, 2);
const deleted = new Set<string>();
let received: boolean | undefined;
let failLast = false;
test.beforeEach(async ({ page }) => {
  deleted.clear(); received = undefined; failLast = false;
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
});
test.use({ graphqlOverrides: {
  tasks: () => tasks.filter(task => !deleted.has(task.id)),
  deleteTasks: ({ ids, deleteStashScene }: { ids: string[]; deleteStashScene: boolean }) => {
    received = deleteStashScene;
    const results = ids.map(id => {
      const failed = failLast && id === tasks[1].id;
      if (!failed) deleted.add(id);
      return { taskId: id, status: failed ? "FAILED" : "SUCCEEDED", reasonCode: failed ? "STASH_DELETE_FAILED" : "DELETED", task: tasks.find(task => task.id === id) };
    });
    return { batchId: "delete-batch", summary: { requestedCount: ids.length, succeededCount: deleted.size, skippedCount: 0, failedCount: ids.length - deleted.size }, results };
  }
} });

for (const mode of ["default", "stash", "partial"]) {
  test(`batch deletion ${mode}`, async ({ page }) => {
    failLast = mode === "partial";
    await page.goto("/tasks");
    await page.getByRole("button", { name: "Enter multi-select mode", exact: true }).click();
    await page.getByRole("button", { name: "Select this page (2)", exact: true }).click();
    await page.getByRole("button", { name: "Delete (2)", exact: true }).click();
    const checkbox = page.locator(".confirm-delete").getByRole("checkbox");
    await expect(checkbox).not.toBeChecked();
    if (mode !== "default") await checkbox.check();
    await page.locator(".confirm-delete__confirm").click();
    await expect.poll(() => received).toBe(mode !== "default");
    await expect(page.locator(".confirm-delete")).toHaveCount(0);
    await expect(page.locator(".task-card")).toHaveCount(mode === "partial" ? 1 : 0);
    if (mode === "partial") {
      await expect(page.locator(".task-batch-results")).toContainText("Stash deletion incomplete; task kept");
      await expect(page.locator(".task-card")).toContainText(tasks[1].id);
    } else {
      await expect(page.getByText("Successfully processed 2 tasks.", { exact: true })).toBeVisible();
    }
  });
}
