import { expect } from "@playwright/test";
import { downloaderSnapshot, test } from "./fixtures";

test.beforeEach(async ({ page }, info) => { await page.addInitScript(locale => localStorage.setItem("moji:locale", locale), info.title.startsWith("zh-CN") ? "zh-CN" : "en"); });

test("home keeps torrents hidden and opens the shared modal with keyboard focus restoration", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".home-system .home-downloader")).toBeVisible();
  await expect(page.locator(".home-downloader .home-diagnostics")).toHaveCount(0);
  await expect(page.locator(".home-downloader-info")).toHaveCount(2);
  await expect(page.locator(".home-downloader-info").first()).toHaveAttribute("title", /Free space refers to the default download disk/);
  await expect(page.locator(".download-manager")).toHaveCount(0);
  await expect(page.getByRole("link", { name: "Downloader", exact: true })).toHaveCount(0);
  const opener = page.getByRole("button", { name: "View torrents", exact: true });
  await opener.click();
  const dialog = page.getByRole("dialog", { name: "Torrent list (2)", exact: true });
  await expect(dialog).toBeVisible();
  const refresh = dialog.getByRole("button", { name: "Refresh qB status" });
  await expect(refresh).toHaveAttribute("title", "Refresh qB status");
  await expect(refresh).toHaveText("");
  const search=dialog.getByPlaceholder("Search name, code, hash, category or tags");
  await expect(search).toBeVisible();
  await expect(dialog.getByText(/Last sync:/)).toHaveCount(0);
  await expect(dialog.getByText("2 torrents",{exact:true})).toHaveCount(0);
  await expect(dialog.getByRole("button",{name:"Next",exact:true})).toHaveCount(0);
  const centers=await dialog.locator(".download-manager__filters").evaluate(el => [...el.children].map(child=>{const r=child.getBoundingClientRect();return r.y+r.height/2;}));
  expect(Math.max(...centers)-Math.min(...centers)).toBeLessThan(1);

  const torrent = dialog.locator(".torrent-row").first();
  await expect(dialog.getByRole("columnheader",{name:"Torrent",exact:true})).toHaveCount(1);
  await expect(dialog.getByRole("columnheader",{name:"Linked tasks",exact:true})).toHaveCount(1);
  await torrent.getByRole("button",{name:"More actions"}).click();
  await page.keyboard.press("Escape");
  await expect(dialog.getByRole("menu")).toHaveCount(0);
  await expect(dialog).toBeVisible();
  await expect(torrent.getByRole("button",{name:"More actions"})).toBeFocused();
  await torrent.getByRole("button",{name:"More actions"}).click();
  await dialog.getByRole("menuitem", { name: "Remove torrent and download files", exact: true }).click();
  await expect(dialog.getByRole("group", { name: "Confirm torrent action" })).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await torrent.getByRole("button",{name:"More actions"}).click();
  await dialog.getByRole("menuitem",{name:"Details",exact:true}).click();
  await expect(dialog.getByText("/downloads/MOJI-301", { exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(opener).toBeFocused();
});

for (const locale of ["en", "zh-CN"] as const) {
 test(`${locale} torrent modal layout`, async ({ page }) => {
  await page.goto(`/?__locale=${locale}`);
  await expect(page.locator("html")).toHaveAttribute("lang", locale);
  await page.getByRole("button", { name: locale === "en" ? "View torrents" : "查看种子列表", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.locator(".torrent-row")).toHaveCount(2);
  await page.evaluate(() => document.fonts.ready);
  expect(await dialog.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
  await expect(page).toHaveScreenshot(`${locale}-torrents.png`);
 });
}

test("downloader settings select one cleanup trigger and native qB preferences", async ({ page }) => {
 await page.goto("/settings/downloader?panel=cleanup");
 await page.getByRole("switch",{name:"Enable automatic cleanup",exact:true}).check();
 await expect(page.getByLabel("Cleanup trigger",{exact:true})).toHaveValue("AFTER_INGEST");
 const saved = page.waitForResponse(response => response.url().endsWith("/graphql") && response.request().postDataJSON().operationName === "UpdateDownloaderPolicyDocument" && response.request().postDataJSON().variables.input.capacityEnabled === true);
 await page.getByLabel("Cleanup trigger",{exact:true}).selectOption("CAPACITY");
 await expect(page.getByLabel("Free space threshold (GB)")).toHaveValue("50");
 await expect(page.getByRole("checkbox",{name:"Permanently delete downloaded files too"})).toBeDisabled();
 await page.getByLabel("Cleanup order",{exact:true}).selectOption("RATIO");
 await saved;
 await expect(page.locator(".downloader-cleanup").getByRole("alert")).toHaveCount(0);
 await page.getByRole("tab", {name:"Transfer settings",exact:true}).click();
 await expect(page.getByLabel("Upload limit (KiB/s)")).toBeVisible();
 await expect(page.getByLabel("Maximum active downloads (-1: unlimited)")).toHaveValue("3");
 await page.getByRole("checkbox",{name:"Enable share ratio limit",exact:true}).check();
 await page.getByLabel("When ratio reaches",{exact:true}).fill("2.5");
 const patched = page.waitForResponse(response=>response.url().endsWith("/graphql")&&response.request().postDataJSON().operationName==="UpdateDownloaderTransferPreferencesDocument"&&response.request().postDataJSON().variables.input.maxRatio===2.5);
 await page.getByLabel("Maximum active downloads (-1: unlimited)").fill("6");
 await patched;
 await expect(page.getByLabel("When ratio reaches",{exact:true})).toHaveValue("2.5");
});

test.describe("current qB sources", () => {
 test.use({graphqlOverrides:{downloader:{...downloaderSnapshot,attentionCount:1,torrents:[
  {...downloaderSnapshot.torrents[0],state:"missingFiles",present:true,needsAttention:false},
  {...downloaderSnapshot.torrents[1],state:"missing",present:false,name:"Absent task source",needsAttention:true},
  {...downloaderSnapshot.torrents[1],hash:"c".repeat(40),state:"missingFiles",present:true,needsAttention:true}
 ]}}});
 test("omits absent sources while showing current missing-file torrents and task associations", async ({page}) => {
  await page.goto("/");await page.getByRole("button",{name:"View torrents",exact:true}).click();
  const dialog=page.getByRole("dialog");const rows=dialog.locator(".torrent-row");
  await expect(rows).toHaveCount(2);
  await expect(dialog.getByText("Absent task source")).toHaveCount(0);
  await expect(rows.first()).not.toHaveClass(/torrent-row--attention/);
  await rows.first().getByRole("button",{name:"More actions"}).click();
  await dialog.getByRole("menuitem",{name:"Details",exact:true}).click();
  await expect(dialog.getByText(/the library copy is unaffected/)).toBeVisible();
  await expect(rows.nth(1)).toHaveClass(/torrent-row--attention/);
 });
});

for (const locale of ["en", "zh-CN"] as const) {
 test(`${locale} enabled cleanup file removal tooltip`, async ({ page }) => {
  await page.goto("/settings/downloader?panel=cleanup");
  const enable = locale === "en" ? "Enable automatic cleanup" : "启用自动清理";
  await page.getByRole("switch", {name:enable,exact:true}).check();
  await expect(page.getByLabel(locale === "en" ? "Cleanup trigger" : "清理条件",{exact:true})).toBeVisible();
  const panel=page.locator(".downloader-cleanup");
  await expect(panel.locator(".settings-save-status")).toHaveCount(0);
  await panel.locator(".settings-info").last().hover();
  const tooltip=page.getByRole("tooltip");
  await expect(tooltip).toBeVisible();
  const bounds=await tooltip.boundingBox();
  expect(bounds!.x).toBeGreaterThanOrEqual(0);
  expect(bounds!.x+bounds!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
  expect(await tooltip.evaluate(el => {
    const node=el as HTMLElement;
    const previous=node.style.pointerEvents;
    node.style.pointerEvents="auto";
    try {const rect=node.getBoundingClientRect();return node.contains(document.elementFromPoint(rect.x+rect.width/2,rect.bottom-8));}
    finally {node.style.pointerEvents=previous;}
  })).toBe(true);
  await page.evaluate(()=>document.fonts.ready);
  await expect(page).toHaveScreenshot(`${locale}-cleanup-enabled-tooltip.png`, {fullPage:true});
 });
}

for (const locale of ["en","zh-CN"] as const) {
 for (const theme of ["light","dark"] as const) {
  test(`${locale} ${theme} transfer preferences`,async({page})=>{
   await page.addInitScript(value=>localStorage.setItem("moji:theme",value),theme);
   await page.emulateMedia({colorScheme:theme});
   await page.goto("/settings/downloader?panel=speed");
   await expect(page.getByRole("heading",{name:locale==="en"?"Seeding limits":"做种限制",exact:true})).toBeVisible();
   await expect(page.getByRole("heading",{name:locale==="en"?"Torrent queueing":"Torrent 排队",exact:true})).toBeVisible();
   await page.evaluate(()=>document.fonts.ready);
   expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
   await expect(page).toHaveScreenshot(`${locale}-${theme}-transfer.png`,{fullPage:true});
   await page.getByLabel(locale==="en"?"Torrent inactivity timer (seconds)":"种子非活跃计时（秒）",{exact:true}).scrollIntoViewIfNeeded();
   await expect(page).toHaveScreenshot(`${locale}-${theme}-queue.png`,{fullPage:true});
  });
 }
}
