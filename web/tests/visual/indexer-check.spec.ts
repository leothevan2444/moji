import { expect, type Page } from "@playwright/test";
import { test, indexerInstances } from "./fixtures";


type CheckInput = { instanceId: string; clientId: string; requestId: string; connectionOnly: boolean; search: boolean; download: boolean };
async function mockActivity(page: Page) {
  const entries = new Map<string, CheckInput & { state: "QUEUED" | "RUNNING" }>();
  const watchers = new Map<string, (items: { requestId: string; instanceId: string; state: string }[]) => void>();
  const publish = () => {
    for (const [clientId, send] of watchers) send([...entries.values()].filter(item => item.clientId === clientId).map(({ requestId, instanceId, state }) => ({ requestId, instanceId, state })));
  };
  await page.routeWebSocket("**/graphql", socket => socket.onMessage(raw => {
    const message = JSON.parse(String(raw));
    if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
    if (message.type === "subscribe" && message.payload.query.includes("subscription IndexerCheckActivity")) {
      watchers.set(message.payload.variables.clientId, items => socket.send(JSON.stringify({ id: message.id, type: "next", payload: { data: { indexerCheckActivity: items } } })));
      publish();
    }
  }));
  return {
    set(input: CheckInput, state: "QUEUED" | "RUNNING") { entries.set(input.requestId, { ...input, state }); publish(); },
    remove(input: CheckInput) { entries.delete(input.requestId); publish(); }
  };
}

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
  expect(inputs).toHaveLength(1);
  expect(inputs[0]).toMatchObject({ instanceId: "builtin-one", expectedRevision: indexerInstances.revision, connectionOnly: true, search: false, download: false });
  expect(inputs[0].expectedConnectionRevision).toBe(indexerInstances.items[0].connectionRevision);
  expect(inputs[0].requestId).toMatch(/^[a-f0-9]{32}$/);
  expect(inputs[0].clientId).toMatch(/^[a-f0-9]{32}$/);
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
    indexerInstances: { ...indexerInstances, items: [...indexerInstances.items, ...Array.from({ length: 4 }, (_, i) => ({ ...indexerInstances.items[0], id: `builtin-${i + 2}`, name: `Extra site ${i + 1}`, enabled: false }))] }
  } });
  test("the backend controls queue states while configuration remains available", async ({ page }) => {
    const inputs: CheckInput[] = [];
    const release: (() => void)[] = [];
    const activity = await mockActivity(page);
    await page.route("**/graphql", async route => {
      const body = route.request().postDataJSON();
      if (body.operationName !== "TestIndexer") return route.fallback();
      const input = body.variables.input as CheckInput;
      inputs.push(input);
      activity.set(input, inputs.length <= 4 ? "RUNNING" : "QUEUED");
      await new Promise<void>(resolve => release.push(resolve));
      activity.remove(input);
      await route.fulfill({ json: { data: { testIndexer: report(input.instanceId, input.instanceId === "builtin-one") } } });
    });
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto("/settings/indexers");
    await page.getByRole("button", { name: "Test all", exact: true }).click();
    await expect.poll(() => inputs.length).toBe(6);
    expect(inputs.map(input => input.instanceId).sort()).toEqual(["builtin-one", "jackett/onejav", "builtin-2", "builtin-3", "builtin-4", "builtin-5"].sort());
    await expect(page.getByRole("button", { name: "Queued", exact: true })).toHaveCount(2);
    await expect(page.locator(".indexer-row").getByRole("button", { name: "Testing", exact: true })).toHaveCount(4);
    await expect(page.locator(".settings-card-actions").getByRole("button", { name: "Testing", exact: true })).toBeDisabled();
    await expect(page.getByRole("button", { name: "Add site", exact: true })).toBeEnabled();
    await expect(page.getByRole("button", { name: "Refresh Jackett sites", exact: true })).toBeEnabled();
    const row = page.locator('[data-indexer-id="builtin-one"]');
    await expect(row.getByRole("switch")).toBeEnabled();
    await expect(row.getByRole("button", { name: "Remove site", exact: true })).toBeEnabled();
    await expect(row.locator(".indexer-drag")).toHaveAttribute("draggable", "true");
    await row.getByRole("button", { name: "Configure", exact: true }).click();
    const editor = page.getByRole("dialog", { name: "Configure site" });
    await expect(editor.getByRole("textbox", { name: "Name", exact: true })).toBeEnabled();
    await expect(editor.getByRole("button", { name: "Test", exact: true })).toBeDisabled();
    await editor.getByRole("button", { name: "Close", exact: true }).click();
    expect(inputs).toHaveLength(6);
    await expect(page).toHaveScreenshot("backend-connection-queue.png", { fullPage: true });
    for (const finish of release) finish();
    await expect(page.getByRole("button", { name: "Test all", exact: true })).toBeEnabled();
    await expect(page.getByRole("button", { name: "Test passed", exact: true })).toHaveCount(5);
    await expect(page.getByRole("button", { name: "Test failed", exact: true })).toHaveCount(1);
    for (const input of inputs) expect(input).toMatchObject({ connectionOnly: true, search: false, download: false });
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
  expect(inputs).toHaveLength(1);
  expect(inputs[0]).toMatchObject({ instanceId: "builtin-one", expectedRevision: indexerInstances.revision, draft: { name: "Unsaved name" }, connectionOnly: true, search: false, download: false });
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
  await expect(editor.getByRole("button", { name: "Awaiting response", exact: true })).toBeDisabled();
  await editor.getByRole("textbox", { name: "Name", exact: true }).fill("A newer draft");
  release();
  await expect(editor.getByRole("button", { name: "Test", exact: true })).toBeEnabled();
  await expect(editor.getByRole("button", { name: "Test passed", exact: true })).toHaveCount(0);
});

