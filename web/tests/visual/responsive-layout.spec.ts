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
    return node.scrollWidth<=node.clientWidth+1 && [...new Set([...node.children,...node.querySelectorAll("input,select,button,a")])].filter(child=>getComputedStyle(child).display!=="none" && getComputedStyle(child).visibility!=="hidden").every(child=>{
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
    await page.locator(".workspace-toolbar__panel.is-active button[data-control=group]").click();
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
    const toolbar=page.locator(".workspace-toolbar");
    for(const width of [800,320,960,800]) {
      await page.setViewportSize({width,height:1000});await contained(toolbar);
      await capture(toolbar,info,`performer-toolbar-${width}`);
    }
    await page.setViewportSize({width:600,height:1000});const pagination=page.locator(".pagination").first();
    await sameRow(pagination.locator(".pagination__viewport > .pagination__row"));await capture(pagination,info,"pagination-600");
    for(const width of [320,600]){await page.setViewportSize({width,height:1000});await expect.poll(()=>pagination.evaluate(node=>node.scrollWidth<=node.clientWidth+1)).toBe(true);await sameRow(pagination.locator(".pagination__viewport > .pagination__row"));}
    await page.setViewportSize({width:800,height:1000});await page.goto(route("/performers/responsive-actor"));
    const band=page.locator(".band-head").first(),identity=page.locator(".performer-detail-hero__topline");
    await sameRow(band);await sameRow(identity);await capture(identity,info,"identity-800");
    await page.setViewportSize({width:320,height:1000});await contained(identity);await contained(band);await expect.poll(()=>page.locator(".pagination").first().evaluate(node=>node.scrollWidth<=node.clientWidth+1)).toBe(true);
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

// Exercise every stage with large text as well as the normal translated headings.
// Widths come from rendered content so this guards the order, not fixed breakpoints.
for (const variant of ["tasks", "performers"] as const) {
 test.describe(`responsive ${variant}`, () => {
  test.use({graphqlOverrides:{...catalog,settings:{...settingsSnapshot,system:{...settingsSnapshot.system,language:"zh-CN"}}}});
  test("icons precede wrapping and wrapped labels return when they fit",async({page},info)=>{
    await page.setViewportSize({width:2000,height:1000});
    await page.goto(variant==="tasks"?"/tasks":"/performers");
    const heading=page.locator(`.filter-count-heading--${variant}`);
    await expect(heading).toBeVisible();
    const largeText=await page.addStyleTag({content:".filter-count-heading h2 {font-size:128px!important}"});
    const widths=await heading.evaluate(node=>{
      const probes=node.querySelector(".filter-count__measurement")!;
      const title=probes.querySelector("h2")!.scrollWidth;
      const groups=[...probes.querySelectorAll(".filter-count__items")].map(item=>item.scrollWidth);
      const gap=parseFloat(getComputedStyle(node).columnGap);
      return {groups,title,gap};
    });
    const modes=["inline","stacked","selected","icons"];
    const thresholds=widths.groups.map(width=>Math.ceil(widths.title+widths.gap+width+2));
    expect(thresholds[3]).toBeGreaterThan(widths.groups[0]+4);
    const stages=[
      {width:thresholds[0]+2,density:"inline",wrap:false},
      ...thresholds.slice(0,3).map((width,index)=>({width:width-3,density:modes[index+1],wrap:false})),
      {width:thresholds[3]-3,density:"inline",wrap:true},
      ...widths.groups.slice(0,3).map((width,index)=>({width:width-1,density:modes[index+1],wrap:true})),
    ];
    const group=heading.locator(":scope > .filter-count__items");
    const names=await group.locator(":scope > *").evaluateAll(items=>items.map(item=>item.getAttribute("aria-label")));
    if(variant==="performers") await group.getByRole("button").first().focus();
    for(const [index,stage] of [...stages,...stages.slice().reverse()].entries()) {
      await heading.evaluate((node,width)=>{(node as HTMLElement).style.width=`${width}px`;},stage.width);
      await expect.poll(()=>heading.evaluate(node=>({density:node.querySelector("[role=toolbar]")!.getAttribute("data-density"),wrap:node.classList.contains("is-wrapped")}))).toEqual({density:stage.density,wrap:stage.wrap});
      await contained(heading);
      await expect(group.locator("svg")).toHaveCount(5);
      if(stage.density==="icons") await expect(group.locator(".is-active .filter-count__label")).toBeHidden();
      else await expect(group.locator(".is-active .filter-count__label")).toBeVisible();
      for(const label of await group.locator(".filter-count__item:not(.is-active) .filter-count__label").all()) {
        if(["selected","icons"].includes(stage.density)) await expect(label).toBeHidden();
        else await expect(label).toBeVisible();
      }
      const geometry=await group.getByRole("button").first().evaluate(node=>{
        const identity=node.querySelector(".filter-count__identity")!,number=node.querySelector(".filter-count__number")!;
        const a=identity.getBoundingClientRect(),b=number.getBoundingClientRect();
        return {below:b.top>=a.bottom-1,right:b.left>=a.right-1,font:parseFloat(getComputedStyle(number).fontSize)};
      });
      expect(geometry.below).toBe(stage.density!=="inline");
      if(stage.density==="inline") expect(geometry.right).toBe(true);
      expect(geometry.font).toBe(stage.density==="inline"?12:11);
      await expect(group.locator(".filter-count__number")).toHaveCount(5);
      expect(await group.locator(":scope > *").evaluateAll(items=>items.map(item=>item.getAttribute("aria-label")))).toEqual(names);
      for (const [index, name] of names.entries()) {
        await expect(group.getByRole("button").nth(index)).toHaveAttribute("title", new RegExp(name!.split(" ")[0]));
      }
      if(index<stages.length) await capture(heading,info,`${variant}-stage-${index}`);
    }
    await largeText.evaluate(node=>{node.parentNode?.removeChild(node);});
    await heading.evaluate(node=>{(node as HTMLElement).style.removeProperty("width");});
    await expect(heading.locator(":scope > h2")).toHaveCSS("font-size","24px");
    for(const zoom of [0.8,1,1.25]) {
      await page.evaluate(value=>{document.documentElement.style.zoom=String(value);},zoom);
      for(const width of [1200,800,600,400,320,800,1200]) {
        if(width/zoom<320) continue;
        await page.setViewportSize({width,height:1000});
        await contained(heading);
      }
    }
    if(variant==="performers") {
      await group.getByRole("button",{name:/^Stash 收藏 /}).click();
      await expect(group.getByRole("button",{name:/^Stash 收藏 /})).toHaveAttribute("aria-pressed","true");
    }
  });
 });
}
