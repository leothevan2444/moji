import { expect } from "@playwright/test";
import { downloaderSnapshot, homeOverview, torrentSearchSnapshot, searchCandidate, test } from "./fixtures";

const waiting = homeOverview.attentionTasks[1];
const completed = { ...homeOverview.recentTasks[0], torrentHash: "a".repeat(40), torrentName: "MOJI-301 - Completed and seeding", torrentUrl: "magnet:?xt=urn:btih:" + "a".repeat(40), files: [
  { index: 0, path: "MOJI-301/movie.mp4", size: 12000000000, kind: "MAIN", selected: true, part: 1, reason: "TARGET_CODE" }
] };
const candidate = { title: "MOJI-102 " + "Long candidate title with release and encoding information ".repeat(4), tracker: "Example private indexer", trackerId: "example", categoryDesc: "Movies", publishDate: "2026-01-14", infoHash: "b".repeat(40), size: 12000000000, seeders: 23, peers: 4,
  magnetUri: "magnet:?xt=urn:btih:" + "b".repeat(40), link: "https://indexer.invalid/download?apikey=fixture", details: "https://indexer.invalid/details/102" };
test.use({ graphqlOverrides: {
  tasks: [waiting, completed], task: ({ id }: { id: string }) => [waiting, completed].find(task => task.id === id),
  torrentSearch: ({id,offset,limit}:{id:string;offset:number;limit:number})=>({...torrentSearchSnapshot,id,offset,limit,taskId:id.startsWith("task-search-")?id.slice(12):null,total:id.startsWith("task-search-")?3:1,candidates:[{...searchCandidate,title:candidate.title,instanceName:candidate.tracker,kind:"JACKETT",size:candidate.size,seeders:candidate.seeders,publishedAt:candidate.publishDate,detailsUrl:candidate.details},...(id.startsWith("task-search-")?[{...searchCandidate,id:"unknown",title:"No link or seeder data supplied",size:null,seeders:null,detailsUrl:null},{...searchCandidate,id:"zero",title:"Zero seeders",seeders:0,detailsUrl:null}]:[])]}),
  downloader: { ...downloaderSnapshot, total: 1, torrents: [downloaderSnapshot.torrents[0]] }
} });

for (const locale of ["en", "zh-CN"] as const) {
  for (const theme of ["light", "dark"] as const) {
    test(`${locale} ${theme} manual candidate links`, async ({ page }) => {
      const errors: string[] = [];
      page.on("pageerror", error => errors.push(error.message));
      await page.addInitScript(({ theme, locale }) => { localStorage.setItem("moji:theme", theme); localStorage.setItem("moji:locale", locale); }, { theme, locale });
      await page.goto(`/tasks/${waiting.id}/resolve?__locale=${locale}`);
      const candidates = page.locator(".sourcing-resolution__candidates");
      await expect(candidates.locator("li")).toHaveCount(3);
      await expect(candidates.getByRole("link",{name:locale==="en"?"Indexer details":"站点详情"})).toHaveCount(1);
      await expect(candidates).toContainText(locale === "en" ? "Seeders unknown" : "做种数未知");
      await expect(candidates).toContainText(locale === "en" ? "0 seeders" : "0 个做种者");
      expect(await candidates.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
      await expect(page.getByRole("button",{name:locale==="en"?"Search again":"重新搜索候选",exact:true})).toBeEnabled();
      await page.evaluate(()=>document.fonts.ready);
      await page.evaluate(()=>Promise.all(document.getAnimations().filter(animation=>animation.effect?.getComputedTiming().iterations!==Infinity).map(animation=>animation.finished.catch(()=>undefined))));
      await expect(page).toHaveScreenshot(`${locale}-${theme}-candidates.png`);
      expect(errors).toEqual([]);
    });

    test(`${locale} ${theme} integrated torrent information`, async ({ page }) => {
      const errors: string[] = [];
      page.on("pageerror", error => errors.push(error.message));
      await page.addInitScript(({ theme, locale }) => { localStorage.setItem("moji:theme", theme); localStorage.setItem("moji:locale", locale); }, { theme, locale });
      await page.goto(`/tasks/${completed.id}?__locale=${locale}`);
      const info = page.locator(".task-files");
      await info.scrollIntoViewIfNeeded();
      await expect(info.locator(".download-manager")).toHaveCount(1);
      await expect(page.locator(".download-manager")).toHaveCount(1);
      await expect(info.locator(".download-torrent")).toHaveCount(1);
      expect(await info.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
      await expect(info).toHaveScreenshot(`${locale}-${theme}-torrent-information.png`);
      await info.getByRole("button", { name: locale === "en" ? "Remove torrent and download files" : "移除种子和下载文件", exact: true }).click();
      await expect(info.locator(".image-cache-confirm")).toBeVisible();
      expect(errors).toEqual([]);
    });
  }
}

test("discovery uses server candidates and keeps detail links and seeder counts", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/discover?source=indexers&q=MOJI-102&run=search-one&fastRules=0&fileRules=0");
  const card = page.locator(".candidate-card--indexer");
  await expect(card).toHaveCount(1);
  await expect(card).toContainText("23 seeders");
  await expect(card.getByRole("link")).toHaveCount(1);
  expect(await card.evaluate(el => el.scrollWidth <= el.clientWidth && el.scrollHeight <= el.clientHeight)).toBe(true);
  expect(await card.evaluate(el => el.querySelector(".candidate-card__head")!.getBoundingClientRect().bottom <= el.querySelector(".candidate-card__foot")!.getBoundingClientRect().top)).toBe(true);
  await expect(card).toHaveScreenshot("discovery-candidate.png");
});
