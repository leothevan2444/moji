import { expect } from "@playwright/test";
import { test } from "./fixtures";

const artwork = "data:image/svg+xml," + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360"><rect width="640" height="360" fill="#234567"/><circle cx="320" cy="180" r="100" fill="#d9a234"/></svg>');

test.use({ graphqlOverrides: {
  releaseReviews: {
    totalCount: 1, pendingCount: 1, failedCount: 0, page: 1, pageSize: 24, totalPages: 1, performers: [],
    items: [{ key: "mask-test", code: "TEST-123", title: "Example release", date: "2026-01-12", url: null,
      imageUrl: artwork, source: "example", vr: false, status: "PENDING", reasonCode: "group_behavior_review",
      error: null, taskId: null, seenAt: "2026-01-15T12:00:00Z", reviewedAt: null, performerNames: [], performers: [] }],
  },
} });

test("masks media immediately, keeps functional images readable and restores after reload", async ({ page }, info) => {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.goto("/performers/review?__locale=en");
  const toggle = page.getByRole("button", { name: "NSFW image masking", exact: true });
  const image = page.locator("img[data-nsfw]").first();
  await expect(image).toBeVisible();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await expect(toggle).toHaveText("NSFW");
  await expect(image).toHaveCSS("filter", "none");
  await toggle.focus();
  await page.keyboard.press("Space");
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await expect(toggle).toHaveText("NSFW");
  await expect(image).toHaveCSS("filter", "blur(12px)");
  // Unmarked functional images, such as verification challenges, stay readable.
  await page.evaluate(src => {
    const image = document.createElement("img");
    image.src = src;
    image.id = "functional-image";
    document.querySelector("main")!.append(image);
  }, artwork);
  await expect(page.locator("#functional-image")).toHaveCSS("filter", "none");
  await page.reload();
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await expect(image).toHaveCSS("filter", "blur(12px)");
  await page.screenshot({ path: info.outputPath("masked-media.png"), fullPage: true });
  await page.locator('.masthead__tools a[href="/stats"]').click();
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await toggle.click();
  expect(await page.evaluate(() => localStorage.getItem("moji:nsfw-mask"))).toBe("false");
  expect(errors).toEqual([]);
});
