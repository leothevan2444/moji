import {expect,type Page} from "@playwright/test";
import {test} from "./fixtures";
const path="/settings/indexers?panel=definitions";
async function policyServer(page:Page,conflict=false){
 let current={revision:"policy-1",source:"prowlarr",schemaVersion:11,checkIntervalHours:24,autoUpdate:false};let serial=1;const writes:{expectedRevision:string;input:Record<string,unknown>}[]=[];const operations:string[]=[];
 await page.route("**/graphql",async route=>{
  const body=route.request().postDataJSON();operations.push(body.operationName);
  if(body.operationName==="IndexerDefinitionPolicy"){await route.fulfill({contentType:"application/json",body:JSON.stringify({data:{indexerDefinitionPolicy:current}})});return}
  if(body.operationName==="UpdateIndexerDefinitionPolicy"){
   writes.push(body.variables);if(conflict){conflict=false;current={...current,revision:"external",checkIntervalHours:72}}
   if(body.variables.expectedRevision!==current.revision){await route.fulfill({contentType:"application/json",body:JSON.stringify({errors:[{message:"redacted",extensions:{code:"CONFIG_REVISION_CONFLICT"}}],data:null})});return}
   current={...current,...body.variables.input,revision:`policy-${++serial}`};await route.fulfill({contentType:"application/json",body:JSON.stringify({data:{updateIndexerDefinitionPolicy:current}})});return
  }
  await route.fallback();
 });
 return {writes,operations,get current(){return current}};
}
test("policy saves partial edits with acknowledged revisions and survives refresh",async({page})=>{
 const server=await policyServer(page);await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(path);const policy=page.locator(".indexer-definition-policy");await expect(policy.getByRole("spinbutton")).toHaveValue("24");expect(server.writes).toEqual([]);
 await policy.getByRole("spinbutton").fill("48");await expect.poll(()=>server.writes.length).toBe(1);expect(server.writes[0]).toEqual({expectedRevision:"policy-1",input:{checkIntervalHours:48}});
 await policy.getByRole("checkbox").check();await expect.poll(()=>server.writes.length).toBe(2);expect(server.writes[1]).toEqual({expectedRevision:"policy-2",input:{autoUpdate:true}});
 await policy.getByRole("checkbox").uncheck();await expect.poll(()=>server.writes.length).toBe(3);expect(server.writes[2]).toEqual({expectedRevision:"policy-3",input:{autoUpdate:false}});
 await expect(policy.locator(".settings-save-status")).toHaveCount(0);await page.reload();await expect(policy.getByRole("spinbutton")).toHaveValue("48");await expect(policy.getByRole("checkbox")).not.toBeChecked();expect(server.writes).toHaveLength(3);
 expect(server.operations.some(name=>["InstallIndexerDefinitions","UpdateIndexerDefinitions","CheckIndexerDefinitionUpdate"].includes(name))).toBe(false);
});
test("policy conflict keeps the draft until the user confirms a fresh read",async({page})=>{
 const server=await policyServer(page,true);await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(path);const policy=page.locator(".indexer-definition-policy");await policy.getByRole("spinbutton").fill("48");await expect.poll(()=>server.writes.length).toBe(1);await expect(policy.getByRole("spinbutton")).toHaveValue("48");const reload=policy.getByRole("button",{name:"Load current settings",exact:true});await expect(reload).toBeVisible();await reload.click();const dialog=page.getByRole("dialog");await expect(policy.getByRole("spinbutton")).toHaveValue("48");await dialog.getByRole("button",{name:"Load current settings",exact:true}).click();await expect(dialog).toHaveCount(0);await expect(policy.getByRole("spinbutton")).toHaveValue("72");expect(server.writes).toHaveLength(1);
});
test("invalid interval remains editable and never reaches the server",async({page})=>{
 const server=await policyServer(page);await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(path);const policy=page.locator(".indexer-definition-policy");await policy.getByRole("spinbutton").fill("0");await expect(policy.getByRole("button",{name:"Retry",exact:true})).toBeVisible();await expect(policy.getByRole("spinbutton")).toHaveValue("0");expect(server.writes).toEqual([]);
});
for(const mode of [{locale:"en",theme:"light"},{locale:"zh-CN",theme:"light"},{locale:"ar-XB",theme:"light"},{locale:"en",theme:"dark"}] as const){
 test(`${mode.locale} ${mode.theme} rule update policy remains readable`,async({page})=>{
  await page.addInitScript(value=>{localStorage.setItem("moji:locale",value.locale==="ar-XB"?"en":value.locale);localStorage.setItem("moji:theme",value.theme)},mode);await page.goto(path+(mode.locale==="ar-XB"?"&__locale=ar-XB":""));await expect(page.locator(".indexer-definition-policy").getByRole("spinbutton")).toHaveValue("24");await expect(page.locator("html")).toHaveAttribute("dir",mode.locale==="ar-XB"?"rtl":"ltr");expect(await page.locator(".settings-main").evaluate(element=>element.scrollWidth<=element.clientWidth)).toBe(true);await page.evaluate(()=>document.fonts.ready);await expect(page.locator(".settings-main")).toHaveScreenshot(`${mode.locale}-${mode.theme}-definition-policy.png`);
 });
}
