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
    return { activities: items.slice((page - 1) * pageSize, page * pageSize).map(actor=>{const rank=actor.id==="actor-0"?4:actor.id==="actor-2"?1:actor.id==="actor-3"?5:null;return {id:actor.id,value:rank===4?3.25:rank===1?0:rank===5?9:null,rank,lowerBound:false,evaluationStatus:rank?"READY":"NOT_CHECKED",observedAt:rank?"2026-01-15T12:00:00Z":null};}), performers: { items: items.slice((page - 1) * pageSize, page * pageSize), page, pageSize, totalCount: items.length, totalPages: Math.ceil(items.length / pageSize), hasPrevPage: page > 1, hasNextPage: page * pageSize < items.length }, subscribedPerformers: states };
  },
  stashPerformerDetail: { latestRelease: {code:"NEW-001",title:"Latest source release, independent of check history",date:"2026-01-14"}, performer: actors[0], disambiguation: "Profile information remains readable on small screens.", birthdate: "2000-01-01", ethnicity: null, country: "JP", eyeColor: null, heightCm: 165, rating100: null, urls: [], matchedStashBox: { name: "Example StashBox", endpoint: "https://example.com/graphql", performerId: "remote-0", performerName: actors[0].name }, totalSceneCount: 30, stashSceneCount: 10, stashBoxSceneCount: 30, dedupedSceneCount: 30, activeTaskCount: 2, sceneStatsComplete: true, sceneStatsStale: false },
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
    const cards=page.locator(".profile-card");
    const known=cards.nth(0).locator(".performer-activity-icons");
    await expect(known.locator("svg")).toHaveCount(5);
    await expect(known.locator(".is-active")).toHaveCount(4);
    await expect(known).toHaveAttribute("title",locale==="en"?"Activity level 4 / 5":"活跃度 4 / 5");
    await expect(cards.nth(0)).not.toContainText("3.25");
    const unknown=cards.nth(1).locator(".performer-activity-icons");
    await expect(unknown).toHaveClass(/is-unknown/);
    await expect(unknown.locator("svg")).toHaveCount(5);
    await expect(unknown.locator(".is-active")).toHaveCount(0);
    await expect(unknown).toHaveAttribute("aria-label",locale==="en"?"Activity level 0 / 5":"活跃度 0 / 5");
    await expect(unknown).toHaveAttribute("title",locale==="en"?/Level 0 means no reliable rating/:/0 档：尚无可靠评分/);
    expect(await unknown.evaluate(el=>getComputedStyle(el,"::after").borderTopWidth)).toBe("1px");
    await expect(cards.nth(2).locator(".performer-activity-icons .is-active")).toHaveCount(1);
    await expect(cards.nth(2).locator(".performer-activity-icons")).not.toHaveClass(/is-unknown/);
    await expect(cards.nth(3).locator(".performer-activity-icons .is-active")).toHaveCount(5);
    expect(await cards.evaluateAll(els=>els.every(card=>card.querySelector(".performer-activity-icons")!.getBoundingClientRect().right<=card.querySelector(".profile-actions")!.getBoundingClientRect().left))).toBe(true);
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
    const checks = page.getByRole("region", { name: locale === "en" ? "Checks" : "检查", exact: true });
    const works = page.getByRole("region", { name: locale === "en" ? "Scenes and tasks" : "作品与任务", exact: true });
    const activity = page.getByRole("region", {name: locale === "en" ? "Activity level" : "活跃度", exact:true});
    await expect(activity.locator("dt")).toHaveCount(2);
    await expect(checks.locator("dt")).toHaveCount(3);
    await expect(checks.locator("dt")).toContainText([locale === "en" ? "Estimated next check" : "预计下次检查",locale === "en" ? "Last check" : "上次检查"]);
    await expect(activity.locator("dd").last()).toHaveText("3.25");
    await expect(activity.locator("dd").last()).not.toHaveAttribute("title",/Rating updated|评分更新/);
    await expect(page.locator(".performer-detail-hero__description")).toContainText("NEW-001");
    await expect(page.locator(".performer-detail-hero__description")).not.toContainText("ABC-123");
    await expect(activity.locator(".performer-activity-icons svg")).toHaveCount(5);
    await expect(activity.locator(".performer-activity-icons .is-active")).toHaveCount(4);
    await expect(works.locator("dd")).toHaveText(["10", "30", "2"]);
    expect(await checks.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
    expect(await works.evaluate(el => el.getBoundingClientRect().height)).toBeLessThanOrEqual(66);
    if (page.viewportSize()!.width > 1100) {
      expect(await checks.evaluate(el => el.getBoundingClientRect().height)).toBeLessThanOrEqual(66);
      expect(await checks.evaluate(el => el.getBoundingClientRect().width)).toBeGreaterThan(await works.evaluate(el => el.getBoundingClientRect().width) * 1.25);
    }
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
  await expect(page).toHaveScreenshot("en-performers-selection.png");
  await page.getByRole("button", { name: "Exit performer selection" }).click();
  await expect(page.getByRole("region", { name: "Selected performer actions" })).toBeHidden();
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
      const checks = page.getByRole("region", { name: locale === "en" ? "Checks" : "检查", exact: true });
      const reason = checks.getByRole("button", { name: locale === "en" ? "View reason" : "查看原因" });
      await expect(reason).toBeVisible();
      const geometry = await checks.evaluate(el => {
        const status = el.querySelector(".performer-check-state")!.getBoundingClientRect();
        const button = el.querySelector("button")!.getBoundingClientRect();
        return { height: el.getBoundingClientRect().height, sameLine: Math.abs(status.top - button.top) <= 1, beside: button.left >= status.right, fits: button.right <= el.getBoundingClientRect().right };
      });
      expect(geometry).toMatchObject({ sameLine: true, beside: true, fits: true });
      await expect(page.locator(".performer-detail-card")).toHaveScreenshot(`${locale}-performer-check-failed.png`);
      await reason.click();
      await expect(page.getByRole("dialog")).toContainText(states[1].lastError!);
      expect(await checks.evaluate(el => el.getBoundingClientRect().height)).toBe(geometry.height);
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
  await expect(page.locator(".performer-scene-selection")).toBeHidden();
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
  await expect(page.locator(".performer-scene-selection")).toBeHidden();
  await expect(page.locator(".performer-scene-card__selector")).toHaveCount(0);
  await page.getByRole("button", { name: "Select scenes", exact: true }).click();
  await expect(page.locator(".performer-scene-card.is-selected")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Create download tasks", exact: true })).toBeDisabled();
});


