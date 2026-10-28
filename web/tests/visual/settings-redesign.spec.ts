import { expect } from "@playwright/test";
import { settingsSnapshot, test } from "./fixtures";

test.use({ graphqlOverrides: { logs: [] } });

for (const locale of ["zh-CN", "en"] as const) {
  test(`${locale} all settings tabs share layout and fit the viewport`, async ({ page }, info) => {
    await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale);
    for (const tab of ["indexers", "downloader", "stash", "tasks", "system", "logs", "about"]) {
      await page.goto(`/settings/${tab}`);
      await expect(page.locator(".settings-tabs [aria-current=page]")).toHaveCount(1);
      await expect(page.locator(".settings-workspace .drawer-card").first()).toBeVisible();
      await expect(page.locator(".skeleton, .settings-loading")).toHaveCount(0);
      await page.evaluate(() => document.fonts.ready);
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      await page.screenshot({ path: info.outputPath(`${locale}-${tab}.png`), fullPage: true });
    }
    for (const [old, current] of [["connections", "indexers"], ["ingest", "stash"], ["automation", "tasks"]]) {
      await page.goto(`/settings/${old}`);
      await expect(page).toHaveURL(new RegExp(`/settings/${current}$`));
    }
  });
}

for (const locale of ["qps-ploc", "ar-XB"] as const) {
  test(`${locale} settings support long text and direction`, async ({ page }, info) => {
    await page.goto(`/settings/tasks?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("dir", locale === "ar-XB" ? "rtl" : "ltr");
    await expect(page.locator(".settings-workspace .settings-form").first()).toBeVisible();
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await page.screenshot({ path: info.outputPath(`${locale}-tasks.png`), fullPage: true });
  });
}

test.describe("incremental autosave", () => {
  const saves: any[] = [];
  let current: any;
  test.use({ graphqlOverrides: {
    settings: () => current ?? settingsSnapshot,
    patchSettings: ({ input }: any) => {
      saves.push(input);
      if(input.tasks?.sourcing_retry_interval_hours !== undefined) current.automation.sourcingRetryIntervalHours=input.tasks.sourcing_retry_interval_hours;
      if(input.tasks?.torrent_selection?.enabled !== undefined) current.automation.torrentSelection.enabled=input.tasks.torrent_selection.enabled;
      return current;
    }
  }});
  test("debounces edits, saves independent fields, and continues across tab navigation", async ({ page }) => {
    saves.length=0;current=structuredClone(settingsSnapshot);current.system.language="en";
    await page.goto("/settings/tasks");
    await expect(page.locator('.settings-form button[type="submit"]')).toHaveCount(0);
    const interval=page.getByRole("spinbutton",{name:/Resource search interval/});
    await interval.fill("12");await interval.fill("18");
    await page.getByLabel("Enable rule chain",{exact:true}).check();
    await page.locator(".settings-tabs").getByRole("button",{name:"Stash",exact:true}).click();
    await expect.poll(()=>saves.length).toBe(2);
    expect(saves).toContainEqual({tasks:{sourcing_retry_interval_hours:18}});
    expect(saves).toContainEqual({tasks:{torrent_selection:{enabled:true}}});
    await page.locator(".settings-tabs").getByRole("button",{name:"Tasks & subscriptions",exact:true}).click();
    await expect(page.getByRole("spinbutton",{name:/Resource search interval/})).toHaveValue("18");
    await expect(page.getByLabel("Enable rule chain",{exact:true})).toBeChecked();
  });
  test("keeps failed edits and retries the same incremental fields",async({page})=>{
    saves.length=0;current=structuredClone(settingsSnapshot);current.system.language="en";let failed=false;
    await page.route("**/graphql",async route=>{
      if(route.request().postDataJSON().operationName==="PatchSettings" && !failed){failed=true;await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({errors:[{message:"Temporary save failure"}]})});return;}
      await route.fallback();
    });
    await page.goto("/settings/tasks");
    await page.getByRole("spinbutton",{name:/Resource search interval/}).fill("12");
    await expect(page.getByRole("alert")).toContainText("Your changes are retained");
    await expect(page.getByRole("spinbutton",{name:/Resource search interval/})).toHaveValue("12");
    await page.getByRole("button",{name:"Retry",exact:true}).click();
    await expect.poll(()=>saves.length).toBe(1);
    expect(saves[0]).toEqual({tasks:{sourcing_retry_interval_hours:12}});
    await expect(page.getByRole("alert")).toHaveCount(0);
  });
});

test("dark settings preserve navigation and keyboard access", async ({ page }, info) => {
  await page.addInitScript(() => localStorage.setItem("moji:theme", "dark"));
  await page.goto("/settings/indexers");
  await expect(page.locator(".settings-tabs [aria-current=page]")).toHaveText("Indexers");
  const show = page.getByRole("button", { name: "Show credential" }).first();
  await show.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("button", { name: "Hide credential" })).toBeFocused();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  await page.screenshot({ path: info.outputPath("dark-indexers.png"), fullPage: true });
});

test.describe("system language and task deletion ownership", () => {
  let language = "en";
  let policy = "KEEP_ONLY";
  test.use({graphqlOverrides:{
    settings: () => ({...settingsSnapshot,system:{...settingsSnapshot.system,language,taskDeletePolicy:policy}}),
    patchSettings: ({input}:any) => {
      if(input.system?.language !== undefined)language=input.system.language;
      if(input.tasks?.task_delete_policy !== undefined)policy=input.tasks.task_delete_policy;
      return {...settingsSnapshot,system:{...settingsSnapshot.system,language,taskDeletePolicy:policy}};
    },
  }});
  test("saves instance language, refreshes it and keeps deletion in tasks",async({page})=>{
    language="en";policy="KEEP_ONLY";
    await page.goto("/settings/indexers");
    await expect(page.getByRole("combobox",{name:"Language",exact:true})).toHaveCount(0);
    await page.goto("/settings/tasks");
    const deletion=page.getByRole("combobox",{name:/Task deletion policy/});
    await deletion.selectOption("REMOVE_TORRENT");
    await expect.poll(()=>policy).toBe("REMOVE_TORRENT");
    await page.goto("/settings/system");
    await expect(page.getByRole("combobox",{name:/Task deletion policy/})).toHaveCount(0);
    await page.getByRole("combobox",{name:"Language",exact:true}).selectOption("zh-CN");
    await expect(page.locator("html")).toHaveAttribute("lang","en");
    await expect(page.locator("html")).toHaveAttribute("lang","zh-CN");
    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("lang","zh-CN");
    await expect(page.getByRole("combobox",{name:"语言",exact:true})).toHaveValue("zh-CN");
    expect(policy).toBe("REMOVE_TORRENT");
  });
});

test("slow settings queries keep the current service and section titles",async({page})=>{
  let release!:()=>void;
  const gate=new Promise<void>(resolve=>{release=resolve;});
  await page.route("**/graphql",async route=>{
    const name=route.request().postDataJSON().operationName;
    if(name==="ServiceConnectionSettings" || name==="SettingsPolicySection")await gate;
    await route.fallback();
  });
  await page.goto("/settings/indexers",{waitUntil:"domcontentloaded"});
  try {
    await expect(page.getByRole("heading",{name:"Jackett",exact:true})).toBeVisible();
    await expect(page.getByRole("heading",{name:"Indexer query order",exact:true})).toBeVisible();
    await expect(page.getByRole("heading",{name:"Connections",exact:true})).toHaveCount(0);
    await expect(page.getByRole("heading",{name:"Automation",exact:true})).toHaveCount(0);
  } finally {release();}
  await expect(page.locator(".settings-loading")).toHaveCount(0);
});
