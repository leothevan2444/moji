import { expect, type Locator } from "@playwright/test";
import { test, homeOverview, settingsSnapshot } from "./fixtures";

const actors=Array.from({length:211},(_,index)=>({id:`pager-actor-${index}`,name:`Performer ${index}`,aliasList:[],favorite:false,imagePath:null,sceneCount:211,subscribed:false}));
const scenes=Array.from({length:211},(_,index)=>({key:`pager-scene-${index}`,primarySource:"STASHBOX",sourceSceneId:`pager-scene-${index}`,code:`PAGE-${index}`,title:`Scene ${index}`,date:"2026-01-01",studioName:"Synthetic",performerCount:2,tagCount:0,performers:[],tags:[],imageUrl:null,url:null,inLibrary:false,matchedStashSceneId:null,hasStashSource:false,hasStashBoxSource:true,stashBoxSceneId:`pager-scene-${index}`,stashBoxEndpoint:"https://metadata.example/graphql",sourceLabels:["StashBox"],stashIds:[],mojiTask:null}));
const profile={performer:actors[0],disambiguation:null,birthdate:null,ethnicity:null,country:null,eyeColor:null,heightCm:null,rating100:null,urls:[],matchedStashBox:null,totalSceneCount:211,stashSceneCount:0,stashBoxSceneCount:211,dedupedSceneCount:211,activeTaskCount:0,sceneStatsComplete:true,sceneStatsStale:false};
const catalog={
  tasks:Array.from({length:211},(_,index)=>({...homeOverview.activeTasks[0],id:`pager-task-${index}`,code:`PAGE-${index}`})),
  performerWorkspace:({page=1,pageSize=24}:{page?:number;pageSize?:number})=>({activities:[],performers:{items:actors.slice((page-1)*pageSize,page*pageSize),page,pageSize,totalCount:211,totalPages:Math.ceil(211/pageSize),hasPrevPage:page>1,hasNextPage:page*pageSize<211},subscribedPerformers:[]}),
  stashPerformerProfile:profile,stashPerformerDetail:profile,
  stashPerformerScenes:({input}:{input:{page?:number;pageSize?:number;cast?:string}})=>{
    const page=input.page??1,pageSize=input.pageSize??24,exact=input.cast!=="GROUP"||page>=4;
    const totalCount=exact?211:64,totalPages=exact?Math.ceil(totalCount/pageSize):null;
    return {items:scenes.slice((page-1)*pageSize,Math.min(page*pageSize,totalCount)),page,pageSize,totalCount,totalPages,hasPrevPage:exact&&page>1,hasNextPage:!exact||page*pageSize<211,stashSceneCount:0,stashBoxCount:211,dedupedCount:totalCount,totalCountExact:exact,stashBoxRemoteCount:211,stashBoxLoadedCount:totalCount,cacheComplete:exact,cacheUpdatedAt:null,cacheStale:false};
  },
};
const views=[{name:"tasks",path:"/tasks",key:"page",size:"pageSize"},{name:"performers",path:"/performers",key:"page",size:"pageSize"},{name:"scenes",path:"/performers/pager-actor-0",key:"scenePage",size:"scenePageSize"}] as const;
test.use({graphqlOverrides:catalog});

async function singleRow(pager:Locator) {
  await expect.poll(()=>pager.locator(".pagination__viewport > .pagination__row").evaluate(row=>{
    const boxes=[...row.children].filter(child=>getComputedStyle(child).display!=="none").map(child=>child.getBoundingClientRect());
    return boxes.length>2&&Math.max(...boxes.map(box=>box.top))<Math.min(...boxes.map(box=>box.bottom));
  })).toBe(true);
}

