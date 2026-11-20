import { expect } from "@playwright/test";
import { test, homeOverview, downloaderSnapshot } from "./fixtures";

const hash = "a".repeat(40);
const film = {endpoint:"https://catalog.test/graphql",sceneId:"film-detail",code:"DEMO-001",title:"A complete film title with a long subtitle for task identification",date:"2026-01-10",studio:"Example studio",performers:["Alice Example","Beth Example","Cara Example","Diana Example"],imageUrl:"/test-cover.svg",url:"",confirmationDigest:"fixture"};
const files = [
  {index:0,path:"DEMO-001/movie.mp4",size:1200000000,kind:"MAIN",selected:true,part:1,reason:"TARGET_CODE"},
  {index:1,path:"DEMO-001/movie.en.srt",size:24000,kind:"SUBTITLE",selected:true,part:0,reason:"SUBTITLE_POLICY"},
  {index:2,path:"DEMO-001/sample.mp4",size:120000,kind:"DISCARD",selected:false,part:0,reason:"ADVERTISEMENT_OR_SAMPLE"},
];
const running = {...homeOverview.activeTasks[0],id:"detail-running",code:film.code,film,sourcingMode:"AUTO",category:"private-category",torrentHash:hash,torrentName:"Observed qB torrent name",contentPath:"/qb/downloads/DEMO-001",savePath:"/qb/downloads",files};
const blocked = {...running,id:"detail-blocked",stageStatus:"BLOCKED",stageErrorCode:"FILE_SELECTION_REQUIRED",files:[{...files[0],kind:"UNKNOWN_VIDEO",selected:false,reason:"IDENTITY_UNCERTAIN"},...files.slice(1)]};
const waiting = {...running,id:"detail-waiting",stage:"SOURCING",stageStatus:"PENDING",torrentHash:"",torrentName:"",contentPath:"",files:[],sourcingLastCheckedAt:"2026-01-15T11:00:00Z",sourcingNextCheckAt:"2026-01-15T18:00:00Z"};
const completed = {...running,id:"detail-completed",stage:"COMPLETED",stageStatus:"DONE",progress:1,stashSceneIds:["actual-scene"],stashScanPath:"/library/DEMO-001"};
const stashFilm = {id:"actual-scene",title:"Actual Stash movie title",code:film.code,date:"2026-01-12",studio:"Actual Stash studio",performers:["Actual Stash performer"],imageUrl:"/test-stash-cover.svg",url:"https://stash.invalid/scenes/actual-scene"};
const tasks = [running,blocked,waiting,completed];
test.use({graphqlOverrides:{tasks,taskStashScenes:[stashFilm],task:({id}:{id:string})=>tasks.find(task=>task.id===id),downloader:{...downloaderSnapshot,total:1,torrents:[{...downloaderSnapshot.torrents[0],hash,name:running.torrentName,savePath:running.savePath,taskIds:tasks.map(task=>task.id),taskCodes:[film.code]}]}}});

