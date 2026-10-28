import { expect } from "@playwright/test";
import { settingsSnapshot, test } from "./fixtures";

for (const locale of ["zh-CN", "en", "qps-ploc", "ar-XB"] as const) {
  test.describe(locale, () => {
  test.use({graphqlOverrides:{settings:{...settingsSnapshot,system:{...settingsSnapshot.system,language:locale === "zh-CN" ? "zh-CN" : "en"}}}});
  test(`${locale} navigation fits and keeps destinations reachable`, async ({ page }, info) => {
    await page.goto(`/tasks?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    const header = page.locator(".masthead");
    const toggle = header.locator(".masthead__toggle");
    if (info.project.name === "mobile") {
      await expect(toggle).toBeVisible();
      await toggle.focus();
      await page.keyboard.press("Enter");
      await expect(toggle).toHaveAttribute("aria-expanded", "true");
    }
    await expect(header.locator('nav [aria-current="page"]')).toHaveAttribute("href", "/tasks");
    const selected = header.locator('nav [aria-current="page"]');
    expect(await selected.evaluate(node => {
      const style = getComputedStyle(node);
      return { background: style.backgroundColor, shadow: style.boxShadow, border: style.borderTopWidth };
    })).toEqual({background:"rgba(0, 0, 0, 0)",shadow:"none",border:"0px"});
    await page.evaluate(() => document.fonts.ready);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-navigation.png`, {clip: await header.evaluate(el => ({x:el.getBoundingClientRect().x,y:el.getBoundingClientRect().y,width:el.getBoundingClientRect().width,height:220}))});
    if (info.project.name === "mobile") {
      await page.keyboard.press("Escape");
      await expect(toggle).toBeFocused();
      await expect(header.locator("nav")).toBeHidden();
      await toggle.click();
    }
    await header.locator('nav a[href="/performers"]').click();
    await expect(page).toHaveURL(/\/performers$/);
    if (info.project.name === "mobile") await expect(toggle).toHaveAttribute("aria-expanded", "false");
    if (locale === "qps-ploc" || locale === "ar-XB") {
      await page.goto(`/performers?__locale=${locale}`);
      await expect(page.locator("html")).toHaveAttribute("lang", locale);
    }
    await expect(header.locator('.masthead__tools a[href="/stats"]')).toBeVisible();
    await expect(header.locator('.masthead__tools a[href="/settings"]')).toBeVisible();
    await expect(header.locator('a[href="/settings/system"]')).toHaveCount(0);
    await expect(page).toHaveScreenshot(`${locale}-tools.png`, { clip: await header.evaluate(el => ({x:el.getBoundingClientRect().x,y:el.getBoundingClientRect().y,width:el.getBoundingClientRect().width,height:320})) });
    await header.locator('a[href="/settings"]').click();
    await expect(page).toHaveURL(/\/settings\/indexers$/);
  });
  });
}

test("visible tools and theme support keyboard focus", async ({ page }) => {
  await page.goto("/");
  const header = page.locator(".masthead");
  const stats = header.getByRole("link", {name:"Statistics",exact:true});
  await stats.focus();
  await page.keyboard.press("Tab");
  await expect(header.getByRole("link", {name:"Settings",exact:true})).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(header.getByRole("button", {name:"Help",exact:true})).toBeFocused();
  const theme = header.locator(".theme-menu > button");
  await theme.focus();
  await page.keyboard.press("ArrowDown");
  const options = page.getByRole("menuitemradio");
  await expect(page.getByRole("menuitemradio", { checked:true })).toBeFocused();
  await page.keyboard.press("Home");
  await expect(options.nth(0)).toBeFocused();
  await page.keyboard.press("ArrowUp");
  await expect(options.nth(2)).toBeFocused();
  await page.keyboard.press("End");
  await page.keyboard.press("ArrowDown");
  await expect(options.nth(0)).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(theme).toBeFocused();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.keyboard.press("Enter");
  await page.keyboard.press("Escape");
  await expect(theme).toBeFocused();
  await page.keyboard.press("Enter");
  await page.keyboard.press("Tab");
  expect(await page.locator("main").evaluate(node => node.contains(document.activeElement))).toBe(true);
  await expect(page.getByRole("menu")).toHaveCount(0);
});

