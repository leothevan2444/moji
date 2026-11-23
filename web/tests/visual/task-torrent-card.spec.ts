import { expect } from "@playwright/test";
import { test,homeOverview,downloaderSnapshot } from "./fixtures";

const longName="Observed qB torrent name with release, encoding and subtitle information ".repeat(4);
const cases=[
  {id:"torrent-downloading",state:"downloading",progress:.62,stage:"DOWNLOADING",stale:false,present:true,name:longName},
  {id:"torrent-uploading",state:"uploading",progress:1,stage:"COMPLETED",stale:false,present:true,name:"Observed qB seeding torrent"},
  {id:"torrent-paused",state:"stoppedDL",progress:.62,stage:"DOWNLOADING",stale:false,present:true,name:"Observed qB paused torrent"},
  {id:"torrent-missing",state:"missing",progress:1,stage:"COMPLETED",stale:false,present:false,name:"Previously observed qB torrent"},
  {id:"torrent-stale",state:"uploading",progress:1,stage:"COMPLETED",stale:true,present:true,name:"Last confirmed qB torrent"},
].map((item,index)=>({...item,hash:(index+10).toString(16).repeat(40)}));
const files=[
  {index:0,path:"release/movie.mp4",size:1200000000,kind:"MAIN",selected:true,part:1,reason:"TARGET_CODE"},
  {index:1,path:"release/movie.en.srt",size:24000,kind:"SUBTITLE",selected:true,part:0,reason:"SUBTITLE_POLICY"},
  {index:2,path:"release/sample.mp4",size:120000,kind:"DISCARD",selected:false,part:0,reason:"ADVERTISEMENT_OR_SAMPLE"},
];
const tasks=cases.map(item=>({...homeOverview.activeTasks[0],id:item.id,code:"DEMO-001",stage:item.stage,stageStatus:item.stage==="COMPLETED"?"DONE":"RUNNING",progress:item.progress,torrentHash:item.hash,torrentName:"Earlier task observation",torrentUrl:item.id==="torrent-paused"?"indexer-resource:opaque-server-reference":`magnet:?xt=urn:btih:${item.hash}`,contentPath:"/downloads/release",savePath:"/downloads",files}));
test.use({graphqlOverrides:{tasks,task:({id}:{id:string})=>tasks.find(task=>task.id===id),downloader:({taskId}:{taskId:string})=>{
  const item=cases.find(item=>item.id===taskId)??cases[0];
  return {...downloaderSnapshot,connected:!item.stale,stale:item.stale,total:1,torrents:[{...downloaderSnapshot.torrents[0],hash:item.hash,name:item.name,state:item.state,progress:item.progress,present:item.present,completedBytes:Math.round(12000000000*item.progress),downloadSpeed:item.state==="downloading"?12500000:0,uploadSpeed:item.state==="uploading"?2400000:0,eta:item.state==="downloading"?360:0,allCompleted:item.stage==="COMPLETED",canDeleteFiles:item.stage==="COMPLETED"&&item.present,taskIds:[item.id],taskCodes:["DEMO-001"]}]};
},torrentPieces:({hash}:{hash:string})=>{
 const item=cases.find(item=>item.hash===hash)!;const complete=item.progress===1;
 return {hash,total:32,observedAt:"2026-01-15T12:00:00Z",buckets:Array.from({length:32},(_,i)=>({start:i,count:1,missing:!complete&&i>=24?1:0,downloading:!complete&&i>=20&&i<24?1:0,downloaded:complete||i<20?1:0}))};
}}});

