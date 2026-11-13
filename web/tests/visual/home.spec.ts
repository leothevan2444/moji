import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => { if (!localStorage.getItem("moji:locale")) localStorage.setItem("moji:locale", "en"); });
});

test("indexers and downloader are peers and fixed gateway cards are removed", async ({ page }) => {
  await page.goto("/");
  const indexers = page.getByRole("region", { name: "Indexers", exact: true });
  const downloader = page.getByRole("region", { name: "Downloader", exact: true });
  await expect(indexers.getByText("Enabled sites", { exact: true })).toBeVisible();
  await expect(indexers.locator("dd").nth(0)).toHaveText("2");
  await expect(indexers.getByText("Search success rate")).toBeVisible();
  await expect(indexers.getByText("95%")).toBeVisible();
  await expect(indexers.getByRole("button", {name:"View indexer records"})).toBeVisible();
  await expect(indexers.locator("..")).toHaveClass("home-sidebar");
  await expect(downloader.locator("..")).toHaveClass("home-sidebar");
  await page.locator(".home-system .home-diagnostics summary").click();
  await expect(page.locator(".home-system").getByText("Jackett", { exact: true })).toHaveCount(0);
  await expect(page.locator(".home-system").getByText("qBittorrent", { exact: true })).toHaveCount(0);
  await downloader.getByRole("link", { name: "Configure", exact: true }).click();
  await expect(page).toHaveURL(/settings\/downloader/);
});

test.describe("unloaded indexer directory",()=>{
 test.use({graphqlOverrides:{homeIndexerSummary:{revision:"unknown",total:0,hours:72,statistics:null,statisticsErrorCode:""}}});
 test("shows a numeric count before the directory loads and provides management",async({page})=>{
  await page.goto("/");const section=page.getByRole("region",{name:"Indexers",exact:true});
  await expect(section.locator("dd").first()).toHaveText("0");
  await section.getByRole("link",{name:"Manage indexers"}).click();await expect(page).toHaveURL(/settings\/indexers/);
 });
});
test("homepage opens retained indexer history and restores focus",async({page})=>{
 await page.goto("/");const section=page.getByRole("region",{name:"Indexers",exact:true});
 const button=section.getByRole("button",{name:"View indexer records"});await button.click();
 await expect(page.getByRole("dialog",{name:"Execution history",exact:true})).toBeVisible();
 await page.keyboard.press("Escape");await expect(button).toBeFocused();await expect(page).toHaveURL("/");
});

