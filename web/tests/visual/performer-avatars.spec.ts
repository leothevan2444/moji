import { expect } from "@playwright/test";
import { test } from "./fixtures";

const imagePath = `/api/images/${"a".repeat(64)}`;
const performer = { id: "avatar-test", name: "Avatar test", aliasList: [], favorite: false, imagePath, sceneCount: 0, subscribed: false };
const profile = { performer, disambiguation: null, birthdate: null, ethnicity: null, country: null, eyeColor: null, heightCm: null, rating100: null, urls: [], matchedStashBox: null, totalSceneCount: 0, stashSceneCount: 0, stashBoxSceneCount: 0, dedupedSceneCount: 0, activeTaskCount: 0, sceneStatsComplete: true, sceneStatsStale: false };

test.use({ graphqlOverrides: {
  performerWorkspace: { activities: [], performers: { items: [performer], page: 1, pageSize: 24, totalCount: 1, totalPages: 1, hasPrevPage: false, hasNextPage: false }, subscribedPerformers: [] },
  stashPerformerProfile: profile,
  stashPerformerDetail: profile,
  stashPerformerScenes: { items: [], page: 1, pageSize: 24, totalCount: 0, totalPages: 1, hasPrevPage: false, hasNextPage: false, stashSceneCount: 0, stashBoxCount: 0, dedupedCount: 0, totalCountExact: true, stashBoxRemoteCount: 0, stashBoxLoadedCount: 0, cacheComplete: true, cacheStale: false },
} });

test("list uses thumbnail and detail keeps original", async ({ page }) => {
  const imageRequests: string[] = [];
  await page.route("**/api/images/**", route => {
    imageRequests.push(new URL(route.request().url()).search);
    return route.fulfill({ contentType: "image/png", body: Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=", "base64") });
  });
  await page.goto("/performers");
  const avatar = page.locator(".profile-card .avatar__image");
  await expect(avatar).toHaveAttribute("src", `${imagePath}?variant=avatar`);
  await expect.poll(() => avatar.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(1);
  expect(imageRequests).toEqual(["?variant=avatar"]);
  await page.getByRole("link", { name: "Avatar test", exact: true }).click();
  const detail = page.locator(".performer-detail-hero__avatar img");
  await expect(detail).toHaveAttribute("src", imagePath);
  await expect.poll(() => detail.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(1);
  expect(imageRequests).toContain("");
});

test("masked avatars cover the placeholder edges in list and detail", async ({ page }, info) => {
  await page.route("**/api/images/**", route => route.fulfill({
    contentType: "image/svg+xml",
    body: '<svg xmlns="http://www.w3.org/2000/svg" width="200" height="300"><rect width="200" height="300" fill="#d6bfa5"/></svg>',
  }));
  await page.goto("/performers?__locale=en");
  const toggle = page.getByRole("button", {name:"NSFW image masking",exact:true});
  await toggle.click();
  for (const view of ["list", "detail"]) {
    const frame = page.locator(view === "list" ? ".profile-card .avatar" : ".performer-detail-hero__avatar");
    const image = frame.locator("img");
    await expect.poll(() => image.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(200);
    await expect(image).toHaveCSS("filter", "blur(12px)");
    const bounds = await image.boundingBox();
    const crop = await frame.boundingBox();
    expect(bounds!.x).toBeCloseTo(crop!.x - 36, 0);
    expect(bounds!.width).toBeCloseTo(crop!.width + 72, 0);
    await frame.screenshot({path:info.outputPath(`masked-avatar-${view}.png`)});
    if (view === "list") await page.getByRole("link", {name:"Avatar test",exact:true}).click();
  }
  await toggle.click();
  await expect(page.locator(".performer-detail-hero__avatar img")).toHaveCSS("filter", "none");
});
