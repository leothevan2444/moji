import { expect } from "@playwright/test";
import { test, serviceStatusSnapshot } from "./fixtures";

const cacheStatus = {
  ...serviceStatusSnapshot,
  imageCache: { ...serviceStatusSnapshot.imageCache, usedBytes: 4096, entryCount: 2 },
  stashBoxDataCache: { ...serviceStatusSnapshot.stashBoxDataCache, usedBytes: 4096, sceneCount: 2, performerCount: 1 }
};

test.use({ graphqlOverrides: { settingsStatus: cacheStatus } });

for (const locale of ["en", "zh-CN"] as const) {
  for (const theme of ["light", "dark"] as const) {
    test(`${locale} ${theme} release review uses shared surfaces and fallback artwork`, async ({ page }, info) => {
      await page.addInitScript(({ locale, theme }) => {
        localStorage.setItem("moji:locale", locale);
        localStorage.setItem("moji:theme", theme);
      }, { locale, theme });
      await page.goto("/performers/review");
      const cards = page.locator(".release-review__item");
      await expect(cards).toHaveCount(2);
      await expect(cards.first().locator(".performer-scene-card__image-placeholder")).toBeVisible();
      await expect(cards.first().locator(".release-review__meta")).toHaveCSS("color", theme === "dark" ? "rgb(147, 161, 181)" : "rgb(95, 107, 122)");
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await page.evaluate(() => document.fonts.ready);
      await expect(page).toHaveScreenshot(`${locale}-${theme}-release-review.png`, { fullPage: true });
    });

    for (const kind of ["image", "stashbox"] as const) {
      test(`${locale} ${theme} ${kind} cache confirms, cancels and retains failed operations`, async ({ page }, info) => {
        await page.addInitScript(({ locale, theme }) => {
          localStorage.setItem("moji:locale", locale);
          localStorage.setItem("moji:theme", theme);
        }, { locale, theme });
        let attempts = 0;
        let finish: (() => void) | undefined;
        await page.route("**/graphql", async route => {
          const operation = route.request().postDataJSON().operationName;
          if (operation !== (kind === "image" ? "ClearImageCacheDocument" : "ClearStashBoxDataCacheDocument")) return route.fallback();
          attempts++;
          if (attempts === 1) {
            await route.fulfill({ contentType: "application/json", body: JSON.stringify({ data: null, errors: [{ message: "Clear failed", extensions: { code: "INTERNAL_ERROR" } }] }) });
          } else {
            await new Promise<void>(resolve => { finish = resolve; });
            const field = kind === "image" ? "clearImageCache" : "clearStashBoxDataCache";
            const result = kind === "image" ? { ...cacheStatus.imageCache, entryCount: 0, usedBytes: 0 } : { ...cacheStatus.stashBoxDataCache, sceneCount: 0, performerCount: 0, usedBytes: 0 };
            await route.fulfill({ contentType: "application/json", body: JSON.stringify({ data: { [field]: result } }) });
          }
        });
        await page.goto(kind === "image" ? "/settings/system?panel=imageCache" : "/settings/stash?panel=metadata");
        const opener = page.locator(".image-cache-management__clear");
        await expect(opener).toBeEnabled();
        await opener.click();
        const dialog = page.getByRole("dialog");
        await expect(dialog).toHaveCount(1);
        await page.keyboard.press("Escape");
        await expect(dialog).toHaveCount(0);
        await expect(opener).toBeFocused();
        expect(attempts).toBe(0);
        await opener.click();
        await page.evaluate(() => document.fonts.ready);
        expect(await dialog.evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
        await expect(dialog).toHaveScreenshot(`${locale}-${theme}-${kind}-cache-confirm.png`);
        const confirm = dialog.locator(".confirmation-actions__confirm");
        await confirm.click();
        await expect(dialog.getByRole("alert")).toBeVisible();
        expect(attempts).toBe(1);
        await confirm.click();
        await expect(confirm).toBeDisabled();
        await page.keyboard.press("Escape");
        await expect(dialog).toBeVisible();
        await expect.poll(() => attempts).toBe(2);
        finish!();
        await expect(dialog).toHaveCount(0);
      });
    }
  }
}

for (const theme of ["light", "dark"] as const) {
  test(`automatic ${theme} theme keeps shared semantic colors available`, async ({ page }) => {
    await page.emulateMedia({ colorScheme: theme });
    await page.addInitScript(() => { localStorage.setItem("moji:theme", "auto"); localStorage.setItem("moji:locale", "en"); });
    await page.goto("/performers/review");
    await expect(page.locator(".release-review__item")).toHaveCount(2);
    await expect(page.locator(".release-review__meta").first()).toHaveCSS("color", theme === "dark" ? "rgb(147, 161, 181)" : "rgb(95, 107, 122)");
    expect(await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue("--danger").trim())).toBe(theme === "dark" ? "#fca5a5" : "#c2412f");
  });
}

for (const locale of ["qps-ploc", "ar-XB"] as const) {
  test(`${locale} release review keeps filters and actions reachable`, async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto(`/performers/review?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("dir", locale === "ar-XB" ? "rtl" : "ltr");
    await expect(page.locator(".release-review__item")).toHaveCount(2);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.evaluate(() => document.fonts.ready);
    await expect(page).toHaveScreenshot(`${locale}-release-review.png`, { fullPage: true });
  });
}