test("single checks remain independent across ordering and unrelated account edits", async ({ page }) => {
  let snapshot = structuredClone(indexerInstances);
  const inputs: CheckInput[] = [];
  const finish = new Map<string, () => void>();
  const activity = await mockActivity(page);
  await page.route("**/graphql", async route => {
    const body = route.request().postDataJSON();
    if (body.operationName === "IndexerInstances") return route.fulfill({ json: { data: { indexerInstances: snapshot } } });
    if (body.operationName === "ReorderIndexers") {
      snapshot = { ...snapshot, revision: "reordered", order: body.variables.ids, items: body.variables.ids.map((id: string) => snapshot.items.find(item => item.id === id)!) };
      return route.fulfill({ json: { data: { reorderIndexers: snapshot } } });
    }
    if (body.operationName === "UpdateBuiltinIndexer") {
      snapshot = { ...snapshot, revision: "changed-one", items: snapshot.items.map(item => item.id === body.variables.id ? { ...item, name: body.variables.input.name, connectionRevision: "changed-account" } : item) };
      return route.fulfill({ json: { data: { updateBuiltinIndexer: { revision: snapshot.revision, instance: snapshot.items.find(item => item.id === body.variables.id) } } } });
    }
    if (body.operationName !== "TestIndexer") return route.fallback();
    const input = body.variables.input as CheckInput;
    inputs.push(input); activity.set(input, "RUNNING");
    await new Promise<void>(resolve => finish.set(input.instanceId, resolve));
    activity.remove(input);
    await route.fulfill({ json: { data: { testIndexer: report(input.instanceId) } } }).catch(() => {});
  });
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.goto("/settings/indexers");
  const first = page.locator('[data-indexer-id="builtin-one"]'), second = page.locator('[data-indexer-id="jackett/onejav"]');
  await first.getByRole("button", { name: "Test", exact: true }).click();
  await expect(second.getByRole("button", { name: "Test", exact: true })).toBeEnabled();
  await expect(page.getByRole("button", { name: "Test all", exact: true })).toBeEnabled();
  await second.getByRole("button", { name: "Test", exact: true }).click();
  await expect.poll(() => inputs.length).toBe(2);
  await first.locator(".indexer-drag").press("ArrowDown");
  await expect(first.getByRole("button", { name: "Testing", exact: true })).toBeDisabled();
  await expect(second.getByRole("button", { name: "Testing", exact: true })).toBeDisabled();
  await first.getByRole("button", { name: "Configure", exact: true }).click();
  const editor = page.getByRole("dialog", { name: "Configure site" });
  await editor.getByRole("textbox", { name: "Name", exact: true }).fill("Changed account");
  await editor.getByRole("button", { name: "Save", exact: true }).click();
  await expect(editor).toHaveCount(0);
  await expect(first.getByRole("button", { name: "Test", exact: true })).toBeEnabled();
  await expect(second.getByRole("button", { name: "Testing", exact: true })).toBeDisabled();
  finish.get("builtin-one")!(); finish.get("jackett/onejav")!();
  await expect(second.getByRole("button", { name: "Test passed", exact: true })).toBeVisible();
  await expect(first.getByRole("button", { name: "Test", exact: true })).toBeEnabled();
  await expect(first.getByRole("button", { name: "Test passed", exact: true })).toHaveCount(0);
  expect(inputs).toHaveLength(2);
});
