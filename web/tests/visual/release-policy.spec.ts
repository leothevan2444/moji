import { expect } from "@playwright/test";
import { test } from "./fixtures";

for (const locale of ["zh-CN", "en"] as const) {
  test(`${locale} release policy fits the viewport and explains manual review`, async ({ page }, testInfo) => {
    await page.addInitScript((value) => localStorage.setItem("moji:locale", value), locale);
    await page.goto("/settings/automation");
    const vr = page.getByRole("combobox", { name: /VR/ });
    await expect(vr).toHaveValue("REVIEW");
    await expect(vr.locator("option")).toHaveText(locale === "zh-CN"
      ? ["下载", "人工确认", "跳过"]
      : ["Download", "Manual review", "Skip"]);
    await expect(page.getByText(locale === "zh-CN"
      ? "人工确认的影片可在演员页面的“发行复核”中处理。保存配置不会重新处理已记录的影片。"
      : "Handle releases awaiting review from Release review on the Performers page. Saving settings does not reprocess recorded releases.")).toBeVisible();
    await page.evaluate(() => document.fonts.ready);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await page.screenshot({ path: testInfo.outputPath(`${locale}-release-policy.png`), fullPage: true });
  });
}