for (const workspace of ["scenes", "performers"] as const) {
  test(`${workspace} selection expands and collapses smoothly`, async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "no-preference" });
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto(workspace === "scenes" ? "/performers/actor-0" : "/performers");
    const toggle = page.getByRole("button", { name: workspace === "scenes" ? "Select scenes" : "Select multiple performers", exact: true });
    await expect(toggle).toBeVisible();
    const motion = await page.evaluate(async () => {
      const toggle = document.querySelector<HTMLButtonElement>(".performer-toolbar__toggle")!;
      const panel = document.querySelector<HTMLElement>(".performer-toolbar__collapse")!;
      const initial = panel.getBoundingClientRect().height;
      let expanding = false;
      const started = (event: TransitionEvent) => {
        if (event.target === panel && event.propertyName === "grid-template-rows") expanding = true;
      };
      panel.addEventListener("transitionrun", started);
      toggle.click();
      await new Promise(requestAnimationFrame);
      await new Promise(requestAnimationFrame);
      const animations = panel.getAnimations();
      expanding ||= animations.some(animation => (animation as CSSTransition).transitionProperty === "grid-template-rows");
      await Promise.all(animations.map(animation => animation.finished));
      panel.removeEventListener("transitionrun", started);
      return { initial, expanding, expanded: panel.getBoundingClientRect().height };
    });
    expect(motion.initial).toBe(0);
    expect(motion.expanding).toBe(true);
    expect(motion.expanded).toBeGreaterThan(0);
    await page.getByRole("button", { name: workspace === "scenes" ? "Exit selection" : "Exit performer selection", exact: true }).click();
    await expect(page.locator(".performer-subtoolbar")).toBeHidden();
    await expect.poll(() => page.locator(".performer-toolbar__collapse").evaluate(el => el.getBoundingClientRect().height)).toBe(0);
  });

}


