import { expect, type Page } from "@playwright/test";
import { test, indexerInstances } from "./fixtures";

const definition = { categories: ["Movies", "Movies/HD", "TV"], description: "A synthetic site with a long description for checking narrow dialogs and wrapped actions.", language: "en-US", privacy: "public", loginSupported: false, digest: "digest", links: ["https://example.test/"], modes: ["search"], warnings: [], settings: [] };
const definitions = [
  { ...definition, id: "fresh-public", name: "Fresh Public" },
  { ...definition, id: "private-login", name: "Private Account", privacy: "private", loginSupported: true },
  { ...definition, id: "semi", name: "Semi Private", privacy: "semi-private" },
  { ...definition, id: "sukebei", name: "Sukebei" },
  ...Array.from({ length: 26 }, (_, i) => ({ ...definition, id: `catalog-${i}`, name: `Catalog site ${i + 1}` }))
];
const catalog = { revision: "a".repeat(40), totalCount: 30, offset: 0, limit: 30, items: definitions };
async function openCatalog(page: Page) {
  await page.locator(".settings-subpanel:not([hidden]) .settings-card-actions > button").first().click();
  return page.getByRole("dialog");
}

test.describe("site catalog usability", () => {
  test.use({ graphqlOverrides: { indexerDefinitions: catalog } });
  for (const scenario of [{ locale: "en", theme: "light" }, { locale: "zh-CN", theme: "dark" }, { locale: "ar-XB", theme: "dark" }]) {
    test(`${scenario.locale} ${scenario.theme} tall catalog stays inside a short viewport`, async ({ page }, info) => {
      const width = info.project.name === "mobile" ? 412 : 1440;
      await page.setViewportSize({ width, height: 560 });
      await page.addInitScript(value => { localStorage.setItem("moji:locale", value.locale === "ar-XB" ? "en" : value.locale); localStorage.setItem("moji:theme", value.theme); }, scenario);
      await page.goto(`/settings/indexers${scenario.locale === "ar-XB" ? "?__locale=ar-XB" : ""}`);
      const dialog = await openCatalog(page);
      await expect(dialog.locator("[data-definition-id]")).toHaveCount(30);
      const box = await dialog.boundingBox();
      expect(box).not.toBeNull();
      expect(box!.y).toBeGreaterThanOrEqual(0);
      expect(box!.y + box!.height).toBeLessThanOrEqual(560);
      expect(box!.x).toBeGreaterThanOrEqual(0);
      expect(box!.x + box!.width).toBeLessThanOrEqual(width);
      await expect(dialog.locator(".drawer__head")).toBeInViewport({ ratio: 1 });
      await page.evaluate(() => document.fonts.ready);
      await expect(page).toHaveScreenshot(`${scenario.locale}-${scenario.theme}-site-catalog.png`, { fullPage: true });
      await dialog.locator(".drawer-body").evaluate(element => { element.scrollTop = element.scrollHeight; });
      await expect(dialog.locator(".drawer__head")).toBeInViewport({ ratio: 1 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await page.keyboard.press("Escape");
      await expect(dialog).toHaveCount(0);
      await expect(page.locator(".settings-subpanel:not([hidden]) .settings-card-actions > button").first()).toBeFocused();
    });
  }

  test("public quick add persists enabled defaults once and updates the catalog status", async ({ page }) => {
    let snapshot = structuredClone(indexerInstances);
    const inputs: Record<string, unknown>[] = [];
    let releaseSave!: () => void;
    const saveGate = new Promise<void>(resolve => { releaseSave = resolve; });
    await page.route("**/graphql", async route => {
      const request = route.request().postDataJSON();
      if (request.operationName === "IndexerInstances") return route.fulfill({ json: { data: { indexerInstances: snapshot } } });
      if (request.operationName !== "AddBuiltinIndexer") return route.fallback();
      inputs.push(request.variables.input);
      await saveGate;
      const instance = { ...indexerInstances.items[0], id: "new-public", definitionId: request.variables.input.definitionId, name: request.variables.input.name || request.variables.input.definitionId, enabled: request.variables.input.enabled, settings: [] };
      snapshot = { ...snapshot, revision: "config-2", items: [...snapshot.items, instance], order: [...snapshot.order, instance.id] };
      return route.fulfill({ json: { data: { addBuiltinIndexer: { revision: snapshot.revision, instance } } } });
    });
    await page.goto("/settings/indexers");
    const dialog = await openCatalog(page);
    const row = dialog.locator('[data-definition-id="fresh-public"]');
    const add = row.getByRole("button", { name: "Add and enable", exact: true });
    await add.click();
    await expect(add).toBeDisabled();
    expect(inputs).toEqual([{ definitionId: "fresh-public", name: "Fresh Public", enabled: true }]);
    releaseSave();
    await expect(row).toContainText("Already added");
    await expect(row).toContainText("Enabled");
    await expect(add).toHaveCount(0);
    await expect(row.getByRole("button", { name: "Add another configuration" })).toBeEnabled();
    await expect(dialog.locator('[data-definition-id="private-login"]').getByRole("button", { name: "Add and enable" })).toHaveCount(0);
    await expect(dialog.locator('[data-definition-id="sukebei"]')).toContainText("Already added");
    await page.keyboard.press("Escape");
    await expect(page.locator('[data-indexer-id="new-public"]')).toContainText("Fresh Public");
    await expect(page.locator('[data-indexer-id="new-public"]').getByRole("switch")).toBeChecked();
    await page.reload();
    await expect(page.locator('[data-indexer-id="new-public"]')).toContainText("Fresh Public");
    await expect(page.locator('[data-indexer-id="new-public"]').getByRole("switch")).toBeChecked();
    expect(inputs).toHaveLength(1);
  });

  test("a rejected quick add stays retryable and does not claim the site was added", async ({ page }) => {
    await page.route("**/graphql", async route => {
      const request = route.request().postDataJSON();
      if (request.operationName !== "AddBuiltinIndexer") return route.fallback();
      await route.fulfill({ json: { data: null, errors: [{ message: "INDEXER_SETTINGS_INVALID", extensions: { code: "INDEXER_SETTINGS_INVALID" } }] } });
    });
    await page.goto("/settings/indexers");
    const dialog = await openCatalog(page);
    const row = dialog.locator('[data-definition-id="fresh-public"]');
    await row.getByRole("button", { name: "Add and enable" }).click();
    await expect(dialog.getByRole("alert")).toBeVisible();
    await expect(row.getByRole("button", { name: "Add and enable" })).toBeEnabled();
    await expect(row).not.toContainText("Already added");
    await row.getByRole("button", { name: "Configure", exact: true }).click();
    await expect(page.getByRole("dialog").getByLabel("Name", { exact: true })).toHaveValue("Fresh Public");
    await expect(page.getByRole("dialog").getByRole("button", { name: "Add", exact: true })).toBeEnabled();
  });
});

test("login availability follows declared flow rather than privacy and switches save state", async ({ page }) => {
  let snapshot = { ...indexerInstances, items: [
    { ...indexerInstances.items[0], id: "public-login", name: "Public with login", loginSupported: true },
    { ...indexerInstances.items[0], id: "private-no-login", name: "Private without login", privacy: "private", loginSupported: false }
  ] };
  const changes: unknown[] = [];
  await page.route("**/graphql", async route => {
    const request = route.request().postDataJSON();
    if (request.operationName === "IndexerInstances") return route.fulfill({ json: { data: { indexerInstances: snapshot } } });
    if (request.operationName !== "SetIndexerEnabled") return route.fallback();
    changes.push(request.variables);
    snapshot = { ...snapshot, revision: "config-2", items: snapshot.items.map(site => site.id === request.variables.id ? { ...site, enabled: request.variables.enabled } : site) };
    return route.fulfill({ json: { data: { setIndexerEnabled: snapshot } } });
  });
  await page.goto("/settings/indexers");
  await expect(page.locator('[data-indexer-id="public-login"]').getByRole("button", { name: "Log in to site" })).toBeVisible();
  const row = page.locator('[data-indexer-id="private-no-login"]');
  await expect(row.getByRole("button", { name: "Log in to site" })).toHaveCount(0);
  await expect(row.getByRole("switch")).toHaveCSS("opacity", "0");
  const track = await row.locator(".switch__track").boundingBox();
  expect(track?.width).toBe(38); expect(track?.height).toBe(22);
  await row.getByRole("switch").click();
  await expect(row.getByRole("switch")).not.toBeChecked();
  expect(changes).toEqual([{ id: "private-no-login", enabled: false, expectedRevision: "config-1" }]);
  await page.reload();
  await expect(row.getByRole("switch")).not.toBeChecked();
});

test("sorting starts from the visible handle with real mouse or touch and survives reload", async ({ page }, info) => {
  const base = indexerInstances.items[1];
  let snapshot = { ...indexerInstances, order: ["jackett/a", "jackett/b", "jackett/c"], items: ["a", "b", "c"].map(id => ({ ...base, id: `jackett/${id}`, name: id.toUpperCase() })) };
  const orders: string[][] = [];
  await page.route("**/graphql", async route => {
    const request = route.request().postDataJSON();
    if (request.operationName === "IndexerInstances") return route.fulfill({ json: { data: { indexerInstances: snapshot } } });
    if (request.operationName !== "ReorderIndexers") return route.fallback();
    const ids = request.variables.ids as string[];
    orders.push(ids);
    snapshot = { ...snapshot, revision: `config-${orders.length + 1}`, order: ids, items: ids.map(id => snapshot.items.find(item => item.id === id)!) };
    return route.fulfill({ json: { data: { reorderIndexers: snapshot } } });
  });
  await page.goto("/settings/indexers");
  await page.evaluate(() => {
    const log: string[] = [];
    (window as unknown as { dragEvents: string[] }).dragEvents = log;
    for (const name of ["dragstart", "pointercancel", "dragover", "drop", "dragend"]) document.addEventListener(name, event => {
      log.push(`${event.type} ${(event as PointerEvent).pointerType ?? ""} ${(event.target as Element).closest("[data-indexer-id]")?.getAttribute("data-indexer-id")} dragging=${Boolean(document.querySelector(".is-dragging"))}`);
    }, true);
  });
  const handle = await page.getByRole("button", { name: "Drag A to reorder", exact: true }).boundingBox();
  const target = await page.locator('[data-indexer-id="jackett/c"]').boundingBox();
  expect(handle).not.toBeNull(); expect(target).not.toBeNull();
  const x = handle!.x + handle!.width / 2, y = handle!.y + handle!.height / 2;
  if (info.project.name === "mobile") {
    const touch = await page.context().newCDPSession(page);
    await touch.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [{ x, y }] });
    await touch.send("Input.dispatchTouchEvent", { type: "touchMove", touchPoints: [{ x, y: target!.y + target!.height / 2 }] });
    await touch.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
    await touch.detach();
  } else {
    await page.mouse.move(x, y); await page.mouse.down();
    await page.mouse.move(x + 4, y + 4, { steps: 4 });
    await page.mouse.move(x, target!.y + target!.height / 2, { steps: 16 });
    await page.mouse.move(x, target!.y + target!.height / 2 + 4);
    await page.mouse.up();
  }
  await info.attach("drag-events.json", { body: JSON.stringify(await page.evaluate(() => (window as unknown as { dragEvents: string[] }).dragEvents)), contentType: "application/json" });
  await expect.poll(() => orders).toEqual([["jackett/b", "jackett/c", "jackett/a"]]);
  await page.reload();
  await expect(page.locator(".indexer-row").last()).toHaveAttribute("data-indexer-id", "jackett/a");
  await page.getByRole("button", { name: "Drag A to reorder", exact: true }).focus(); await page.keyboard.press("ArrowUp");
  await expect.poll(() => orders.length).toBe(2);
  expect(orders[1]).toEqual(["jackett/b", "jackett/a", "jackett/c"]);
});

