import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
});

test("home opens file confirmation directly and restores focus after Escape", async ({ page }) => {
  await page.goto("/");
  const confirm = page.getByRole("button", { name: "Select movie files: MOJI-101" });
  await confirm.click();
  await expect(page).toHaveURL(/task=MOJI-101&files=1/);
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("checkbox")).toBeVisible();
  await expect(dialog.getByRole("heading", { name: "Task overview" })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(page).toHaveURL("/");
  await expect(confirm).toBeFocused();
});

test("home search preserves query and source when returning", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("combobox").selectOption("jackett");
  await page.getByRole("textbox", { name: "Search movies or resources" }).fill("MOJI-101");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page).toHaveURL(/discover\?q=MOJI-101&source=jackett/);
  await page.goBack();
  await expect(page.getByRole("textbox", { name: "Search movies or resources" })).toHaveValue("MOJI-101");
  await expect(page.getByRole("combobox")).toHaveValue("jackett");
});

test.describe("review queue", () => {
  test.use({ graphqlOverrides: { tasks: homeOverview.attentionTasks } });
  test("waits for tasks before opening the session and returns Home on close", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Review queue" }).click();
    await expect(page.getByRole("dialog", { name: "Task workbench" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "MOJI-101", exact: true })).toBeVisible();
    await expect(page.getByText("0 handled · 3 remaining")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("search", { name: "Search movies or resources" })).toBeVisible();
    await expect(page).toHaveURL("/");
  });
});

for (const locale of ["qps-ploc", "ar-XB"] as const) {
  test(`${locale} home layout`, async ({ page }) => {
    await page.goto(`/?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    await expect(page.locator(".home-metric")).toHaveCount(4);
    await expect(page.locator(".home-loading")).toHaveCount(0);
    await page.evaluate(() => document.fonts.ready);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-home.png`, { fullPage: true });
  });
}

test("dark home layout", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  await expect(page.locator(".home-loading")).toHaveCount(0);
  await expect.poll(() => page.locator("html").evaluate(element => getComputedStyle(element).colorScheme)).toBe("dark");
  await page.evaluate(() => document.fonts.ready);
  await expect(page).toHaveScreenshot("dark-home.png", { fullPage: true });
});
