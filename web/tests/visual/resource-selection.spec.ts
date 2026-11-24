import { expect } from "@playwright/test";
import { test, searchCandidate, settingsSnapshot, homeOverview } from "./fixtures";

test.use({graphqlOverrides:{patchSettings:()=>({...settingsSnapshot,system:{...settingsSnapshot.system,language:"auto"}})}});

const rulePolicy={version:7,activeGroup:"daily",rules:[{id:"label",name:"User label",conditions:[{scope:"TITLE",value:"user-label",minimum:"",maximum:""}]}],groups:[{id:"daily",name:"Daily",fallback:true,require:["label","extra-1","extra-2"],exclude:["extra-0"],tiers:[{id:"first",name:"First choice",rules:["label"]}]},{id:"alternate",name:"Alternate",fallback:false,require:[],exclude:[],tiers:[{id:"first",name:"Alternate priority",rules:["label"]}]}]};
rulePolicy.rules.push(...["Subtitle labels", "Resolution labels", "A long user-defined rule name that wraps within its card"].map((name,index)=>({id:`extra-${index}`,name,conditions:[{scope:"TITLE",value:"user-defined",minimum:"",maximum:""}]})));
rulePolicy.rules.push({id:"unused",name:"Unused rule",conditions:[{scope:"TITLE",value:"unused",minimum:"",maximum:""}]});
const ruleSettings={...settingsSnapshot,automation:{...settingsSnapshot.automation,torrentSelection:rulePolicy}};
test.describe("rule library and groups",()=>{
 test.use({graphqlOverrides:{settings:()=>ruleSettings,patchSettings:()=>({...ruleSettings,system:{...ruleSettings.system,language:"auto"}})}});
 for(const locale of ["zh-CN","en"] as const){
  for(const theme of ["light","dark"] as const){
   test(`${locale} ${theme} reusable rules and priority groups`,async({page})=>{
    await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme);},{locale,theme});
    await page.goto("/settings/tasks?panel=resources");const panel=page.locator(".resource-selection");await expect(panel).toBeVisible();
    await expect(panel.locator(":scope > .drawer-card")).toHaveCount(2);
    await expect(panel.locator("select")).toHaveCount(0);
    await expect(panel.getByRole("button",{name:locale==="en"?"Using rule group “Daily”":"正在使用规则组“Daily”",exact:true})).toBeDisabled();
    const rules=panel.locator(".resource-selection__rule-grid");await expect(rules.locator(".resource-selection__rule-edit")).toHaveCount(5);
    await expect(rules.locator("input,select")).toHaveCount(0);await expect(rules.getByRole("button",{name:"User label",exact:true})).toHaveText("User label");
    await page.evaluate(()=>document.fonts.ready);await expect(page).toHaveScreenshot(`${locale}-${theme}-resource-cards.png`);
    const groupCard=panel.locator(":scope > .drawer-card").nth(1);await groupCard.scrollIntoViewIfNeeded();await expect(groupCard).toHaveScreenshot(`${locale}-${theme}-group-cards.png`);
    const rule=rules.getByRole("button",{name:"User label",exact:true});await rule.click();
    const ruleDialog=page.getByRole("dialog",{name:"User label",exact:true});
    await expect(ruleDialog.locator(".drawer__footer")).toContainText("Daily、Alternate");
    await expect(ruleDialog.locator(".drawer-body")).not.toContainText("Daily、Alternate");
    await expect(ruleDialog.getByRole("button",{name:locale==="en"?"Delete":"删除",exact:true})).toBeEnabled();
    await expect(ruleDialog).toHaveScreenshot(`${locale}-${theme}-rule-editor.png`);
    await page.keyboard.press("Escape");await expect(ruleDialog).not.toBeVisible();await expect(rule).toBeFocused();
    const group=panel.getByRole("button",{name:"Daily",exact:true});await group.click();
    const groupDialog=page.getByRole("dialog",{name:"Daily",exact:true});
    await expect(groupDialog.locator(".resource-selection__tier")).toHaveCount(1);
    await expect(groupDialog.locator(".resource-selection__references legend")).toHaveCount(0);
    await expect(groupDialog.locator(".drawer__footer").getByRole("switch")).toHaveCount(1);
    await expect(groupDialog.locator(".drawer-body").getByRole("switch")).toHaveCount(0);
    const fallbackLabel=groupDialog.locator(".resource-selection__fallback > .settings-field__label"),fallbackSwitch=groupDialog.locator(".resource-selection__fallback > .switch");
    const labelBox=(await fallbackLabel.boundingBox())!,switchBox=(await fallbackSwitch.boundingBox())!;
    expect(switchBox.x-labelBox.x-labelBox.width).toBeGreaterThanOrEqual(0);expect(switchBox.x-labelBox.x-labelBox.width).toBeLessThanOrEqual(12);
    if(locale==="zh-CN")await expect(groupDialog).not.toContainText("条件");
    await expect(groupDialog.locator(".resource-selection__tier-heading").getByRole("button")).toHaveCount(1);
    await expect(groupDialog.getByRole("group",{name:locale==="en"?"Admission rules":"准入规则",exact:true}).getByRole("combobox")).toHaveCount(1);
    await expect(groupDialog.getByRole("group",{name:locale==="en"?"Exclusion rules":"排除规则",exact:true}).getByRole("combobox")).toHaveCount(1);
    await expect(groupDialog.getByRole("combobox",{name:locale==="en"?"Match relation":"匹配关系",exact:true})).toHaveCount(0);
    await expect.poll(()=>groupDialog.evaluate(el=>el.scrollWidth<=el.clientWidth)).toBe(true);
    await expect(groupDialog).toHaveScreenshot(`${locale}-${theme}-group-editor.png`);
    const firstTier=groupDialog.locator(".resource-selection__tier").first();await firstTier.scrollIntoViewIfNeeded();await expect(firstTier).toHaveScreenshot(`${locale}-${theme}-tier-tags.png`);
    await page.keyboard.press("Escape");await expect(group).toBeFocused();
   });
  }
 }
 test("rule conditions save as a fixed-any list with independent fields",async({page})=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");
  await page.getByRole("button",{name:"User label",exact:true}).click();const dialog=page.getByRole("dialog",{name:"User label",exact:true});
  await dialog.getByRole("button",{name:"Add condition",exact:true}).click();
  await dialog.getByRole("combobox",{name:"Match field",exact:true}).nth(1).selectOption("SEEDERS");
  await dialog.getByRole("textbox",{name:"Minimum",exact:true}).fill("5");
  await expect(dialog.getByRole("combobox")).toHaveCount(2);
  for(const row of await dialog.locator(".resource-condition-row").all()){
   const box=(await row.boundingBox())!;expect(box.height).toBeLessThanOrEqual(44);await expect.poll(()=>row.evaluate(el=>el.scrollWidth<=el.clientWidth)).toBe(true);
   const select=(await row.getByRole("combobox").boundingBox())!,button=(await row.getByRole("button").boundingBox())!;expect(Math.abs(select.y+select.height/2-button.y-button.height/2)).toBeLessThan(2);
  }
  await expect(dialog).toContainText("The rule matches when any condition matches.");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.rules?.[0]?.conditions?.[1]?.minimum==="5")).toBe(true);
  const rule=patches.find(p=>p.tasks?.torrent_selection?.rules?.[0]?.conditions?.[1]?.minimum==="5").tasks.torrent_selection.rules[0];
  expect(rule.conditions).toEqual([{scope:"TITLE",value:"user-label",minimum:"",maximum:""},{scope:"SEEDERS",value:"",minimum:"5",maximum:""}]);expect(rule).not.toHaveProperty("mode");
  await expect(dialog).toHaveScreenshot("fixed-any-rule-editor.png");
  await dialog.getByRole("button",{name:"Delete condition",exact:true}).first().click();
  await expect(dialog.getByRole("combobox",{name:"Match field",exact:true})).toHaveValue("SEEDERS");
  await expect(dialog.getByRole("button",{name:"Delete condition",exact:true})).toBeDisabled();
 });
 test("global and tier rules save only positive scalar IDs",async({page})=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");
  await page.locator(".resource-selection").getByRole("button",{name:"Daily",exact:true}).click();
  const dialog=page.getByRole("dialog",{name:"Daily",exact:true});
  await expect(dialog.getByRole("menu")).toHaveCount(0);
  await expect(dialog.getByRole("button",{name:"Actions for User label",exact:true})).toHaveCount(0);
  await expect(dialog.locator(".resource-selection__references")).toContainText("User label");
  const fallback=dialog.getByRole("switch",{name:"Allow fallback selection",exact:true});
  await fallback.uncheck();
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.fallback===false)).toBe(true);
  const saved=patches.find(p=>p.tasks?.torrent_selection?.groups?.[0]?.fallback===false).tasks.torrent_selection.groups[0];
  expect(saved).not.toHaveProperty("requireMode");expect(saved).not.toHaveProperty("excludeMode");
  expect(saved.tiers).toHaveLength(1);expect(saved.tiers[0].rules).toEqual(["label"]);expect(saved.tiers[0]).not.toHaveProperty("mode");expect(saved.tiers[0]).not.toHaveProperty("matchAll");
  await dialog.getByRole("button",{name:"Delete tier",exact:true}).click();
  await expect(dialog).toContainText("No priority tiers configured and fallback is off.");
  await fallback.check();await expect(dialog).toContainText("Choose among resources passing the pre-filter.");
  await fallback.scrollIntoViewIfNeeded();await expect(dialog).toHaveScreenshot("fallback-only-group.png");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.fallback===true&&p.tasks.torrent_selection.groups[0].tiers.length===0)).toBe(true);

 });

 for(const kind of ["rule","group"] as const){for(const entry of ["card","editor"] as const){
  test(`${kind} deletion from ${entry} confirms and can be cancelled`,async({page})=>{
   const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
   await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");const panel=page.locator(".resource-selection");
   const name=kind==="rule"?"Unused rule":"Daily";
   if(entry==="editor")await panel.getByRole("button",{name,exact:true}).click();
   const trigger=entry==="card"?panel.getByRole("button",{name:`Delete ${kind==="rule"?"rule":"rule group"} “${name}”`,exact:true}):page.getByRole("dialog").getByRole("button",{name:kind==="rule"?"Delete":"Delete rule group",exact:true});
   await trigger.click();let confirm=page.getByRole("dialog",{name:kind==="rule"?"Delete rule":"Delete rule group",exact:true});await expect(confirm).toBeVisible();
   if(kind==="group")await expect(confirm).toContainText('Deleting it will activate “Alternate”.');
   await expect(confirm).toHaveScreenshot(`${kind}-${entry}-delete-confirm.png`);
   await confirm.getByRole("button",{name:"Cancel",exact:true}).click();await expect(confirm).not.toBeVisible();await expect(trigger).toBeFocused();
   expect(patches).toHaveLength(0);
   await trigger.click();confirm=page.getByRole("dialog",{name:kind==="rule"?"Delete rule":"Delete rule group",exact:true});await confirm.getByRole("button",{name:"Delete",exact:true}).click();
   await expect(confirm).not.toBeVisible();await expect(panel.getByRole("button",{name,exact:true})).toHaveCount(0);
   if(kind==="rule")await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.rules&&!p.tasks.torrent_selection.rules.some((r:any)=>r.id==="unused"))).toBe(true);
   else{await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.active_group==="alternate")).toBe(true);await expect(panel.getByRole("button",{name:"Using rule group “Alternate”",exact:true})).toBeDisabled();await expect(panel.getByRole("button",{name:"User label",exact:true})).toBeVisible();}
  });
 }}
 test("group activation is direct and pre-filter pickers are mutually exclusive",async({page})=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");const panel=page.locator(".resource-selection");
  await panel.getByRole("button",{name:"Use rule group “Alternate”",exact:true}).click();await expect(page.getByRole("dialog")).toHaveCount(0);await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.active_group==="alternate")).toBe(true);
  await panel.getByRole("button",{name:"Daily",exact:true}).click();const dialog=page.getByRole("dialog",{name:"Daily",exact:true});const admission=dialog.getByRole("group",{name:"Admission rules",exact:true});const excluded=dialog.getByRole("group",{name:"Exclusion rules",exact:true});
  await expect(excluded.getByRole("option",{name:"User label",exact:true})).toHaveCount(0);await expect(admission.getByRole("option",{name:"Subtitle labels",exact:true})).toHaveCount(0);
  await admission.getByRole("button",{name:"Remove reference to Resolution labels",exact:true}).click();await excluded.getByRole("combobox").selectOption("extra-1");
  await expect(admission.getByRole("option",{name:"Resolution labels",exact:true})).toHaveCount(0);
  await dialog.getByRole("textbox",{name:"Tier name",exact:true}).fill("Edited inline");await expect(page.getByRole("dialog")).toHaveCount(1);
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.tiers?.[0]?.name==="Edited inline")).toBe(true);
 });
 test("library drag touch and keyboard sorting only change rule order",async({page},info)=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");const library=page.locator(".resource-selection__rule-grid");const cards=library.getByRole("listitem");const order=()=>cards.evaluateAll(items=>items.map(item=>item.getAttribute("data-library-rule-id")));
  await cards.last().scrollIntoViewIfNeeded();
  if(info.project.name==="mobile"){
   const source=(await cards.first().locator(".resource-selection__drag").boundingBox())!,target=(await cards.last().boundingBox())!;const touch=await page.context().newCDPSession(page);
   await touch.send("Input.dispatchTouchEvent",{type:"touchStart",touchPoints:[{x:source.x+source.width/2,y:source.y+source.height/2}]});await touch.send("Input.dispatchTouchEvent",{type:"touchMove",touchPoints:[{x:target.x+target.width/2,y:target.y+target.height/2}]});await touch.send("Input.dispatchTouchEvent",{type:"touchEnd",touchPoints:[]});await touch.detach();
  }else await cards.first().locator(".resource-selection__drag").dragTo(cards.last());
  await expect.poll(order).toEqual(["extra-0","extra-1","extra-2","unused","label"]);
  await library.getByRole("button",{name:"Drag User label to reorder",exact:true}).press("ArrowLeft");await expect.poll(order).toEqual(["extra-0","extra-1","extra-2","label","unused"]);
  await expect(page.getByRole("dialog")).toHaveCount(0);await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.rules?.[0]?.id==="extra-0")).toBe(true);expect(patches.every(p=>!p.tasks?.torrent_selection?.groups)).toBe(true);
 });

 test("group cards reorder without activating or opening an editor",async({page},info)=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");const library=page.locator(".resource-selection__group-grid");const cards=library.locator(":scope > [data-library-group-id]");
  const order=()=>cards.evaluateAll(items=>items.map(item=>item.getAttribute("data-library-group-id")));
  const handle=cards.first().locator(".resource-selection__drag");await cards.last().scrollIntoViewIfNeeded();await handle.scrollIntoViewIfNeeded();
  await expect(handle).toHaveCSS("border-top-width","0px");
  await expect(library.getByRole("button",{name:"Use rule group “Alternate”",exact:true}).locator("path")).toHaveCSS("fill","none");
  if(info.project.name==="mobile"){
   const source=(await handle.boundingBox())!,target=(await cards.last().boundingBox())!;const touch=await page.context().newCDPSession(page);
   await touch.send("Input.dispatchTouchEvent",{type:"touchStart",touchPoints:[{x:source.x+source.width/2,y:source.y+source.height/2}]});
   await touch.send("Input.dispatchTouchEvent",{type:"touchMove",touchPoints:[{x:target.x+20,y:target.y+20}]});
   await touch.send("Input.dispatchTouchEvent",{type:"touchEnd",touchPoints:[]});await touch.detach();
  }else await handle.dragTo(cards.last());
  await expect.poll(order).toEqual(["alternate","daily"]);
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.id==="alternate")).toBe(true);
  await library.getByRole("button",{name:"Drag rule group Daily to reorder",exact:true}).press("ArrowLeft");await expect.poll(order).toEqual(["daily","alternate"]);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(library.getByRole("button",{name:"Using rule group “Daily”",exact:true})).toHaveText("In use");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.id==="alternate")).toBe(true);
  expect(patches.every(p=>!p.tasks?.torrent_selection?.active_group)).toBe(true);
 });
 test("whole card surfaces open editors and referenced deletion clears all references",async({page})=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");
  await page.locator(".resource-selection__rule-tile").first().click({position:{x:2,y:2}});await expect(page.getByRole("dialog",{name:"User label",exact:true})).toBeVisible();await page.keyboard.press("Escape");await expect(page.getByRole("button",{name:"User label",exact:true})).toBeFocused();
  await page.locator(".resource-selection__card-actions").first().click({position:{x:48,y:3}});await expect(page.getByRole("dialog",{name:"Daily",exact:true})).toBeVisible();await page.keyboard.press("Escape");
  await page.getByRole("button",{name:"Delete rule “User label”",exact:true}).click();const confirm=page.getByRole("dialog",{name:"Delete rule",exact:true});await expect(confirm).toContainText("Daily、Alternate");await expect(confirm).toContainText("keeps priority tiers");await expect(confirm).toHaveScreenshot("referenced-rule-delete-confirm.png");
  await confirm.getByRole("button",{name:"Cancel",exact:true}).click();await expect(page.getByRole("button",{name:"User label",exact:true})).toBeVisible();
  await page.getByRole("button",{name:"Delete rule “User label”",exact:true}).click();await page.getByRole("dialog").getByRole("button",{name:"Delete",exact:true}).click();
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.rules?.length===4)).toBe(true);
  const saved=patches.find(p=>p.tasks?.torrent_selection?.rules?.length===4).tasks.torrent_selection;
  expect(saved.groups[0].require).toEqual(["extra-1","extra-2"]);expect(saved.groups.every((g:any)=>g.tiers.length===1&&g.tiers[0].rules.length===0)).toBe(true);
 });
 test("exclusion and priority pickers stay mutually exclusive in both directions",async({page},info)=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");await page.getByRole("button",{name:"Daily",exact:true}).click();const dialog=page.getByRole("dialog");
  const tiers=dialog.locator(".resource-selection__tier-grid"),tier=tiers.locator(".resource-selection__tier").first(),excluded=dialog.getByRole("group",{name:"Exclusion rules",exact:true});
  await expect(tier.getByRole("option",{name:"Subtitle labels",exact:true})).toHaveCount(0);await expect(excluded.getByRole("option",{name:"User label",exact:true})).toHaveCount(0);
  await tier.getByRole("combobox").selectOption("unused");await expect(excluded.getByRole("option",{name:"Unused rule",exact:true})).toHaveCount(0);
  await tier.getByRole("button",{name:"Remove reference to Unused rule",exact:true}).click();
  await excluded.getByRole("combobox").selectOption("unused");await expect(tier.getByRole("option",{name:"Unused rule",exact:true})).toHaveCount(0);
  await dialog.getByRole("button",{name:"Add priority tier",exact:true}).click();await dialog.getByRole("button",{name:"Add priority tier",exact:true}).click();
  if(info.project.name==="desktop") {const parent=(await tiers.boundingBox())!,card=(await tier.boundingBox())!;expect(card.width/parent.width).toBeGreaterThan(.48);expect(card.width/parent.width).toBeLessThan(.51);}
  await expect(tiers).toHaveScreenshot("compact-priority-cards.png");
 });
 test("empty condition drafts never reach saved settings",async({page})=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");await page.getByRole("button",{name:"User label",exact:true}).click();const dialog=page.getByRole("dialog");
  await dialog.getByRole("textbox",{name:"Match expression",exact:true}).fill("");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.rules?.[0]?.conditions?.length===0)).toBe(true);
  await dialog.getByRole("button",{name:"Add condition",exact:true}).click();await dialog.getByRole("combobox",{name:"Match field",exact:true}).nth(1).selectOption("SEEDERS");
  await dialog.getByRole("textbox",{name:"Minimum",exact:true}).fill("0");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.rules?.[0]?.conditions?.[0]?.minimum==="0")).toBe(true);
  expect(patches.every(p=>!p.tasks?.torrent_selection?.rules?.[0]?.conditions?.some((c:any)=>c.scope==="TITLE"&&!c.value.trim()))).toBe(true);
  await expect(dialog.locator(".resource-condition-row")).toHaveCount(2);
 });

 test("card keyboard focus covers the whole card and group controls use their new positions",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");
  await page.keyboard.press("Tab");
  for(const kind of ["rule","group"]){
   const card=page.locator(`.resource-selection__${kind}-tile`).first();await card.scrollIntoViewIfNeeded();
   await card.locator(`.resource-selection__${kind}-edit`).focus();
   await expect(card).toHaveCSS("outline-width","2px");await expect(card.locator(`.resource-selection__${kind}-edit`)).toHaveCSS("outline-style","none");
   await expect(card).toHaveScreenshot(`${kind}-whole-card-focus.png`);
  }
  const card=page.locator(".resource-selection__group-tile").first(),handle=(await card.locator(".resource-selection__drag").boundingBox())!,name=(await card.locator("strong").boundingBox())!,state=(await card.locator(".resource-selection__activate").boundingBox())!;
  expect(handle.x+handle.width).toBeLessThanOrEqual(name.x);expect(Math.abs(handle.y+handle.height/2-name.y-name.height/2)).toBeLessThan(2);expect(state.y).toBeGreaterThan(name.y);
  expect(Math.abs(state.x-handle.x)).toBeLessThan(20);
 });
 test("priority handles reorder empty tiers and share the standard dialog close style",async({page},info)=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");await page.getByRole("button",{name:"Daily",exact:true}).click();const dialog=page.getByRole("dialog",{name:"Daily",exact:true});
  const closeFont=await dialog.getByRole("button",{name:"Close",exact:true}).evaluate(el=>({size:getComputedStyle(el).fontSize,family:getComputedStyle(el).fontFamily}));
  await dialog.getByRole("button",{name:"Delete rule group",exact:true}).click();const confirm=page.getByRole("dialog",{name:"Delete rule group",exact:true});
  expect(await confirm.getByRole("button",{name:"Close",exact:true}).evaluate(el=>({size:getComputedStyle(el).fontSize,family:getComputedStyle(el).fontFamily}))).toEqual(closeFont);await confirm.getByRole("button",{name:"Cancel",exact:true}).click();
  await dialog.getByRole("button",{name:"Add priority tier",exact:true}).click();const grid=dialog.locator(".resource-selection__tier-grid"),cards=grid.locator(".resource-selection__tier");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.tiers?.length===2&&p.tasks.torrent_selection.groups[0].tiers[1].rules.length===0)).toBe(true);
  expect(await cards.last().locator(".resource-rule-tags__tag").count()).toBe(0);
  const handle=cards.last().locator(".resource-selection__drag");await handle.scrollIntoViewIfNeeded();await expect(handle).toHaveCSS("border-top-width","0px");
  if(info.project.name==="mobile"){
   const source=(await handle.boundingBox())!,target=(await cards.first().boundingBox())!;const touch=await page.context().newCDPSession(page);
   await touch.send("Input.dispatchTouchEvent",{type:"touchStart",touchPoints:[{x:source.x+source.width/2,y:source.y+source.height/2}]});
   await touch.send("Input.dispatchTouchEvent",{type:"touchMove",touchPoints:[{x:target.x+20,y:Math.min(target.y+target.height-20,source.y-30)}]});await touch.send("Input.dispatchTouchEvent",{type:"touchEnd",touchPoints:[]});await touch.detach();
  }else await handle.dragTo(cards.first());
  await expect(dialog.getByRole("textbox",{name:"Tier name",exact:true}).first()).toHaveValue("Priority tier 2");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.tiers?.[0]?.name==="Priority tier 2")).toBe(true);
  await cards.first().locator(".resource-selection__drag").press("ArrowRight");await expect(dialog.getByRole("textbox",{name:"Tier name",exact:true}).first()).toHaveValue("First choice");
  await dialog.locator(".resource-selection__prefilter").scrollIntoViewIfNeeded();await expect(dialog.locator(".resource-selection__prefilter")).toHaveScreenshot("compact-prefilter.png");
 });

 test("editable titles and header add icons replace separate name and add rows",async({page})=>{
  const patches:any[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="PatchSettings")patches.push(body.variables.input);}});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");await page.mouse.move(0,0);
  const cards=page.locator(".resource-selection__group-tile");expect(await cards.first().evaluate(el=>getComputedStyle(el).borderColor)).toBe(await cards.last().evaluate(el=>getComputedStyle(el).borderColor));
  await page.getByRole("button",{name:"User label",exact:true}).click();const editor=page.locator(".resource-selection-dialog");
  await expect(editor).toHaveAccessibleName("User label");await expect(editor.locator(".drawer-body").getByRole("textbox",{name:"Name",exact:true})).toHaveCount(0);
  await editor.locator(".drawer__head").getByRole("textbox",{name:"Name",exact:true}).fill("Renamed rule");await expect(editor).toHaveAccessibleName("Renamed rule");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.rules?.[0]?.name==="Renamed rule")).toBe(true);await page.keyboard.press("Escape");
  await page.getByRole("button",{name:"Daily",exact:true}).click();await editor.locator(".drawer__head").getByRole("textbox",{name:"Group name",exact:true}).fill("Renamed group");await expect(editor).toHaveAccessibleName("Renamed group");
  for(const filter of await editor.locator(".resource-rule-tags").all()) {
   const picker=filter.locator(".resource-rule-picker"),heading=filter.locator("h4"),box=(await picker.boundingBox())!,title=(await heading.boundingBox())!;
   expect(box.x).toBeGreaterThan(title.x);expect(box.y).toBeGreaterThanOrEqual(title.y-1);expect(box.y-title.y).toBeLessThan(2);await expect(picker.getByRole("combobox")).toHaveCSS("opacity","0");
  }
  const tier=editor.locator(".resource-selection__tier").first();await tier.getByRole("textbox",{name:"Tier name",exact:true}).fill("Renamed priority");
  const picker=tier.locator(".resource-rule-picker"),remove=tier.getByRole("button",{name:"Delete tier",exact:true}),box=(await picker.boundingBox())!,deleteBox=(await remove.boundingBox())!;
  expect(box.x+box.width).toBeLessThanOrEqual(deleteBox.x);expect(Math.abs(box.y-deleteBox.y)).toBeLessThan(2);
  await picker.getByRole("combobox").selectOption("unused");await expect(tier).toContainText("Unused rule");
  await expect.poll(()=>patches.some(p=>p.tasks?.torrent_selection?.groups?.[0]?.name==="Renamed group"&&p.tasks.torrent_selection.groups[0].tiers[0].name==="Renamed priority")).toBe(true);
  await expect(editor).toHaveScreenshot("editable-titles-and-add-icons.png");
 });

 test("editor focus title width square icons and left alignment",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");const library=page.locator(".resource-selection");
  for(const icon of await library.locator(".icon-button").all()) {const box=(await icon.boundingBox())!;expect(Math.abs(box.width-box.height)).toBeLessThan(1);}
  const card=library.locator(".resource-selection__group-tile").first(),body=(await card.locator(".resource-selection__group-body").boundingBox())!,inset=await card.locator(".resource-selection__group-body").evaluate(el=>parseFloat(getComputedStyle(el).paddingInlineStart));
  const textX=body.x+inset,grip=(await card.locator(".resource-selection__drag svg").boundingBox())!,state=(await card.locator(".resource-selection__activate svg").boundingBox())!;
  expect(Math.abs(grip.x-textX)).toBeLessThan(2);expect(Math.abs(state.x-textX)).toBeLessThan(2);
  for(const [name,label] of [["User label","Name"],["Daily","Group name"]]){
   const opener=library.getByRole("button",{name,exact:true});await opener.click();const editor=page.locator(".resource-selection-dialog");
   await expect(editor).toBeFocused();const title=editor.locator(".drawer__head").getByRole("textbox",{name:label,exact:true});await expect(title).not.toBeFocused();
   const box=(await title.boundingBox())!,parent=(await editor.locator(".drawer__head > div").boundingBox())!;expect(box.width/parent.width).toBeGreaterThan(.38);expect(box.width/parent.width).toBeLessThan(.42);
   await page.keyboard.press("Tab");await expect(title).toBeFocused();
   if(name==="User label"){
    const add=editor.getByRole("button",{name:"Add condition",exact:true});expect(await add.locator("svg").count()).toBe(1);await add.click();await expect(editor.locator(".resource-condition-row")).toHaveCount(2);
    const header=editor.locator(".resource-rule-editor__condition-heading"),button=(await add.boundingBox())!,head=(await header.boundingBox())!;expect(Math.abs(button.x+button.width-head.x-head.width)).toBeLessThan(2);
   }
   for(const icon of await editor.locator(".icon-button, .resource-rule-tags__remove").all()) {const box=(await icon.boundingBox())!;expect(Math.abs(box.width-box.height)).toBeLessThan(1);}
   if(name==="Daily") {const tier=editor.locator(".resource-selection__tier").first(),handle=(await tier.locator(".resource-selection__drag svg").boundingBox())!,references=(await tier.locator(".resource-selection__references").boundingBox())!;expect(Math.abs(handle.x-references.x)).toBeLessThan(2);}
   await page.keyboard.press("Escape");await expect(opener).toBeFocused();
  }
 });

 test("only indexer result fields are configurable and inspection controls are absent",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");
  const panel=page.locator(".resource-selection");await expect(panel.locator("input[type=number]")).toHaveCount(0);await expect(panel.locator(".settings-info")).toHaveCount(0);
  await panel.getByRole("button",{name:"User label",exact:true}).click();
  expect(await page.getByRole("dialog").getByRole("combobox",{name:"Match field",exact:true}).locator("option").evaluateAll(items=>items.map(item=>(item as HTMLOptionElement).value))).toEqual(["TITLE","SIZE","SEEDERS"]);
 });

 for(const locale of ["qps-ploc","ar-XB"] as const){
  test(`${locale} resource cards and editors fit long text and direction`,async({page})=>{
   await page.goto(`/settings/tasks?panel=resources&__locale=${locale}`);
   await expect(page.locator("html")).toHaveAttribute("dir",locale==="ar-XB"?"rtl":"ltr");
   const panel=page.locator(".resource-selection");await expect(panel.locator(":scope > .drawer-card")).toHaveCount(2);
   await expect.poll(()=>panel.evaluate(el=>el.scrollWidth<=el.clientWidth)).toBe(true);
   await page.evaluate(()=>document.fonts.ready);await expect(page).toHaveScreenshot(`${locale}-resource-cards.png`);
   await panel.getByRole("button",{name:"Daily",exact:true}).click();
   const dialog=page.getByRole("dialog");await expect(dialog).toBeVisible();
   await expect.poll(()=>dialog.evaluate(el=>el.scrollWidth<=el.clientWidth)).toBe(true);
   await expect(dialog).toHaveScreenshot(`${locale}-group-editor.png`);
   await page.keyboard.press("Escape");await expect(panel.getByRole("button",{name:"Daily",exact:true})).toBeFocused();
  });
 }

});

