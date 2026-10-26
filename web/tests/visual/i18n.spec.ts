import { expect, type Page } from "@playwright/test";
import { test } from "./fixtures";

const routes = [
  ["home", "/"],
  ["tasks", "/tasks"],
  ["performers", "/performers"],
  ["discover", "/discover"],
  ["settings", "/settings/system"],
  ["downloader-settings", "/settings/downloader"]
] as const;

async function prepare(page: Page, locale: "zh-CN" | "en") {
  await page.addInitScript((value) => localStorage.setItem("moji:locale", value), locale);
}

async function expectReady(page: Page) {
  await expect(page.locator(".app-shell main .section-band").first()).toBeVisible();
  await expect(page.locator(".skeleton, .settings-loading, .is-spinning")).toHaveCount(0);
  await page.evaluate(() => document.fonts.ready);
  await expect.poll(() => page.evaluate(() =>
    document.documentElement.scrollWidth <= document.documentElement.clientWidth,
  ), { message: "Page must fit the viewport without horizontal scrolling" }).toBe(true);
}

for (const locale of ["zh-CN", "en"] as const) {
  for (const [name, path] of routes) {
    test(`${locale} ${name}`, async ({ page }, testInfo) => {
      await prepare(page, locale);
      await page.goto(path);
      await expect(page.locator("html")).toHaveAttribute("lang", locale);
      await expectReady(page);
      if (name === "home") await expect(page.locator(".home-metrics .home-metric")).toHaveCount(4);
      if (name === "settings" || name === "downloader-settings") await expect(page.locator(".settings-form").first()).toBeVisible();
      // Fractional desktop card positions can vary glyph edges by 217 pixels
      // in Chromium; keep this allowance local to the affected screenshot.
      const homeGlyphTolerance = locale === "zh-CN" && name === "home" && testInfo.project.name === "desktop"
        ? { maxDiffPixels: 250 } : {};
      await expect(page).toHaveScreenshot(`${locale}-${name}.png`, { fullPage: true, ...homeGlyphTolerance });
    });
  }
}

for (const locale of ["qps-ploc", "ar-XB"] as const) {
  test(`${locale} development locale`, async ({ page }) => {
    await prepare(page, "en");
    await page.goto(`/discover?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    await expect(page.locator("html")).toHaveAttribute("dir", locale === "ar-XB" ? "rtl" : "ltr");
    await expectReady(page);
    await expect(page).toHaveScreenshot(`${locale}-discover.png`, { fullPage: true });
  });
}
