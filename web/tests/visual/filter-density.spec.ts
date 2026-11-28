import {expect} from "@playwright/test";
import {test} from "./fixtures";

// Keep the counts in React's input, including measurement copies and rerenders.
let count=1;
const connection=()=>({items:[],page:1,pageSize:24,totalCount:count,totalPages:Math.ceil(count/24),hasPrevPage:false,hasNextPage:count>24});
test.use({graphqlOverrides:{
 taskPage:()=>({...connection(),totalCount:count*4,totalPages:Math.ceil(count*4/24),sequence:0,streamId:"visual",filterCounts:{total:count*4,active:count,waiting:count,completed:count,attention:count},stats:{total:count*4,active:count*3,waiting:count,completed:count,attention:count,downloading:count,pendingScans:0,failed:count}}),
 performerWorkspace:()=>({activities:[],subscribedPerformers:[],performers:connection(),filterCounts:{total:count,subscribed:count,favorite:count,review:count,error:count}}),
}});

for(const path of ['tasks','performers']) test(`${path} compact filters stay beside their title with up to four count digits`,async({page},info)=>{
 await page.addInitScript(()=>localStorage.setItem('moji:locale','zh-CN'));
 const measurements=[];
  for(const digits of ['1','12','123','1234']) {
   count=Number(digits);
   await page.goto('/'+path);
   const heading=page.locator('.filter-count-heading'), group=heading.getByRole('toolbar');
   await expect(group.locator('.filter-count__number').first()).toHaveText(path==='tasks'?String(count*4):digits);
   for(const width of [430,393,375,360,320,360,430]) {
    await page.setViewportSize({width,height:1000});
    await expect(heading.locator(":scope > h2")).toHaveCSS("font-size","24px");
    await expect.poll(()=>heading.evaluate(node=>node.scrollWidth<=node.clientWidth+1)).toBe(true);
    if(width>=375 || (width>=360 && path==='tasks')) {
     await expect(heading).not.toHaveClass(/is-wrapped/);
     await expect.poll(()=>heading.evaluate(node=>{
      const title=node.querySelector('h2')!.getBoundingClientRect(),group=node.querySelector('[role=toolbar]')!.getBoundingClientRect();
      return title.top<group.bottom && group.top<title.bottom;
     })).toBe(true);
     const rows=await group.getByRole('button').evaluateAll(buttons=>new Set(buttons.map(button=>Math.round(button.getBoundingClientRect().top))).size);
     expect(rows).toBe(1);
    }
    for(const [index,button] of (await group.getByRole('button').all()).entries()) {
     await expect(button.locator('.filter-count__number')).toHaveText(path==='tasks'&&index===0?String(count*4):digits);
     if(info.project.name==='mobile') {
      const box=(await button.boundingBox())!;
      expect(box.width).toBeGreaterThanOrEqual(44);expect(box.height).toBeGreaterThanOrEqual(44);
     }
    }
    if(digits==='1234' && [360,393].includes(width)) await heading.screenshot({path:info.outputPath(`${path}-${width}.png`)});
    measurements.push(await heading.evaluate((node,context)=>({...context,available:node.clientWidth,compact:node.querySelector('.filter-count__measurement [data-density=icons]')!.scrollWidth,wrapped:node.classList.contains('is-wrapped')}),{path,digits,width,device:info.project.name}));
   }
  }
 await info.attach('layout-measurements.json',{body:JSON.stringify(measurements),contentType:'application/json'});
});
