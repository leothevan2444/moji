import { expect } from "@playwright/test";
import { test, indexerInstances, indexerLibrary } from "./fixtures";

const observedSearch = { startedAt: "2026-01-15T11:00:00Z", finishedAt: "2026-01-15T11:00:01Z", errorCode: "", correlationId: "00000000-0000-4000-8000-000000000001", retryAt: null };
const observedFetch = { startedAt: "2026-01-15T11:01:00Z", finishedAt: "2026-01-15T11:01:01Z", errorCode: "INDEXER_AUTHENTICATION_FAILED", correlationId: "00000000-0000-4000-8000-000000000002", retryAt: null };

test.describe("empty select default", () => {
  test.use({ graphqlOverrides: {
    indexerInstances: { ...indexerInstances, items: [{ ...indexerInstances.items[0], settings: [{ name: "choice", type: "select", label: "Choice", secret: false, configured: false, value: null, defaultValue: {string:"",boolean:null,number:null}, options: [{value:"a",label:"First"},{value:"b",label:"Second"}] }] }] }
  } });
  test("an unset selection remains visibly empty until the user chooses", async ({ page }, testInfo) => {
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto("/settings/indexers");
    await page.locator(".indexer-row").first().getByRole("button", {name:"Edit",exact:true}).click();
    await expect(page.getByLabel("Choice", {exact:true})).toHaveValue("");
    await expect(page.getByRole("option", {name:"No selection",exact:true})).toHaveJSProperty("selected", true);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({path:testInfo.outputPath("empty-select.png"),fullPage:true});
    await page.getByLabel("Choice", {exact:true}).selectOption("b");
    await expect(page.getByLabel("Choice", {exact:true})).toHaveValue("b");
  });
});

test.describe("search cache management", () => {
  let entries = 3;
  const scopes: Array<string | null> = [];
  const status = () => ({ revision: "config-1", enabled: true, ttlSeconds: 300, emptyTtlSeconds: 30, maxSizeMb: 32, entries, usedBytes: entries ? 1024 : 0, activeRequests: 1, activeWaiters: 2 });
  test.use({ graphqlOverrides: {
    indexerCacheStatus: status,
    clearIndexerSearchCache: ({ instanceId }: { instanceId: string | null }) => { scopes.push(instanceId); entries = 0; return status(); }
  } });
  test("cache clear confirms a captured site scope and rereads server counts", async ({ page }) => {
    entries = 3; scopes.length = 0;
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    try {
      await page.goto("/settings/indexers?panel=cache");
      await expect(page.getByRole("tab", { name: "Network & cache", exact: true })).toHaveAttribute("aria-selected", "true");
      await expect(page.locator(".indexer-cache-metrics dd").nth(0)).toHaveText("3");
      await expect(page).toHaveScreenshot("indexer-cache.png", { fullPage: true });
      await page.getByLabel("Clear scope", { exact: true }).selectOption("builtin-one");
      await page.getByRole("button", { name: "Clear search cache", exact: true }).click();
      const dialog = page.getByRole("dialog", { name: "Clear search cache", exact: true });
      await expect(dialog.getByText("Sukebei", { exact: true })).toBeVisible();
      expect(scopes).toEqual([]);
      await dialog.getByRole("button", { name: "Clear search cache", exact: true }).click();
      await expect(dialog).toHaveCount(0);
      expect(scopes).toEqual(["builtin-one"]);
      await expect(page.locator(".indexer-cache-metrics dd").nth(0)).toHaveText("0");
      await page.reload();
      await expect(page.locator(".indexer-cache-metrics dd").nth(0)).toHaveText("0");
      await expect(page.locator(".indexer-cache-metrics dd").nth(2)).toHaveText("1");
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    } finally { entries = 3; scopes.length = 0; }
  });
});
test.describe("persisted site observations", () => {
  test.use({ graphqlOverrides: {
    indexerInstances: { ...indexerInstances, items: [{ ...indexerInstances.items[0], health: { search: observedSearch, fetch: observedFetch } }] },
    homeIndexerSummary: { definitionJob: null, definitionErrorCode: "", revision: "observed", total: 1, builtinEnabled: 1, builtinUnverified: 0, builtinSearchSucceeded: 1, healthErrorCode: "", attention: 1, issues: [{ id: "builtin-one", name: "Sukebei", errorCode: observedFetch.errorCode, operation: "FETCH", observation: observedFetch }], jackett: null }
  } });
  test("search success and acquisition failure remain distinct after refresh", async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto("/settings/indexers");
    const site = page.locator(".indexer-row").first();
    await expect(site.getByText(/^Last search succeeded/)).toBeVisible();
    await expect(site.getByText(/^Last acquisition failed/)).toBeVisible();
    await page.reload();
    await expect(site.getByText(/^Last acquisition failed/)).toBeVisible();
    await site.locator("details summary").click();
    await expect(site.getByText(observedFetch.correlationId, { exact: true })).toBeVisible();
    await expect(page).toHaveScreenshot("indexer-operation-observations.png", { fullPage: true });
    await page.goto("/");
    const summary = page.getByRole("region", { name: "Indexers", exact: true });
    await expect(summary.getByText("Last search succeeded for 1 built-in site", { exact: true })).toBeVisible();
    await expect(summary.getByText(/^Last acquisition failed/)).toBeVisible();
    await expect(summary.getByText(/has not been verified/)).toHaveCount(0);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  });
});