for (const locale of ["en","zh-CN"] as const) {
  for (const theme of ["light","dark"] as const) {
    test(`${locale} ${theme} compact task header and folded skipped files`, async ({page}) => {
      await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme);},{locale,theme});
      await page.route("**/test*cover.svg", route => route.fulfill({contentType:"image/svg+xml",body:'<svg xmlns="http://www.w3.org/2000/svg" width="480" height="320" viewBox="0 0 480 270"><rect width="480" height="270" fill="#36566f"/><circle cx="360" cy="80" r="38" fill="#e5c688"/><path d="M0 270 150 95 290 270M160 270 340 135 480 270" fill="#6d968e"/></svg>'}));
      await page.goto(`/tasks/${running.id}`);
      const dialog = page.getByRole("dialog");
      const overview = dialog.locator(".task-detail-overview");
      await expect(overview.locator(".task-detail-summary h3")).toHaveText(film.title);
      await expect(overview.getByText(film.code,{exact:true})).toHaveCount(0);
      const cover = overview.locator(".task-detail-summary__artwork img");
      await expect(cover).toBeVisible();
      const coverSize = await cover.boundingBox();
      expect(coverSize!.width / coverSize!.height).toBeCloseTo(480/320,1);
      await expect(dialog.locator(".task-stash-scenes")).toHaveCount(0);
      await expect(overview.locator(".task-detail-summary")).toContainText(film.studio);
      await expect(overview.locator(".task-detail-summary")).toContainText(film.performers[0]);
      await expect(overview.getByText(film.endpoint,{exact:true})).toBeHidden();
      await expect(overview.getByText("private-category",{exact:true})).toBeHidden();
      expect(await overview.evaluate(node=>node.scrollWidth<=node.clientWidth)).toBe(true);
      await expect(overview).toHaveScreenshot(`${locale}-${theme}-header.png`);
      const metadata = overview.locator(".task-detail-extra");
      await metadata.locator("summary").click();
      await expect(overview.getByText(film.endpoint,{exact:true})).toBeVisible();
      await expect(overview.getByText("private-category",{exact:true})).toBeVisible();
      await metadata.scrollIntoViewIfNeeded();
      await expect(metadata.getByText(locale === "en" ? "Created" : "创建时间",{exact:true})).toBeInViewport();
      await expect(metadata.getByText(locale === "en" ? "Last updated" : "最近更新",{exact:true})).toBeInViewport();
      await expect(metadata).toHaveScreenshot(`${locale}-${theme}-metadata.png`);
      const groups = dialog.locator(".task-files__groups");
      await groups.evaluate(element => element.scrollIntoView({block:"start",behavior:"instant"}));
      // Natural image heights can put this crop between device pixels.
      // Align only the captured section so text rasterization is deterministic.
      await groups.evaluate(element => {
        const top = element.getBoundingClientRect().top;
        element.style.transform = `translateY(${Math.round(top)-top}px)`;
      });
      await expect(groups).toHaveScreenshot(`${locale}-${theme}-file-groups.png`);
      await groups.evaluate(element => element.style.removeProperty("transform"));
      const skipped = dialog.locator(".task-file-group--skipped details");
      await expect(skipped.getByText("sample.mp4",{exact:true})).toBeHidden();
      await skipped.locator("summary").click();
      await expect(skipped.getByText("sample.mp4",{exact:true})).toBeVisible();
      await skipped.locator("summary").focus();
      await page.keyboard.press("Enter");
      await expect(skipped.getByText("sample.mp4",{exact:true})).toBeHidden();
      await page.goto(`/tasks/${blocked.id}`);
      await expect(dialog.locator('.task-file-group--choose input[type="checkbox"]')).toHaveCount(1);
      await expect(dialog.locator(".task-file-group--skipped details").getByText("sample.mp4",{exact:true})).toBeHidden();
      await expect(dialog.locator(".task-detail-overview")).toHaveScreenshot(`${locale}-${theme}-blocked.png`);
      await page.goto(`/tasks/${waiting.id}`);
      await expect(dialog.locator(".task-timeline > [aria-current=step] .task-detail-schedule")).toBeVisible();
      await expect(dialog.locator(".task-detail-overview")).toHaveScreenshot(`${locale}-${theme}-waiting.png`);
      await expect(dialog.locator(".task-detail-extra").getByText("private-category",{exact:true})).toBeHidden();
      await page.goto(`/tasks/${completed.id}`);
      const completedOverview = dialog.locator(".task-detail-overview");
      await expect(completedOverview.locator("h3")).toHaveText(stashFilm.title);
      await expect(completedOverview).toContainText(stashFilm.studio);
      await expect(completedOverview).toContainText(stashFilm.performers[0]);
      await expect(completedOverview).not.toContainText(film.title);
      const stashLink = completedOverview.getByRole("link");
      await expect(stashLink).toHaveAttribute("href",stashFilm.url);
      await expect(completedOverview.locator(".task-detail-summary .status-chip")).toHaveCount(0);
      await expect(completedOverview.locator(".task-timeline")).toContainText(locale === "en" ? "In library" : "已入库");
      await expect(dialog.locator(".task-detail-overview")).toHaveScreenshot(`${locale}-${theme}-completed.png`);
      expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
    });
  }
}