const film={endpoint:"https://metadata.example/graphql",sceneId:"film-one",code:"MOJI-102",title:"Synthetic target film",date:"2026-01-01",studio:"Synthetic studio",performers:[],imageUrl:"",url:"",durationSeconds:7200,confirmationDigest:"confirmed-digest"};
const evaluation={size:12000000000,seeders:42,groupName:"Daily",tierName:"First choice",evidenceOmitted:0,candidateId:searchCandidate.id,title:searchCandidate.title,status:"ACCEPTABLE",preferred:true,reasons:["TIER_MATCHED"],unknownRules:[],evidence:[]};
const completed={searchId:"search-one",searchRevision:1,partial:false,total:1,offset:0,limit:24,candidates:[searchCandidate],evaluations:[evaluation]};
test.describe("indexer-only preview",()=>{
 test.use({graphqlOverrides:{settings:()=>ruleSettings,taskFilmCandidates:()=>[film],taskFilm:()=>film,previewTorrentSelection:()=>completed}});
 test("confirmed film preview is synchronous and never starts inspection",async({page})=>{
  const requests:Array<{operationName:string;variables:any}>=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST")requests.push(request.postDataJSON());});
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/discover?source=indexers&q=MOJI-102&run=search-one");
  await expect(page.getByRole("button",{name:"Preview selection",exact:true})).toBeDisabled();
  await page.getByRole("button",{name:"Choose a film to preview rules",exact:true}).click();await page.getByRole("button",{name:"MOJI-102 · Synthetic target film",exact:true}).click();
  await page.getByRole("combobox",{name:"Preview rule group",exact:true}).selectOption("alternate");
  await expect(page.getByLabel("Inspect torrent contents",{exact:true})).toHaveCount(0);await page.getByRole("button",{name:"Preview selection",exact:true}).click();
  await expect(page.locator(".resource-evaluation")).toContainText("Priority tier matched");await expect(page).not.toHaveURL(/preview=/);
  const calls=requests.filter(r=>r.operationName==="PreviewTorrentSelection");expect(calls).toHaveLength(1);expect(calls[0].variables.input).toMatchObject({endpoint:film.endpoint,sceneId:film.sceneId,confirmationDigest:film.confirmationDigest,groupID:"alternate"});expect(calls[0].variables.input).not.toHaveProperty("inspect");
  await page.reload();await expect(page.locator(".resource-evaluation")).toHaveCount(0);expect(requests.filter(r=>r.operationName==="PreviewTorrentSelection")).toHaveLength(1);
 });
});
const task={...homeOverview.attentionTasks[1],id:"selection-task",code:"MOJI-102"};
test.describe("task selection",()=>{
 test.use({graphqlOverrides:{task:()=>task,previewTorrentSelection:()=>({...completed,searchId:"task-search-selection-task"})}});
 test("task recommendations use immediate matching without a background job",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/tasks/selection-task/resolve?sourcingTask=selection-task&sourcingRun=task-search-selection-task");
  await page.getByRole("button",{name:"Recommend a resource",exact:true}).click();await expect(page.getByRole("button",{name:"Select recommended resource",exact:true})).toBeVisible();await expect(page).not.toHaveURL(/sourcingPreview=/);
  await page.reload();await expect(page.getByRole("button",{name:"Select recommended resource",exact:true})).toHaveCount(0);
 });
});
test.describe("selection report",()=>{
 test.use({graphqlOverrides:{task:()=>({...task,selectionReport:{groupName:"Daily",end:"PREFERRED",partial:false,selected:searchCandidate.title,evaluations:[{...evaluation,selected:true,excludedRules:[]}]}})}});
 test("task detail explains the persisted decision",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/tasks/selection-task");
  const sourcing=page.locator(".task-timeline > li").first();
  await sourcing.getByRole("button",{name:"View selection report",exact:true}).click();
  const report=page.getByRole("dialog",{name:"Selection report · MOJI-102",exact:true});
  await expect(report).toContainText("First-priority resource found");await expect(report).toContainText("First choice");
  await expect(report).toHaveScreenshot("task-selection-report.png");
 });
});

