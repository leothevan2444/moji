import { expect } from "@playwright/test";
import { test } from "./fixtures";

const actors = Array.from({ length: 25 }, (_, index) => ({
  id: `actor-${index}`, name: index === 0 ? "長い名前のテスト演员 · Example Performer" : `Performer ${String(index).padStart(2, "0")}`,
  aliasList: ["Example alias"], favorite: index < 2, imagePath: null, sceneCount: 25 - index, subscribed: index < 2,
}));
const states = actors.slice(0, 2).map((performer, index) => ({
  performer, lastCheckedAt: index === 0 ? "2026-01-15T11:30:00Z" : null, lastError: index === 0 ? null : "The release source could not be reached. Try checking again. ".repeat(12),
  pendingReleaseCount: 3, processedReleaseCount: 4, reviewReleaseCount: 2,
  recentReleases: [{ key: `release-${index}`, title: "An example release title", code: "ABC-123", date: "2026-01-10", seenAt: "2026-01-15T10:00:00Z" }],
}));
const scenes = Array.from({ length: 30 }, (_, index) => ({
  key: `scene-${index}`, primarySource: "STASHBOX", sourceSceneId: `scene-${index}`, code: `ABC-${100 + index}`, title: "An example scene with a longer title for layout verification", date: "2026-01-10", studioName: "Example Studio", performerCount: 1, tagCount: 2, performers: [], tags: [], imageUrl: null, url: null, inLibrary: index === 2, matchedStashSceneId: null,
  hasStashSource: index === 2, hasStashBoxSource: true, stashBoxSceneId: `scene-${index}`, stashBoxEndpoint: "https://example.com/graphql", sourceLabels: ["StashBox"], stashIds: [],
  mojiTask: index < 2 ? { id: `task-${index}`, stage: index === 0 ? "SOURCING" : "DOWNLOADING", stageStatus: index === 0 ? "PENDING" : "BLOCKED", stageLabel: "", stageStatusLabel: "", progress: index === 0 ? 0 : .5 } : null,
}));

const graphqlOverrides = {
  performerWorkspace: ({ search = "", page = 1, pageSize = 24, options }: { search?: string | null; page?: number; pageSize?: number; options?: { filter: string; sort: string; }; }) => {
    let items = actors.filter(actor => actor.name.toLowerCase().includes((search ?? "").toLowerCase()));
    if (options?.filter === "SUBSCRIBED") items = items.filter(actor => actor.subscribed);
    if (options?.filter === "FAVORITE") items = items.filter(actor => actor.favorite);
    if (options?.filter === "REVIEW") items = items.filter(actor => actor.id === "actor-24");
    if (options?.filter === "ERROR") items = items.filter(actor => actor.id === "actor-1");
    if (options?.sort === "NAME") items.sort((a, b) => a.name.localeCompare(b.name));
    return { performers: { items: items.slice((page - 1) * pageSize, page * pageSize), page, pageSize, totalCount: items.length, totalPages: Math.ceil(items.length / pageSize), hasPrevPage: page > 1, hasNextPage: page * pageSize < items.length }, subscribedPerformers: states };
  },
  stashPerformerDetail: { performer: actors[0], disambiguation: "Profile information remains readable on small screens.", birthdate: "2000-01-01", ethnicity: null, country: "JP", eyeColor: null, heightCm: 165, rating100: null, urls: [], matchedStashBox: { name: "Example StashBox", endpoint: "https://example.com/graphql", performerId: "remote-0", performerName: actors[0].name }, totalSceneCount: 30, stashSceneCount: 10, stashBoxSceneCount: 30, dedupedSceneCount: 30, activeTaskCount: 2, sceneStatsComplete: true, sceneStatsStale: false },
  stashPerformerScenes: ({ input }: { input: { page?: number; }; }) => { const page = input.page ?? 1; return { items: scenes.slice((page - 1) * 24, page * 24), page, pageSize: 24, totalCount: 30, totalPages: 2, hasPrevPage: page > 1, hasNextPage: page < 2, stashSceneCount: 10, stashBoxCount: 30, dedupedCount: 30, totalCountExact: true, stashBoxRemoteCount: 30, stashBoxLoadedCount: 30, cacheComplete: true, cacheUpdatedAt: "2026-01-15T11:30:00Z", cacheStale: false }; },
};
test.use({ graphqlOverrides: { ...graphqlOverrides, stashPerformerProfile: graphqlOverrides.stashPerformerDetail } });

