import { expect, type Locator, type TestInfo } from "@playwright/test";
import { test, settingsSnapshot, homeOverview } from "./fixtures";

const actor = {id:"responsive-actor",name:"Alex",aliasList:[],favorite:false,imagePath:null,sceneCount:1,subscribed:false};
const profile = {performer:actor,disambiguation:null,birthdate:null,ethnicity:null,country:null,eyeColor:null,heightCm:null,rating100:null,urls:[],matchedStashBox:null,totalSceneCount:0,stashSceneCount:0,stashBoxSceneCount:0,dedupedSceneCount:0,activeTaskCount:0,sceneStatsComplete:true,sceneStatsStale:false};
const film = {endpoint:"https://metadata.example/graphql",sceneId:"responsive-film",code:"RESP-1",title:"Synthetic film",date:"2026-01-01",studio:"Synthetic studio",performers:[],imageUrl:"",url:"",durationSeconds:60,confirmationDigest:"confirmed-digest"};
const catalog = {
  tasks:[...homeOverview.attentionTasks,...homeOverview.activeTasks],
  performerWorkspace:{activities:[],performers:{items:[actor],page:1,pageSize:24,totalCount:240,totalPages:10,hasPrevPage:false,hasNextPage:true},subscribedPerformers:[]},
  stashPerformerProfile:profile,stashPerformerDetail:profile,
  stashPerformerScenes:{items:[],page:1,pageSize:24,totalCount:0,totalPages:1,hasPrevPage:false,hasNextPage:false,stashSceneCount:0,stashBoxCount:0,dedupedCount:0,totalCountExact:true,stashBoxRemoteCount:0,stashBoxLoadedCount:0,cacheComplete:true,cacheUpdatedAt:null,cacheStale:false},
  taskFilmSources:()=>[{name:"Synthetic",endpoint:film.endpoint}],taskFilmCandidates:()=>[film],taskFilm:()=>film,
};

async function sameRow(locator:Locator) {
  await expect(locator).toBeVisible();
  await expect.poll(()=>locator.evaluate(node=>{
    const boxes=[...node.children].filter(child=>getComputedStyle(child).display!=="none").map(child=>child.getBoundingClientRect()).filter(box=>box.width>0);
    return boxes.length>1 && Math.max(...boxes.map(box=>box.top)) < Math.min(...boxes.map(box=>box.bottom));
  })).toBe(true);
}
async function contained(locator:Locator) {
  await expect.poll(()=>locator.evaluate(node=>{
    const frame=node.getBoundingClientRect();
    return node.scrollWidth<=node.clientWidth+1 && [...new Set([...node.children,...node.querySelectorAll("input,select,button,a")])].filter(child=>getComputedStyle(child).display!=="none").every(child=>{
      const box=child.getBoundingClientRect();
      return !box.width || (box.left>=frame.left-1 && box.right<=frame.right+1);
    });
  })).toBe(true);
}
async function capture(locator:Locator,info:TestInfo,name:string) {
  await locator.screenshot({path:info.outputPath(`${name}.png`)});
}