const detailedReport = {
 groupName:"Daily",end:"BEST_AVAILABLE",partial:true,selected:searchCandidate.title,
 evaluations:[{selected:true,title:searchCandidate.title,size:12000000000,seeders:42,tierName:"First choice",status:"ACCEPTABLE",reasons:["TIER_MATCHED"],excludedRules:[]},{selected:false,title:"MOJI-102 a very long alternative resource name with user-defined labels and additional release information",status:"INSUFFICIENT_DATA",size:null,seeders:null,tierName:"",reasons:[],excludedRules:[]},{selected:false,title:"MOJI-102 excluded resource with unwanted-label",status:"REJECTED",size:0,seeders:0,tierName:"",reasons:["EXCLUDED_RULE_MATCHED","REQUIRED_RULE_FAILED"],excludedRules:["Excluded labels"]}]
};
test.describe("selection report hierarchy",()=>{
 test.use({graphqlOverrides:{task:()=>({...task,selectionReport:detailedReport})}});
 for(const locale of ["zh-CN","en"] as const) for(const theme of ["light","dark"] as const) {
  test(`${locale} ${theme} sourcing explanation shows compact candidate summaries`,async({page})=>{
   await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme);},{locale,theme});
   await page.goto("/tasks/selection-task");await page.evaluate(()=>document.fonts.ready);
   const sourcing=page.locator(".task-timeline > li").first(),entry=sourcing.locator(".task-selection-report-entry");
   await expect(page.locator(".task-selection-report-dialog")).toHaveCount(0);
   await expect(sourcing.locator(".task-timeline__actions button").first()).toHaveText(locale==="en"?"View selection report":"查看选种报告");
   await expect(sourcing).toHaveScreenshot(`${locale}-${theme}-selection-report-entry.png`);
   await entry.focus();await page.keyboard.press("Enter");
   const report=page.getByRole("dialog",{name:locale==="en"?"Selection report · MOJI-102":"选种报告 · MOJI-102",exact:true});
   await expect(report).toBeVisible();
   await expect(report.getByText("Daily",{exact:true})).toHaveCount(1);
   await expect(report.getByRole("img",{name:locale==="en"?"Meets selection conditions":"符合选择条件",exact:true})).toBeVisible();
   await expect(page.locator(".drawer--task")).toHaveJSProperty("inert",true);
   await page.keyboard.press("Tab");await expect(report.getByRole("button",{name:locale==="en"?"Close":"关闭",exact:true})).toBeFocused();
   await expect(report.getByRole("img",{name:locale==="en"?"Insufficient information to confirm":"信息不足，无法确认",exact:true})).toBeVisible();
   await expect(report.getByRole("img",{name:locale==="en"?"Does not meet selection conditions":"未通过选择条件",exact:true})).toBeVisible();
   const rejected=report.locator(".task-selection-report__candidate").nth(2);
   await expect(rejected).toContainText(locale==="en"?"Excluded by: Excluded labels":"命中排除条件：Excluded labels");
   await expect(report.locator("details, .task-selection-report__evidence")).toHaveCount(0);
   await expect(report.locator(".task-selection-report__candidate").first()).toContainText(locale==="en"?"Priority tier matched: First choice":"命中优先级：First choice");
   const chosen=report.locator(".task-selection-report__candidate.is-selected");await expect(chosen).toHaveCount(1);
   const watermark=chosen.locator(".task-selection-report__watermark");await expect(watermark).toBeVisible();
   const selectedBox=(await chosen.boundingBox())!,markBox=(await watermark.boundingBox())!;
   expect(Math.abs(selectedBox.x+selectedBox.width/2-markBox.x-markBox.width/2)).toBeLessThan(2);
   expect(Math.abs(selectedBox.y+selectedBox.height/2-markBox.y-markBox.height/2)).toBeLessThan(2);
   const reasons=rejected.locator(".task-selection-report__reasons li");await expect(reasons).toHaveCount(2);
   expect(await reasons.last().evaluate(el=>getComputedStyle(el,"::before").content)).toContain("·");
   expect(await reasons.last().evaluate(el=>getComputedStyle(el).display)).toBe("inline");
   await expect(report.getByText(locale==="en"?"Selected resource":"选中资源",{exact:true})).toHaveCount(0);
   const icon=chosen.locator(".task-selection-report__status-icon"),title=chosen.locator(".task-selection-report__candidate-title");
   expect((await icon.boundingBox())!.x).toBeLessThan((await title.boundingBox())!.x);
   await expect(icon).toHaveText("");
   const meta=chosen.locator(".task-selection-report__candidate-meta"),facts=meta.locator("dl"),reason=meta.locator("ul");
   const factsBox=(await facts.boundingBox())!,reasonBox=(await reason.boundingBox())!;
   expect(reasonBox.x).toBeGreaterThan(factsBox.x);expect(Math.abs(reasonBox.y-factsBox.y)).toBeLessThan(8);
   await expect(report).toHaveScreenshot(`${locale}-${theme}-selection-report-hierarchy.png`);
   expect(await report.evaluate(element=>element.scrollWidth<=element.clientWidth)).toBe(true);
   const box=(await report.boundingBox())!;expect(box.x+box.width).toBeLessThanOrEqual(page.viewportSize()!.width);
   await page.keyboard.press("Escape");await expect(report).toHaveCount(0);await expect(entry).toBeFocused();
   await expect(page.locator(".drawer--task")).toHaveJSProperty("inert",false);
  });
 }
});

