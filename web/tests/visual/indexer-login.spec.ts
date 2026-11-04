import { readFileSync } from "node:fs";
import { expect, type Page } from "@playwright/test";
import { test, indexerInstances } from "./fixtures";

const image = `data:image/png;base64,${readFileSync(new URL("./assets/indexer-challenge.png", import.meta.url)).toString("base64")}`;
async function loginServer(page: Page) {
  const calls: { operation: string; variables: Record<string, unknown> }[] = [];
  let current: Record<string, unknown> | null = null;
  let starts = 0;
  await page.route("**/graphql", async route => {
    const request = route.request().postDataJSON();
    if (request.operationName === "IndexerInstances") return route.fulfill({ json: { data: { indexerInstances: { ...indexerInstances, items: indexerInstances.items.map(site => site.id === "builtin-one" ? { ...site, loginSupported: true, privacy: "private" } : site) } } } });
    if (!["IndexerLogin", "BeginIndexerLogin", "AnswerIndexerChallenge", "CancelIndexerLogin"].includes(request.operationName)) return route.fallback();
    calls.push({ operation: request.operationName, variables: request.variables });
    let field = "indexerLogin", value: unknown = current;
    if (request.operationName === "BeginIndexerLogin") {
      starts++;
      const id = `flow-${starts}`;
      current = { __typename: "IndexerLoginReport", id, instanceId: request.variables.input.instanceId, configRevision: request.variables.input.expectedRevision, ruleRevision: "a".repeat(40), state: "WAITING", expiresAt: new Date(Date.parse("2026-01-15T12:00:00Z") + starts * 300000).toISOString(), challenge: { __typename: "IndexerLoginChallenge", id, type: "image", prompt: "Enter the digits shown below.", imageData: image } };
      field = "beginIndexerLogin"; value = current;
    } else if (request.operationName === "AnswerIndexerChallenge") {
      field = "answerIndexerChallenge";
      if (request.variables.answer !== "42") { current = null; return route.fulfill({ json: { data: null, errors: [{ message: "INDEXER_AUTHENTICATION_FAILED" }] } }); }
      value = { ...current, state: "READY", challenge: null }; current = null;
    } else if (request.operationName === "CancelIndexerLogin") { field = "cancelIndexerLogin"; value = true; current = null; }
    return route.fulfill({ json: { data: { [field]: value } } });
  });
  return calls;
}
async function openLogin(page: Page, chinese = false) {
  await page.locator('.indexer-row[data-indexer-id="builtin-one"]').getByRole("button", { name: chinese ? "登录站点" : "Log in to site", exact: true }).click();
  return page.getByRole("dialog", { name: chinese ? "登录站点" : "Log in to site", exact: true });
}
test("login starts explicitly and an idle challenge survives close and page reload", async ({ page }) => {
  const calls = await loginServer(page);
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en")); await page.goto("/settings/indexers");
  expect(await page.locator('.indexer-row[data-indexer-id="jackett/onejav"]').getByRole("button", { name: "Log in to site", exact: true }).count()).toBe(0);
  let dialog = await openLogin(page);
  await expect(dialog.getByRole("button", { name: "Start login" })).toBeEnabled();
  expect(calls.map(call => call.operation)).toEqual(["IndexerLogin"]);
  await dialog.getByRole("button", { name: "Start login" }).click();
  await expect(dialog.getByRole("textbox", { name: "Challenge answer" })).toBeVisible();
  expect(calls[1].variables).toEqual({ input: { instanceId: "builtin-one", expectedRevision: indexerInstances.revision } });
  await dialog.getByRole("button", { name: "Close", exact: true }).click();
  await page.reload(); dialog = await openLogin(page);
  await expect(dialog.getByRole("img", { name: "Site challenge" })).toBeVisible();
  expect(calls.filter(call => call.operation === "BeginIndexerLogin")).toHaveLength(1);
  await dialog.getByRole("textbox", { name: "Challenge answer" }).fill("42");
  await dialog.getByRole("button", { name: "Submit answer" }).click();
  await expect(dialog.getByText(/The login session is available/)).toBeVisible();
  expect(calls.find(call => call.operation === "AnswerIndexerChallenge")?.variables).toEqual({ id: "flow-1", input: { instanceId: "builtin-one", expectedRevision: indexerInstances.revision }, answer: "42" });
});
test("failed answer and explicit cancellation do not claim a successful login", async ({ page }) => {
  const calls = await loginServer(page);
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en")); await page.goto("/settings/indexers"); const dialog = await openLogin(page);
  await dialog.getByRole("button", { name: "Start login" }).click(); await dialog.getByRole("textbox").fill("wrong"); await dialog.getByRole("button", { name: "Submit answer" }).click();
  await expect(dialog.getByRole("alert")).toBeVisible(); await expect(dialog.getByText(/The login session is available/)).toHaveCount(0);
  await dialog.getByRole("button", { name: "Start login" }).click(); await expect(dialog.getByRole("textbox")).toHaveValue("");
  await dialog.getByRole("button", { name: "Cancel login" }).click(); await expect(dialog.getByText("Login verification cancelled.")).toBeVisible();
  expect(calls.filter(call => call.operation === "CancelIndexerLogin")).toHaveLength(1);
});
for (const scenario of [{ locale: "en", theme: "light" }, { locale: "zh-CN", theme: "light" }, { locale: "ar-XB", theme: "light" }, { locale: "en", theme: "dark" }]) {
  test(`${scenario.locale} ${scenario.theme} challenge stays readable and keyboard accessible`, async ({ page }) => {
    await loginServer(page);
    await page.addInitScript(value => { localStorage.setItem("moji:locale", value.locale === "ar-XB" ? "en" : value.locale); localStorage.setItem("moji:theme", value.theme); }, scenario);
    await page.goto(`/settings/indexers${scenario.locale === "ar-XB" ? "?__locale=ar-XB" : ""}`);
    const chinese = scenario.locale === "zh-CN"; const dialog = await openLogin(page, chinese);
    await dialog.getByRole("button", { name: chinese ? "开始登录" : "Start login" }).click();
    await expect(dialog.getByRole("textbox")).toBeFocused();
    await expect(dialog.getByRole("img")).toBeVisible();
    expect(await dialog.evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
    await page.evaluate(() => document.fonts.ready);
    await expect(dialog).toHaveScreenshot(`${scenario.locale}-${scenario.theme}-indexer-login.png`);
    await page.keyboard.press("Escape"); await expect(dialog).toHaveCount(0);
  });
}

test("expiry stops answer entry without polling and restart cancels the old capability first", async ({page}) => {
 const calls=await loginServer(page);
 await page.clock.install({time:new Date("2026-01-15T12:00:00Z")});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/indexers");const dialog=await openLogin(page);
 await dialog.getByRole("button",{name:"Start login"}).click();await expect(dialog.getByRole("textbox")).toBeVisible();
 await page.clock.fastForward(300001);
 await expect(dialog.getByText("This challenge expired. Restart login.")).toBeVisible();
 expect(calls.map(call=>call.operation)).toEqual(["IndexerLogin","BeginIndexerLogin"]);
 await dialog.getByRole("button",{name:"Restart login"}).click();await expect(dialog.getByRole("textbox")).toBeVisible();
 expect(calls.map(call=>call.operation)).toEqual(["IndexerLogin","BeginIndexerLogin","CancelIndexerLogin","BeginIndexerLogin"]);
});

test.describe("long saved site names",()=>{
 const name="SavedSiteWithAnUnbrokenName".repeat(9);
 test.use({graphqlOverrides:{indexerInstances:{...indexerInstances,items:indexerInstances.items.map(site=>site.id==="builtin-one"?{...site,name}:site)}}});
 test("site names wrap inside the login panel",async({page})=>{
  await loginServer(page);await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/settings/indexers");const dialog=await openLogin(page);
  await expect(dialog.getByText(name,{exact:true})).toBeVisible();
  expect(await dialog.locator(".drawer-body").evaluate(element=>element.scrollWidth<=element.clientWidth)).toBe(true);
 });
});
