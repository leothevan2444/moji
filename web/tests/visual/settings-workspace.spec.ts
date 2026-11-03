import { expect } from "@playwright/test";
import { test, settingsSnapshot, indexerInstances } from "./fixtures";

test.use({graphqlOverrides:{logs:[],patchSettings:()=>({...settingsSnapshot,system:{...settingsSnapshot.system,language:"en"}})}});

const panels = {
  indexers:["sites","definitions","connection","cache"], downloader:["connection","speed","cleanup"],
  stash:["connection","delivery","metadata"], tasks:["schedule","releases","resources","deletion"],
  system:["general","imageCache"], logs:[], about:[]
} as const;

for (const locale of ["zh-CN", "en"] as const) {
  test(`${locale} grouped settings panels match the handoff`, async ({page}, info) => {
    await page.addInitScript(value => {localStorage.setItem("moji:locale",value);localStorage.setItem("moji:theme","dark");}, locale);
    for (const [section, subpages] of Object.entries(panels)) {
      await page.goto(`/settings/${section}`);
      await expect(page.locator(".settings-tabs [aria-current=page]")).toHaveCount(1);
      await expect(page.locator(".settings-main .drawer-card:visible").first()).toBeVisible();
      await expect(page.locator(".settings-loading, .skeleton")).toHaveCount(0);
      for (const panel of subpages.length ? subpages : [""]) {
        if (panel) await page.locator(`#settings-tab-${panel}`).click();
        if (panel === "resources") {
          const expand = page.locator(".torrent-rules__expand");
          if (await expand.getAttribute("aria-expanded") === "false") await expand.click();
        }
        await expect(page.locator(".settings-main .drawer-card:visible").first()).toBeVisible();
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
        expect(await page.locator(".settings-main").evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
        await page.evaluate(() => document.fonts.ready);
        await page.screenshot({path:info.outputPath(`${locale}-${section}-${panel || "main"}.png`),fullPage:true});
        if (["connection","resources","delivery","imageCache"].includes(panel)) {
          await expect(page).toHaveScreenshot(`${locale}-${section}-${panel}.png`,{fullPage:true});
        }
      }
    }
  });
}

test("panel URL, keyboard navigation and fixed sidebar preserve context", async ({page},info) => {
  await page.goto("/settings/tasks?panel=releases");
  const selected = page.getByRole("tab",{selected:true});
  await expect(selected).toHaveText("Release policy");
  await selected.focus();
  await page.keyboard.press("ArrowRight");
  await expect(page).toHaveURL(/panel=resources/);
  await expect(page.getByRole("tab",{name:"Resource selection",exact:true})).toBeFocused();
  await page.keyboard.press("End");
  await expect(page.getByRole("tab",{name:"Deletion policy",exact:true})).toBeFocused();
  await page.goBack();
  await expect(page.getByRole("tab",{name:"Resource selection",exact:true})).toHaveAttribute("aria-selected","true");
  await page.reload();
  await expect(page.getByRole("tab",{name:"Resource selection",exact:true})).toHaveAttribute("aria-selected","true");
  await expect(page.locator(".settings-loading")).toHaveCount(0);
  await page.getByLabel("Enable rule chain",{exact:true}).check();
  const expand = page.locator(".torrent-rules__expand");
  if (await expand.getAttribute("aria-expanded") === "false") await expand.click();
  const before = await page.locator(".settings-sidebar").boundingBox();
  await page.locator(".settings-main").evaluate(node => node.scrollTo(0,600));
  const bounds = await page.locator(".settings-sidebar").boundingBox();
  expect(bounds!.y).toBe(before!.y);
  if (info.project.name === "desktop") {
    expect(bounds!.x).toBe(0);
    expect(Math.round(bounds!.y + bounds!.height)).toBe(page.viewportSize()!.height);
  }
  await expect(page.locator(".settings-tabs [aria-current=page]")).toBeVisible();
});

test("paired delivery paths save independently while switching subpages", async ({page}) => {
  const patches:unknown[]=[];
  await page.route("**/graphql",async route => {
    const request=route.request().postDataJSON();
    if(request.operationName === "PatchSettings")patches.push(request.variables.input);
    await route.fallback();
  });
  await page.goto("/settings/stash?panel=delivery");
  const download = page.getByRole("group",{name:"Download directory mapping",exact:true});
  await expect(download.getByRole("textbox")).toHaveCount(2);
  await download.getByRole("textbox").nth(0).fill("/qb/downloads");
  await page.getByRole("tab",{name:"Metadata & cache",exact:true}).click();
  await expect.poll(()=>patches.length).toBe(1);
  expect(patches[0]).toEqual({stash:{ingest:{downloads:{qb_root:"/qb/downloads"}}}});
  await page.getByRole("tab",{name:"Media delivery",exact:true}).click();
  await expect(download.getByRole("textbox").nth(0)).toHaveValue("/qb/downloads");
});

test("each connection tests its current fields and reports a result", async ({page}) => {
  const sent:any[]=[];
  await page.route("**/graphql", async route => {
    const request=route.request().postDataJSON();
    if(request.operationName !== "TestServiceConnection") {await route.fallback();return;}
    sent.push(request.variables.input);
    await route.fallback();
  });
  for (const [section,service] of [["indexers","JACKETT"],["downloader","QBITTORRENT"],["stash","STASH"]]) {
    await page.goto(`/settings/${section}?panel=connection`);
    await expect(page.locator(".settings-main .status-chip")).toHaveCount(0);
    const address=page.getByRole("textbox",{name:"Service address",exact:true});
    await address.fill(`http://${section}.example`);
    await page.getByRole("button",{name:"Test connection",exact:true}).click();
    await expect(page.getByRole("status").filter({hasText:"Connection successful."})).toBeVisible();
    expect(sent.at(-1)).toMatchObject({service,url:`http://${section}.example`});
  }
});

test("connection results expire on edits and stale responses cannot overwrite the new fields", async ({page}) => {
  let release!:()=>void;
  let code="OK";
  await page.route("**/graphql",async route=>{
    const request=route.request().postDataJSON();
    if(request.operationName!=="TestServiceConnection"){await route.fallback();return;}
    const current=code;
    await new Promise<void>(resolve=>{release=resolve;});
    await route.fulfill({contentType:"application/json",body:JSON.stringify({data:{testServiceConnection:{code:current}}})});
  });
  await page.goto("/settings/stash");
  const button=page.getByRole("button",{name:"Test connection",exact:true});
  const address=page.getByRole("textbox",{name:"Service address",exact:true});
  const original=await address.inputValue();
  await button.click();
  await expect(page.getByRole("button",{name:"Testing…",exact:true})).toBeDisabled();
  await expect.poll(()=>typeof release).toBe("function");
  await address.fill("http://changed.example");
  await address.fill(original);
  release();
  await expect(button).toBeEnabled();
  await expect(page.getByText("Connection successful.",{exact:true})).toHaveCount(0);
  for(const result of ["OK","FAILED","TIMEOUT","INVALID_URL"]){
    code=result;release=undefined as any;
    await button.click();await expect.poll(()=>typeof release).toBe("function");release();
    await expect(button).toBeEnabled();
    await expect(page.locator(".settings-connection-test__result")).toBeVisible();
    await expect(page.locator(".settings-connection-test__result")).toContainText(result==="OK" ? "successful" : result==="TIMEOUT" ? "timed out" : result==="INVALID_URL" ? "HTTP" : "failed");
  }
});

test("switch tracks and thumbs align in normal and compact sizes", async ({page}) => {
  for (const path of ["/settings/system", "/settings/tasks?panel=resources"]){
    await page.goto(path);
    await expect(page.locator(".settings-main .settings-form:visible").first()).toBeVisible();
    const expand=page.locator(".torrent-rules__expand");
    if(await expand.count() && await expand.getAttribute("aria-expanded")==="false")await expand.click();
    const switches=page.locator(".settings-main .switch:visible");
    expect(await switches.count()).toBeGreaterThan(0);
    for(const control of await switches.all()) {
      const geometry = await control.evaluate(node=>{
        const track=node.querySelector(".switch__track")!.getBoundingClientRect();
        const thumb=node.querySelector(".switch__thumb")!.getBoundingClientRect();
        const box=node.getBoundingClientRect();
        return {height:box.height,track:track.height,center:Math.abs((thumb.y+thumb.height/2)-(track.y+track.height/2)),fits:thumb.x>=track.x && thumb.right<=track.right};
      });
      expect(geometry.height).toBe(geometry.track);
      expect(geometry.center).toBeLessThan(0.6);
      expect(geometry.fits).toBe(true);
    }
  }
});

test("connection feedback is plain text and disappears after five seconds", async ({page}) => {
  await page.clock.install();
  await page.goto("/settings/stash");
  const testButton = page.getByRole("button", {name:"Test connection", exact:true});
  await testButton.click();
  const result = page.locator(".settings-connection-test__result");
  await expect(result).toHaveText("Connection successful.");
  expect(await result.evaluate(node => getComputedStyle(node).backgroundColor)).toBe("rgba(0, 0, 0, 0)");
  await page.clock.fastForward(5001);
  await expect(result).toHaveCount(0);
  await testButton.click();
  await expect(result).toBeVisible();
  await page.clock.fastForward(5001);
  await expect(result).toHaveCount(0);
});

test("indexer rows preserve metadata, touch dragging and scoped order", async ({page}, info) => {
  const base=indexerInstances.items[1];
  let snapshot={...indexerInstances,order:["jackett/a","jackett/b","jackett/c"],items:[
    {...base,id:"jackett/a",name:"Alpha",siteUrl:"https://alpha.example/"},
    {...base,id:"jackett/b",name:"Beta",privacy:"private",language:"zh-CN"},
    {...base,id:"jackett/c",name:"Gamma",siteUrl:""}
  ]};
  const orders:string[][]=[];
  await page.route("**/graphql",async route=>{
    const request=route.request().postDataJSON();
    if(request.operationName==="ReorderIndexers"){
      const ids=request.variables.ids as string[];orders.push(ids);
      snapshot={...snapshot,revision:`config-${orders.length+1}`,order:ids,items:ids.map(id=>snapshot.items.find(item=>item.id===id)!)};
      await route.fulfill({contentType:"application/json",body:JSON.stringify({data:{reorderIndexers:snapshot}})});return;
    }
    if(request.operationName==="IndexerInstances"){await route.fulfill({contentType:"application/json",body:JSON.stringify({data:{indexerInstances:snapshot}})});return;}
    await route.fallback();
  });
  await page.goto("/settings/indexers?panel=queryOrder");
  const rows=page.locator(".indexer-list .indexer-row");await expect(rows).toHaveCount(3);
  await expect(rows.nth(0).getByRole("link")).toHaveAttribute("href","https://alpha.example/");await expect(rows.nth(1)).toContainText("zh-CN");
  await page.evaluate(()=>document.fonts.ready);await expect(page).toHaveScreenshot("indexer-query-order.png",{fullPage:true});
  if(info.project.name==="mobile"){
    const source=await rows.nth(0).locator(".indexer-drag").boundingBox();const target=await rows.nth(2).boundingBox();
    const touch=await page.context().newCDPSession(page);
    await touch.send("Input.dispatchTouchEvent",{type:"touchStart",touchPoints:[{x:source!.x+4,y:source!.y+4}]});
    await touch.send("Input.dispatchTouchEvent",{type:"touchMove",touchPoints:[{x:target!.x+target!.width/2,y:target!.y+target!.height/2}]});
    await touch.send("Input.dispatchTouchEvent",{type:"touchEnd",touchPoints:[]});await touch.detach();
  }else{await rows.nth(0).locator(".indexer-drag").dragTo(rows.nth(2));}
  await expect.poll(()=>orders.length).toBe(1);expect(orders[0]).toEqual(["jackett/b","jackett/c","jackett/a"]);await expect(rows.nth(2)).toContainText("Alpha");
  await rows.nth(2).getByRole("button",{name:"Move Alpha up",exact:true}).click();await expect.poll(()=>orders.length).toBe(2);expect(orders[1]).toEqual(["jackett/b","jackett/a","jackett/c"]);
});