test.describe("selection report actions",()=>{
 test.use({graphqlOverrides:{task:()=>({...task,stage:"SOURCING",stageStatus:"PENDING",stageErrorCode:"",sourcingNextCheckAt:"2026-01-15T18:00:00Z",selectionReport:detailedReport})}});
 for(const locale of ["zh-CN","en"] as const) for(const theme of ["light","dark"] as const) {
  test(`${locale} ${theme} sourcing actions share dimensions and interaction with distinct priority`,async({page})=>{
   await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme);},{locale,theme});
   await page.goto("/tasks/selection-task");await page.evaluate(()=>document.fonts.ready);
   const sourcing=page.locator(".task-timeline > li").first(),actions=sourcing.locator(".task-timeline__actions"),buttons=actions.getByRole("button");
   await expect(buttons).toHaveCount(3);await expect(buttons.first()).toHaveText(locale==="en"?"View selection report":"查看选种报告");
   const styles=await buttons.evaluateAll(elements=>elements.map(element=>{const s=getComputedStyle(element);return {height:element.getBoundingClientRect().height,fontSize:s.fontSize,fontWeight:s.fontWeight,borderRadius:s.borderRadius,padding:s.padding,transform:s.transform,color:s.color,borderColor:s.borderColor,background:s.backgroundColor};}));
   for(const style of styles)for(const key of ["height","fontSize","fontWeight","borderRadius","padding","transform"] as const)expect(style[key]).toBe(styles[0][key]);
   const touchSizing=await page.evaluate(()=>matchMedia("(max-width: 640px), (pointer: coarse)").matches);
   expect(styles[0].height).toBe(touchSizing?44:38);expect(styles[0].color).toBe(styles[1].color);expect(styles[0].color).not.toBe(styles[2].color);
   for(let i=0;i<3;i++){await buttons.nth(i).hover();await expect(buttons.nth(i)).toHaveCSS("transform","none");}
   await page.mouse.move(0,0);await actions.scrollIntoViewIfNeeded();await expect(actions).toHaveScreenshot(`${locale}-${theme}-sourcing-action-styles.png`);
  });
 }
});

