import { expect } from "@playwright/test";
import { test, homeOverview } from "./fixtures";

const actor={id:'audit-actor',name:'Example Performer With A Longer Display Name',aliasList:['Example alias'],favorite:true,imagePath:null,sceneCount:3,subscribed:false};
const profile={performer:actor,disambiguation:null,birthdate:null,ethnicity:null,country:null,eyeColor:null,heightCm:null,rating100:null,urls:[],matchedStashBox:null,totalSceneCount:3,stashSceneCount:1,stashBoxSceneCount:3,dedupedSceneCount:3,activeTaskCount:1,sceneStatsComplete:true,sceneStatsStale:false};
const scene={key:'scene-0',primarySource:'STASHBOX',sourceSceneId:'scene-0',code:'AUDIT-101',title:'Example film title used to review selection states',date:'2026-01-01',studioName:'Example Studio',performerCount:2,tagCount:0,performers:[],tags:[],imageUrl:null,url:'https://example.test/film',inLibrary:false,matchedStashSceneId:null,hasStashSource:false,hasStashBoxSource:true,stashBoxSceneId:'scene-0',stashBoxEndpoint:'https://example.test/graphql',sourceLabels:['StashBox'],stashIds:[],mojiTask:null};
const scenes=[scene,{...scene,key:'scene-1',code:'AUDIT-102',inLibrary:true},{...scene,key:'scene-2',code:'AUDIT-103',mojiTask:{id:'audit-task',stage:'DOWNLOADING',stageStatus:'RUNNING',stageLabel:'Downloading',stageStatusLabel:'Running',progress:0.5}}];
test.use({graphqlOverrides:{tasks:[...homeOverview.attentionTasks,...homeOverview.activeTasks],performerWorkspace:{activities:[],performers:{items:[actor,{...actor,id:'audit-actor-2',name:'Another Performer',favorite:false}],page:1,pageSize:24,totalCount:2,totalPages:1,hasPrevPage:false,hasNextPage:false},subscribedPerformers:[]},stashPerformerProfile:profile,stashPerformerDetail:profile,stashPerformerScenes:{items:scenes,page:1,pageSize:24,totalCount:3,totalPages:1,hasPrevPage:false,hasNextPage:false,stashSceneCount:1,stashBoxCount:3,dedupedCount:3,totalCountExact:true,stashBoxRemoteCount:3,stashBoxLoadedCount:3,cacheComplete:true,cacheUpdatedAt:null,cacheStale:false}}});

const views = [
  { id: "tasks", path: "/tasks", card: ".task-card", input: ".task-card__checkbox" },
  { id: "actors", path: "/performers", card: ".profile-card", input: ".performer-card-selector" },
  { id: "scenes", path: "/performers/audit-actor", card: ".performer-scene-card", input: ".performer-scene-card__selector" },
];
for (const theme of ["light", "dark"]) for (const view of views) {
  test(`${view.id} ${theme} shares stable selection geometry and keyboard behavior`, async ({ page }) => {
    const locale = theme === "dark" ? "zh-CN" : "en";
    await page.addInitScript(({ theme, locale }) => {
      localStorage.setItem("moji:theme", theme);
      localStorage.setItem("moji:locale", locale);
    }, { theme, locale });
    await page.goto(view.path);
    const card = page.locator(view.card).first();
    const toggle = page.locator(".workspace-toolbar__toggle");
    const geometry = () => card.evaluate(node => {
      const card = node.getBoundingClientRect(), title = node.querySelector("h3")!.getBoundingClientRect();
      return { width: card.width, height: card.height, titleWidth: title.width, titleLeft: title.left - card.left, titleTop: title.top - card.top };
    });
    for (const width of [1440, 600, 393, 320, 1440]) {
      await page.setViewportSize({ width, height: 1000 });
      await expect(card).toBeVisible();
      const normal = await geometry();
      await toggle.click();
      const checkbox = card.getByRole("checkbox");
      await expect(checkbox).toBeVisible();
      await expect(checkbox).not.toBeChecked();
      await expect(checkbox).toHaveCSS("opacity", "1");
      await expect(checkbox.locator("..")).toHaveCSS("opacity", "1");
      if (view.id === "tasks") {
        await expect.poll(async () => {
          const selected = await geometry();
          return { width: selected.width, titleShift: selected.titleLeft - normal.titleLeft };
        }).toEqual({ width: normal.width, titleShift: 32 });
      } else {
        await expect.poll(geometry).toEqual(normal);
      }
      const selection = await geometry();
      expect(await checkbox.evaluate(node => ({ width: node.getBoundingClientRect().width, height: node.getBoundingClientRect().height }))).toEqual({ width: 24, height: 24 });
      expect(await checkbox.locator("..").evaluate(node => ({ width: node.getBoundingClientRect().width, height: node.getBoundingClientRect().height }))).toEqual({ width: 44, height: 44 });
      const control = await checkbox.boundingBox(), bounds = await card.boundingBox();
      expect(control!.x - bounds!.x).toBeLessThan(view.id === "actors" ? 15 : 28);
      expect(control!.y - bounds!.y).toBeLessThan(view.id === "actors" ? 15 : 28);
      await checkbox.check();
      await expect(checkbox).toBeChecked();
      await expect(card).toHaveClass(/is-selected/);
      await expect.poll(geometry).toEqual(selection);
      await expect(card).toHaveCSS("transform", "none");
      if (width === 320) await expect(card).toHaveScreenshot(`${view.id}-${theme}-selection.png`);
      await checkbox.uncheck();
      await checkbox.focus();
      await checkbox.press("Space");
      await expect(checkbox).toBeChecked();
      await expect(checkbox).toHaveCSS("outline-style", "solid");
      await card.locator("h3").click();
      await expect(checkbox).not.toBeChecked();
      if (view.id === "tasks") await expect(card.getByRole("button", { name: /Delete task|删除任务/ })).toHaveCount(0);
      if (view.id === "scenes") {
        await expect(card.getByRole("button", { name: /Create task|创建任务/ })).toHaveCount(0);
        for (const index of [1, 2]) {
          const unavailable = page.locator(view.card).nth(index).getByRole("checkbox");
          await expect(unavailable).toBeDisabled();
          await expect(unavailable).toBeVisible();
          await expect(unavailable).toHaveCSS("opacity", "0.45");
          await expect(unavailable).toHaveAttribute("aria-description", /.+/);
        }
        await expect(card.locator(".performer-scene-card__media")).toHaveCSS("background-color", "rgba(148, 163, 184, 0.12)");
        expect(await card.locator(".performer-scene-card__media").evaluate(node => getComputedStyle(node, "::after").content)).toBe("none");
      }
      await toggle.click();
      await expect(card.getByRole("checkbox")).toHaveCount(0);
      await expect.poll(geometry).toEqual(normal);
    }
    await page.emulateMedia({ reducedMotion: "reduce" });
    await toggle.click();
    await expect(card.locator(".card-selection-control")).toHaveCSS("animation-name", "none");
    if (view.id === "actors") {
      await card.getByRole("checkbox").check();
      await card.locator("h3").click();
      await expect(card.getByRole("checkbox")).not.toBeChecked();
    }
  });
}