for(const locale of ["zh-CN","en","qps-ploc","ar-XB"] as const) {
 test.describe(locale,()=>{
  test.use({graphqlOverrides:{...catalog,settings:{...settingsSnapshot,system:{...settingsSnapshot.system,language:locale==="zh-CN"?"zh-CN":"en"}}}});
  const route=(path:string)=>`${path}${path.includes("?")?"&":"?"}__locale=${locale}`;
  test.beforeEach(async({page})=>{await page.emulateMedia({colorScheme:locale==="en"||locale==="ar-XB"?"dark":"light"});});

  test("search keeps a useful query field and wraps its submit only when needed",async({page},info)=>{
    await page.setViewportSize({width:560,height:1000});await page.goto(route("/"));
    await expect(page.locator("html")).toHaveAttribute("lang",locale);
    const form=page.locator(".home-search");
    await sameRow(form);await contained(form);
    expect(await form.locator("input").evaluate(node=>node.clientWidth)).toBeGreaterThan(140);
    await capture(form,info,"search-560");
    await page.setViewportSize({width:320,height:1000});await contained(form);
    expect(await form.locator("input").evaluate(node=>node.clientWidth)).toBeGreaterThan(50);
    await capture(form,info,"search-320");
    await page.setViewportSize({width:560,height:1000});await sameRow(form);
    await form.getByRole("combobox").selectOption("indexers");
    await form.locator("input").fill("RESP-1");await form.locator("input").press("Enter");
    await expect(page).toHaveURL(/\/discover\?.*source=indexers/);
  });

  test("page headings group actions and preview help use available width",async({page},info)=>{
    await page.setViewportSize({width:800,height:1000});await page.goto(route("/tasks"));
    const heading=page.locator(".band-head").first(),group=page.locator(".task-group-section__head").first();
    await sameRow(heading);await sameRow(group);await capture(group,info,"task-group-800");
    await page.setViewportSize({width:320,height:1000});await contained(heading);await contained(group);
    await page.setViewportSize({width:800,height:1000});await sameRow(heading);await sameRow(group);
    await page.goto(route("/discover"));const preview=page.locator(".section-band--preview .preview-panel");
    await sameRow(preview);await capture(preview,info,"preview-800");
    await page.setViewportSize({width:320,height:1000});await contained(preview);
  });

  test("performer controls wrap in groups and compact pagination stays inline",async({page},info)=>{
    await page.setViewportSize({width:800,height:1000});await page.goto(route("/performers"));
    const toolbar=page.locator(".toolbar-inline--performer-list"),actions=toolbar.locator(".performer-list-actions");
    for(const width of [800,320,960,800]) {
      await page.setViewportSize({width,height:1000});await contained(toolbar);await sameRow(actions);await contained(toolbar.locator(".performer-list-options"));
      if(width>=800) await sameRow(toolbar.locator(".performer-list-options"));
      await capture(toolbar,info,`performer-toolbar-${width}`);
    }
    await page.setViewportSize({width:600,height:1000});const pagination=page.locator(".pagination-bar--performers").first();
    await sameRow(pagination);await capture(pagination,info,"pagination-600");
    for(const width of [320,600]){await page.setViewportSize({width,height:1000});await contained(pagination);}
    await page.setViewportSize({width:800,height:1000});await page.goto(route("/performers/responsive-actor"));
    const band=page.locator(".band-head").first(),identity=page.locator(".performer-detail-hero__topline");
    await sameRow(band);await sameRow(identity);await capture(identity,info,"identity-800");
    await page.setViewportSize({width:320,height:1000});await contained(identity);await contained(band);
  });

  test("settings source and cache actions stay inline when they fit",async({page},info)=>{
    await page.setViewportSize({width:700,height:1000});await page.goto(route("/settings/stash?panel=metadata"));
    const source=page.locator(".stashbox-source__head");await sameRow(source);await capture(source,info,"metadata-700");
    await page.setViewportSize({width:320,height:1000});await contained(source);
    await page.setViewportSize({width:560,height:1000});await page.goto(route("/settings/indexers?panel=cache"));
    const cache=page.locator(".indexer-cache-actions");await sameRow(cache);await capture(cache,info,"cache-560");
    await page.setViewportSize({width:320,height:1000});await contained(cache);
    await sameRow(cache.locator(".indexer-actions"));
  });

  test("film confirmation actions stay on one line when room allows",async({page},info)=>{
    await page.setViewportSize({width:640,height:1000});await page.goto(route("/discover?source=indexers&q=RESP-1&run=search-one"));
    const card=page.locator(".candidate-card--indexer").first();await expect(card).toBeVisible();await card.getByRole("button").last().click();
    await page.locator(".film-confirmation__candidate").first().click();
    const actions=page.locator(".film-confirmation .inline-actions");await sameRow(actions);await contained(actions);await capture(actions,info,"confirmation-640");
    await page.setViewportSize({width:320,height:1000});await contained(actions);
    for(const button of await actions.getByRole("button").all()) expect((await button.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    await capture(actions,info,"confirmation-320");
  });

  test.describe("touch",()=>{
    test.use({hasTouch:true});
    test("wide task badges remain beside identity without being forced down",async({page},info)=>{
      await page.setViewportSize({width:1440,height:1000});await page.goto(route("/tasks"));
      const head=page.locator(".task-card__head").first();await sameRow(head);await contained(head);await capture(head,info,"touch-task-1440");
      await page.setViewportSize({width:320,height:1000});await contained(head);await capture(head,info,"touch-task-320");
    });
  });
 });
}