let tagSettings: typeof ruleSettings;
test.describe("pre-filter tags",()=>{
 test.use({graphqlOverrides:{settings:()=>tagSettings,patchSettings:({input}:{input:any})=>{
  if(input.tasks?.torrent_selection?.groups)tagSettings={...tagSettings,automation:{...tagSettings.automation,torrentSelection:{...tagSettings.automation.torrentSelection,groups:input.tasks.torrent_selection.groups}}};
  return {...tagSettings,system:{...tagSettings.system,language:"auto"}};
 }}});
 test("pre-filter tags follow library order and have no drag handles",async({page})=>{
  tagSettings=structuredClone(ruleSettings);tagSettings.automation.torrentSelection.groups[0].require=["extra-2","label","extra-1"];
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");await page.getByRole("button",{name:"Daily",exact:true}).click();
  const required=page.getByRole("group",{name:"Admission rules",exact:true}),tags=required.locator(".resource-rule-tags__name");
  await expect(tags).toHaveText(["User label","Resolution labels","A long user-defined rule name that wraps within its card"]);
  await expect(required.locator("[draggable]")).toHaveCount(0);
  const title=(await required.locator("h4").boundingBox())!,hint=(await required.locator(".settings-note").boundingBox())!;expect(hint.y-title.y-title.height).toBeGreaterThanOrEqual(0);expect(hint.y-title.y-title.height).toBeLessThanOrEqual(3);
  await required.getByRole("button",{name:"Remove reference to Resolution labels",exact:true}).click();await required.getByRole("combobox").selectOption("extra-1");
  await expect(tags).toHaveText(["User label","Resolution labels","A long user-defined rule name that wraps within its card"]);
  await page.keyboard.press("Escape");
  await page.getByRole("button",{name:"Drag User label to reorder",exact:true}).press("ArrowRight");
  // Library order is now extra-0, label, extra-1, extra-2: move label once more.
  await page.getByRole("button",{name:"Drag User label to reorder",exact:true}).press("ArrowRight");
  await page.getByRole("button",{name:"Daily",exact:true}).click();
  await expect(tags).toHaveText(["Resolution labels","User label","A long user-defined rule name that wraps within its card"]);
 });
});

