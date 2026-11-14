import { expect } from "@playwright/test";
import { serviceStatusSnapshot, settingsSnapshot, test } from "./fixtures";

const labels = {
  en: { connected: "Connected", unavailable: "Unavailable", unknown: "Status unknown", unconfigured: "Not configured", settings: "Stash settings", refresh: "Refresh Stash status", open: "Open Stash" },
  "zh-CN": { connected: "已连接", unavailable: "不可用", unknown: "状态未知", unconfigured: "未配置", settings: "Stash 设置", refresh: "刷新 Stash 状态", open: "打开 Stash" }
};
for (const locale of ["en", "zh-CN"] as const) {
  for (const theme of ["light", "dark"] as const) {
    for (const state of ["connected", "unavailable", "unknown", "unconfigured"] as const) {
      test.describe(`${locale} ${theme} ${state}`, () => {
        const known = state === "connected" || state === "unavailable";
        test.use({ graphqlOverrides: {
          settings: { ...settingsSnapshot, stash: { ...settingsSnapshot.stash, url: state === "unconfigured" ? "" : "http://stash.test" } },
          settingsStatus: { ...serviceStatusSnapshot, stash: { configured: state !== "unconfigured", ready: state === "connected" }, stashStats: { ...serviceStatusSnapshot.stashStats, version: known ? "0.28.0" : null, sceneCount: known ? 1234 : null, pendingMojiScanCount: known ? 2 : 0, okAt: known ? "2026-01-15T12:00:00Z" : null, lastError: state === "unavailable" ? "connection refused" : null } }
        } });
        test("shows Stash metrics and independent refresh and settings actions", async ({ page }) => {
          const errors: string[] = [];
          page.on("pageerror", error => errors.push(error.message));
          let statusReads = 0;
          let homeReads = 0;
          page.on("request", request => {
            if (request.method() !== "POST" || !request.url().endsWith("/graphql")) return;
            const operation = request.postDataJSON()?.operationName;
            if (operation === "HomeServiceStatus") statusReads++;
            if (operation === "HomePageDocument") homeReads++;
          });
          await page.addInitScript(({ locale, theme }) => { localStorage.setItem("moji:locale", locale); localStorage.setItem("moji:theme", theme); }, { locale, theme });
          await page.goto("/");
          const region = page.getByRole("region", { name: "Stash", exact: true });
          await expect(region.getByText(labels[locale][state], { exact: true })).toBeVisible();
          await expect(region.locator("dl > div")).toHaveCount(4);
          await expect(region.locator("details, .home-dot, .home-task-processing")).toHaveCount(0);
          await expect(region.getByText("qBittorrent", { exact: true })).toHaveCount(0);
          if (known) {
            await expect(region.getByText("0.28.0", { exact: true })).toBeVisible();
            await expect(region.getByText("1,234", { exact: true })).toBeVisible();
          } else await expect(region.getByText("—", { exact: true })).toHaveCount(3);
          const external = region.getByRole("link", { name: labels[locale].open, exact: true });
          if (state === "unconfigured") await expect(external).toHaveCount(0);
          else {
            await expect(external).toHaveAttribute("href", "http://stash.test/");
            await expect(external).toHaveAttribute("rel", "noopener noreferrer");
          }
          await page.evaluate(() => document.fonts.ready);
          await expect(region).toHaveScreenshot(`${locale}-${theme}-${state}.png`);
          const beforeStatus = statusReads, beforeHome = homeReads;
          await region.getByRole("button", { name: labels[locale].refresh, exact: true }).click();
          await expect.poll(() => statusReads).toBeGreaterThan(beforeStatus);
          expect(homeReads).toBe(beforeHome);
          const settings = region.getByRole("link", { name: labels[locale].settings, exact: true });
          await settings.focus(); await page.keyboard.press("Enter");
          await expect(page).toHaveURL(/\/settings\/stash\?panel=connection$/);
          expect(errors).toEqual([]);
        });
      });
    }
  }
}

test("task settings provide an interval without a global task pause switch", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/settings/tasks?panel=schedule");
  const interval = page.locator(".task-processing-settings select");
  await expect(interval).toBeEnabled();
  await expect(interval.locator("option")).toHaveCount(4);
  await expect(page.locator(".task-processing-settings [role=switch]")).toHaveCount(0);
  await expect(page.locator(".task-processing-notice")).toHaveCount(0);
});
