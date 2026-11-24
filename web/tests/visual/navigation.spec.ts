import { expect } from "@playwright/test";
import { settingsSnapshot, test } from "./fixtures";

for (const locale of ["zh-CN", "en", "qps-ploc", "ar-XB"] as const) {
  test.describe(locale, () => {
  test.use({graphqlOverrides:{settings:{...settingsSnapshot,system:{...settingsSnapshot.system,language:locale === "zh-CN" ? "zh-CN" : "en"}}}});
  test(`${locale} navigation fits and keeps destinations reachable`, async ({ page }) => {
    await page.goto(`/tasks?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    const header = page.locator(".masthead");
    await expect(header.locator(".masthead__toggle")).toHaveCount(0);
    await expect(header.locator("nav a")).toHaveCount(4);
    for (const link of await header.locator("nav a").all()) await expect(link).toBeVisible();
    await expect(header.locator('nav [aria-current="page"]')).toHaveAttribute("href", "/tasks");
    const selected = header.locator('nav [aria-current="page"]');
    expect(await selected.evaluate(node => {
      const style = getComputedStyle(node);
      return { background: style.backgroundColor, shadow: style.boxShadow, border: style.borderTopWidth };
    })).toEqual({background:"rgba(0, 0, 0, 0)",shadow:"none",border:"0px"});
    await page.evaluate(() => document.fonts.ready);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-navigation.png`, {clip: await header.evaluate(el => ({x:el.getBoundingClientRect().x,y:el.getBoundingClientRect().y,width:el.getBoundingClientRect().width,height:220}))});
    await header.locator('nav a[href="/performers"]').click();
    await expect(page).toHaveURL(/\/performers$/);
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
  await expect(header.getByRole("button", {name:"NSFW image masking",exact:true})).toBeFocused();
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
    await expect(header.locator(".masthead__tools > a, .masthead__tools > button")).toHaveCount(5);
    expect(await header.locator(".masthead__tools > a, .masthead__tools > button").evaluateAll(items => items.map(item => item.getAttribute("aria-label"))))
      .toEqual(["Performance monitoring", "Statistics", "Settings", "Help", "NSFW image masking"]);
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
    for (const name of ["Performance monitoring", "Statistics", "Settings", "Help", "NSFW image masking"]) {
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


test("navigation labels follow actual available width and keep focus when resized", async ({ page }) => {
  await page.setViewportSize({width:960,height:800});
  await page.goto("/tasks?__locale=en");
  const header = page.locator(".masthead");
  const labels = header.locator("nav a span");
  await expect(header).not.toHaveClass(/masthead--icons/);
  for (const label of await labels.all()) await expect(label).toBeVisible();
  const tasks = header.getByRole("link", {name:"Tasks",exact:true});
  await tasks.focus();
  // Increase the space consumed by existing tools without changing the viewport.
  await header.locator(".masthead__tools").evaluate(node => { (node as HTMLElement).style.width = "600px"; });
  await expect(header).toHaveClass(/masthead--icons/);
  for (const label of await labels.all()) await expect(label).toBeHidden();
  await expect(tasks).toBeFocused();
  await expect(tasks).toHaveAttribute("title", "Tasks");
  await header.locator(".masthead__tools").evaluate(node => { (node as HTMLElement).style.removeProperty("width"); });
  await expect(header).not.toHaveClass(/masthead--icons/);
  const enlargedText = await page.addStyleTag({content:".masthead__link { font-size: 40px; }"});
  await expect(header).toHaveClass(/masthead--icons/);
  await enlargedText.evaluate(node => { node.parentNode?.removeChild(node); });
  await expect(header).not.toHaveClass(/masthead--icons/);
  await page.setViewportSize({width:640,height:800});
  await expect(header).toHaveClass(/masthead--icons/);
  await expect(tasks).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(header.getByRole("link", {name:"Performers",exact:true})).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/performers$/);
});


test("navigation hides labels before wrapping tools", async ({page}) => {
  await page.setViewportSize({width:1100,height:800});
  await page.goto("/tasks?__locale=en");
  const header = page.locator(".masthead");
  await expect(header).not.toHaveClass(/masthead--icons/);
  // Read layout in the same task as resizing, before ResizeObserver can run.
  // The full-label layout must never wrap ahead of the icon-only decision.
  const wrappedBeforeObservation = await header.evaluate(header => {
    const nav = header.querySelector("nav")!;
    const tools = header.querySelector(".masthead__tools")!;
    const brand = header.querySelector(".masthead__brand")!;
    const gap = parseFloat(getComputedStyle(header).columnGap);
    const navGap = parseFloat(getComputedStyle(nav).columnGap);
    const links = [...nav.querySelectorAll("a")];
    const required = brand.getBoundingClientRect().width + tools.getBoundingClientRect().width
      + links.reduce((width, link) => width + link.getBoundingClientRect().width, 0)
      + (links.length - 1) * navGap + gap * 2;
    (header as HTMLElement).style.width = `${Math.floor(required) - 1}px`;
    return tools.getBoundingClientRect().top > nav.getBoundingClientRect().bottom;
  });
  expect(wrappedBeforeObservation).toBe(false);
  await expect(header).toHaveClass(/masthead--icons/);
  const tools = header.locator(".masthead__tools");
  await expect.poll(async () => {
    const navBounds = await header.locator("nav").boundingBox();
    const toolBounds = await tools.boundingBox();
    return Math.abs(navBounds!.y - toolBounds!.y);
  }).toBeLessThan(1);
  // Once even the icon-only layout cannot fit, wrapping is the fallback.
  await header.evaluate(node => { (node as HTMLElement).style.width = "292px"; });
  await expect(header).toHaveClass(/masthead--icons/);
  await expect.poll(async () => {
    const navBounds = await header.locator("nav").boundingBox();
    const toolBounds = await tools.boundingBox();
    return toolBounds!.y > navBounds!.y + navBounds!.height;
  }).toBe(true);
  await header.evaluate(node => { (node as HTMLElement).style.removeProperty("width"); });
  await expect(header).not.toHaveClass(/masthead--icons/);
});


test("scaled headers hide labels before controls leave the viewport", async ({page}, info) => {
  await page.goto("/tasks?__locale=en");
  for (const zoom of [0.8, 1, 1.25]) {
    await page.evaluate(value => { document.documentElement.style.zoom = String(value); }, zoom);
    for (const width of [960,800,720,680,640,600,560,520,480,400,320]) {
      if (width / zoom < 320) continue; // The application has a 320 CSS-pixel minimum.
      await page.setViewportSize({width,height:800});
      await expect.poll(() => page.locator(".masthead").evaluate(header => {
        const viewport = window.innerWidth;
        const links = [...header.querySelectorAll("nav a, .masthead__tools > a, .masthead__tools button")];
        return links.every(node => {
          const rect = node.getBoundingClientRect();
          return rect.left >= -1 && rect.right <= viewport + 1;
        });
      }), {message:`controls stay visible at width ${width} and scale ${zoom}`}).toBe(true);
      if (zoom === 0.8 && width === 520) {
        const header = page.locator(".masthead");
        await expect(header).toHaveClass(/masthead--icons/);
        const nav = await header.locator("nav").boundingBox();
        const tools = await header.locator(".masthead__tools").boundingBox();
        expect(Math.abs(nav!.y - tools!.y)).toBeLessThan(1);
        await header.screenshot({path:info.outputPath("scaled-icon-header.png")});
      }
    }
  }
});

test.describe("wrapped navigation labels", () => {
  test.use({graphqlOverrides:{settings:{...settingsSnapshot,system:{...settingsSnapshot.system,language:"zh-CN"}}}});
test("wrapped navigation restores labels when its own row has room", async ({page}, info) => {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.setViewportSize({width:1100,height:800});
  await page.goto("/tasks?__locale=zh-CN");
  const header = page.locator(".masthead");
  await expect(page.locator("html")).toHaveAttribute("lang", "zh-CN");
  await expect(header).not.toHaveClass(/masthead--icons/);
  const sizes = await header.evaluate(node => {
    const brand = (node.querySelector(".masthead__brand") as HTMLElement).offsetWidth;
    const tools = (node.querySelector(".masthead__tools") as HTMLElement).offsetWidth;
    const full = node.querySelector(".masthead__measure")!.scrollWidth;
    const icons = node.querySelector(".masthead__measure--icons")!.scrollWidth;
    const gap = parseFloat(getComputedStyle(node).columnGap);
    return {fullSingle:brand + full + tools + gap * 2 + 2,
      iconsSingle:brand + icons + tools + gap * 2 + 2,
      fullWrapped:brand + full + gap + 2};
  });
  expect(sizes.fullWrapped).toBeLessThan(sizes.iconsSingle);
  const states = [
    {name:"single-labels", width:sizes.fullSingle + 20, icons:false, wrapped:false},
    {name:"single-icons", width:Math.floor((sizes.fullSingle + sizes.iconsSingle) / 2), icons:true, wrapped:false},
    {name:"wrapped-labels", width:Math.floor((sizes.iconsSingle + sizes.fullWrapped) / 2), icons:false, wrapped:true},
    {name:"wrapped-icons", width:sizes.fullWrapped - 20, icons:true, wrapped:true},
  ];
  for (const state of [...states, ...states.slice(0, -1).reverse()]) {
    await header.evaluate((node, width) => { (node as HTMLElement).style.width = `${width}px`; }, state.width);
    if (state.icons) await expect(header).toHaveClass(/masthead--icons/);
    else await expect(header).not.toHaveClass(/masthead--icons/);
    if (state.wrapped) await expect(header).toHaveClass(/masthead--wrapped/);
    else await expect(header).not.toHaveClass(/masthead--wrapped/);
    const nav = await header.locator("nav").boundingBox();
    const tools = await header.locator(".masthead__tools").boundingBox();
    expect(tools!.y > nav!.y + nav!.height).toBe(state.wrapped);
    for (const label of await header.locator("nav a span").all()) {
      if (state.icons) await expect(label).toBeHidden();
      else await expect(label).toBeVisible();
    }
    expect(await header.evaluate(node => [...node.querySelectorAll("nav a, .masthead__tools")].every(child => {
      const frame = node.getBoundingClientRect();
      const bounds = child.getBoundingClientRect();
      return bounds.left >= frame.left - 1 && bounds.right <= frame.right + 1;
    }))).toBe(true);
    await header.screenshot({path:info.outputPath(`${state.name}.png`)});
  }
  expect(errors).toEqual([]);
});

});
