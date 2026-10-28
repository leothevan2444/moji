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

test.describe("partition saves", () => {
  const saves: any[] = [];
  test.use({ graphqlOverrides: {
    updateSettingsSection: ({ input }: any) => { saves.push(input); return settingsSnapshot; },
  } });
  test("saving resources preserves the unsaved task draft", async ({ page }) => {
    saves.length = 0;
    await page.goto("/settings/tasks");
    const interval = page.getByRole("spinbutton", { name: /Resource search interval/ });
    await interval.fill("12");
    await page.getByRole("button", { name: "Save resource rules", exact: true }).click();
    await expect.poll(() => saves.length).toBe(1);
    expect(Object.keys(saves[0]).sort()).toEqual(["resources", "section"]);
    await expect(interval).toHaveValue("12");
    await page.locator(".settings-tabs").getByRole("button", { name: "Stash", exact: true }).click();
    await page.locator(".settings-tabs").getByRole("button", { name: "Tasks & subscriptions", exact: true }).click();
    await expect(page.getByRole("spinbutton", { name: /Resource search interval/ })).toHaveValue("12");
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
    updateSystemSettings: ({input}:any) => { language=input.language;return {...settingsSnapshot,system:{...settingsSnapshot.system,language}}; },
    updateSettingsSection: ({input}:any) => {policy=input.taskDeletePolicy;return settingsSnapshot;},
  }});
  test("saves instance language, refreshes it and keeps deletion in tasks",async({page})=>{
    language="en";policy="KEEP_ONLY";
    await page.goto("/settings/indexers");
    await expect(page.getByRole("combobox",{name:"Language",exact:true})).toHaveCount(0);
    await page.goto("/settings/tasks");
    const deletion=page.getByRole("combobox",{name:/Task deletion policy/});
    await deletion.selectOption("REMOVE_TORRENT");
    await page.getByRole("button",{name:"Save changes",exact:true}).click();
    await expect.poll(()=>policy).toBe("REMOVE_TORRENT");
    await page.goto("/settings/system");
    await expect(page.getByRole("combobox",{name:/Task deletion policy/})).toHaveCount(0);
    await page.getByRole("combobox",{name:"Language",exact:true}).selectOption("zh-CN");
    await expect(page.locator("html")).toHaveAttribute("lang","en");
    await page.getByRole("button",{name:"Save system settings",exact:true}).click();
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
