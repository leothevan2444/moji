import { expect } from "@playwright/test";
import { test, indexerLibrary } from "./fixtures";

const failure = { id: "synthetic-rule-job", kind: "UPDATE", state: "FAILED", targetRevision: "b".repeat(40), expectedActiveRevision: "a".repeat(40), downloadedBytes: 0, errorCode: "DEFINITION_SOURCE_UNAVAILABLE", correlationId: "synthetic-rule-failure-correlation", startedAt: "2026-01-15T12:00:00Z", finishedAt: "2026-01-15T12:00:01Z", revision: 3 };
const overview = { revision: "config-1", total: 2, builtinEnabled: 1, builtinUnverified: 1, builtinSearchSucceeded: 0, healthErrorCode: "", attention: 1, issues: [], definitionJob: failure, definitionErrorCode: "", jackett: { sites: 1, stale: false, updatedAt: null, errorCode: "" } };

test("Home restores a failed rule job after reload and opens definitions without starting work", async ({ page }) => {
  const operations: string[] = [];
  page.on("request", request => { if (request.url().endsWith("/graphql") && request.method() === "POST") operations.push(request.postDataJSON().operationName); });
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.route("**/graphql", async route => {
    const name = route.request().postDataJSON().operationName;
    if (name === "HomeIndexerSummary") return route.fulfill({ json: { data: { homeIndexerSummary: overview } } });
    if (name === "IndexerDefinitionStatus") return route.fulfill({ json: { data: { indexerDefinitionStatus: { ...indexerLibrary, job: failure } } } });
    return route.fallback();
  });
  await page.goto("/");
  const section = page.getByRole("region", { name: "Indexers", exact: true });
  await expect(section.getByText("Definition update failed", { exact: true })).toBeVisible();
  await page.reload();
  await expect(section.getByText("Definition update failed", { exact: true })).toBeVisible();
  await section.getByRole("link", { name: "View definitions", exact: true }).click();
  await expect(page).toHaveURL(/\/settings\/indexers\?panel=definitions/);
  await expect(page.getByText("Definition update failed", { exact: true })).toBeVisible();
  expect(operations.filter(name => /^(CheckIndexerDefinitionUpdate|InstallIndexerDefinitions|UpdateIndexerDefinitions|RepairIndexerDefinitions|RollbackIndexerDefinitions)$/.test(name))).toEqual([]);
});

test("definition notifications reread current state without polling and supersede an old failure", async ({ page }) => {
  let current: Record<string, unknown> = { ...overview, definitionJob: null, attention: 0 };
  let reads = 0;
  let emit = () => {};
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.route("**/graphql", async route => {
    if (route.request().postDataJSON().operationName === "HomeIndexerSummary") { reads++; return route.fulfill({ json: { data: { homeIndexerSummary: current } } }); }
    return route.fallback();
  });
  await page.routeWebSocket("**/graphql", socket => {
    const ids = new Set<string>(); let jobRevision = 0;
    emit = () => { jobRevision++; for (const id of ids) socket.send(JSON.stringify({ id, type: "next", payload: { data: { indexerDefinitionEvents: { streamId: "definitions", revision: jobRevision, job: { id: failure.id, revision: jobRevision } } } } })); };
    socket.onMessage(raw => {
      const message = JSON.parse(String(raw));
      if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
      if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
      if (message.type === "subscribe" && message.payload?.query?.includes("subscription HomeIndexerDefinitionEvents")) { ids.add(message.id); emit(); }
      if (message.type === "complete") ids.delete(message.id);
    });
  });
  await page.goto("/");
  const section = page.getByRole("region", { name: "Indexers", exact: true });
  await expect(section.getByRole("link", { name: "Manage indexers", exact: true })).toBeVisible();
  await expect.poll(() => reads).toBeGreaterThan(1);
  const initial = reads; await page.waitForTimeout(750); expect(reads).toBe(initial);
  current = { ...overview };
  emit();
  await expect(section.getByText("Definition update failed", { exact: true })).toBeVisible();
  current = { ...overview, definitionJob: null, attention: 0 };
  emit();
  await expect(section.getByText("Definition update failed", { exact: true })).toHaveCount(0);
  const settled = reads; await page.waitForTimeout(750); expect(reads).toBe(settled);
});