const fixedHeightSettings=structuredClone(ruleSettings);
fixedHeightSettings.automation.torrentSelection.groups[0].name="A long rule group name that wraps over more than two lines while keeping its card the same height as an empty group";
fixedHeightSettings.automation.torrentSelection.groups[0].tiers=Array.from({length:6},(_,index)=>({id:`tier-${index}`,name:`Priority ${index+1}`,rules:["label"]}));
fixedHeightSettings.automation.torrentSelection.groups[1].tiers=[];
test.describe("fixed rule group cards",()=>{
 test.use({graphqlOverrides:{settings:()=>fixedHeightSettings}});
 test("empty and truncated previews use the same fixed card height",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/tasks?panel=resources");
  const cards=page.locator(".resource-selection__group-grid > [data-library-group-id]");await expect(cards).toHaveCount(2);
  for(const card of await cards.all()){const box=(await card.boundingBox())!;expect(box.height).toBe(248);expect(await card.evaluate(el=>el.scrollHeight<=el.clientHeight)).toBe(true);}
  const lastPreview=(await cards.first().locator(".resource-selection__tier-preview li").last().boundingBox())!,footer=(await cards.first().locator(".resource-selection__card-actions").boundingBox())!;expect(lastPreview.y+lastPreview.height).toBeLessThanOrEqual(footer.y);
  await expect(cards.first().locator(".resource-selection__tier-preview li")).toHaveCount(4);await expect(cards.first()).toContainText("3 more");
  await expect(cards.last().locator(".resource-selection__tier-preview")).toHaveCount(0);
  await cards.first().scrollIntoViewIfNeeded();await expect(cards.first()).toHaveScreenshot("fixed-height-truncated-group.png");
 });
});
