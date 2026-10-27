import { expect } from "@playwright/test";
import { test } from "./fixtures";

const tasks = Array.from({length:277},(_,n)=>({
  id:`task-${String(n).padStart(4,"0")}`,code:`TASK-${String(n).padStart(4,"0")}`,source:"MANUAL",
  version:"00000000000000000001:00000000000000000001",revision:1,
  stage:"DOWNLOADING",stageStatus:n===0||n===276?"BLOCKED":"RUNNING",stageLabel:"Downloading",stageStatusLabel:"Running",
  stageErrorCode:n===0||n===276?"TORRENT_ERROR":null,stageErrorMessage:null,progress:.25,
  torrentName:`Download ${n}`,qbittorrentState:"downloading",stashIngestPhase:null,stashSceneId:null,
  sourcingNextCheckAt:null,sourcingLastCheckedAt:null,sourcingCompletedAt:null,downloadCompletedAt:null,ingestCompletedAt:null,lastProgressAt:null,
  createdAt:new Date(Date.UTC(2026,0,1,0,n)).toISOString(),updatedAt:"2026-01-15T12:00:00Z",
  files:[],torrentUrl:"",stashSceneIds:[],candidate:{title:"",tracker:"",infoHash:"",link:"",magnetUri:"",size:0,seeders:0,peers:0},
}));
let snapshotCursor={sequence:0,streamId:"visual"};
test.beforeEach(()=>{snapshotCursor={sequence:0,streamId:"visual"}});
test.use({graphqlOverrides:{tasks,taskSnapshotCursor:()=>snapshotCursor,task:({id}:{id:string})=>tasks.find(task=>task.id===id)}});

test("bounds cards, preserves cross-page selection and keeps detail navigation on its page", async ({page},testInfo)=>{
  const queries:Array<{query:string;variables:any}>=[];
  page.on("request",r=>{if(r.url().endsWith("/graphql")&&r.method()==="POST")queries.push(r.postDataJSON())});
  await page.goto("/tasks");
  await expect(page.locator(".task-card")).toHaveCount(24);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.screenshot({path:testInfo.outputPath("synthetic-task-page.png")});
  expect(queries.some(q=>q.query.includes("query TasksOverviewDocument"))).toBe(false);
  await page.getByRole("button",{name:"Enter multi-select mode",exact:true}).click();
  await page.getByRole("button",{name:"Select this page (24)",exact:true}).click();
  await expect(page.locator(".task-selection-bar")).toContainText("24");
  await page.getByRole("button",{name:"Next",exact:true}).click();
  await expect(page).toHaveURL(/page=2/);
  await expect(page.locator(".task-card")).toHaveCount(24);
  await expect(page.locator(".task-selection-bar")).toContainText("24");
  await expect(page.locator(".task-card__open").first()).toContainText("TASK-0252");
  await expect(page.locator(".task-card__checkbox").first()).not.toBeChecked();
  await page.locator(".task-card__checkbox").first().check();
  await expect(page.locator(".task-selection-bar")).toContainText("25");
  await page.getByRole("button",{name:"Exit multi-select mode",exact:true}).click();
  await page.locator(".task-card__open").first().click();
  await expect(page).toHaveURL(/\/tasks\/task-\d+\?page=2/);
  await expect(page.locator(".task-detail-overview")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page).toHaveURL(/\/tasks\?page=2/);
  await page.locator(".toolbar-inline--tasks input").fill("TASK-0000");
  await expect(page.locator(".task-card")).toHaveCount(1);
  await expect(page.locator(".task-card")).toContainText("TASK-0000");
  expect(queries.filter(q=>q.query.includes("query TasksWorkspace")).at(-1)?.variables.input.page).toBe(1);
});

test("updates progress without refetching or reordering, and opens an independent review queue", async ({page})=>{
  let send:((event:any)=>void)|undefined;const requests:string[]=[];
  await page.routeWebSocket("**/graphql",socket=>socket.onMessage(raw=>{
    const msg=JSON.parse(String(raw));if(msg.type==="connection_init")socket.send(JSON.stringify({type:"connection_ack"}));
    if(msg.type==="subscribe"&&msg.payload.query.includes("subscription TaskEvents")){
      send=event=>socket.send(JSON.stringify({id:msg.id,type:"next",payload:{data:{taskEvents:event}}}));
      send({__typename:"TaskEvent",sequence:0,streamId:"visual",type:"READY",taskId:"",listChanged:false,task:null,dashboardStats:null});
    }
  }));
  page.on("request",r=>{if(r.url().endsWith("/graphql")&&r.method()==="POST")requests.push(r.postDataJSON().query)});
  await page.goto("/tasks");await expect(page.locator(".task-card")).toHaveCount(24);
  await expect.poll(()=>Boolean(send)).toBe(true);
  const before=await page.locator(".task-card__open").allTextContents();
  const reads=requests.filter(q=>q.includes("query TasksWorkspace")).length;
  send!({__typename:"TaskEvent",sequence:1,streamId:"visual",type:"UPDATED",taskId:tasks[275].id,listChanged:false,
    task:{...tasks[275],__typename:"TaskSummary",version:"00000000000000000001:00000000000000000002",progress:.71},
    dashboardStats:{__typename:"DashboardStats",total:277,active:277,completed:0,attention:2,waiting:0,downloading:275,pendingScans:0,failed:2}});
  await expect(page.locator(".task-card").filter({hasText:"TASK-0275"}).getByRole("progressbar")).toHaveAttribute("aria-valuenow","71");
  expect(await page.locator(".task-card__open").allTextContents()).toEqual(before);
  expect(requests.filter(q=>q.includes("query TasksWorkspace")).length).toBe(reads);
  await page.locator(".toolbar-inline--tasks input").fill("TASK-0100");
  await expect(page.locator(".task-card")).toHaveCount(1);
  await page.getByRole("button",{name:/Open task workbench/}).click();
  const dialog=page.getByRole("dialog");await expect(dialog).toContainText("0 handled · 2 remaining");
  await expect(dialog.getByRole("heading",{name:"TASK-0000",exact:true})).toBeVisible();
});

