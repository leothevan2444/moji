import { expect } from "@playwright/test";
import { test, homeOverview } from "./fixtures";

const actor = { id: "toolbar-actor", name: "Alex", aliasList: [], favorite: false, imagePath: null, sceneCount: 1, subscribed: false };
const profile = { performer: actor, disambiguation: null, birthdate: null, ethnicity: null, country: null, eyeColor: null, heightCm: null, rating100: null, urls: [], matchedStashBox: null, totalSceneCount: 0, stashSceneCount: 0, stashBoxSceneCount: 0, dedupedSceneCount: 0, activeTaskCount: 0, sceneStatsComplete: true, sceneStatsStale: false };
test.use({
  graphqlOverrides: {
    tasks: [...homeOverview.attentionTasks, ...homeOverview.activeTasks],
    performerWorkspace: { activities: [], performers: { items: [actor], page: 1, pageSize: 24, totalCount: 1, totalPages: 1, hasPrevPage: false, hasNextPage: false }, subscribedPerformers: [] },
    stashPerformerProfile: profile, stashPerformerDetail: profile,
    stashPerformerScenes: { items: [], page: 1, pageSize: 24, totalCount: 0, totalPages: 1, hasPrevPage: false, hasNextPage: false, stashSceneCount: 0, stashBoxCount: 0, dedupedCount: 0, totalCountExact: true, stashBoxRemoteCount: 0, stashBoxLoadedCount: 0, cacheComplete: true, cacheUpdatedAt: null, cacheStale: false },
  }
});

for (const locale of ["zh-CN", "en", "qps-ploc", "ar-XB"]) for (const path of ["/tasks", "/performers", "/performers/toolbar-actor"]) {
  test(`${locale} ${path} compresses and restores toolbar in at most three rows`, async ({ page }, info) => {
    await page.emulateMedia({ reducedMotion: "no-preference" });
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale === "zh-CN" ? "zh-CN" : "en");
    await page.goto(`${path}?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    const toolbar = page.locator(".workspace-toolbar"), toggle = toolbar.locator(".workspace-toolbar__toggle"), active = toolbar.locator(".workspace-toolbar__panel.is-active");
    await expect(toggle).toBeVisible();
    for (const selecting of [false, true, false]) {
      if (await toggle.getAttribute("aria-pressed") !== String(selecting)) await toggle.click();
      await expect(toggle).toHaveAttribute("aria-pressed", String(selecting));
      await expect(toggle.locator("svg")).toHaveAttribute("data-icon", selecting ? "xmark" : "list-check");
      for (const width of [1440, 1000, 700, 500, 400, 320, 400, 700, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await expect.poll(() => active.locator(".workspace-toolbar__row").first().evaluate(row => {
          const frame = row.getBoundingClientRect();
          const controls = [...row.children].filter(child => getComputedStyle(child).display !== "none");
          const boxes = controls.map(child => child.getBoundingClientRect()).filter(box => box.width > 0);
          const rows = new Set(boxes.map(box => Math.round(box.top)));
          return boxes.every(box => box.left >= frame.left - 1 && box.right <= frame.right + 1) && rows.size <= 3;
        })).toBe(true);
        const more = active.getByRole("button", { expanded: false });
        if (width === 320 && await more.count()) {
          await more.click();
          const menu = page.locator(".workspace-toolbar-menu");
          await expect(menu).toBeVisible();
          await menu.press("Escape");
          await expect(menu).toHaveCount(0);
          await expect(more).toBeFocused();
        }
      }
      await expect(active.locator(".workspace-toolbar__row").first()).not.toHaveClass(/is-compact/);
      await toolbar.screenshot({ path: info.outputPath(`toolbar-${selecting ? "selection" : "normal"}.png`) });
    }
    const search = active.getByRole("searchbox");
    await search.fill("ABC");
    await toggle.click(); await toggle.click();
    await expect(search).toHaveValue("ABC");
    await page.emulateMedia({ reducedMotion: "reduce" });
    await expect(active).toHaveCSS("transition-duration", "0s");
  });
}

for (const path of ["/tasks", "/performers", "/performers/toolbar-actor"]) {
  test(`${path} keeps overflow controls usable under zoom and invokes collected actions`, async ({ page }) => {
    await page.setViewportSize({width:320,height:1000});
    await page.goto(path);
    const toolbar=page.locator(".workspace-toolbar");
    for (const zoom of [0.8,1.25,2,1]) {
      await page.evaluate(value=>document.documentElement.style.zoom=String(value),zoom);
      await expect.poll(()=>toolbar.evaluate(node=>{
        const frame=node.getBoundingClientRect();
        return [...node.querySelectorAll<HTMLElement>(".workspace-toolbar__panel.is-active > .workspace-toolbar__row > *")]
          .filter(child=>getComputedStyle(child).display!=="none")
          .every(child=>{const box=child.getBoundingClientRect();return box.left>=frame.left-1&&box.right<=frame.right+1;});
      })).toBe(true);
      const more=toolbar.locator(".workspace-toolbar__panel.is-active button.workspace-toolbar__more");
      if (await more.count()) {
        await more.scrollIntoViewIfNeeded();
        // Allow container and visual viewport ResizeObservers to settle after changing zoom.
        await page.evaluate(()=>new Promise<void>(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve()))));
        await more.focus();
        await expect.poll(()=>more.evaluate(node=>{
          const box=node.getBoundingClientRect();
          return node.contains(document.elementFromPoint(box.x+box.width/2,box.y+box.height/2));
        })).toBe(true);
        await more.press("Enter");
        const menu=page.locator(".workspace-toolbar-menu");
        await expect(menu).toBeVisible();
        const box=(await menu.boundingBox())!;
        expect(box.x).toBeGreaterThanOrEqual(0);
        expect(box.x+box.width).toBeLessThanOrEqual(321);
        await menu.press("Escape");
        await expect(more).toBeFocused();
      }
    }
    if(path==="/performers") {
      await toolbar.locator(".workspace-toolbar__panel.is-active button.workspace-toolbar__more").click();
      await page.locator(".workspace-toolbar-menu").getByRole("link",{name:/Release review/}).click();
      await expect(page).toHaveURL(/\/performers\/review/);
    }
    if(path==="/tasks") {
      await toolbar.locator(".workspace-toolbar__toggle").click();
      let selectPage=toolbar.getByRole("button",{name:/^Select this page/});
      if(!await selectPage.isVisible()) {
        await toolbar.locator(".workspace-toolbar__panel.is-active button.workspace-toolbar__more").click();
        selectPage=page.locator(".workspace-toolbar-menu").getByRole("button",{name:/^Select this page/});
      }
      await selectPage.click();
      await expect(page.locator(".task-card__checkbox").first()).toBeChecked();
    }
  });
}
