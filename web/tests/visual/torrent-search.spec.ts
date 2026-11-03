import { expect } from "@playwright/test";
import { test, torrentSearchSnapshot, searchCandidate } from "./fixtures";
const url = "/discover?source=indexers&q=MOJI-102&run=search-one&fastRules=1&fileRules=1";

test("refresh restores a run without starting or inspecting again", async ({page})=>{
 const operations:string[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST")operations.push(request.postDataJSON().operationName)});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url);await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);
 await page.reload();await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);await expect(page).toHaveURL(/run=search-one/);
 expect(operations.filter(name=>name==="StartTorrentSearch"||name==="PreviewTorrentSelection")).toEqual([]);
});

test("site scope requires an explicit search and admission persists its run URL", async ({page})=>{
 const starts:unknown[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="StartTorrentSearch")starts.push(body.variables.input)}});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/discover?source=indexers&q=MOJI-102");
 await page.locator(".filter-panel summary").click();await page.locator(".filter-panel__chip").filter({hasText:"Sukebei"}).click();expect(starts).toHaveLength(0);
 await page.locator(".torrent-search__actions").getByRole("button",{name:"Search resources",exact:true}).click();await expect(page).toHaveURL(/run=search-one/);await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);
 expect(starts).toEqual([{keywords:"MOJI-102",instanceIds:["builtin-one"],refresh:true}]);
});

test.describe("empty rule preview",()=>{
 test.use({graphqlOverrides:{previewTorrentSelection:{searchId:"search-one",searchRevision:1,total:0,offset:0,limit:10,candidates:[],meta:{appliedFastRules:true,appliedFileRules:true,inspectedCount:1,inspectableCount:1,inspectionCandidateLimit:5}}}});
 test("empty filtered results never fall back to unfiltered candidates",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url);await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);
  await page.getByRole("button",{name:"Apply preview",exact:true}).click();await expect(page.getByText("No results match the current rules.")).toBeVisible();await expect(page.locator(".candidate-card--indexer")).toHaveCount(0);
  await expect(page.getByText("Inspected 1; this request allows up to 5")).toBeVisible();await page.getByRole("button",{name:"Clear preview",exact:true}).click();await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);
 });
});

test.describe("durable cancellation",()=>{
 let requested=false;
 test.beforeEach(()=>{requested=false});
 test.use({graphqlOverrides:{torrentSearch:({id,offset,limit}:{id:string;offset:number;limit:number})=>({...torrentSearchSnapshot,id,offset,limit,state:"RUNNING",finishedAt:null,cancelRequested:requested}),cancelTorrentSearch:()=>{requested=true;return {...torrentSearchSnapshot,state:"RUNNING",finishedAt:null,cancelRequested:true}}}});
 test("cancel requested does not invent a terminal state",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url);await page.getByRole("button",{name:"Cancel search",exact:true}).click();await expect(page.getByRole("button",{name:"Cancelling search",exact:true})).toBeDisabled();await expect(page.locator(".torrent-search__actions")).toContainText("Searching");
  await page.reload();await expect(page.getByRole("button",{name:"Cancelling search",exact:true})).toBeDisabled();
 });
});

test("candidate admission sends code and server identities only",async({page})=>{
 const queued:unknown[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="QueueTorrentCandidate")queued.push(body.variables.input)}});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url);await page.locator(".candidate-card--indexer").getByRole("button",{name:"Create task",exact:true}).click();await page.getByRole("textbox",{name:"Code",exact:true}).fill("MOJI-102");await page.getByRole("button",{name:"Confirm creation",exact:true}).click();await expect(page).toHaveURL(/tasks\/queued-candidate/);
 expect(queued).toEqual([{candidateId:"candidate-one",searchId:"search-one",code:"MOJI-102"}]);
});

test("an expired URL reports expiry without rerunning the search",async({page})=>{
 let starts=0;await page.route("**/graphql",async route=>{const body=route.request().postDataJSON();if(body.operationName==="StartTorrentSearch")starts++;if(body.operationName==="TorrentSearch"){await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({errors:[{message:"redacted",extensions:{code:"TORRENT_SEARCH_EXPIRED"}}],data:null})});return;}await route.fallback()});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url);await expect(page.getByText("These search results expired. Start a new search.")).toBeVisible();expect(starts).toBe(0);
});