for (const [locale, theme] of [["zh-CN", "light"], ["en", "dark"], ["ar-XB", "dark"]] as const) {
  test(`${locale} persisted rule failure fits Home`, async ({ page }) => {
    await page.addInitScript(value => { localStorage.setItem("moji:locale", value.locale); localStorage.setItem("moji:theme", value.theme); }, { locale, theme });
    await page.route("**/graphql", async route => {
      if (route.request().postDataJSON().operationName === "HomeIndexerSummary") return route.fulfill({ json: { data: { homeIndexerSummary: { ...overview, definitionJob: { ...failure, state: "INTERRUPTED", errorCode: "DEFINITION_UPDATE_INTERRUPTED", correlationId: "synthetic-".repeat(30) } } } } });
      return route.fallback();
    });
    await page.goto("/" + (locale === "ar-XB" ? "?__locale=ar-XB" : ""));
    await expect(page.locator("html")).toHaveAttribute("dir", locale === "ar-XB" ? "rtl" : "ltr");
    const section = page.getByRole("region", { name: locale === "zh-CN" ? "索引器" : "Indexers", exact: true });
    await expect(section.locator(".home-indexer-definition-job")).toBeVisible();
    await section.locator(".home-indexer-definition-job summary").click();
    expect(await section.evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
    await expect(section).toHaveScreenshot(`${locale}-home-rule-failure.png`);
  });
}

test("job status read failure keeps site overview and retries only the local snapshot", async ({ page }) => {
  let unavailable = true; const operations: string[] = [];
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.route("**/graphql", async route => {
    const name = route.request().postDataJSON().operationName; operations.push(name);
    if (name === "HomeIndexerSummary") return route.fulfill({ json: { data: { homeIndexerSummary: { ...overview, definitionJob: null, definitionErrorCode: unavailable ? "DEFINITION_LIBRARY_UNAVAILABLE" : "", attention: unavailable ? 1 : 0 } } } });
    return route.fallback();
  });
  await page.goto("/");
  const section = page.getByRole("region", { name: "Indexers", exact: true });
  await expect(section.getByText("Definition job status is unavailable. Try again.", { exact: true })).toBeVisible();
  await expect(section.getByText("Jackett", { exact: true })).toBeVisible();
  unavailable = false;
  await section.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(section.getByText("Definition job status is unavailable. Try again.", { exact: true })).toHaveCount(0);
  expect(operations.some(name => /^(CheckIndexerDefinitionUpdate|InstallIndexerDefinitions|UpdateIndexerDefinitions)$/.test(name))).toBe(false);
});

test("subscription reconnect recovers a failed job missed while disconnected", async ({ page }) => {
  let current: Record<string, unknown> = { ...overview, definitionJob: null, attention: 0 };
  let disconnect = () => {}; let connections = 0;
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.route("**/graphql", async route => {
    if (route.request().postDataJSON().operationName === "HomeIndexerSummary") return route.fulfill({ json: { data: { homeIndexerSummary: current } } });
    return route.fallback();
  });
  await page.routeWebSocket("**/graphql", socket => {
    connections++; disconnect = () => socket.close({ code: 1012, reason: "synthetic restart" });
    socket.onMessage(raw => {
      const message = JSON.parse(String(raw));
      if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
      if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
      if (message.type === "subscribe" && message.payload?.query?.includes("subscription HomeIndexerDefinitionEvents")) socket.send(JSON.stringify({ id: message.id, type: "next", payload: { data: { indexerDefinitionEvents: { streamId: `definitions-${connections}`, revision: 1, job: null } } } }));
    });
  });
  await page.goto("/");
  const section = page.getByRole("region", { name: "Indexers", exact: true });
  await expect(section.getByRole("link", { name: "Manage indexers", exact: true })).toBeVisible();
  await expect.poll(() => connections).toBe(1);
  current = { ...overview };
  disconnect();
  await expect.poll(() => connections, { timeout: 15000 }).toBeGreaterThan(1);
  await expect(section.getByText("Definition update failed", { exact: true })).toBeVisible();
});