for (const locale of ["zh-CN", "en"] as const) {
  test(`${locale} performer list with records`, async ({ page }) => {
    await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale);
    await page.goto("/performers");
    await expect(page.locator(".profile-card")).toHaveCount(24);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    await expect(page.locator(".skeleton")).toHaveCount(0);
    const cardGeometry = await page.locator(".profile-card").evaluateAll(cards => cards.map(card => {
      const avatar = card.querySelector(".avatar")!;
      const style = getComputedStyle(card);
      return { avatarWidth: avatar.getBoundingClientRect().width, imageRatio: innerWidth <= 640 ? 88 / 178 : 104 / 178, height: card.getBoundingClientRect().height, avatar: avatar.getBoundingClientRect().height, contentHeight: card.clientHeight - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom) };
    }));
    expect(cardGeometry.every(card => card.height === 180 && Math.abs(card.avatar - card.contentHeight) < 1 && Math.abs(card.avatarWidth - card.avatar * card.imageRatio) < 1)).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-performers-populated.png`);
    await page.locator(".performer-card-error-button").first().click();
    await expect(page.getByRole("dialog")).toBeVisible();
    expect(await page.locator(".profile-card").nth(1).evaluate(card => card.getBoundingClientRect().height)).toBe(180);
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    const pagination = page.locator(".pagination-bar--performers").first();
    expect(await pagination.evaluate(e => e.scrollWidth <= e.clientWidth)).toBe(true);
    await expect(pagination.locator('button[type="submit"]')).toBeInViewport();
  });

  test(`${locale} performer detail keeps identity and metrics readable`, async ({ page }) => {
    await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale);
    await page.goto("/performers/actor-0");
    await expect(page.locator(".performer-scene-card")).toHaveCount(24);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    await expect(page.locator(".performer-detail-hero__name")).toHaveText(actors[0].name);
    const checks = page.getByRole("region", { name: locale === "en" ? "Latest check" : "最近检查", exact: true });
    const works = page.getByRole("region", { name: locale === "en" ? "Scenes and tasks" : "作品与任务", exact: true });
    await expect(checks.locator("dt")).toHaveCount(2);
    await expect(works.locator("dd")).toHaveText(["10", "30", "2"]);
    expect(await checks.evaluate(el => el.getBoundingClientRect().height)).toBeLessThanOrEqual(50);
    expect(await works.evaluate(el => el.getBoundingClientRect().height)).toBeLessThanOrEqual(50);
    const cardInsets = await page.locator(".performer-detail-card").evaluate(card => {
      const rect = card.getBoundingClientRect();
      const content = card.querySelector(".performer-detail-hero")!.getBoundingClientRect();
      return { top: content.top - rect.top, bottom: rect.bottom - content.bottom };
    });
    expect(Math.abs(cardInsets.top - cardInsets.bottom)).toBeLessThanOrEqual(1);
    expect(await page.locator(".performer-detail-hero__name").evaluate(e => e.scrollWidth <= e.clientWidth)).toBe(true);
    const metrics = page.locator(".performer-summary-metrics");
    expect(await metrics.locator("dt").evaluateAll(es => es.every(e => e.scrollWidth <= e.clientWidth))).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-performer-detail.png`);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await expect(page.getByRole("combobox", { name: locale === "en" ? "Scene source" : "影片来源" })).toBeVisible();
    await expect(page.locator(".performer-results-meta")).toContainText(locale === "en" ? "30 results" : "共 30 条");
    await expect(page.locator(".performer-results-meta")).toContainText(locale === "en" ? "24 on this page" : "本页 24 条");
  });
}

test("performer selection survives pagination and filters", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/performers");
  await page.getByRole("button", { name: "Select multiple performers" }).click();
  await page.locator(".profile-card input[type=checkbox]").first().check();
  await page.locator(".pagination-bar--performers").first().getByRole("button", { name: "Next", exact: true }).click();
  await expect(page.locator(".profile-card")).toHaveCount(1);
  await page.locator(".profile-card input[type=checkbox]").check();
  await expect(page.getByRole("region", { name: "Selected performer actions" })).toContainText("2 performers selected");
  await page.getByRole("button", { name: "Awaiting review", exact: true }).click();
  await expect(page).toHaveURL(/filter=review/);
  await expect(page.locator(".profile-card input[type=checkbox]")).toBeChecked();
  await expect(page.getByRole("region", { name: "Selected performer actions" })).toContainText("2 performers selected");
});


