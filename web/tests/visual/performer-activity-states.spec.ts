import { expect } from "@playwright/test";
import { test } from "./fixtures";

const actor={id:"activity-actor",name:"Activity performer",aliasList:[],favorite:false,sceneCount:0,imagePath:null,subscribed:true};
const profile={performer:actor,urls:[]};
const common={
 stashPerformerProfile:profile,
 stashPerformerDetail:{...profile,totalSceneCount:0,stashSceneCount:0,stashBoxSceneCount:0,dedupedSceneCount:0,activeTaskCount:0,sceneStatsComplete:true,sceneStatsStale:false},
 stashPerformerScenes:{items:[],page:1,pageSize:24,totalCount:0,totalPages:1,hasPrevPage:false,hasNextPage:false,stashSceneCount:0,stashBoxCount:0,dedupedCount:0,totalCountExact:true,stashBoxRemoteCount:0,stashBoxLoadedCount:0,cacheComplete:true,cacheStale:false,cacheUpdatedAt:null},
};
const cases=[
 ["NOT_CHECKED","Not assessed","Not assessed yet"],
 ["INCOMPLETE_CATALOG","Partial catalog","Partial catalog"],
 ["INVALID_DATES","Incomplete dates","Release dates are incomplete"],
 ["INVALID_MEMBERSHIP","Unverified link","Performer links are unverified"],
 ["CHECK_FAILED","Check failed","Release check failed"],
] as const;
for(const [status,label,explanation] of cases){
 test.describe(status,()=>{
  test.use({graphqlOverrides:{...common,performerActivity:({id}:{id:string})=>({id,evaluationStatus:status,value:null,rank:null,lowerBound:false,observedAt:status==="NOT_CHECKED"||status==="CHECK_FAILED"?null:"2026-01-15T12:00:00Z",lastSuccessAt:null,nextCheckAt:null})}});
  test("explains missing ratings using the source evidence",async({page})=>{
   await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));
   await page.goto("/performers/activity-actor");
   const section=page.getByRole("region",{name:"Activity level",exact:true});
   const value=section.locator("dd").last();
   await expect(value).toHaveText(label);
   await expect(value).toHaveAttribute("title",new RegExp(explanation));
   await expect(section.locator(".performer-activity-icons")).toHaveAttribute("title",new RegExp(explanation));
   await expect(section.locator(".performer-activity-icons svg")).toHaveCount(5);
   await expect(section.locator(".performer-activity-icons .is-active")).toHaveCount(0);
   expect(await section.evaluate(el=>el.scrollWidth<=el.clientWidth)).toBe(true);
   if(page.viewportSize()!.width>1100)expect(await section.evaluate(el=>el.getBoundingClientRect().height)).toBeLessThanOrEqual(66);
   await page.setViewportSize({width:800,height:900});
   expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  });
 });
}

for (const failed of [false,true]) {
 test.describe(failed?"read failure":"partial statistics",()=>{
  test.use({graphqlOverrides:{...common,
   stashPerformerDetail:{...common.stashPerformerDetail,sceneStatsComplete:false,activeTaskCount:2},
   performerActivity:({id}:{id:string})=>{return {id,evaluationStatus:"READY",value:3.25,rank:4,lowerBound:false,observedAt:"2026-01-15T12:00:00Z",lastSuccessAt:"2026-01-15T12:00:00Z",nextCheckAt:null};},
  }});
  test("keeps warnings inside the fixed information layout",async({page})=>{
   if(failed)await page.route("**/graphql",async route=>{
    const request=route.request().postDataJSON();
    if(request.operationName==="PerformerActivity" || request.query?.includes("query PerformerActivity("))await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({data:null,errors:[{message:"read unavailable"}]})});
    else await route.fallback();
   });
   await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));
   await page.goto("/performers/activity-actor");
   const section=page.getByRole("region",{name:"Activity level",exact:true});
   await expect(section.locator(".performer-activity-icons svg")).toHaveCount(5);
   if(failed)await expect(section.getByRole("button",{name:/Unable to update/})).toBeVisible();
   const tasks=page.getByRole("region",{name:"Scenes and tasks",exact:true});
   await expect(tasks.locator("dd").last()).toHaveText("≥ 2");
   await expect(tasks.locator("dd").last()).toHaveAttribute("title",/loaded/);
   await expect(page.locator(".performer-summary-note")).toHaveCount(0);
   expect(await section.evaluate(el=>el.getBoundingClientRect().height)).toBe(64);
   expect(await tasks.evaluate(el=>el.getBoundingClientRect().height)).toBe(64);
   const fonts=await section.locator("dt").evaluateAll(els=>els.map(el=>{const s=getComputedStyle(el);return s.fontSize+"/"+s.fontWeight;}));
   expect(new Set(fonts).size).toBe(1);
  });
 });
}

test.describe("confirmed minimum and help",()=>{
 test.use({graphqlOverrides:{...common,
  performerActivity:({id}:{id:string})=>({id,evaluationStatus:"READY",value:15.151186205122569,rank:5,lowerBound:true,observedAt:"2026-01-15T12:00:00Z",lastSuccessAt:"2026-01-15T12:00:00Z",nextCheckAt:null}),
 }});
 test("opens the matching help topic and returns to the actor",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","zh-CN"));
  await page.goto("/performers/activity-actor");
  const section=page.getByRole("region",{name:"活跃度",exact:true});
  await expect(section.locator("dd").last()).toHaveText("≥ 15.15");
  await expect(section.locator(".is-active")).toHaveCount(5);
  await expect(section.locator("dd").last()).toHaveAttribute("title",/活跃值下限/);
  await expect(section.locator("dd").last()).not.toHaveAttribute("title",/评分更新/);
  const open=section.getByRole("button",{name:"查看活跃度与活跃值说明",exact:true});
  await open.click();
  const dialog=page.getByRole("dialog");
  await expect(dialog.locator(".help__tab.is-active")).toHaveText("活跃度与活跃值");
  await expect(dialog.getByRole("heading",{name:"为什么活跃值带有 ≥"})).toBeVisible();
  await expect(dialog.locator(".help__card")).toContainText("每 24 小时检查");
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(open).toBeFocused();
  await expect(page).toHaveURL(/\/performers\/activity-actor$/);
 });
});
