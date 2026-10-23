import { expect } from "@playwright/test";
import { test } from "./fixtures";

for (const locale of ["zh-CN", "en"] as const) {
  test(`${locale} release review list`, async ({ page }, testInfo) => {
    await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale);
    await page.goto("/performers/review");
    await expect(page.getByRole("heading", { name: locale === "zh-CN" ? "发行复核" : "Release review", exact: true })).toBeVisible();
    await expect(page.locator(".release-review__item")).toHaveCount(2);
    await expect(page.getByRole("button", { name: locale === "zh-CN" ? "下载此片" : "Download release", exact: true })).toHaveCount(2);
    await expect(page.getByRole("button", { name: locale === "zh-CN" ? "不下载" : "Do not download", exact: true })).toHaveCount(2);
    await expect(page.getByText(locale === "zh-CN" ? "原因：VR 策略为人工确认" : "Reason: VR releases require review", { exact: true })).toBeVisible();
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({ path: testInfo.outputPath(`${locale}-release-review.png`), fullPage: true });
  });
}