test.describe("indexer invalidations", () => {
  let changed = false;
  test.use({ graphqlOverrides: { homeIndexerSummary: ({hours=72}:{hours?:number}) => ({hours,revision: changed ? "after" : "before", total: changed ? 3 : 2, statisticsErrorCode:"", statistics:{windowStartedAt:new Date(Date.parse("2026-01-15T12:00:00Z")-hours*3600000).toISOString(),windowEndedAt:"2026-01-15T12:00:00Z",searches:120,successRate:95,averageElapsedMs:2400,records:240}}) } });
  test("server changes refresh only the indexer region without reloading Home", async ({ page }) => {
    changed = false;
    let notify: (() => void) | undefined;
    let homeRequests = 0;
    page.on("request", request => { if (request.method() === "POST" && request.postDataJSON()?.operationName === "HomePageDocument") homeRequests++; });
    await page.routeWebSocket("**/graphql", socket => {
      socket.onMessage(data => {
        const message = JSON.parse(String(data)) as { id: string; type: string; payload?: { query?: string } };
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription IndexerEvents")) {
          const send = (sequence: number, kind: string) => socket.send(JSON.stringify({ id: message.id, type: "next", payload: { data: { indexerEvents: { streamId: "live", sequence, kind } } } }));
          send(0, "READY");
          notify = () => send(3, "CATALOG");
        }
      });
    });
    try {
      await page.goto("/");
      const indexers = page.getByRole("region", { name: "Indexers", exact: true });
      await expect(indexers.locator("dd").first()).toHaveText("2");
      await expect.poll(() => Boolean(notify)).toBe(true);
      const before = homeRequests;
      expect(before).toBeGreaterThan(0);
      changed = true; notify!();
      await expect(indexers.locator("dd").first()).toHaveText("3");
      expect(homeRequests).toBe(before);
      await expect(page).toHaveURL("/");
    } finally { changed = false; }
  });

  test("retry reopens a rejected indexer subscription", async ({ page }) => {
    changed = false;
    let opens = 0;
    await page.routeWebSocket("**/graphql", socket => {
      socket.onMessage(data => {
        const message = JSON.parse(String(data)) as { id: string; type: string; payload?: { query?: string } };
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription IndexerEvents")) {
          opens++;
          if (opens === 1) socket.send(JSON.stringify({ id: message.id, type: "error", payload: [{ message: "Indexer subscription unavailable", extensions: { code: "INDEXER_BUSY", correlationId: "synthetic" } }] }));
          else socket.send(JSON.stringify({ id: message.id, type: "next", payload: { data: { indexerEvents: { streamId: "recovered", sequence: 0, kind: "READY" } } } }));
        }
      });
    });
    try {
      await page.goto("/");
      const indexers = page.getByRole("region", { name: "Indexers", exact: true });
      await expect(indexers.getByRole("button", { name: "Retry", exact: true })).toBeVisible();
      changed = true;
      await indexers.getByRole("button", { name: "Retry", exact: true }).click();
      await expect.poll(() => opens).toBe(2);
      await expect(indexers.locator("dd").first()).toHaveText("3");
      await expect(indexers.getByRole("button", { name: "Retry", exact: true })).toHaveCount(0);
    } finally { changed = false; }
  });
});

test("home opens file confirmation directly and restores focus after Escape", async ({ page }) => {
  await page.goto("/");
  const confirm = page.getByRole("button", { name: "Select movie files: MOJI-101" });
  await confirm.click();
  await expect(page).toHaveURL(/task=MOJI-101&files=1/);
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("checkbox")).toBeVisible();
  await expect(dialog.getByRole("heading", { name: "Task overview" })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(page).toHaveURL("/");
  await expect(confirm).toBeFocused();
});

test("home search preserves query and source when returning", async ({ page }) => {
  await page.goto("/");
  await page.locator(".home-search").getByRole("combobox").selectOption("indexers");
  await page.getByRole("textbox", { name: "Search movies or resources" }).fill("MOJI-101");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page).toHaveURL(/discover\?q=MOJI-101&source=indexers/);
  await page.goBack();
  await expect(page.getByRole("textbox", { name: "Search movies or resources" })).toHaveValue("MOJI-101");
  await expect(page.locator(".home-search").getByRole("combobox")).toHaveValue("indexers");
});

test.describe("review queue", () => {
  test.use({ graphqlOverrides: { tasks: homeOverview.attentionTasks } });
  test("waits for tasks before opening the session and returns Home on close", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Review queue" }).click();
    await expect(page.getByRole("dialog", { name: "Task workbench" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "MOJI-101", exact: true })).toBeVisible();
    await expect(page.getByText("0 handled · 3 remaining")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("search", { name: "Search movies or resources" })).toBeVisible();
    await expect(page).toHaveURL("/");
  });
});

for (const locale of ["qps-ploc", "ar-XB"] as const) {
  test(`${locale} home layout`, async ({ page }) => {
    await page.goto(`/?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    await expect(page.locator(".home-metric")).toHaveCount(4);
    await expect(page.locator(".home-loading")).toHaveCount(0);
    await page.evaluate(() => document.fonts.ready);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-home.png`, { fullPage: true });
  });
}

test("dark home layout", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  await expect(page.locator(".home-loading")).toHaveCount(0);
  await expect.poll(() => page.locator("html").evaluate(element => getComputedStyle(element).colorScheme)).toBe("dark");
  await page.evaluate(() => document.fonts.ready);
  await expect(page).toHaveScreenshot("dark-home.png", { fullPage: true });
});

