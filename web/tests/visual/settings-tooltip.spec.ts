import { expect } from "@playwright/test";
import { test } from "./fixtures";

for (const scenario of [{ locale: "en", theme: "light" }, { locale: "zh-CN", theme: "dark" }, { locale: "ar-XB", theme: "dark" }]) {
  test(`${scenario.locale} settings tooltip stays in the viewport without widening content`, async ({ page }) => {
    await page.addInitScript(value => { localStorage.setItem("moji:locale", value.locale === "ar-XB" ? "en" : value.locale); localStorage.setItem("moji:theme", value.theme); }, scenario);
    await page.goto(`/settings/indexers?panel=connection${scenario.locale === "ar-XB" ? "&__locale=ar-XB" : ""}`);
    const info = page.locator(".indexer-external-toggle .settings-info");
    const tooltip = page.getByRole("tooltip");
    let widthBefore = await page.locator(".settings-main").evaluate(node => node.scrollWidth);
    const assertContained = async () => {
      await expect(tooltip).toBeVisible();
      const box = (await tooltip.boundingBox())!, viewport = page.viewportSize()!;
      expect(box.x).toBeGreaterThanOrEqual(7);
      expect(box.y).toBeGreaterThanOrEqual(7);
      expect(box.x + box.width).toBeLessThanOrEqual(viewport.width - 7);
      expect(box.y + box.height).toBeLessThanOrEqual(viewport.height - 7);
      expect(await page.locator(".settings-main").evaluate(node => node.scrollWidth)).toBe(widthBefore);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    };
    await info.hover();
    await info.focus();
    await assertContained();
    await expect(page).toHaveScreenshot(`${scenario.locale}-jackett-tooltip.png`, { fullPage: true });
    await page.keyboard.press("Escape");
    await expect(tooltip).toHaveCount(0);

    // A short viewport leaves no room below the real header control.
    await page.mouse.move(0, 0);
    await page.setViewportSize({ width: page.viewportSize()!.width, height: 330 });
    widthBefore = await page.locator(".settings-main").evaluate(node => node.scrollWidth);
    await info.hover();
    await info.focus();
    await assertContained();
    const anchor = (await info.boundingBox())!, popup = (await tooltip.boundingBox())!;
    expect(popup.y + popup.height).toBeLessThanOrEqual(anchor.y - 7);
    await page.keyboard.press("Escape");
    await expect(tooltip).toHaveCount(0);
  });
}
