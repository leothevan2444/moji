import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

const base = { ...homeOverview.attentionTasks[2], stashIngestPhase: "IDENTIFY", progress: 1, stashSceneIds: ["scene-1"], stashScanPath: "/library/MOJI-103" };
const conflict = { ...base, id: "studio-conflict", stageErrorCode: "STASH_STUDIO_CONFLICT" };
const removed = { ...base, id: "scene-removed", stageErrorCode: "STASH_SCENE_MISSING" };
const relations = { ...base, id: "code-relations", stageErrorCode: "STASH_CODE_RELATIONS_REQUIRED" };
test.use({ graphqlOverrides: {
  tasks: [conflict, removed, relations],
  task: ({ id }: { id: string }) => [conflict, removed, relations].find(task => task.id === id),
  taskStashScenes: ({ id }: { id: string }) => id === removed.id ? [] : [{ id: "scene-1", title: "", code: "", date: "", studio: "", performers: [], imageUrl: null, url: "https://stash.invalid/scenes/scene-1" }],
} });

for (const locale of ["en", "zh-CN"] as const) {
  for (const theme of ["light", "dark"] as const) {
    test(`${locale} ${theme} explains studio conflicts, missing fields and removed scenes`, async ({ page }, testInfo) => {
      const errors: string[] = [];
      page.on("pageerror", error => errors.push(error.message));
      await page.addInitScript(({ locale, theme }) => {
        localStorage.setItem("moji:locale", locale);
        localStorage.setItem("moji:theme", theme);
      }, { locale, theme });
      await page.goto(`/tasks/${conflict.id}`);
      await expect(page.locator(".task-detail-overview")).toContainText(locale === "en" ? "studio name conflicts" : "片商名称与已有记录冲突");
      const info = page.locator(".task-stash-scenes");
      await expect(info).toContainText(locale === "en" ? "title, code, release date, studio, performers" : "标题、番号、发行日期、片商、演员");
      await info.scrollIntoViewIfNeeded();
      expect(await info.evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
      await info.screenshot({ path: testInfo.outputPath("missing-fields.png") });
      await page.reload();
      await expect(info).toContainText(locale === "en" ? "Stash identification is missing" : "Stash 识别结果缺少");
      await page.goto(`/tasks/${removed.id}`);
      await expect(info).toContainText(locale === "en" ? "Linked movies were not found in Stash" : "未找到关联的 Stash 影片");
      await expect(info).not.toContainText(locale === "en" ? "temporarily unavailable" : "暂时无法加载");
      await info.scrollIntoViewIfNeeded();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await info.screenshot({ path: testInfo.outputPath("removed-scene.png") });
      await page.goto(`/tasks/${relations.id}`);
      await expect(page.locator(".task-detail-overview")).toContainText(locale === "en" ? "link or create the scraped studio" : "未能关联或创建片商");
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await page.locator(".task-detail-overview").screenshot({ path: testInfo.outputPath("code-relations.png") });
      expect(errors).toEqual([]);
    });
  }
}
