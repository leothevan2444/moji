import { expect } from "@playwright/test";
import { test, torrentSearchSnapshot, searchCandidate, homeOverview } from "./fixtures";
const task=homeOverview.attentionTasks[1];
const base=`/tasks/${task.id}/resolve`;
test.use({graphqlOverrides:{tasks:[task]}});
test("task resolution restores the run URL without another search",async({page})=>{
 const starts:unknown[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="StartTorrentSearch")starts.push(body.variables.input)}});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(base);await expect(page.locator(".sourcing-candidate")).toHaveCount(1);await expect(page).toHaveURL(/sourcingRun=task-search-MOJI-102/);expect(starts).toEqual([{keywords:task.code,taskId:task.id,refresh:false}]);
 await page.reload();await expect(page.locator(".sourcing-candidate")).toHaveCount(1);expect(starts).toHaveLength(1);await expect(page.getByRole("textbox",{name:"Magnet link"})).toBeEnabled();
});
test("selecting a candidate sends task version and identities, not a download URL",async({page})=>{
 const selected:unknown[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="ResolveTaskSourcingCandidate")selected.push(body.variables.input)}});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(base);await page.getByRole("button",{name:`Select: ${searchCandidate.title}`}).click();await expect.poll(()=>selected.length).toBe(1);expect(selected).toEqual([{taskId:task.id,searchId:`task-search-${task.id}`,candidateId:searchCandidate.id,expectedRevision:1}]);
});
test("an expired task run does not admit a replacement search on reload",async({page})=>{
 let starts=0;await page.route("**/graphql",async route=>{const body=route.request().postDataJSON();if(body.operationName==="StartTorrentSearch")starts++;if(body.operationName==="TorrentSearch"){await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({data:null,errors:[{message:"redacted",extensions:{code:"TORRENT_SEARCH_EXPIRED"}}]})});return}await route.fallback()});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(base+`?sourcingTask=${task.id}&sourcingRun=expired`);await expect(page.getByText("These search results expired. Start a new search.")).toBeVisible();await expect(page.getByRole("textbox",{name:"Magnet link"})).toBeEnabled();expect(starts).toBe(0);
});
test("a foreign task run cannot be selected",async({page})=>{
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(base+`?sourcingTask=${task.id}&sourcingRun=search-one`);await expect(page.locator(".sourcing-candidate")).toHaveCount(0);await expect(page.getByText("This search belongs to another task. Search again for the current task.")).toBeVisible();
});
test.describe("rejected version",()=>{
 test("a changed task preserves candidates and keeps direct input available",async({page})=>{
  await page.route("**/graphql",async route=>{const body=route.request().postDataJSON();if(body.operationName==="ResolveTaskSourcingCandidate"){await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({data:null,errors:[{message:"redacted",extensions:{code:"TASK_REVISION_CONFLICT"}}]})});return}await route.fallback()});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(base);await page.getByRole("button",{name:`Select: ${searchCandidate.title}`}).click();await expect(page.locator(".sourcing-resolution__error")).toContainText("The task changed. Load its current state before choosing a resource.");await expect(page.locator(".sourcing-candidate")).toHaveCount(1);await expect(page.getByRole("textbox",{name:"Magnet link"})).toBeEnabled();
 });
});
for(const locale of ["en","zh-CN","ar-XB"] as const){
 test.describe(locale,()=>{
  test.use({graphqlOverrides:{torrentSearch:({id,offset,limit}:{id:string;offset:number;limit:number})=>({...torrentSearchSnapshot,id,offset,limit,taskId:task.id,state:"PARTIAL",partial:true,errorCode:"INDEXER_SEARCH_INCOMPLETE",sites:[...torrentSearchSnapshot.sites,{...torrentSearchSnapshot.sites[0],instanceId:"failed",name:"Synthetic private site",state:"FAILED",errorCode:"INDEXER_AUTHENTICATION_FAILED"}],total:3,candidates:[{...searchCandidate,title:"MOJI-102 "+"A long resource candidate title with encoding metadata ".repeat(4)},{...searchCandidate,id:"zero",title:"MOJI-102 zero seeders",seeders:0},{...searchCandidate,id:"unknown",title:"MOJI-102 unknown size and seeder count",detailsUrl:null}]})}});
  test("task resource selection keeps failures and long candidates readable",async({page})=>{
   await page.addInitScript(value=>localStorage.setItem("moji:locale",value),locale==="ar-XB"?"en":locale);await page.goto(base+`?sourcingTask=${task.id}&sourcingRun=task-search-${task.id}`+(locale==="ar-XB"?"&__locale=ar-XB":""));await expect(page.locator(".sourcing-candidate")).toHaveCount(3);await expect(page.locator("html")).toHaveAttribute("dir",locale==="ar-XB"?"rtl":"ltr");const candidates=page.locator(".sourcing-resolution__candidates");await candidates.scrollIntoViewIfNeeded();expect(await candidates.evaluate(element=>element.scrollWidth<=element.clientWidth)).toBe(true);await page.evaluate(()=>document.fonts.ready);await expect(candidates).toHaveScreenshot(`${locale}-task-resources.png`);
  });
 });
}

test("saving a task run keeps the task-list return path",async({page})=>{
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/tasks?status=failed&q=MOJI");const card=page.locator(".task-card").filter({hasText:task.code});await card.locator(".task-primary-action").click();await expect(page.locator(".sourcing-candidate")).toHaveCount(1);await expect(page).toHaveURL(/sourcingRun=/);await page.keyboard.press("Escape");await expect(page).toHaveURL("/tasks?status=failed&q=MOJI");
});