test("indexer period changes only search metrics and keeps enabled sites independent",async({page})=>{
 const periods:number[]=[];page.on("request",request=>{if(request.method()==="POST"&&request.url().endsWith("/graphql")){const body=request.postDataJSON();if(body.operationName==="HomeIndexerSummary")periods.push(body.variables.hours);}});
 await page.goto("/");const section=page.getByRole("region",{name:"Indexers",exact:true});const selector=section.getByRole("combobox",{name:"Search statistics period",exact:true});
 await expect(selector).toHaveValue("72");await expect(section.locator("dd").first()).toHaveText("2");
 await expect(section.locator("dl > div").first()).toHaveAttribute("title",/independent of/);
 for(const hours of [24,336,72]){await selector.selectOption(String(hours));await expect.poll(()=>periods.at(-1)).toBe(hours);await expect(section.locator("dd").first()).toHaveText("2");await expect(section.getByText("95%",{exact:true})).toBeVisible();}
});
test.describe("empty search statistics",()=>{
 test.use({graphqlOverrides:{homeIndexerSummary:({hours=72}:{hours?:number})=>({revision:"empty",total:7,hours,statisticsErrorCode:"",statistics:{windowStartedAt:"2026-01-12T12:00:00Z",windowEndedAt:"2026-01-15T12:00:00Z",searches:0,successRate:null,averageElapsedMs:null,records:37}})}});
 test("does not turn missing samples into unknown site counts or zero rates",async({page})=>{
  await page.goto("/");const section=page.getByRole("region",{name:"Indexers",exact:true});await expect(section.locator("dd").first()).toHaveText("7");
  await expect(section.getByText("No actual searches were recorded for this period.",{exact:true})).toBeVisible();await expect(section.locator("dd").nth(2)).toHaveText("—");await expect(section.locator("dd").nth(3)).toHaveText("-");await expect(section.getByText("37 saved records",{exact:true})).toBeVisible();
 });
});


for (const averageElapsedMs of [null,0,1500]) {
 test.describe(`partial search timing ${averageElapsedMs}`,()=>{
  test.use({graphqlOverrides:{homeIndexerSummary:({hours=72}:{hours?:number})=>({revision:"history",total:8,hours,statisticsErrorCode:"",statistics:{windowStartedAt:"2026-01-12T12:00:00Z",windowEndedAt:"2026-01-15T12:00:00Z",searches:34,successRate:73.5,averageElapsedMs,records:37}})}});
  test("shows the available mean or a dash without a missing-timing notice",async({page})=>{
   await page.goto("/");const section=page.getByRole("region",{name:"Indexers",exact:true});
   await expect(section.locator("dd").nth(1)).toHaveText("34");await expect(section.locator("dd").nth(2)).toHaveText("73.5%");await expect(section.locator("dd").nth(3)).toHaveText(averageElapsedMs?"1.5 s":"-");
   await expect(section.locator(".home-indexer-note")).toHaveCount(0);
  });
 });
}

 test("compact subscription summary keeps the large count and omits issues", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "zh-CN"));
  await page.goto("/");
  const summary = page.locator(".home-subscriptions");
  const checks = page.locator(".home-checks");
  await expect(summary.locator(".home-big-number")).toContainText("24");
  await expect(summary.locator("details")).toHaveCount(0);
  await expect(summary.getByText(/订阅检查异常/)).toHaveCount(0);
  await expect(checks.getByText("自动检查已启用", { exact: true })).toBeVisible();
  await expect(checks.locator(".home-check-history")).toHaveCount(0);
  expect((await summary.boundingBox())!.height).toBeLessThan(130);
  expect((await checks.boundingBox())!.height).toBeLessThan(190);
  await expect(summary).toHaveScreenshot("compact-subscriptions-zh.png");
  await expect(checks).toHaveScreenshot("compact-checks-zh.png");
  const manage = summary.getByRole("link", { name: "管理演员与订阅" });
  await manage.focus();
  await expect(manage).toBeFocused();
  await expect(manage).toHaveAttribute("href", "/performers");
  await checks.locator(".home-check-statistics").focus();
  await expect(checks.locator(".home-check-statistics")).toHaveAttribute("title", /0 次已结束运行/);
 });