test("unrelated pages never request task history or a task page",async({page})=>{
 const queries:string[]=[];page.on("request",r=>{if(r.url().endsWith("/graphql")&&r.method()==="POST")queries.push(r.postDataJSON().query)});
 await page.goto("/discover");await expect(page.locator(".discovery-shell")).toBeVisible();
 await page.waitForTimeout(500);
 expect(queries.some(q=>q.includes("query TasksOverviewDocument")||q.includes("query TasksWorkspace"))).toBe(false);
});


test("a delayed page cannot overwrite a newer socket update",async({page})=>{
 let release!:()=>void;const barrier=new Promise<void>(resolve=>{release=resolve});let blocked=false;
 await page.route("**/graphql",async route=>{const body=route.request().postDataJSON();if(body.query.includes("query TasksWorkspace")&&!blocked){blocked=true;await barrier}await route.fallback()});
 let send:((event:any)=>void)|undefined;
 await page.routeWebSocket("**/graphql",socket=>socket.onMessage(raw=>{const msg=JSON.parse(String(raw));if(msg.type==="connection_init")socket.send(JSON.stringify({type:"connection_ack"}));if(msg.type==="subscribe"&&msg.payload.query.includes("subscription TaskEvents")){send=event=>socket.send(JSON.stringify({id:msg.id,type:"next",payload:{data:{taskEvents:event}}}));send({__typename:"TaskEvent",sequence:0,streamId:"visual",type:"READY",taskId:"",listChanged:false,task:null,dashboardStats:null})}}));
 await page.goto("/tasks");await expect.poll(()=>blocked&&Boolean(send)).toBe(true);
 send!({__typename:"TaskEvent",sequence:1,streamId:"visual",type:"UPDATED",taskId:tasks[275].id,listChanged:false,task:{...tasks[275],__typename:"TaskSummary",version:"00000000000000000001:00000000000000000002",progress:.83},dashboardStats:null});
 release();await expect(page.locator(".task-card").filter({hasText:"TASK-0275"}).getByRole("progressbar")).toHaveAttribute("aria-valuenow","83");
});

test("a reconnect recovers the current page without fetching history",async({page})=>{
 let send:((event:any)=>void)|undefined;const reads:any[]=[];
 await page.routeWebSocket("**/graphql",socket=>socket.onMessage(raw=>{const msg=JSON.parse(String(raw));if(msg.type==="connection_init")socket.send(JSON.stringify({type:"connection_ack"}));if(msg.type==="subscribe"&&msg.payload.query.includes("subscription TaskEvents")){send=event=>socket.send(JSON.stringify({id:msg.id,type:"next",payload:{data:{taskEvents:event}}}));send({__typename:"TaskEvent",sequence:0,streamId:"visual",type:"READY",taskId:"",listChanged:false,task:null,dashboardStats:null})}}));
 page.on("request",request=>{if(request.url().endsWith("/graphql")&&request.method()==="POST")reads.push(request.postDataJSON())});
 await page.goto("/tasks?page=3");await expect(page.locator(".task-card")).toHaveCount(24);await expect.poll(()=>Boolean(send)).toBe(true);
 const before=reads.filter(q=>q.query.includes("query TasksWorkspace")).length;
 snapshotCursor={sequence:20,streamId:"restarted"};
 send!({__typename:"TaskEvent",sequence:20,streamId:"restarted",type:"READY",taskId:"",listChanged:false,task:null,dashboardStats:null});
 await expect.poll(()=>reads.filter(q=>q.query.includes("query TasksWorkspace")).length).toBeGreaterThan(before);
 await expect(page).toHaveURL(/page=3/);
 expect(reads.filter(q=>q.query.includes("query TasksWorkspace")).every(q=>q.variables.input.page===3)).toBe(true);
 expect(reads.some(q=>q.query.includes("query TasksOverviewDocument"))).toBe(false);
 await expect(page.locator(".band-note")).toContainText("275");
});
