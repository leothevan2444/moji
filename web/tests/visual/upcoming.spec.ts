import { expect } from "@playwright/test";
import { test, settingsSnapshot } from "./fixtures";

const actors = [{ id: "p1", name: "Subscribed performer" }, { id: "p2", name: "Another performer" }];
const records = Array.from({ length: 10 }, (_, index) => ({ key: `upcoming-${index}`, code: `MOJI-${100 + index}`, title: index === 0 ? "A future release with a deliberately long title to check wrapping and small screens" : `Upcoming film ${index + 1}`, date: "2026-11-12", imageUrl: index === 0 ? "/test-upcoming-poster.svg" : index === 3 ? "/test-broken-poster.png" : "", sourceUrl: "https://example.org/release", state: index === 1 ? "UNCONFIRMED" : "WAITING", decision: index === 2 ? "QUEUED" : "DOWNLOADED", errorCode: index === 1 ? "METADATA_UNAVAILABLE" : "", metadataCheckedAt: "2026-10-31T00:00:00Z", performers: index === 0 ? actors : actors.slice(0, 1) }));
for (const locale of ["zh-CN", "en"] as const) {
 for (const theme of ["light", "dark"] as const) {
 test.describe(`${locale} ${theme} upcoming`, () => {
  test.use({ graphqlOverrides: {
   upcomingReleases: ({ input }: { input: { pageSize?: number; search?: string } }) => { const filtered = records.filter(r => !input.search || r.code.includes(input.search));return { items: filtered.slice(0, input.pageSize ?? 24), performers: actors, totalCount: filtered.length, page: 1, pageSize: input.pageSize ?? 24, totalPages: 1, subscribedCount: 2, automatic: true, timezone: "Asia/Shanghai",today:"2026-10-31" }; },
   refreshUpcomingRelease: () => ({ ...records[0], date: "2026-11-20" })
  } });
  test("cards, filtering and detail", async ({ page }, testInfo) => {
   await page.clock.setFixedTime(new Date("2026-10-31T00:00:00Z"));
   const errors: string[] = [];page.on("pageerror", e => errors.push(e.message));page.on("console", m => {if(m.type()==="error")errors.push(m.text())});
   await page.addInitScript(({locale,theme}) => {localStorage.setItem("moji:locale", locale);localStorage.setItem("moji:theme",theme)}, {locale,theme});
   await page.route("**/test-upcoming-poster.svg",route=>route.fulfill({contentType:"image/svg+xml",body:'<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360" viewBox="0 0 640 360"><rect width="640" height="360" fill="#20344b"/><path d="M0 300L280 0H640V360H0" fill="#425c6b"/><circle cx="450" cy="140" r="85" fill="#c2ac89"/><text x="32" y="320" fill="white" font-family="sans-serif" font-size="32">MOJI 100</text></svg>'}));
   await page.route("**/test-broken-poster.png",route=>route.fulfill({contentType:"image/png",body:"invalid image"}));
   await page.goto("/discover");
   await expect(page.locator(".upcoming-card")).toHaveCount(8);
   await expect(page.locator(".upcoming-card").first()).toHaveClass(/performer-scene-card/);
   await expect(page.locator(".upcoming-card").nth(3).locator(".performer-scene-card__image-placeholder")).toBeVisible();
   await expect(page.getByText(locale === "zh-CN" ? "发行后自动选种" : "Find a resource after release",{exact:true})).toHaveCount(0);
   await expect(page.getByRole("heading", { name: locale === "zh-CN" ? "待发行 · 10" : "Upcoming releases · 10" })).toBeVisible();
   await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
   await page.evaluate(() => document.fonts.ready);
   await page.screenshot({ path: testInfo.outputPath(`${locale}-${theme}-upcoming.png`), fullPage: true });
   await page.locator(".upcoming-card__open").first().click();
   await expect(page.getByRole("dialog")).toBeVisible();
   const dialog = page.getByRole("dialog");
   await expect(dialog.locator(".upcoming-player")).toBeVisible();
   await expect(dialog.getByText(locale === "zh-CN" ? "暂无预告片" : "No trailer available", {exact:true})).toBeVisible();
   await expect(dialog.locator("video, audio")).toHaveCount(0);
   await expect(dialog.getByRole("button", { name: locale === "zh-CN" ? "刷新" : "Refresh", exact: true })).toBeVisible();
   await expect(dialog.getByRole("link", {name:locale === "zh-CN" ? "来源" : "Source",exact:true})).toHaveAttribute("rel","noreferrer");
   await expect.poll(()=>dialog.evaluate(element=>element.scrollWidth<=element.clientWidth)).toBe(true);
   await dialog.screenshot({path:testInfo.outputPath(`${locale}-${theme}-upcoming-detail.png`)});
   await dialog.getByRole("button",{name:locale === "zh-CN" ? "刷新" : "Refresh",exact:true}).click();
   await expect(dialog.locator("time")).toHaveText(locale === "zh-CN" ? "预计 2026-11-20" : "Expected 2026-11-20");
   await page.keyboard.press("Escape");await expect(page.getByRole("dialog")).toHaveCount(0);
   await expect(page.locator(".upcoming-card__open").first()).toBeFocused();
   await page.keyboard.press("Enter");await expect(dialog).toBeVisible();
   await page.keyboard.press("Tab");await expect.poll(()=>dialog.evaluate(element=>element.contains(document.activeElement))).toBe(true);
   await page.keyboard.press("Escape");
   await page.getByRole("button", {name:locale==="zh-CN"?"查看全部":"View all",exact:true}).click();
   await expect(page).toHaveURL(/section=upcoming/);await expect(page.locator(".upcoming-card")).toHaveCount(10);
   await page.getByLabel(locale==="zh-CN"?"番号或标题":"Code or title",{exact:true}).fill("MOJI-100");
   await expect(page.locator(".upcoming-card")).toHaveCount(1);
   await page.reload();await expect(page.locator(".upcoming-card")).toHaveCount(1);
   expect(errors).toEqual([]);
  });
 });
 }
}

test.describe("release timezone settings", () => {
 test.use({graphqlOverrides:{patchSettings:({input}:{input:{system?:{release_timezone?:string}}})=>({...settingsSnapshot,system:{...settingsSnapshot.system,language:"auto",releaseTimezone:input.system?.release_timezone??"auto",effectiveReleaseTimezone:input.system?.release_timezone??"Asia/Shanghai"}})}});

for (const locale of ["zh-CN", "en"] as const) {
 test(`${locale} system release timezone supports auto and manual values`, async ({ page }, info) => {
  await page.addInitScript(value => localStorage.setItem("moji:locale",value),locale);
  const patches: Array<Record<string, unknown>> = [];
  await page.route("**/graphql",async route => {
   const request=route.request().postDataJSON();if(request.operationName==="PatchSettings")patches.push(request.variables.input);
   await route.fallback();
  });
  await page.goto("/settings/system?panel=general");
  const field=page.getByLabel(locale==="zh-CN"?"发行日期时区":"Release date timezone",{exact:true});
  await expect(field).toHaveValue("auto");
  await expect(page.getByText(locale==="zh-CN"? /当前生效：Asia\/Shanghai/:/Effective timezone: Asia\/Shanghai/)).toBeVisible();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);
  await page.screenshot({path:info.outputPath(`${locale}-release-timezone.png`),fullPage:true});
  await field.fill("America/New_York");
  await expect.poll(()=>patches.length).toBe(1);
  expect(patches[0]).toEqual({system:{release_timezone:"America/New_York"}});
 });
}

});