test.describe("manual checks for unsubscribed performers", () => {
  test.use({
    graphqlOverrides: {
      ...graphqlOverrides,
      stashPerformerProfile: { ...graphqlOverrides.stashPerformerDetail, performer: actors[3] },
      stashPerformerDetail: { ...graphqlOverrides.stashPerformerDetail, performer: actors[3] },
      startSubscriptionCheck: { id: "manual-check" },
    }
  });
  for (const locale of ["zh-CN", "en"] as const) {
    for (const location of ["list", "detail"] as const) {
      test(`${locale} ${location} confirms an unsubscribed performer check`, async ({ page }) => {
        await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale);
        const submissions: string[][] = [];
        page.on("request", request => {
          if (!request.url().endsWith("/graphql") || request.method() !== "POST") return;
          const body = request.postDataJSON();
          if (body.query?.includes("mutation StartSubscriptionCheck")) submissions.push(body.variables.ids);
        });
        await page.goto(location === "list" ? "/performers" : "/performers/actor-3");
        const check = page.getByRole("button", { name: locale === "en" ? "Check Performer 03 for updates" : "检查 Performer 03 的作品更新", exact: true });
        await expect(check).toBeEnabled();
        await check.click();
        const dialog = page.getByRole("dialog");
        await expect(dialog).toContainText(locale === "en" ? "one-time manual check" : "本次仅进行手动检查");
        expect(submissions).toEqual([]);
        if (location === "list") await expect(dialog).toHaveScreenshot(`${locale}-manual-check-confirm.png`);
        await dialog.getByRole("button", { name: locale === "en" ? "Cancel" : "取消", exact: true }).click();
        await expect(dialog).toHaveCount(0);
        expect(submissions).toEqual([]);
        await check.click();
        await dialog.getByRole("button", { name: locale === "en" ? "Start check" : "开始检查", exact: true }).click();
        await expect.poll(() => submissions).toEqual([["actor-3"]]);
        await expect(dialog).toHaveCount(0);
      });
    }
  }
});

for (const locale of ["zh-CN", "en"] as const) {
 for (const theme of ["light", "dark"] as const) {
  test.describe(`${locale} ${theme} persisted upcoming cards`, () => {
   test.use({graphqlOverrides:{...graphqlOverrides,stashPerformerProfile:graphqlOverrides.stashPerformerDetail,
    stashPerformerScenes:(args:{input:{page?:number}})=>{
     const connection=graphqlOverrides.stashPerformerScenes(args);
     return {...connection,items:connection.items.map((scene,index)=>({...scene,upcomingState:index===3?"WAITING":index===4?"UNCONFIRMED":index===5?"HANDOFF_PENDING":null,date:index>=3&&index<=5?"2026-11-12":scene.date}))};
    }
   }});
   test("state replaces creation and is excluded from page selection",async({page},info)=>{
    await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme);},{locale,theme});
    await page.goto("/performers/actor-0");
    const cards=page.locator(".performer-scene-card");await expect(cards).toHaveCount(24);
    for (const [index,label] of [[3,locale==="en"?"Upcoming":"待发行"],[4,locale==="en"?"Unconfirmed":"待确认"],[5,locale==="en"?"Processing release":"发行处理中"]] as const) {
     await expect(cards.nth(index).locator(".performer-scene-card__business-state")).toHaveText(label);
     await expect(cards.nth(index).locator(".performer-scene-card__create-task")).toHaveCount(0);
    }
    await page.getByRole("button",{name:locale==="en"?"Select scenes":"多选作品",exact:true}).click();
    await expect(page.getByRole("button",{name:locale==="en"?"Select this page (18)":"全选本页（18）",exact:true})).toBeVisible();
    await expect(cards.nth(3).locator(".performer-scene-card__selector")).toHaveCount(0);
    await cards.nth(3).click();await expect(cards.nth(3)).not.toHaveClass(/is-selected/);
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);
    await cards.nth(3).screenshot({path:info.outputPath(`${locale}-${theme}-performer-upcoming.png`)});
    await page.reload();await expect(cards.nth(3).locator(".performer-scene-card__business-state")).toHaveText(locale==="en"?"Upcoming":"待发行");
   });
  });
 }
}


test.describe("activity observations",()=>{
 test.use({graphqlOverrides:{...graphqlOverrides,stashPerformerProfile:graphqlOverrides.stashPerformerDetail,
  performerActivity:({id}:{id:string})=>({id,evaluationStatus:"NOT_CHECKED",value:null,rank:null,lowerBound:false,observedAt:null,lastSuccessAt:null,nextCheckAt:"2026-01-17T12:00:00Z"})}});
 test("unknown rating retains five unlit icons and a planned check",async({page})=>{
  await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));
  await page.goto("/performers/actor-0");
  const icons=page.locator(".performer-activity-icons");
  await expect(icons).toHaveAttribute("aria-label","Not assessed");
  await expect(icons.locator("svg")).toHaveCount(5);
  await expect(icons.locator(".is-active")).toHaveCount(0);
  await expect(page.getByText("Estimated next check",{exact:true})).toBeVisible();
  await expect(page.locator(".performer-summary-metrics--activity").getByText("Not assessed",{exact:true})).toBeVisible();
 });
});

test("dark performer activity remains readable",async({page})=>{
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));
 await page.emulateMedia({colorScheme:"dark"});
 await page.goto("/performers/actor-0");
 await expect.poll(()=>page.locator("html").evaluate(el=>getComputedStyle(el).colorScheme)).toBe("dark");
 await expect(page.locator(".performer-activity-icons .is-active")).toHaveCount(4);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await expect(page.locator(".performer-detail-card")).toHaveScreenshot("dark-performer-activity.png");
});

