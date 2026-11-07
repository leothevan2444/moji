import { expect } from "@playwright/test";
import { test } from "./fixtures";

const initial=()=>({flareSolverrMaxTimeoutMs:90000,proxyEnabled:false,proxyUrl:"",proxyUsername:"",proxyPasswordConfigured:false,flareSolverrEnabled:false,flareSolverrUrl:"",requestTimeoutSeconds:30,siteTimeoutSeconds:120});
test.describe("shared builtin indexer network autosave",()=>{
 let revision=1;let saved=initial();let failure:""|"transient"|"conflict"="";
 const calls:Array<{expectedRevision:string;input:Record<string,unknown>}>=[];
 const status=()=>({revision:`config-${revision}`,...saved});
 test.use({graphqlOverrides:{indexerNetworkSettings:status}});
 test.beforeEach(async({page})=>{
  revision=1;saved=initial();failure="";calls.length=0;
  await page.route("**/graphql",async route=>{
   const body=route.request().postDataJSON();if(body.operationName!=="UpdateIndexerNetwork")return route.fallback();calls.push(body.variables);
   if(failure){const conflict=failure==="conflict";failure="";if(conflict){revision++;saved.proxyUrl="http://external.example:8080";}return route.fulfill({json:{errors:[{message:"save failed",extensions:{code:conflict?"CONFIG_REVISION_CONFLICT":"INDEXER_REQUEST_FAILED"}}]}});}
   const {proxyPassword,clearProxyPassword,...patch}=body.variables.input;saved={...saved,...patch};if(proxyPassword)saved.proxyPasswordConfigured=true;if(clearProxyPassword)saved.proxyPasswordConfigured=false;revision++;
   await route.fulfill({json:{data:{updateIndexerNetwork:status()}}});
  });
 });
 test("automatically saves shared network and timeouts, preserving blank passwords",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/indexers?panel=cache");await page.getByRole("tab",{name:"Network & cache"}).click();
  await expect(page.getByRole("button",{name:"Save",exact:true})).toHaveCount(0);
  const username=page.getByLabel("Username",{exact:true});const password=page.getByLabel("Password",{exact:true});
  await expect(username).toHaveAttribute("placeholder","Blank: no username");await expect(password).toHaveAttribute("placeholder","Blank: no password");
  await page.getByLabel("Proxy URL",{exact:true}).fill("socks5h://proxy.example:1080");await username.fill("synthetic-user");await password.fill("synthetic-secret");
  await page.getByRole("switch",{name:"Enable proxy",exact:true}).check();await page.getByLabel("FlareSolverr URL",{exact:true}).fill("http://solver.example:8191");await page.getByRole("switch",{name:"Enable FlareSolverr",exact:true}).check();await page.getByLabel("Site search timeout (seconds)",{exact:true}).fill("150");
  await expect.poll(()=>saved.siteTimeoutSeconds).toBe(150);await page.getByLabel("FlareSolverr Max Timeout (ms)",{exact:true}).fill("45000");await expect.poll(()=>saved.flareSolverrMaxTimeoutMs).toBe(45000);await expect(password).toHaveValue("");await expect(password).toHaveAttribute("placeholder","Blank: keep saved password");
  expect(saved).toMatchObject({proxyEnabled:true,proxyUrl:"socks5h://proxy.example:1080",proxyUsername:"synthetic-user",proxyPasswordConfigured:true,flareSolverrEnabled:true,flareSolverrUrl:"http://solver.example:8191"});
  const previous=calls.length;await page.getByRole("switch",{name:"Enable proxy",exact:true}).uncheck();await expect.poll(()=>saved.proxyEnabled).toBe(false);expect(calls[previous].input).toEqual({proxyEnabled:false});expect(saved.proxyPasswordConfigured).toBe(true);
  await username.fill("");await expect.poll(()=>saved.proxyUsername).toBe("");expect(saved.proxyPasswordConfigured).toBe(true);
  await page.getByRole("button",{name:"Clear password",exact:true}).click();await expect.poll(()=>saved.proxyPasswordConfigured).toBe(false);await expect(password).toHaveAttribute("placeholder","Blank: no password");
  await page.reload();await page.getByRole("tab",{name:"Network & cache"}).click();await expect(page.getByLabel("FlareSolverr Max Timeout (ms)",{exact:true})).toHaveValue("45000");await expect(page.getByLabel("Proxy URL",{exact:true})).toHaveValue("socks5h://proxy.example:1080");await expect(page.getByRole("switch",{name:"Enable proxy",exact:true})).not.toBeChecked();
  await page.getByRole("tab",{name:"Sites",exact:true}).click();await page.locator('[data-indexer-id="builtin-one"]').getByRole("button",{name:"Configure",exact:true}).click();await expect(page.getByRole("dialog").getByLabel("Network profile",{exact:true})).toHaveCount(0);
 });
 test("failed autosave retains the draft across navigation and supports retry",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/indexers?panel=cache");await page.getByRole("tab",{name:"Network & cache"}).click();failure="transient";
  await page.getByLabel("Proxy URL",{exact:true}).fill("http://draft.example:8080");await expect(page.locator(".indexer-global-network-form .settings-save-status")).toHaveAttribute("role","alert");
  await page.getByRole("button",{name:"Downloader",exact:true}).click();await page.getByRole("button",{name:"Indexers",exact:true}).click();await page.getByRole("tab",{name:"Network & cache"}).click();
  await expect(page.getByLabel("Proxy URL",{exact:true})).toHaveValue("http://draft.example:8080");await page.locator(".indexer-global-network-form .settings-save-status").getByRole("button",{name:"Retry",exact:true}).click();await expect.poll(()=>saved.proxyUrl).toBe("http://draft.example:8080");
 });
 test("a conflict preserves inputs until explicitly reloading the latest settings",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/indexers?panel=cache");await page.getByRole("tab",{name:"Network & cache"}).click();failure="conflict";
  await page.getByLabel("Proxy URL",{exact:true}).fill("http://draft.example:8080");await expect(page.locator(".indexer-global-network-form .settings-save-status")).toHaveAttribute("role","alert");
  await page.getByRole("button",{name:"Load current settings",exact:true}).click();const dialog=page.getByRole("dialog",{name:"Load current settings",exact:true});await dialog.getByRole("button",{name:"Keep editing",exact:true}).click();await expect(page.getByLabel("Proxy URL",{exact:true})).toHaveValue("http://draft.example:8080");
  await page.getByRole("button",{name:"Load current settings",exact:true}).click();await dialog.getByRole("button",{name:"Load current settings",exact:true}).click();await expect(page.getByLabel("Proxy URL",{exact:true})).toHaveValue("http://external.example:8080");
  await page.getByLabel("Username",{exact:true}).fill("new-user");await expect.poll(()=>saved.proxyUsername).toBe("new-user");expect(calls.at(-1)?.expectedRevision).toBe("config-2");
 });
 for(const [locale,theme] of [["en","light"],["zh-CN","dark"]] as const)test(`${locale} ${theme} shared network autosave layout`,async({page},testInfo)=>{
  const errors:string[]=[];page.on("pageerror",error=>errors.push(error.message));await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme);},{locale,theme});await page.goto(`/settings/indexers?panel=cache&__locale=${locale}`);await page.getByRole("tab",{name:locale==="en"?"Network & cache":"网络与缓存"}).click();
  await expect(page.getByLabel("Browser verification timeout (seconds)",{exact:true})).toHaveCount(0);
  const form=page.locator(".indexer-global-network-form");
  if(testInfo.project.name==="mobile"){
   for(const [index,name] of ["proxy","solver","timeouts"].entries()){const card=form.locator(":scope > .drawer-card").nth(index);await card.scrollIntoViewIfNeeded();await expect(card).toHaveScreenshot(`${locale}-${theme}-shared-network-${name}.png`);}
  }else{await form.scrollIntoViewIfNeeded();await expect(form).toHaveScreenshot(`${locale}-${theme}-shared-network.png`);}
  for(const key of ["requestTimeoutSeconds","siteTimeoutSeconds"]){
   const field=form.locator(".indexer-timeout-fields .settings-field").nth(key==="requestTimeoutSeconds"?0:1);const info=field.locator(".settings-info");await info.focus();const tip=page.getByRole("tooltip");await expect(tip).toBeVisible();await expect(tip).toContainText("Jackett");const box=await tip.boundingBox();expect(box).not.toBeNull();expect(box!.x).toBeGreaterThanOrEqual(0);expect(box!.x+box!.width).toBeLessThanOrEqual(page.viewportSize()!.width);await info.press("Escape");await expect(tip).toHaveCount(0);
  }
  expect(errors).toEqual([]);
 });
});