for(const locale of ["en","zh-CN"] as const) for(const theme of ["light","dark"] as const) {
  test(`${locale} ${theme} compact torrent states and progressive details`,async({page})=>{
    const errors:string[]=[];page.on("pageerror",error=>errors.push(error.message));
    await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme);},{locale,theme});
    for(const item of cases) {
      const observation=page.waitForResponse(response=>response.url().endsWith("/graphql")&&response.request().method()==="POST"&&response.request().postDataJSON().operationName==="DownloaderDocument"&&response.request().postDataJSON().variables.taskId===item.id);
      await page.goto(`/tasks/${item.id}`);
      await (await observation).finished();
      const info=page.locator(".task-files");
      await expect(info.getByText(item.name,{exact:true})).toHaveCount(1);
      await expect(info.getByText("Earlier task observation",{exact:true})).toHaveCount(0);
      await expect(info.locator(".torrent-information")).not.toHaveAttribute("open");
      await expect(info.locator(".task-file-group--skipped details")).not.toHaveAttribute("open");
      await expect(info.locator(".download-metrics > div")).toHaveCount(item.state==="downloading"?6:item.state==="uploading"?5:item.present?4:2);
      await expect(info.getByRole("progressbar")).toHaveCount(0);
      await expect(info.locator(".torrent-identity dt")).toBeHidden();
      const titleStyle=await info.getByRole("heading",{name:locale==="en"?"Torrent information":"种子信息",exact:true}).evaluate(element=>{const style=getComputedStyle(element);return [style.fontSize,style.fontWeight,style.lineHeight]});
      const actionsStyle=await page.locator(".task-detail-operations h3").evaluate(element=>{const style=getComputedStyle(element);return [style.fontSize,style.fontWeight,style.lineHeight]});
      expect(titleStyle).toEqual(actionsStyle);
      expect(await info.locator(".torrent-identity .task-value__text").evaluate(element=>parseFloat(getComputedStyle(element).fontSize)>=16&&Number(getComputedStyle(element).fontWeight)>=600)).toBe(true);
      await expect(info.getByRole("button",{name:locale==="en"?"More actions":"更多操作",exact:true})).toHaveCount(0);
      if(item.present) {await expect(info.getByText("8 (40)",{exact:true})).toBeVisible();await expect(info.getByText("12 (100)",{exact:true})).toBeVisible();}
      if(!item.stale&&item.present) await expect(info.getByRole("img",{name:/pieces|分片|共/})).toBeVisible();
      const refresh=info.getByRole("button",{name:locale==="en"?"Refresh qB status":"刷新 qB 状态",exact:true});
      const refreshBox=await refresh.boundingBox();
      expect(Math.abs(refreshBox!.width-refreshBox!.height)).toBeLessThan(1);
      expect(await info.locator(".download-metrics").evaluate(element=>getComputedStyle(element).gridTemplateColumns.split(" ").length)).toBe((page.viewportSize()?.width??1440)<=600?2:3);
      const primary=info.locator(".task-torrent-actions .torrent-actions > button").first();
      if(item.stale||!item.present) await expect(primary).toBeDisabled();
      else await expect(primary).toBeEnabled();
      await info.scrollIntoViewIfNeeded();
      expect(await info.evaluate(element=>element.scrollWidth<=element.clientWidth)).toBe(true);
      if(item.name===longName) expect(await info.locator(".torrent-identity .task-value__text").evaluate(element=>element.getBoundingClientRect().height<=parseFloat(getComputedStyle(element).lineHeight)*2+1)).toBe(true);
      await expect(info).toHaveScreenshot(`${locale}-${theme}-${item.id}.png`);
      await info.locator(".torrent-information summary").click();
      await expect(info.getByText(item.hash,{exact:true})).toBeVisible();
      await expect(info.locator(".torrent-information").getByText(locale==="en"?"Selected file size":"选中文件总大小",{exact:true})).toHaveCount(0);
      await expect(info.locator(".torrent-information").getByText(locale==="en"?"Active time":"活动时间",{exact:true})).toBeVisible();
      await expect(info.locator(".torrent-information").getByText(locale==="en"?"Last seen complete":"最后完整可见",{exact:true})).toBeVisible();
      await expect(info.locator(".torrent-information").getByText(locale==="en"?"Download speed":"下载速度",{exact:true})).toHaveCount(0);
      await expect(info.locator(".torrent-information").getByText(locale==="en"?"Upload speed":"上传速度",{exact:true})).toHaveCount(0);
      if(item.id==="torrent-paused") await expect(info.getByText("indexer-resource:opaque-server-reference",{exact:true})).toHaveCount(0);
      if(item.id==="torrent-downloading") {
        const details=info.locator(".torrent-information");
        await details.scrollIntoViewIfNeeded();
        await expect(details).toHaveScreenshot(`${locale}-${theme}-technical-details.png`);
        expect(await details.locator(".download-details").evaluate(element=>getComputedStyle(element).gridTemplateColumns.split(" ").length)).toBe((page.viewportSize()?.width??1440)<=600?2:3);
      }
      await expect(info.getByRole("button",{name:locale==="en"?"Copy Content location":"复制内容位置",exact:true})).toBeVisible();
      await expect(info.getByRole("button",{name:locale==="en"?"More actions":"更多操作",exact:true})).toHaveCount(0);
      await expect(info.getByRole("button",{name:locale==="en"?"Recheck":"重新校验",exact:true})).toBeVisible();

    }
    expect(errors).toEqual([]);
  });
}