for (const scenario of [{ locale: "zh-CN", theme: "light", edit: "编辑", library: "规则库" }, { locale: "en", theme: "dark", edit: "Edit", library: "Definitions" }, { locale: "ar-XB", theme: "dark", edit: "", library: "" }]) {
  test(`${scenario.locale} indexer sites, editor and library`, async ({ page }) => {
    await page.addInitScript(value => { localStorage.setItem("moji:locale", value.locale); localStorage.setItem("moji:theme", value.theme); }, scenario);
    await page.goto(`/settings/indexers${scenario.locale === "ar-XB" ? "?__locale=ar-XB" : ""}`);
    await expect(page.locator("html")).toHaveAttribute("dir",scenario.locale === "ar-XB" ? "rtl" : "ltr");
    await expect(page.locator(".indexer-row")).toHaveCount(2);
    await page.evaluate(() => document.fonts.ready);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await expect(page).toHaveScreenshot(`${scenario.locale}-indexers-sites.png`, { fullPage: true });
    await page.locator(".indexer-row").first().getByRole("button", { name: scenario.locale === "zh-CN" ? "编辑" : "Edit", exact: true }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByLabel("Cookie", { exact: true })).toHaveValue("");
    await expect(page).toHaveScreenshot(`${scenario.locale}-indexers-editor.png`, { fullPage: true });
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await page.locator("#settings-tab-definitions").click();
    await expect(page.locator(".settings-subpanel:not([hidden])")).toContainText("579");
    await expect(page).toHaveScreenshot(`${scenario.locale}-indexers-library.png`, { fullPage: true });
  });
}

