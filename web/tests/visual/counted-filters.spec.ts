import { expect } from "@playwright/test";
import { test, homeOverview } from "./fixtures";

const actors = [
  {id:"a",name:"Alpha",aliasList:[],favorite:true,subscribed:true},
  {id:"b",name:"Beta",aliasList:["Alpha alias"],favorite:true,subscribed:false},
  {id:"c",name:"Gamma",aliasList:[],favorite:false,subscribed:true},
].map(actor=>({...actor,imagePath:null,sceneCount:0}));
const matches = (actor:typeof actors[number], filter:string) => filter==="SUBSCRIBED"?actor.subscribed:filter==="FAVORITE"?actor.favorite:filter==="REVIEW"?actor.id==="b":filter==="ERROR"?actor.id==="a":true;
const performerWorkspace = ({search="",page=1,pageSize=24,options}:{search?:string;page?:number;pageSize?:number;options?:{filter:string}}) => {
  const scope=actors.filter(actor=>[actor.name,...actor.aliasList].some(text=>text.toLowerCase().includes((search??"").toLowerCase().trim())));
  const items=scope.filter(actor=>matches(actor,options?.filter??"ALL"));
  const totalPages=Math.max(1,Math.ceil(items.length/pageSize));page=Math.min(page,totalPages);
  return {activities:[],subscribedPerformers:[],filterCounts:{total:scope.length,subscribed:scope.filter(a=>matches(a,"SUBSCRIBED")).length,favorite:scope.filter(a=>matches(a,"FAVORITE")).length,review:scope.filter(a=>matches(a,"REVIEW")).length,error:scope.filter(a=>matches(a,"ERROR")).length},
    performers:{items:items.slice((page-1)*pageSize,page*pageSize),page,pageSize,totalPages,totalCount:items.length,hasPrevPage:page>1,hasNextPage:page<totalPages}};
};
test.use({graphqlOverrides:{tasks:[...homeOverview.attentionTasks,...homeOverview.activeTasks],performerWorkspace}});

test("actor filters retain search, count aliases and reset pagination while preserving sort",async({page},info)=>{
  await page.goto("/performers?q=Alpha&sort=name&page=2");
  const group=page.getByRole("toolbar",{name:"Filter performers by scope"});
  await expect(group.getByRole("button",{name:"All performers 2"})).toHaveAttribute("aria-pressed","true");
  await expect(group.getByRole("button",{name:"Subscribed 1"})).toBeVisible();
  await expect(group.getByRole("button",{name:"Stash favorites 2"})).toBeVisible();
  await group.getByRole("button",{name:"Awaiting review 1"}).click();
  await expect(page.locator(".profile-card")).toHaveCount(1);
  await expect(page.locator(".profile-card")).toContainText("Beta");
  const url=new URL(page.url());
  expect(url.searchParams.get("q")).toBe("Alpha");expect(url.searchParams.get("sort")).toBe("name");
  expect(url.searchParams.get("filter")).toBe("review");expect(url.searchParams.get("page")??"1").toBe("1");
  await expect(group.getByRole("button",{name:"All performers 2"})).toBeVisible();
  await page.screenshot({path:info.outputPath("counted-actor-filters.png")});
  await group.getByRole("button",{name:"All performers 2"}).click();
  await expect(page.locator(".profile-card")).toHaveCount(2);
  await page.getByRole("searchbox",{name:"Search Stash performers by name or alias"}).fill("Gamma");
  await expect(group.getByRole("button",{name:"Stash favorites 0"})).toBeVisible();
  await group.getByRole("button",{name:"Stash favorites 0"}).click();
  await expect(page.locator(".profile-card")).toHaveCount(0);
  await expect(group.getByRole("button",{name:"Stash favorites 0"})).toHaveAttribute("aria-pressed","true");
});

test("task filters retain the time scope and always expose a clear selected category",async({page},info)=>{
  await page.goto("/tasks?status=completed&completedWithin=24h&sort=progress");
  const group=page.getByRole("toolbar",{name:"Filter tasks by status"});
  await expect(group.getByRole("button",{name:"Completed 0"})).toHaveAttribute("aria-pressed","true");
  await group.getByRole("button",{name:"In progress 0"}).click();
  const url=new URL(page.url());
  expect(url.searchParams.get("completedWithin")).toBe("24h");expect(url.searchParams.get("sort")).toBe("progress");
  await expect(group.getByRole("button",{name:"In progress 0"})).toHaveAttribute("aria-pressed","true");
  await expect(page.getByRole("combobox",{name:"Task category"})).toHaveCount(0);
  await page.screenshot({path:info.outputPath("counted-task-filters.png")});
});

for (const locale of ["zh-CN","en","qps-ploc","ar-XB"]) {
 test(`${locale} counted filters keep filters and badges visible at narrow widths`,async({page},info)=>{
  await page.addInitScript(value=>localStorage.setItem("moji:locale",value),locale==="zh-CN"?"zh-CN":"en");
  await page.emulateMedia({colorScheme:locale==="en"||locale==="ar-XB"?"dark":"light"});
  for(const path of ["tasks","performers"]) {
   await page.goto(`/${path}?__locale=${locale}`);
   await expect(page.locator("html")).toHaveAttribute("lang",locale);
   const heading=page.locator(`.filter-count-heading--${path}`), group=heading.getByRole("toolbar");
   for(const width of [1440,800,600,393,320,600,1440]) {
    await page.setViewportSize({width,height:1000});
    await expect.poll(()=>heading.evaluate(node=>node.scrollWidth<=node.clientWidth+1)).toBe(true);
    if(await heading.evaluate(node=>node.classList.contains("is-compact"))) await expect(group.locator(".is-active .filter-count__label")).toBeHidden();
    else await expect(group.locator(".is-active .filter-count__label")).toBeVisible();
    for(const badge of await group.locator(".filter-count__number").all()) await expect(badge).toBeVisible();
    const rows=await group.getByRole("button").evaluateAll(buttons=>new Set(buttons.map(button=>Math.round(button.getBoundingClientRect().top))).size);
    expect(rows).toBeLessThanOrEqual(2);
    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
    for(const button of await group.getByRole("button").all()) {
     await expect(button).toHaveAttribute("title",/.+/);
     if(info.project.name==="mobile") expect((await button.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    }
    if(width===320) await heading.screenshot({path:info.outputPath(`${locale}-${path}-320.png`)});
   }
  }
 });
}
