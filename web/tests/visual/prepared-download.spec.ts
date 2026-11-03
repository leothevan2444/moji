import {expect} from "@playwright/test";
import {test,homeOverview} from "./fixtures";
const task=homeOverview.attentionTasks[1];
const attachment="/api/indexers/downloads/"+"d".repeat(64);
test.use({graphqlOverrides:{tasks:[task]}});
for(const mode of ["discover","task"] as const){
 test(`${mode} explicitly prepares an owned candidate and downloads only after clicking the ready link`,async({page})=>{
  const operations:{name:string;input:unknown}[]=[];let reads=0;
  page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();operations.push({name:body.operationName,input:body.variables?.input})}});
  await page.context().route("**"+attachment,route=>{reads++;return route.fulfill({status:200,contentType:"application/x-bittorrent",headers:{"content-disposition":'attachment; filename="resource.torrent"',"cache-control":"no-store"},body:"d4:infod6:lengthi1e4:name12:ABCD-123.mp4ee"})});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(mode==="discover"?"/discover?source=indexers&q=MOJI-102&run=search-one":`/tasks/${task.id}/resolve?sourcingTask=${task.id}&sourcingRun=task-search-${task.id}`);
  const candidate=page.locator(mode==="discover"?".candidate-card--indexer":".sourcing-candidate");await expect(candidate).toHaveCount(1);
  expect(operations.filter(op=>op.name==="PrepareTorrentDownload")).toEqual([]);await candidate.getByRole("button",{name:"Get download link",exact:true}).click();const link=candidate.getByRole("link",{name:"Download torrent",exact:true});await expect(link).toHaveAttribute("href",attachment);expect(reads).toBe(0);
  expect(operations.filter(op=>op.name==="PrepareTorrentDownload")).toEqual([{name:"PrepareTorrentDownload",input:{searchId:mode==="discover"?"search-one":`task-search-${task.id}`,candidateId:"candidate-one"}}]);
  const downloaded=page.waitForEvent("download");await link.click();expect((await downloaded).suggestedFilename()).toBe("resource.torrent");expect(reads).toBe(1);
  expect(operations.some(op=>["QueueTorrentCandidate","ResolveTaskSourcingCandidate"].includes(op.name))).toBe(false);
  await page.reload();await expect(candidate.getByRole("button",{name:"Get download link",exact:true})).toBeVisible();expect(operations.filter(op=>op.name==="PrepareTorrentDownload")).toHaveLength(1);
 });
}
test.describe("prepared magnet",()=>{
 test.use({graphqlOverrides:{prepareTorrentDownload:()=>({kind:"MAGNET",downloadUrl:attachment,expiresAt:new Date(Date.now()+300000).toISOString()})}});
 test("offers a native magnet only after explicitly acquiring its local attachment",async({page})=>{
  let reads=0;const magnet="magnet:?xt=urn:btih:synthetic&dn=MOJI-102";
  await page.context().route("**"+attachment,route=>{reads++;return route.fulfill({status:200,contentType:"text/uri-list; charset=utf-8",body:magnet})});await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/discover?source=indexers&q=MOJI-102&run=search-one");const candidate=page.locator(".candidate-card--indexer");await expect(candidate).toHaveCount(1);expect(reads).toBe(0);await candidate.getByRole("button",{name:"Get download link",exact:true}).click();await expect(candidate.getByRole("link",{name:"Magnet link",exact:true})).toHaveAttribute("href",magnet);expect(reads).toBe(1);
 });
});
test("preparation failure retains source selection and allows retry",async({page})=>{
 await page.route("**/graphql",async route=>{const body=route.request().postDataJSON();if(body.operationName==="PrepareTorrentDownload"){await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({errors:[{message:"redacted",extensions:{code:"INDEXER_AUTHENTICATION_FAILED"}}],data:null})});return}await route.fallback()});await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(`/tasks/${task.id}/resolve?sourcingTask=${task.id}&sourcingRun=task-search-${task.id}`);const candidate=page.locator(".sourcing-candidate");await candidate.getByRole("button",{name:"Get download link",exact:true}).click();await expect(candidate.getByRole("alert")).toBeVisible();await expect(candidate.getByRole("button",{name:"Get download link",exact:true})).toBeEnabled();await expect(candidate.getByRole("button",{name:/^Select:/})).toBeEnabled();await expect(page.getByRole("textbox",{name:"Magnet link",exact:true})).toBeEnabled();
});
