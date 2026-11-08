import { expect } from "@playwright/test";
import { test } from "./fixtures";
const report={executed:true,elapsedKnown:true,id:"100",instanceId:"builtin-one",name:"Sukebei",operation:"SEARCH",trigger:"AUTOMATIC",taskId:"task-one",code:"ABCD-123",searchId:"",resourceId:"",configRevision:"a".repeat(64),bundleRevision:"b".repeat(40),startedAt:"2026-01-15T11:00:00Z",finishedAt:"2026-01-15T11:00:01Z",mode:"search",categories:[6000],limit:100,offset:0,errorCode:"INDEXER_SOLVER_FAILED",correlationId:"00000000-0000-4000-8000-000000000001",retryAt:null,results:0,elapsedMs:1000,cacheHit:false,partial:true,truncated:false,issueCount:0,httpStatus:503};
const sites=[{id:"builtin-one",name:"Sukebei"},{id:"two",name:"Other site"}];
const requests:Array<{instanceId?:string;before?:string;query?:string;outcome?:string;operation?:string;since?:string}>=[];
test.use({graphqlOverrides:{indexerDiagnostics:({input}:{input:typeof requests[number]})=>{
 requests.push(input);
 const filtered=input.query||input.outcome||input.operation||input.since;
 const items=filtered?[{...report,id:"50",code:"ABCD-124"}]:input.before?[{...report,id:"1",code:"ABCD-125",operation:"FETCH",resourceId:"opaque-reference",errorCode:"INDEXER_COOLING_DOWN",httpStatus:429}]:Array.from({length:20},(_,i)=>({...report,id:String(100-i)}));
 return {items,sites,total:filtered?1:21,hasMore:!filtered&&!input.before};
}}});
for(const theme of ["light","dark"]){
 test(`${theme} history shows context directly and filters before paging`,async({page})=>{
  requests.length=0;await page.addInitScript(value=>{localStorage.setItem("moji:locale","en");localStorage.setItem("moji:theme",value);},theme);
  await page.goto("/settings/indexers");await page.locator(".indexer-row").first().getByRole("button",{name:"Execution history",exact:true}).click();
  const dialog=page.getByRole("dialog",{name:"Sukebei · Execution history",exact:true});
  await expect(dialog.locator(".indexer-diagnostics > li")).toHaveCount(20);expect(requests.at(-1)?.instanceId).toBe("builtin-one");
  await expect(dialog.getByText("Automatic sourcing",{exact:true}).first()).toBeVisible();
  await expect(dialog.getByText("task-one",{exact:true}).first()).toBeVisible();await expect(dialog.getByText("HTTP 503",{exact:true}).first()).toBeVisible();
  await expect(dialog.locator("details")).toHaveCount(0);
  const edge=await dialog.evaluate(panel=>{const body=panel.querySelector(".drawer-body")!;const a=panel.getBoundingClientRect();const b=body.getBoundingClientRect();const style=getComputedStyle(panel);return Math.max(Math.abs(b.right-(a.right-parseFloat(style.borderRightWidth))),Math.abs(b.left-(a.left+parseFloat(style.borderLeftWidth))));});
  expect(edge).toBeLessThanOrEqual(1);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await expect(page).toHaveScreenshot(`indexer-diagnostics-${theme}.png`);
  await page.context().grantPermissions(["clipboard-read","clipboard-write"]);
  await dialog.locator(".indexer-record").first().getByRole("button",{name:/^Copy Correlation ID:/}).click();
  await expect.poll(()=>page.evaluate(()=>navigator.clipboard.readText())).toBe(report.correlationId);
  await expect(dialog.getByRole("button",{name:"Copied Correlation ID",exact:true})).toBeVisible();
  await dialog.getByRole("button",{name:"Next page",exact:true}).click();await expect(dialog.getByText("ABCD-125",{exact:true})).toBeVisible();await expect(dialog.getByRole("button",{name:"Next page",exact:true})).toBeDisabled();
  await dialog.getByRole("combobox",{name:"Result",exact:true}).selectOption("FAILED");await expect(dialog.getByText("ABCD-124",{exact:true})).toBeVisible();await expect(dialog.getByText("1 matching records",{exact:true})).toBeVisible();expect(requests.at(-1)?.before).toBeUndefined();
  await dialog.getByRole("combobox",{name:"Operation",exact:true}).selectOption("SEARCH");await expect.poll(()=>requests.at(-1)?.operation).toBe("SEARCH");
  await dialog.getByRole("searchbox",{name:"Search records",exact:true}).fill("ABCD-124");await dialog.getByRole("button",{name:"Search records",exact:true}).click();await expect.poll(()=>requests.at(-1)?.query).toBe("ABCD-124");
  await dialog.getByRole("combobox",{name:"Time",exact:true}).selectOption("24");await expect.poll(()=>Boolean(requests.at(-1)?.since)).toBe(true);
  await page.keyboard.press("Escape");await expect(dialog).toHaveCount(0);await page.reload();await page.getByRole("button",{name:"Execution history",exact:true}).first().click();await expect(page.getByRole("dialog",{name:"Execution history",exact:true}).locator(".indexer-diagnostics > li")).toHaveCount(20);expect(requests.at(-1)?.instanceId).toBeUndefined();
 });
}