test("download progress survives reload and cancellation reports the durable outcome", async ({ page, graphqlOverrides }) => {
  const job = { id: "job", kind: "INSTALL", state: "DOWNLOADING", targetRevision: "a".repeat(40), expectedActiveRevision: "", downloadedBytes: 1048576, errorCode: "", correlationId: "correlation", startedAt: "2026-01-15T12:00:00Z", finishedAt: null as string | null, revision: 1 };
  let library = { ...indexerLibrary, activeRevision: "", availableRevision: "", available: false, installed: [], errorCode: "DEFINITION_LIBRARY_NOT_INSTALLED", job: null as typeof job | null };
  let installs = 0;
  const original={...graphqlOverrides};
  try {
  Object.assign(graphqlOverrides, {
    indexerInstances: { ...indexerInstances, ruleRevision: "", jackettEnabled: false, items: [], order: [] },
    indexerDefinitionStatus: () => library, indexerDefinitionEvents: () => library,
    checkIndexerDefinitionUpdate: () => { library = { ...library, availableRevision: "a".repeat(40), revision: 2 }; return library; },
    installIndexerDefinitions: () => { installs++; library = { ...library, job }; return job; },
    cancelIndexerDefinitionUpdate: () => { library = { ...library, job: { ...job, state: "CANCELLED", finishedAt: "2026-01-15T12:01:00Z", revision: 2 } }; return library.job; }
  });
  // The fixture captures overrides while opening the page; use a route to read
  // the mutable owner on each HTTP operation instead of fabricating local UI state.
  await page.route("**/graphql", async route => {
    const request = route.request().postDataJSON();
    const fields: Record<string, string> = { IndexerDefinitionStatus: "indexerDefinitionStatus", CheckIndexerDefinitionUpdate: "checkIndexerDefinitionUpdate", InstallIndexerDefinitions: "installIndexerDefinitions", CancelIndexerDefinitionUpdate: "cancelIndexerDefinitionUpdate" };
    const field = fields[request.operationName];
    if (!field) { await route.fallback(); return; }
    const handler = graphqlOverrides[field]; const value = typeof handler === "function" ? handler(request.variables) : handler;
    await route.fulfill({ contentType: "application/json", body: JSON.stringify({ data: { [field]: value } }) });
  });
  await page.goto("/settings/indexers?panel=definitions");
  await page.getByRole("button", { name: "Download definitions", exact: true }).click();
  await expect(page.getByText("Downloading definitions", { exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByText("Downloading definitions", { exact: true })).toBeVisible();
  expect(installs).toBe(1);
  await page.getByRole("button", { name: "Cancel update" }).click();
  await expect(page.getByText("Update cancelled", { exact: true })).toBeVisible();
  } finally { for(const key of Object.keys(graphqlOverrides))delete graphqlOverrides[key];Object.assign(graphqlOverrides,original); }
});

test("catalog selection saves only explicit source fields and hides saved credentials", async ({ page }) => {
  let sent: Record<string, unknown> | null = null;
  await page.route("**/graphql", async route => {
    const request = route.request().postDataJSON();
    if (request.operationName !== "AddBuiltinIndexer") { await route.fallback(); return; }
    sent = request.variables.input;
    await route.fulfill({ contentType: "application/json", body: JSON.stringify({ data: { addBuiltinIndexer: { revision: "config-2", instance: { ...indexerInstances.items[0], id: "new-source", name: "Another site" } } } }) });
  });
  await page.goto("/settings/indexers");
  await page.getByRole("button", { name: "Add site", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.getByRole("button", { name: "Add another configuration", exact: true }).click();
  await page.getByLabel("Name", { exact: true }).fill("Another site");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(sent).toMatchObject({ definitionId: "sukebei", name: "Another site", enabled: false, networkProfile: "direct" });
  expect(sent).not.toHaveProperty("settings");
});

test("completed installation refreshes the site catalog without leaving settings", async ({ page, graphqlOverrides }) => {
  const original = { ...graphqlOverrides };
  let library = { ...indexerLibrary, activeRevision: "", availableRevision: "", available: false, revision: 1, installed: [] as typeof indexerLibrary.installed };
  let sources = { ...indexerInstances, ruleRevision: "", jackettEnabled: false, order: [] as string[], items: [] as typeof indexerInstances.items };
  const complete = { id: "installed", kind: "INSTALL", state: "COMPLETED", targetRevision: "a".repeat(40), expectedActiveRevision: "", downloadedBytes: 1000, errorCode: "", correlationId: "correlation", startedAt: "2026-01-15T12:00:00Z", finishedAt: "2026-01-15T12:00:01Z", revision: 2 };
  try {
    Object.assign(graphqlOverrides, { indexerDefinitionEvents: () => library });
    await page.route("**/graphql", async route => {
      const operation = route.request().postDataJSON().operationName;
      let data: Record<string, unknown>;
      if (operation === "IndexerInstances") data = { indexerInstances: sources };
      else if (operation === "IndexerDefinitionStatus") data = { indexerDefinitionStatus: library };
      else if (operation === "CheckIndexerDefinitionUpdate") { library = { ...library, availableRevision: "a".repeat(40), revision: 2 }; data = { checkIndexerDefinitionUpdate: library }; }
      else if (operation === "InstallIndexerDefinitions") { library = { ...indexerLibrary, revision: 3 }; sources = { ...sources, ruleRevision: "a".repeat(40) }; data = { installIndexerDefinitions: complete }; }
      else { await route.fallback(); return; }
      await route.fulfill({ contentType: "application/json", body: JSON.stringify({ data }) });
    });
    await page.goto("/settings/indexers");
    await page.getByRole("button", { name: "Download definitions to add sites", exact: true }).click();
    await page.getByRole("button", { name: "Download definitions", exact: true }).click();
    await expect(page.getByText("579 site definitions installed", { exact: true })).toBeVisible();
    await page.locator("#settings-tab-sites").click();
    await expect(page.getByRole("button", { name: "Add site", exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Add site", exact: true }).click();
    await expect(page.getByRole("button", { name: "Configure", exact: true })).toBeVisible();
  } finally { for (const key of Object.keys(graphqlOverrides)) delete graphqlOverrides[key]; Object.assign(graphqlOverrides, original); }
});
