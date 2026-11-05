import { expect } from "@playwright/test";
import { test, indexerInstances } from "./fixtures";

const report = (id: string, failed = false) => ({
  id: "connection-check", instanceId: id, kind: id.startsWith("jackett/") ? "JACKETT" : "BUILTIN",
  configRevision: indexerInstances.revision, ruleRevision: "", startedAt: "2026-01-15T12:00:00Z", finishedAt: "2026-01-15T12:00:01Z",
  results: 0, partial: false, truncated: false,
  stages: [{ name: "CONNECTION", state: failed ? "FAILED" : "PASSED", errorCode: failed ? "INDEXER_TIMEOUT" : "", elapsedMs: 1 }]
});
test("clicking Test checks connectivity immediately and displays retryable outcomes", async ({ page }) => {
  const inputs: Record<string, unknown>[] = [];
  let failed = true;
  await page.route("**/graphql", async route => {
    const body = route.request().postDataJSON();
    if (body.operationName !== "TestIndexer") return route.fallback();
    inputs.push(body.variables.input);
    const response = report(body.variables.input.instanceId, failed);
    if (failed) response.stages[0].errorCode = "SYNTHETIC_UNKNOWN_ERROR";
    await route.fulfill({ json: { data: { testIndexer: response } } });
  });
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/settings/indexers");
  const row = page.locator('[data-indexer-id="builtin-one"]');
  await row.getByRole("button", { name: "Test", exact: true }).click();
  await expect(row.getByRole("button", { name: "Test failed", exact: true })).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(inputs).toEqual([{ instanceId: "builtin-one", expectedRevision: indexerInstances.revision, connectionOnly: true, search: false, download: false }]);
  failed = false;
  await row.getByRole("button", { name: "Test failed", exact: true }).click();
  await expect(row.getByRole("button", { name: "Test passed", exact: true })).toBeVisible();
  await expect(row.locator(".indexer-history, .indexer-error")).toHaveCount(0);
  for (const button of await row.locator(".indexer-row__actions .icon-button").all()) {
    const box = (await button.boundingBox())!;
    expect(box.width).toBe(26); expect(box.height).toBe(26);
  }
  expect((await row.getByRole("button", { name: "Test passed", exact: true }).boundingBox())!.height).toBe(26);
  const bounds = (await row.boundingBox())!, source = (await row.locator(".indexer-source").boundingBox())!;
  expect(Math.abs(bounds.x + bounds.width - source.x - source.width)).toBeLessThanOrEqual(1);
  expect(source.y - bounds.y).toBeLessThanOrEqual(8);
  await expect(page).toHaveScreenshot("inline-connection-result.png", { fullPage: true });
  await page.reload();
  await expect(row.getByRole("button", { name: "Test", exact: true })).toBeVisible();
});

test.describe("test all", () => {
  test.use({ graphqlOverrides: {
    indexerInstances: { ...indexerInstances, items: [...indexerInstances.items, { ...indexerInstances.items[0], id: "builtin-two", name: "Second site", enabled: false }] }
  } });
  test("caps concurrency at two and retains each result after partial failure", async ({ page }) => {
    const inputs: Record<string, unknown>[] = [];
    const release: (() => void)[] = [];
    await page.route("**/graphql", async route => {
      const body = route.request().postDataJSON();
      if (body.operationName !== "TestIndexer") return route.fallback();
      inputs.push(body.variables.input);
      await new Promise<void>(resolve => release.push(resolve));
      await route.fulfill({ json: { data: { testIndexer: report(body.variables.input.instanceId, body.variables.input.instanceId === "builtin-one") } } });
    });
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto("/settings/indexers");
    await page.getByRole("button", { name: "Test all", exact: true }).click();
    await expect.poll(() => release.length).toBe(2);
    expect(inputs).toHaveLength(2);
    await expect(page.locator(".settings-card-actions").getByRole("button", { name: "Testing", exact: true })).toBeDisabled();
    release[0]();
    await expect.poll(() => release.length).toBe(3);
    release[1](); release[2]();
    await expect(page.getByRole("button", { name: "Test all", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Test passed", exact: true })).toHaveCount(2);
    await expect(page.getByRole("button", { name: "Test failed", exact: true })).toHaveCount(1);
    for (const input of inputs) expect(input).toMatchObject({ connectionOnly: true, search: false, download: false });
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page).toHaveScreenshot("all-connection-results.png", { fullPage: true });
  });
});

test("draft connectivity checks do not save and changing the draft clears results", async ({ page }) => {
  const inputs: Record<string, unknown>[] = [];
  const writes: string[] = [];
  await page.route("**/graphql", async route => {
    const body = route.request().postDataJSON();
    if (body.operationName === "TestIndexer") inputs.push(body.variables.input);
    if (["AddBuiltinIndexer", "UpdateBuiltinIndexer", "QueueTorrentCandidate"].includes(body.operationName)) writes.push(body.operationName);
    await route.fallback();
  });
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/settings/indexers");
  await page.locator('[data-indexer-id="builtin-one"]').getByRole("button", { name: "Configure", exact: true }).click();
  const editor = page.getByRole("dialog", { name: "Configure site" });
  await editor.getByRole("textbox", { name: "Name", exact: true }).fill("Unsaved name");
  await editor.getByRole("button", { name: "Test", exact: true }).click();
  await expect(editor.getByRole("button", { name: "Test passed", exact: true })).toBeVisible();
  expect(inputs).toEqual([{ instanceId: "builtin-one", expectedRevision: indexerInstances.revision, draft: { name: "Unsaved name" }, connectionOnly: true, search: false, download: false }]);
  expect(writes).toEqual([]);
  await editor.getByRole("textbox", { name: "Name", exact: true }).fill("Changed again");
  await expect(editor.getByRole("button", { name: "Test", exact: true })).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(1);
});

test("editing a pending draft rejects its late connection result", async ({ page }) => {
  let release!: () => void;
  let received = false;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/graphql", async route => {
    const body = route.request().postDataJSON();
    if (body.operationName !== "TestIndexer") return route.fallback();
    received = true;
    await gate;
    await route.fulfill({ json: { data: { testIndexer: report("builtin-one") } } }).catch(() => {});
  });
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/settings/indexers");
  await page.locator('[data-indexer-id="builtin-one"]').getByRole("button", { name: "Configure", exact: true }).click();
  const editor = page.getByRole("dialog", { name: "Configure site" });
  await editor.getByRole("button", { name: "Test", exact: true }).click();
  await expect.poll(() => received).toBe(true);
  await expect(editor.getByRole("button", { name: "Testing", exact: true })).toBeDisabled();
  await editor.getByRole("textbox", { name: "Name", exact: true }).fill("A newer draft");
  release();
  await expect(editor.getByRole("button", { name: "Test", exact: true })).toBeEnabled();
  await expect(editor.getByRole("button", { name: "Test passed", exact: true })).toHaveCount(0);
});
