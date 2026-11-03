import { test } from "./fixtures";
import { expect } from "@playwright/test";

test.describe("cache policy autosave", () => {
  let policy = { revision: "config-1", enabled: true, ttlSeconds: 300, emptyTtlSeconds: 30, maxSizeMb: 32 };
  const status = () => ({ ...policy, entries: 0, usedBytes: 0, activeRequests: 0, activeWaiters: 0 });
  test.use({ graphqlOverrides: { indexerCacheStatus: status } });
  test.beforeEach(async ({ page }) => {
    policy = { revision: "config-1", enabled: true, ttlSeconds: 300, emptyTtlSeconds: 30, maxSizeMb: 32 };
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  });

  test("serializes edits across tab navigation with the acknowledged revision and explicit false", async ({ page }) => {
    const writes: Array<{ expectedRevision: string; input: Record<string, unknown> }> = [];
    let releaseFirst!: () => void;
    const firstGate = new Promise<void>(resolve => { releaseFirst = resolve; });
    await page.route("**/graphql", async route => {
      const body = route.request().postDataJSON();
      if (body.operationName !== "UpdateIndexerSearchCache") return route.fallback();
      writes.push(body.variables);
      if (writes.length === 1) await firstGate;
      expect(body.variables.expectedRevision).toBe(policy.revision);
      policy = { ...policy, ...body.variables.input, revision: `config-${writes.length + 1}` };
      await route.fulfill({ json: { data: { updateIndexerSearchCache: status() } } });
    });
    try {
      await page.goto("/settings/indexers?panel=cache");
      const retention = page.getByLabel("Successful result retention (seconds)", { exact: true });
      await retention.fill("600");
      await expect.poll(() => writes.length).toBe(1);
      await page.getByLabel("Empty result retention (seconds)", { exact: true }).fill("60");
      await page.getByRole("tab", { name: "Sites", exact: true }).click();
      await page.getByRole("tab", { name: "Network & cache", exact: true }).click();
      await expect(page.getByLabel("Empty result retention (seconds)", { exact: true })).toHaveValue("60");
      releaseFirst();
      await expect.poll(() => writes.length).toBe(2);
      await expect(page.locator(".settings-save-status")).toHaveCount(0);
      expect(writes.slice(0, 2)).toEqual([
        { expectedRevision: "config-1", input: { ttlSeconds: 600 } },
        { expectedRevision: "config-2", input: { emptyTtlSeconds: 60 } }
      ]);
      await page.getByLabel("Enable search cache", { exact: true }).uncheck();
      await expect.poll(() => writes.length).toBe(3);
      await expect(page.locator(".settings-save-status")).toHaveCount(0);
      expect(writes[2]).toEqual({ expectedRevision: "config-3", input: { enabled: false } });
      await page.reload();
      await expect(retention).toHaveValue("600");
      await expect(page.getByLabel("Empty result retention (seconds)", { exact: true })).toHaveValue("60");
      await expect(page.getByLabel("Enable search cache", { exact: true })).not.toBeChecked();
    } finally { releaseFirst(); }
  });

  test("retains a conflicted draft until the user confirms loading the current policy", async ({ page }) => {
    await page.route("**/graphql", async route => {
      const body = route.request().postDataJSON();
      if (body.operationName !== "UpdateIndexerSearchCache") return route.fallback();
      policy = { ...policy, revision: "config-2", enabled: false, ttlSeconds: 700 };
      await route.fulfill({ json: { errors: [{ message: "The settings changed in another window.", extensions: { code: "CONFIG_REVISION_CONFLICT" } }] } });
    });
    await page.goto("/settings/indexers?panel=cache");
    const retention = page.getByLabel("Successful result retention (seconds)", { exact: true });
    await retention.fill("900");
    const load = page.getByRole("button", { name: "Load current settings", exact: true });
    await expect(load).toBeVisible();
    await expect(retention).toHaveValue("900");
    await load.click();
    const dialog = page.getByRole("dialog", { name: "Load current settings", exact: true });
    await expect(dialog.getByText("Discard unsaved changes on this page and load the current server settings.", { exact: true })).toBeVisible();
    await dialog.getByRole("button", { name: "Close", exact: true }).last().click();
    await expect(retention).toHaveValue("900");
    await load.click();
    await dialog.getByRole("button", { name: "Load current settings", exact: true }).click();
    await expect(dialog).toHaveCount(0);
    await expect(retention).toHaveValue("700");
    await expect(page.getByLabel("Enable search cache", { exact: true })).not.toBeChecked();
    await expect(load).toHaveCount(0);
  });
});