test.describe("developer tools", () => {
  test.use({graphqlOverrides:{settings:{...settingsSnapshot,system:{...settingsSnapshot.system,developerMode:true}}}});
  test("shows performance only in saved developer mode and restores focus after Help", async ({ page }) => {
    await page.goto("/");
    const header = page.locator(".masthead");
    await expect(header.getByRole("button", {name:"Performance monitoring"})).toBeVisible();
    await expect(header.locator(".masthead__tools > a, .masthead__tools > button")).toHaveCount(4);
    expect(await header.locator(".masthead__tools > a, .masthead__tools > button").evaluateAll(items => items.map(item => item.getAttribute("aria-label"))))
      .toEqual(["Performance monitoring", "Statistics", "Settings", "Help"]);
    const help = header.getByRole("button", {name:"Help",exact:true});
    await help.click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(help).toBeFocused();
  });
  test("320px keeps every developer tool visible", async ({ page }) => {
    await page.setViewportSize({width:320,height:740});
    await page.goto("/?__locale=en");
    const header = page.locator(".masthead");
    for (const name of ["Performance monitoring", "Statistics", "Settings", "Help"]) {
      const tool = header.getByRole(name === "Statistics" || name === "Settings" ? "link" : "button", {name,exact:true});
      await expect(tool).toBeVisible();
      const bounds = await tool.boundingBox();
      expect(bounds!.x).toBeGreaterThanOrEqual(0);
      expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(320);
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await expect(page).toHaveScreenshot("narrow-developer-header.png", {clip:{x:0,y:0,width:320,height:100}});
  });

});

test("320px dark header and tools stay within the viewport", async ({ page }) => {
  await page.setViewportSize({width:320,height:740});
  await page.addInitScript(() => localStorage.setItem("moji:theme", "dark"));
  await page.goto("/?__locale=en");
  const header = page.locator(".masthead");
  await expect(header.getByRole("link", {name:"Statistics",exact:true})).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  await expect(page).toHaveScreenshot("dark-narrow-tools.png",{clip:{x:0,y:0,width:320,height:350}});
});

test("header palette follows explicit and system themes", async ({ page }) => {
  await page.emulateMedia({colorScheme:"dark"});
  await page.addInitScript(() => localStorage.setItem("moji:theme", "light"));
  await page.goto("/?__locale=en");
  const header = page.locator(".masthead");
  const theme = header.locator(".theme-menu > button");
  const palette = () => header.evaluate(node => ({
    background: getComputedStyle(node, "::before").backgroundColor,
    text: getComputedStyle(node).color,
    page: getComputedStyle(document.documentElement).getPropertyValue("--bg").trim()
  }));
  await expect.poll(palette).toEqual({background:"rgb(255, 255, 255)",text:"rgb(22, 32, 51)",page:"#f5f7fa"});
  await expect(header).toHaveScreenshot("theme-light-header.png");
  await theme.click();
  await page.getByRole("menuitemradio", {name:"Dark",exact:true}).click();
  await expect.poll(palette).toEqual({background:"rgb(22, 32, 51)",text:"rgb(230, 236, 245)",page:"#0b1220"});
  await expect(header).toHaveScreenshot("theme-dark-header.png");
  await page.emulateMedia({colorScheme:"light"});
  await expect.poll(async () => (await palette()).background).toBe("rgb(22, 32, 51)");
  await theme.click();
  await page.getByRole("menuitemradio", {name:/Automatic/}).click();
  await expect.poll(async () => (await palette()).background).toBe("rgb(255, 255, 255)");
  await page.emulateMedia({colorScheme:"dark"});
  await expect.poll(async () => (await palette()).background).toBe("rgb(22, 32, 51)");
});
