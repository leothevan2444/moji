import { expect } from "@playwright/test";
import { test } from "./fixtures";

const imagePath = `/api/images/${"a".repeat(64)}`;
const performer = { id: "avatar-test", name: "Avatar test", aliasList: [], favorite: false, imagePath, sceneCount: 0, subscribed: false };
const profile = { performer, disambiguation: null, birthdate: null, ethnicity: null, country: null, eyeColor: null, heightCm: null, rating100: null, urls: [], matchedStashBox: null, totalSceneCount: 0, stashSceneCount: 0, stashBoxSceneCount: 0, dedupedSceneCount: 0, activeTaskCount: 0, sceneStatsComplete: true, sceneStatsStale: false };

test.use({ graphqlOverrides: {
  performerWorkspace: { performers: { items: [performer], page: 1, pageSize: 24, totalCount: 1, totalPages: 1, hasPrevPage: false, hasNextPage: false }, subscribedPerformers: [] },
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