test("dark performer cards show saved levels and crossed-out unknown ratings",async({page})=>{
 await page.addInitScript(()=>{localStorage.setItem("moji:locale","zh-CN");localStorage.setItem("moji:theme","dark");});
 await page.goto("/performers");
 const cards=page.locator(".profile-card");
 await expect(cards).toHaveCount(24);
 await expect(cards.nth(0).locator(".performer-activity-icons .is-active")).toHaveCount(4);
 await expect(cards.nth(1).locator(".performer-activity-icons")).toHaveClass(/is-unknown/);
 await expect(cards.nth(0)).toHaveScreenshot("dark-performer-card-rated.png");
 await expect(cards.nth(1)).toHaveScreenshot("dark-performer-card-unknown.png");
});

test.describe("cast filter", () => {
  const castScenes = scenes.map((scene, index) => ({ ...scene, performerCount: index % 3 === 0 ? 0 : index % 2 === 0 ? 1 : 2 }));
  const pageForCast = ({ input }: { input: { cast?: string; page?: number; pageSize?: number } }) => {
    const filtered = castScenes.filter(scene => input.cast === "SOLO" ? scene.performerCount === 1 : input.cast === "GROUP" ? scene.performerCount > 1 : true);
    const page = input.page ?? 1, pageSize = input.pageSize ?? 24;
    return { items: filtered.slice((page - 1) * pageSize, page * pageSize), page, pageSize, totalCount: filtered.length, totalPages: Math.ceil(filtered.length / pageSize), hasPrevPage: page > 1, hasNextPage: page * pageSize < filtered.length, stashSceneCount: 10, stashBoxCount: 30, dedupedCount: filtered.length, totalCountExact: true, stashBoxRemoteCount: 30, stashBoxLoadedCount: 30, cacheComplete: true, cacheUpdatedAt: "2026-01-15T11:30:00Z", cacheStale: false };
  };
  test.use({ graphqlOverrides: { ...graphqlOverrides, stashPerformerProfile: graphqlOverrides.stashPerformerDetail, stashPerformerScenes: pageForCast, refreshStashPerformerScenes: pageForCast } });
  for (const locale of ["en", "zh-CN"] as const) {
    for (const theme of ["light", "dark"] as const) {
      test(`${locale} ${theme} filters before paging and preserves URL and selection`, async ({ page }) => {
        await page.addInitScript(({ locale, theme }) => { localStorage.setItem("moji:locale", locale); localStorage.setItem("moji:theme", theme); }, { locale, theme });
        const requests: string[] = [];
        page.on("request", request => { if (request.method() === "POST" && request.url().endsWith("/graphql")) { const body = request.postDataJSON(); if (body.operationName === "StashPerformerScenes") requests.push(body.variables.input.cast); } });
        await page.goto("/performers/actor-0?scenePage=2");
        await expect(page.locator(".performer-scene-card")).toHaveCount(6);
        const cast = page.getByRole("combobox", { name: locale === "en" ? "Cast type" : "出演类型", exact: true });
        await expect(cast.locator("option")).toHaveText(locale === "en" ? ["All", "Single", "Multiple"] : ["全部", "单人作品", "多人作品"]);
        await cast.selectOption("SOLO");
        await expect(page).toHaveURL(/cast=solo/);
        expect(new URL(page.url()).searchParams.has("scenePage")).toBe(false);
        await expect(page.locator(".performer-scene-card")).toHaveCount(10);
        await expect.poll(() => requests.at(-1)).toBe("SOLO");
        await cast.selectOption("GROUP");
        await expect(page.locator(".performer-scene-card")).toHaveCount(10);
        await expect.poll(() => requests.at(-1)).toBe("GROUP");
        await page.getByRole("button", { name: locale === "en" ? "Select scenes" : "多选作品", exact: true }).click();
        const selected = page.locator(".performer-scene-card").filter({ has: page.getByText("ABC-105", { exact: true }) });
        await selected.click();
        await cast.selectOption("SOLO");
        await expect(page.locator(".performer-scene-card")).toHaveCount(10);
        await cast.selectOption("GROUP");
        await expect(selected.locator(".performer-scene-card__selector")).toHaveAttribute("aria-pressed", "true");
        await page.evaluate(() => document.fonts.ready);
        await expect(page.locator(".performer-scene-filters")).toHaveScreenshot(`${locale}-${theme}-cast-filter.png`);
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
        await page.reload();
        await expect(cast).toHaveValue("GROUP");
        await expect(page.locator(".performer-scene-card")).toHaveCount(10);
      });
    }
  }
});