test.describe("failed performer check", () => {
  test.use({
    graphqlOverrides: {
      ...graphqlOverrides,
      stashPerformerProfile: graphqlOverrides.stashPerformerDetail,
      performerWorkspace: (args: Parameters<typeof graphqlOverrides.performerWorkspace>[0]) => ({
        ...graphqlOverrides.performerWorkspace(args),
        subscribedPerformers: states.map(state => ({ ...state, lastError: states[1].lastError })),
      }),
    }
  });

  for (const locale of ["zh-CN", "en"] as const) {
    test(`${locale} check reason stays beside failure status`, async ({ page }) => {
      await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale);
      await page.goto("/performers/actor-0");
      const checks = page.getByRole("region", { name: locale === "en" ? "Latest check" : "最近检查", exact: true });
      const reason = checks.getByRole("button", { name: locale === "en" ? "View reason" : "查看原因" });
      await expect(reason).toBeVisible();
      const geometry = await checks.evaluate(el => {
        const status = el.querySelector(".performer-check-state")!.getBoundingClientRect();
        const button = el.querySelector("button")!.getBoundingClientRect();
        return { height: el.getBoundingClientRect().height, sameLine: Math.abs(status.top - button.top) <= 1, beside: button.left >= status.right, fits: button.right <= el.getBoundingClientRect().right };
      });
      expect(geometry).toEqual({ height: 48, sameLine: true, beside: true, fits: true });
      await expect(page.locator(".performer-detail-card")).toHaveScreenshot(`${locale}-performer-check-failed.png`);
      await reason.click();
      await expect(page.getByRole("dialog")).toContainText(states[1].lastError!);
      expect(await checks.evaluate(el => el.getBoundingClientRect().height)).toBe(48);
      await page.keyboard.press("Escape");
      await expect(page.getByRole("dialog")).toHaveCount(0);
      await expect(reason).toBeFocused();
    });
  }
});


for (const delayed of ["metadata", "profile"] as const) {
  test(`performer ${delayed} loading does not block other detail regions`, async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    let release!: () => void;
    const pending = new Promise<void>(resolve => { release = resolve; });
    await page.route("**/graphql", async route => {
      const body = route.request().postDataJSON();
      const shouldDelay = delayed === "metadata"
        ? /query StashPerformer(Detail|Scenes)\b/.test(body.query)
        : /query StashPerformerProfile\b/.test(body.query);
      if (shouldDelay) await pending;
      await route.fallback();
    });
    try {
      await page.goto("/performers/actor-0");
      await expect(page.getByRole("combobox", { name: "Scene source" })).toBeVisible();
      if (delayed === "metadata") {
        await expect(page.locator(".performer-detail-hero__name")).toHaveText(actors[0].name);
        await expect(page.locator(".performer-summary-metrics--works dd")).toHaveText(["25", "—", "—"]);
        await expect(page.locator(".performer-scene-card")).toHaveCount(0);
      } else {
        await expect(page.locator(".performer-scene-card")).toHaveCount(24);
        await expect(page.locator(".performer-detail-hero__name")).toHaveCount(0);
      }
    } finally { release(); }
    await expect(page.locator(".performer-detail-hero__name")).toHaveText(actors[0].name);
    await expect(page.locator(".performer-summary-metrics--works dd")).toHaveText(["10", "30", "2"]);
    await expect(page.locator(".performer-scene-card")).toHaveCount(24);
  });
}


test("scene multi-select is optional and clears when exited", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/performers/actor-0");
  await expect(page.locator(".performer-scene-card")).toHaveCount(24);
  await expect(page.locator(".performer-scene-selection")).toHaveCount(0);
  await expect(page.locator(".performer-scene-card__selector")).toHaveCount(0);
  await page.locator(".performer-scene-card").nth(3).click();
  await expect(page.locator(".performer-scene-card.is-selected")).toHaveCount(0);

  await page.getByRole("button", { name: "Select scenes", exact: true }).click();
  await expect(page.locator(".performer-scene-card__selector")).toHaveCount(21);
  await page.getByRole("button", { name: "Select this page (21)", exact: true }).click();
  await page.getByRole("button", { name: "Next", exact: true }).click();
  await expect(page.locator(".performer-scene-card")).toHaveCount(6);
  await page.getByRole("button", { name: "Select this page (6)", exact: true }).click();
  await expect(page.getByRole("button", { name: "Create download tasks (27)" })).toBeVisible();
  await expect(page).toHaveScreenshot("en-performer-scene-selection.png");

  await page.getByRole("button", { name: "Exit selection", exact: true }).click();
  await expect(page.locator(".performer-scene-selection")).toHaveCount(0);
  await expect(page.locator(".performer-scene-card__selector")).toHaveCount(0);
  await page.getByRole("button", { name: "Select scenes", exact: true }).click();
  await expect(page.locator(".performer-scene-card.is-selected")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Create download tasks", exact: true })).toBeDisabled();
});