for(const view of views) {
  test(`${view.name} shares upper and lower page state, first/last navigation and page size`,async({page})=>{
    await page.setViewportSize({width:1440,height:1000});
    await page.goto(`${view.path}?${view.key}=3`);
    const pagers=page.locator(".pagination"),top=pagers.first(),bottom=pagers.last();
    await expect(pagers).toHaveCount(2);
    for(const pager of [top,bottom]) await expect(pager.getByRole("textbox",{name:"Page number"})).toHaveValue("3");
    await expect(top.getByRole("status")).toContainText("Page 3 / 9, 211 items");
    const toolbar=page.locator(".workspace-toolbar").first();
    expect((await top.boundingBox())!.y).toBeGreaterThanOrEqual((await toolbar.boundingBox())!.y+(await toolbar.boundingBox())!.height);
    const lastItem=page.locator(view.name==="tasks"?".task-card":view.name==="performers"?".profile-card":".performer-scene-card").last();
    expect((await bottom.boundingBox())!.y).toBeGreaterThanOrEqual((await lastItem.boundingBox())!.y+(await lastItem.boundingBox())!.height);
    const jump=bottom.getByRole("textbox",{name:"Page number"});await jump.fill("7");await jump.press("Enter");
    await expect.poll(()=>new URL(page.url()).searchParams.get(view.key)).toBe("7");
    for(const pager of [top,bottom]) await expect(pager.getByRole("textbox",{name:"Page number"})).toHaveValue("7");
    await top.getByRole("button",{name:"First page",exact:true}).click();
    await expect(top.getByRole("textbox")).toHaveValue("1");
    await bottom.getByRole("button",{name:"Last page",exact:true}).click();
    await expect(top.getByRole("textbox")).toHaveValue("9");
    await top.getByRole("button",{name:"Previous",exact:true}).click();await expect(bottom.getByRole("textbox")).toHaveValue("8");
    await bottom.getByRole("button",{name:"Next",exact:true}).click();await expect(top.getByRole("textbox")).toHaveValue("9");
    await bottom.getByRole("combobox",{name:"Page size",exact:true}).selectOption("96");
    await expect.poll(()=>new URL(page.url()).searchParams.get(view.size)).toBe("96");
    await expect(top.getByRole("status")).toContainText("Page 1 / 3, 211 items");
    expect(new URL(page.url()).searchParams.has(view.key)).toBe(false);
    await page.reload();
    for(const pager of [top,bottom]) await expect(pager.getByRole("combobox",{name:"Page size",exact:true})).toHaveValue("96");
  });

  test(`${view.name} hides size before edge navigation and preserves left middle right placement`,async({page},info)=>{
    await page.setViewportSize({width:1440,height:1000});await page.goto(`${view.path}?${view.key}=3`);
    const top=page.locator(".pagination").first();await expect(top.getByRole("textbox")).toHaveValue("3");
    const [full,partial]=await top.locator(".pagination__measurement > .pagination__row").evaluateAll(nodes=>nodes.map(node=>node.scrollWidth));
    expect(full).toBeGreaterThan(partial+4);
    const stages=[{width:full+12,size:true,edges:true},{width:full+8,size:false,edges:true},{width:partial+12,size:false,edges:true},{width:partial+8,size:false,edges:false}];
    for(const [index,stage] of [...stages,...stages.slice().reverse()].entries()) {
      await top.evaluate((node,width)=>{(node as HTMLElement).style.width=`${width}px`;},stage.width);
      await expect(top.getByRole("combobox",{name:"Page size",exact:true})).toHaveCount(stage.size?1:0);
      await expect(top.getByRole("button",{name:"First page",exact:true})).toHaveCount(stage.edges?1:0);
      await expect(top.getByRole("button",{name:"Last page",exact:true})).toHaveCount(stage.edges?1:0);
      await singleRow(top);
      await expect.poll(()=>top.locator(".pagination__viewport > .pagination__row").evaluate(node=>{
        const navigation=node.querySelector(".pagination__navigation")!.getBoundingClientRect(),count=node.querySelector(".pagination__count")!.getBoundingClientRect(),controls=node.querySelector(".pagination__controls")!.getBoundingClientRect(),row=node.getBoundingClientRect(),viewport=node.parentElement!.getBoundingClientRect();
        return navigation.right<=count.left&&count.right<=controls.left&&Math.abs(navigation.left-row.left)<1&&Math.abs(controls.right-row.right)<1&&Math.abs((row.left+row.right-viewport.left-viewport.right)/2)<1;
      })).toBe(true);
      if(!stage.size) {
        await top.getByRole("button",{name:"More pagination options"}).click();
        const popup=page.getByRole("dialog");await expect(popup.getByRole("combobox",{name:"Page size"})).toBeVisible();
        await expect(popup.getByRole("button",{name:"First page",exact:true})).toHaveCount(stage.edges?0:1);
        await expect(popup.getByRole("button",{name:"Last page",exact:true})).toHaveCount(stage.edges?0:1);
        await page.keyboard.press("Escape");
      }
      if(index<stages.length)await top.screenshot({path:info.outputPath(`${view.name}-stage-${index}.png`)});
    }
  });

  test(`${view.name} collapses options without wrapping and keeps popup and focus inside the viewport`,async({page},info)=>{
    await page.setViewportSize({width:1440,height:1000});await page.goto(`${view.path}?${view.key}=3`);
    const top=page.locator(".pagination").first();
    const size=top.getByRole("combobox",{name:"Page size",exact:true});
    await expect(size).toBeEnabled();await size.focus();await expect(size).toBeFocused();
    await page.setViewportSize({width:320,height:800});
    await expect(top).toHaveClass(/is-compact/);await singleRow(top);
    await expect(top.getByRole("button",{name:"More pagination options"})).toBeFocused();
    await expect(top.getByRole("button",{name:"First page",exact:true})).toHaveCount(0);
    expect(await top.locator(".pagination__viewport").evaluate(node=>node.scrollWidth<=node.clientWidth+1)).toBe(true);
    await top.screenshot({path:info.outputPath(`${view.name}-pagination-320.png`)});
    for(const zoom of [0.8,1,1.25]) {
      await page.setViewportSize({width:Math.round(320*zoom),height:800});await page.evaluate(value=>{document.documentElement.style.zoom=String(value);},zoom);
      const bottom=page.locator(".pagination").last(),more=bottom.getByRole("button",{name:"More pagination options"});
      await more.click();
      const popup=page.getByRole("dialog",{name:"More pagination options"});
      await expect(popup.getByRole("combobox",{name:"Page size"})).toBeFocused();
      await expect.poll(()=>popup.evaluate(node=>{const box=node.getBoundingClientRect();return box.left>=7&&box.right<=innerWidth-7&&box.top>=7&&box.bottom<=innerHeight-7;})).toBe(true);
      await popup.screenshot({path:info.outputPath(`${view.name}-pagination-options-${zoom}.png`)});
      await page.keyboard.press("Escape");await expect(popup).toHaveCount(0);await expect(more).toBeFocused();
    }
    await page.evaluate(()=>{document.documentElement.style.zoom="1";});
    await page.setViewportSize({width:320,height:800});
    const more=top.getByRole("button",{name:"More pagination options"});await more.click();
    await page.keyboard.press("Shift+Tab");await expect(page.getByRole("dialog")).toHaveCount(0);await expect(more).toBeFocused();
    await more.click();await page.keyboard.press("Tab");await page.keyboard.press("Tab");await page.keyboard.press("Tab");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await more.click();await page.getByRole("dialog").getByRole("combobox",{name:"Page size"}).selectOption("48");
    await expect(top.getByRole("textbox")).toHaveValue("1");await expect(more).toBeFocused();
    await more.click();await page.getByRole("dialog").getByRole("button",{name:"Last page",exact:true}).click();
    await expect(top.getByRole("textbox")).toHaveValue("5");
    await more.click();await page.setViewportSize({width:1440,height:1000});
    await expect(top).not.toHaveClass(/is-compact/);await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(top.getByRole("combobox",{name:"Page size",exact:true})).toBeFocused();
  });
}

