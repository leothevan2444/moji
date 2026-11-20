import { expect } from "@playwright/test";
import { test, settingsSnapshot } from "./fixtures";

for (const locale of ["zh-CN", "en"] as const) {
  for (const theme of ["light", "dark"] as const) {
    test(`${locale} ${theme} scheduling timezone and cache spacing`, async ({ page }) => {
      await page.addInitScript(({ locale, theme }) => {
        localStorage.setItem("moji:locale", locale);
        localStorage.setItem("moji:theme", theme);
      }, { locale, theme });
      await page.goto("/settings/tasks?panel=schedule");
      const frequency = page.getByRole("combobox", { name: locale === "zh-CN" ? "任务推进频率" : "Task processing frequency" });
      await expect(frequency).toHaveValue("MEDIUM");
      await expect(page.locator('input[type="checkbox"]:visible')).toHaveCount(0);
      await expect(page).toHaveScreenshot(`${locale}-${theme}-scheduling.png`, { fullPage: true });
      await page.goto("/settings/system?panel=general");
      const timezone = page.getByRole("combobox", { name: locale === "zh-CN" ? "时区" : "Timezone", exact: true });
      await expect(timezone).toHaveValue("auto");
      await expect(timezone.locator('option[value="auto"]')).toHaveText(locale === "zh-CN" ? "跟随系统（UTC）" : "Follow system (UTC)");
      await expect(timezone.locator("..").locator("small")).toHaveCount(0);
      await expect(page).toHaveScreenshot(`${locale}-${theme}-timezone.png`, { fullPage: true, maxDiffPixels: 0 });
      await page.goto("/settings/stash?panel=metadata");
      const ttl = page.getByRole("spinbutton");
      await expect(ttl).toBeVisible();
      const inputBounds = await ttl.boundingBox();
      const managementBounds = await page.locator(".image-cache-management").boundingBox();
      expect(managementBounds!.y - inputBounds!.y - inputBounds!.height).toBeGreaterThanOrEqual(12);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await expect(page).toHaveScreenshot(`${locale}-${theme}-stashbox-cache.png`, { fullPage: true });
    });
  }
}

test.describe("global timezone behavior", () => {
  let current: typeof settingsSnapshot & {system: typeof settingsSnapshot.system & {language:string}};
  test.use({graphqlOverrides:{
    settings: () => current,
    logs: [{sequence:1,time:"2026-11-19T23:30:00Z",level:"Info",message:"timezone test"}],
    patchSettings: ({input}:{input:{system?:{timezone?:string}}}) => {
      if (input.system?.timezone !== undefined) {
        current.system.timezone = input.system.timezone;
        current.system.effectiveTimezone = input.system.timezone === "auto" ? "UTC" : input.system.timezone;
      }
      return current;
    }
  }});
  test("uses server timezone across pages and after reload with a different browser timezone", async ({page}) => {
    current={...structuredClone(settingsSnapshot),system:{...settingsSnapshot.system,language:"en"}};
    await page.goto("/settings/system?panel=general");
    const field=page.getByRole("combobox",{name:"Timezone",exact:true});
    await field.selectOption("Asia/Tokyo");
    await expect.poll(()=>current.system.timezone).toBe("Asia/Tokyo");
    await page.goto("/settings/logs");
    await expect(page.locator(".log-line__time")).toHaveText("Nov 20, 2026, 08:30:00 AM");
    await page.reload();
    await expect(page.locator(".log-line__time")).toHaveText("Nov 20, 2026, 08:30:00 AM");
    await page.goto("/settings/system?panel=general");
    await expect(field.locator('option[value="auto"]')).toHaveText("Follow system (UTC)");
    await field.selectOption("auto");
    await expect.poll(()=>current.system.timezone).toBe("auto");
    await page.goto("/settings/logs");
    await expect(page.locator(".log-line__time")).toHaveText("Nov 19, 2026, 11:30:00 PM");
  });
});
