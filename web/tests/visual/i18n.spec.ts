import { expect, test, type Page } from "@playwright/test";

const routes = [
  ["home", "/"],
  ["tasks", "/tasks"],
  ["performers", "/performers"],
  ["discover", "/discover"],
  ["settings", "/settings/system"]
] as const;

async function prepare(page: Page, locale: "zh-CN" | "en") {
  await page.addInitScript((value) => localStorage.setItem("moji:locale", value), locale);
  await page.route("**/graphql", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ data: {} }) });
  });
}

for (const locale of ["zh-CN", "en"] as const) {
  for (const [name, path] of routes) {
    test(`${locale} ${name}`, async ({ page }) => {
      await prepare(page, locale);
      await page.goto(path);
      await expect(page.locator("html")).toHaveAttribute("lang", locale);
      await expect(page.locator("body")).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      await expect(page).toHaveScreenshot(`${locale}-${name}.png`, { fullPage: true });
    });
  }
}

for (const locale of ["qps-ploc", "ar-XB"] as const) {
  test(`${locale} development locale`, async ({ page }) => {
    await prepare(page, "en");
    await page.goto(`/discover?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    await expect(page.locator("html")).toHaveAttribute("dir", locale === "ar-XB" ? "rtl" : "ltr");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-discover.png`, { fullPage: true });
  });
}