test("unknown scene totals bound direct jumps, allow next-page loading, and recover full navigation",async({page})=>{
  await page.setViewportSize({width:1440,height:1000});
  const reads:number[]=[];page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST"){const body=request.postDataJSON();if(body.operationName==="StashPerformerScenes")reads.push(body.variables.input.page);}});
  await page.goto("/performers/pager-actor-0?cast=group&scenePage=3");
  const pagers=page.locator(".pagination"),top=pagers.first();
  await expect(top.getByRole("status")).toContainText("Page 3 / —, 64 loaded");
  await expect(top.getByRole("button",{name:"Last page",exact:true})).toBeDisabled();
  await expect(top.getByRole("button",{name:"Previous",exact:true})).toBeEnabled();
  const field=top.getByRole("textbox");await field.fill("4");await field.press("Enter");
  expect(await field.evaluate(node=>(node as HTMLInputElement).validity.valid)).toBe(false);expect(reads.includes(4)).toBe(false);
  await top.getByRole("button",{name:"Next",exact:true}).click();
  await expect(top.getByRole("status")).toContainText("Page 4 / 9, 211 items");
  await pagers.last().getByRole("button",{name:"Last page",exact:true}).click();
  await expect(top.getByRole("textbox")).toHaveValue("9");
});

for(const locale of ["zh-CN","en","qps-ploc","ar-XB"] as const) {
 test.describe(locale,()=>{
  test.use({graphqlOverrides:{...catalog,settings:{...settingsSnapshot,system:{...settingsSnapshot.system,language:locale==="zh-CN"?"zh-CN":"en"}}}});
  test("all pagination rows survive widths and scale in both directions",async({page},info)=>{
    for(const view of views) {
      await page.goto(`${view.path}?__locale=${locale}`);
      const pagers=page.locator(".pagination");await expect(pagers).toHaveCount(2);
      for(const zoom of [0.8,1,1.25]) {
        await page.evaluate(value=>{document.documentElement.style.zoom=String(value);},zoom);
        for(const width of [1440,800,400,320,800,1440]) {
          if(width/zoom<320)continue;
          await page.setViewportSize({width,height:1000});
          for(const pager of await pagers.all())await singleRow(pager);
          const geometry=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,client:document.documentElement.clientWidth,viewport:innerWidth,
            overflow:[...document.querySelectorAll("main *")].filter(node=>{const rect=node.getBoundingClientRect();return getComputedStyle(node).visibility!=="hidden"&&!node.closest(".pagination__announcement")&&rect.width>0&&(rect.left < -1 || rect.right > innerWidth+1);}).slice(0,12).map(node=>({tag:node.tagName,class:node.className,left:node.getBoundingClientRect().left,right:node.getBoundingClientRect().right}))}));
          await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth+1),{message:JSON.stringify({view:view.name,locale,width,zoom,...geometry})}).toBe(true);
        }
      }
    }
    await page.evaluate(()=>{document.documentElement.style.zoom="1";});
    await page.emulateMedia({colorScheme:"dark"});await page.setViewportSize({width:320,height:800});
    const top=page.locator(".pagination").first();
    expect(await top.locator("form.pagination__page").evaluate(node=>node.querySelector("input")!.getBoundingClientRect().right<=node.querySelector(".pagination__pages")!.getBoundingClientRect().left)).toBe(true);
    if(locale==="ar-XB") expect(await top.getByRole("button").first().locator("svg").evaluate(node=>getComputedStyle(node).transform)).toBe("matrix(-1, 0, 0, 1, 0, 0)");
    await top.locator("button.pagination__more").click();
    const popup=page.getByRole("dialog");
    await expect.poll(()=>popup.evaluate(node=>{const frame=node.getBoundingClientRect();return [...node.querySelectorAll("button,select")].every(control=>{const box=control.getBoundingClientRect();return box.left>=frame.left&&box.right<=frame.right;});})).toBe(true);
    await popup.screenshot({path:info.outputPath(`${locale}-dark-options.png`)});
    await page.keyboard.press("Escape");await expect(popup).toHaveCount(0);
  });
 });
}