for(const locale of ["en","zh-CN","ar-XB"] as const){
 test.describe(locale,()=>{
  test.use({graphqlOverrides:{torrentSearch:({id,offset,limit}:{id:string;offset:number;limit:number})=>({...torrentSearchSnapshot,id,offset,limit,state:"PARTIAL",partial:true,truncated:true,candidates:[{...searchCandidate,title:"MOJI-102 "+"A long resource title with encoding and release information ".repeat(3)}],sites:[...torrentSearchSnapshot.sites,{...torrentSearchSnapshot.sites[0],instanceId:"jackett/onejav",name:"OneJAV",kind:"JACKETT",state:"FAILED",cacheHit:false,fetchedAt:null,errorCode:"INDEXER_AUTHENTICATION_FAILED"}]})}});
  test("partial progress and unknown observations fit the viewport",async({page})=>{
   await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));if(locale!=="ar-XB")await page.addInitScript(value=>localStorage.setItem("moji:locale",value),locale);
   await page.goto(url+(locale==="ar-XB"?"&__locale=ar-XB":""));await expect(page.locator("html")).toHaveAttribute("dir",locale==="ar-XB"?"rtl":"ltr");await page.locator(".torrent-search__sites summary").click();await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);await page.evaluate(()=>document.fonts.ready);
   const drawer=page.locator(".drawer-stack");expect(await drawer.evaluate(element=>element.scrollWidth<=element.clientWidth)).toBe(true);await expect(page.locator(".drawer--modal")).toHaveScreenshot(`${locale}-search-results.png`);const card=page.locator(".candidate-card--indexer");await card.scrollIntoViewIfNeeded();await expect(card).toHaveScreenshot(`${locale}-search-candidate.png`);
  });
 });
}

 test("dark resource search keeps readable controls and results",async({page})=>{
  await page.addInitScript(()=>{localStorage.setItem("moji:locale","en");localStorage.setItem("moji:theme","dark")});await page.goto(url);await page.locator(".candidate-card--indexer").scrollIntoViewIfNeeded();await expect(page.locator(".drawer--modal")).toHaveScreenshot("dark-search-results.png");
 });

test.describe("preview pagination",()=>{
 test.use({graphqlOverrides:{torrentSearch:({id,offset,limit}:{id:string;offset:number;limit:number})=>({...torrentSearchSnapshot,id,offset,limit,total:21,candidates:Array.from({length:21},(_,i)=>({...searchCandidate,id:`candidate-${i}`,title:`MOJI-102 page-${i}`})).slice(offset,offset+limit)}),previewTorrentSelection:({input}:{input:{searchId:string;offset:number;limit:number}})=>({searchId:input.searchId,searchRevision:1,total:1,offset:input.offset,limit:input.limit,candidates:[{...searchCandidate,title:"MOJI-102 globally locked"}],meta:{appliedFastRules:true,appliedFileRules:false,inspectedCount:0,inspectableCount:0,inspectionCandidateLimit:0}})}});
 test("rules start on page one after evaluating the whole run",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url+"&page=2");await expect(page.locator(".candidate-card--indexer")).toHaveCount(10);await expect(page.locator(".candidate-card--indexer").first()).toContainText("page-10");
  await page.getByRole("button",{name:"Apply preview",exact:true}).click();await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);await expect(page.locator(".candidate-card--indexer")).toContainText("globally locked");await expect.poll(()=>new URL(page.url()).searchParams.get("page")).toBeNull();
 });
 test("out-of-range saved pages return to the available last page",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url+"&page=99");await expect(page.locator(".candidate-card--indexer")).toHaveCount(1);await expect(page.locator(".candidate-card--indexer")).toContainText("page-20");await expect.poll(()=>new URL(page.url()).searchParams.get("page")).toBe("3");
 });
});