test("remove icon opens a visible confirmation and cancellation preserves the source", async ({ page }) => {
  let snapshot = structuredClone(indexerInstances);
  let removes = 0;
  await page.route("**/graphql", async route => {
    const request = route.request().postDataJSON();
    if (request.operationName === "IndexerInstances") return route.fulfill({ json: { data: { indexerInstances: snapshot } } });
    if (request.operationName !== "RemoveBuiltinIndexer") return route.fallback();
    removes++;
    snapshot = { ...snapshot, revision: "config-2", items: snapshot.items.filter(site => site.id !== request.variables.id), order: snapshot.order.filter(id => id !== request.variables.id) };
    return route.fulfill({ json: { data: { removeBuiltinIndexer: snapshot } } });
  });
  await page.goto("/settings/indexers");
  const remove = page.locator('[data-indexer-id="builtin-one"]').getByRole("button", { name: "Remove site", exact: true });
  await remove.click();
  const dialog = page.getByRole("dialog", { name: "Remove site", exact: true });
  await expect(dialog.getByRole("heading", { name: "Remove site", exact: true })).toBeInViewport({ ratio: 1 });
  expect(removes).toBe(0);
  await page.keyboard.press("Escape");
  await expect(remove).toBeFocused();
  expect(removes).toBe(0);
  await remove.click();
  await dialog.getByRole("button", { name: "Remove site", exact: true }).click();
  await expect(dialog).toHaveCount(0);
  await expect(page.locator('[data-indexer-id="builtin-one"]')).toHaveCount(0);
  expect(removes).toBe(1);
});
