import { expect } from "@playwright/test";
import { test } from "./fixtures";

test.describe("network profile management", () => {
  let revision = 1;
  let requestTimeoutSeconds = 30;
  let challengeTimeoutSeconds = 90;
  const profiles = new Map<string, { proxyConfigured: boolean; flareSolverrConfigured: boolean; usedBy: number }>();
  const calls: Array<{ name: string; variables: Record<string, unknown> }> = [];
  const status = () => ({ revision: `config-${revision}`, requestTimeoutSeconds, siteTimeoutSeconds: 120, challengeTimeoutSeconds, profiles: [...profiles].map(([id, profile]) => ({ id, ...profile })) });
  test.use({ graphqlOverrides: { indexerNetworkSettings: status } });
  test.beforeEach(async ({ page }) => {
    revision = 1; requestTimeoutSeconds = 30; challengeTimeoutSeconds = 90; calls.length = 0; profiles.clear();
    profiles.set("direct", { proxyConfigured: false, flareSolverrConfigured: false, usedBy: 0 }); profiles.set("private", { proxyConfigured: true, flareSolverrConfigured: false, usedBy: 1 }); profiles.set("spare", { proxyConfigured: true, flareSolverrConfigured: false, usedBy: 0 });
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.route("**/graphql", async route => {
      const { operationName: name, variables } = route.request().postDataJSON();
      const fields: Record<string, string> = { AddIndexerNetworkProfile: "addIndexerNetworkProfile", UpdateIndexerNetworkProfile: "updateIndexerNetworkProfile", RemoveIndexerNetworkProfile: "removeIndexerNetworkProfile", UpdateIndexerNetworkTimeouts: "updateIndexerNetworkTimeouts" };
      if (!fields[name]) return route.fallback();
      calls.push({ name, variables });
      if (variables.expectedRevision !== `config-${revision}`) return route.fulfill({ json: { errors: [{ message: "Settings changed.", extensions: { code: "CONFIG_REVISION_CONFLICT" } }] } });
      if (name === "AddIndexerNetworkProfile") profiles.set(variables.id, { proxyConfigured: Boolean(variables.input.proxyUrl), flareSolverrConfigured: Boolean(variables.input.flareSolverrUrl), usedBy: 0 });
      if (name === "UpdateIndexerNetworkProfile") { const profile = profiles.get(variables.id)!; if (variables.input.clearProxy) profile.proxyConfigured = false; else if (variables.input.proxyUrl) profile.proxyConfigured = true; if (variables.input.clearFlareSolverr) profile.flareSolverrConfigured = false; else if (variables.input.flareSolverrUrl) profile.flareSolverrConfigured = true; }
      if (name === "RemoveIndexerNetworkProfile") profiles.delete(variables.id);
      if (name === "UpdateIndexerNetworkTimeouts") { requestTimeoutSeconds = variables.input.requestTimeoutSeconds ?? requestTimeoutSeconds; challengeTimeoutSeconds = variables.input.challengeTimeoutSeconds ?? challengeTimeoutSeconds; }
      revision++;
      await route.fulfill({ json: { data: { [fields[name]]: status() } } });
    });
  });
  test("adds masked credentials, protects unsaved edits, clears explicitly and confirms removal", async ({ page }) => {
    await page.goto("/settings/indexers?panel=cache");
    const network = page.locator("article").filter({ has: page.getByRole("heading", { name: "Network profiles", exact: true }) });
    await expect(network.getByRole("button", { name: "Add network profile", exact: true })).toBeVisible();
    await network.scrollIntoViewIfNeeded();
    await expect(network.getByRole("button", { name: "Remove private network profile", exact: true })).toBeDisabled();
    await expect(network).toHaveScreenshot("network-profiles.png");
    await network.getByRole("button", { name: "Add network profile", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "Add network profile", exact: true });
    await dialog.getByLabel("Profile name", { exact: true }).fill("new-proxy");
    await dialog.getByLabel("Proxy URL", { exact: true }).fill("http://synthetic-user:synthetic-password@proxy.example:8080");
    await expect(dialog.getByLabel("Proxy URL", { exact: true })).toHaveAttribute("type", "password");
    await expect(dialog.getByRole("button", { name: "Save", exact: true })).toBeInViewport();
    await expect(page).toHaveScreenshot("network-profile-editor.png", { fullPage: true });
    await dialog.getByRole("button", { name: "Close", exact: true }).click();
    const discard = page.getByRole("dialog", { name: "Discard unsaved changes?", exact: true });
    await discard.getByRole("button", { name: "Keep editing", exact: true }).click();
    await dialog.getByRole("button", { name: "Save", exact: true }).click();
    await expect(dialog).toHaveCount(0);
    expect(calls[0]).toEqual({ name: "AddIndexerNetworkProfile", variables: { id: "new-proxy", expectedRevision: "config-1", input: { proxyUrl: "http://synthetic-user:synthetic-password@proxy.example:8080" } } });
    await network.getByRole("button", { name: "Edit new-proxy network profile", exact: true }).click();
    const edit = page.getByRole("dialog", { name: "Edit network profile", exact: true });
    await expect(edit.getByLabel("Proxy URL", { exact: true })).toHaveValue("");
    await expect(edit.getByLabel("Proxy URL", { exact: true })).toHaveAttribute("placeholder", "Configured; leave blank to keep");
    await edit.getByLabel("Clear proxy URL", { exact: true }).check();
    await edit.getByRole("button", { name: "Save", exact: true }).click();
    await expect(edit).toHaveCount(0);
    expect(calls[1].variables).toEqual({ id: "new-proxy", expectedRevision: "config-2", input: { clearProxy: true } });
    await network.getByRole("button", { name: "Remove new-proxy network profile", exact: true }).click();
    const remove = page.getByRole("dialog", { name: "Remove network profile", exact: true });
    expect(profiles.has("new-proxy")).toBe(true);
    await remove.getByRole("button", { name: "Remove network profile", exact: true }).click();
    await expect(remove).toHaveCount(0);
    await expect(network.getByRole("button", { name: "Edit new-proxy network profile", exact: true })).toHaveCount(0);
    await network.getByLabel("Request timeout (seconds)", { exact: true }).fill("45");
    await expect.poll(() => calls.length).toBe(4);
    await expect(network.locator(".settings-save-status")).toHaveCount(0);
    expect(calls[3].variables).toEqual({ expectedRevision: "config-4", input: { requestTimeoutSeconds: 45 } });
    await page.reload();
    await expect(page.getByLabel("Request timeout (seconds)", { exact: true })).toHaveValue("45");
  });
  test("solver-only profile keeps masked URL, saves timeout and clears explicitly", async ({ page }) => {
    await page.goto("/settings/indexers?panel=cache");
    await page.getByRole("button", { name: "Add network profile", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "Add network profile", exact: true });
    await dialog.getByLabel("Profile name", { exact: true }).fill("browser");
    await expect(dialog.getByRole("button", { name: "Save", exact: true })).toBeDisabled();
    await dialog.getByLabel("FlareSolverr URL", { exact: true }).fill("http://solver.example:8191");
    await expect(dialog.getByLabel("FlareSolverr URL", { exact: true })).toHaveAttribute("type", "password");
    await dialog.getByRole("button", { name: "Save", exact: true }).click();
    await expect(dialog).toHaveCount(0);
    expect(calls[0].variables).toEqual({ id: "browser", expectedRevision: "config-1", input: { flareSolverrUrl: "http://solver.example:8191" } });
    await page.getByRole("button", { name: "Edit browser network profile", exact: true }).click();
    const edit = page.getByRole("dialog", { name: "Edit network profile", exact: true });
    await expect(edit.getByLabel("FlareSolverr URL", { exact: true })).toHaveValue("");
    await expect(edit.getByLabel("FlareSolverr URL", { exact: true })).toHaveAttribute("placeholder", "Configured; leave blank to keep");
    await edit.getByLabel("Proxy URL", { exact: true }).fill("http://proxy.example:8080");
    await edit.getByRole("button", { name: "Save", exact: true }).click();
    await expect(edit).toHaveCount(0);
    expect(calls[1].variables).toEqual({ id: "browser", expectedRevision: "config-2", input: { proxyUrl: "http://proxy.example:8080" } });
    expect(profiles.get("browser")?.flareSolverrConfigured).toBe(true);
    await expect(page.locator(".indexer-network-profiles li").filter({ has: page.getByText("browser", { exact: true }) })).toContainText("Browser verification configured");
    await page.getByLabel("Browser verification timeout (seconds)", { exact: true }).fill("150");
    await expect.poll(() => calls.length).toBe(3);
    await expect(page.locator(".settings-save-status")).toHaveCount(0);
    expect(calls[2].variables).toEqual({ expectedRevision: "config-3", input: { challengeTimeoutSeconds: 150 } });
    await page.reload();
    await expect(page.getByLabel("Browser verification timeout (seconds)", { exact: true })).toHaveValue("150");
    await page.getByRole("button", { name: "Edit browser network profile", exact: true }).click();
    await edit.getByLabel("Clear FlareSolverr URL", { exact: true }).check();
    await expect(edit.getByLabel("FlareSolverr URL", { exact: true })).toBeDisabled();
    await edit.getByRole("button", { name: "Save", exact: true }).click();
    await expect(edit).toHaveCount(0);
    expect(calls[3].variables).toEqual({ id: "browser", expectedRevision: "config-4", input: { clearFlareSolverr: true } });
    expect(profiles.get("browser")).toEqual({ proxyConfigured: true, flareSolverrConfigured: false, usedBy: 0 });
  });
  test("background refresh cannot promote an open credential draft onto another configuration version", async ({ page }) => {
    let notify = () => {};
    await page.routeWebSocket("**/graphql", socket => {
      const ids = new Set<string>(); let sequence = 0;
      const send = (kind: string) => { for (const id of ids) socket.send(JSON.stringify({ id, type: "next", payload: { data: { indexerEvents: { streamId: "network-test", sequence, kind } } } })); };
      notify = () => { sequence++; send("CONFIGURATION"); };
      socket.onMessage(raw => {
        const message = JSON.parse(String(raw));
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
        if (message.type === "subscribe" && message.payload.query.includes("subscription IndexerEvents")) { ids.add(message.id); send("READY"); }
        if (message.type === "complete") ids.delete(message.id);
      });
    });
    await page.goto("/settings/indexers?panel=cache");
    await page.getByRole("button", { name: "Edit spare network profile", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "Edit network profile", exact: true });
    const proxy = dialog.getByLabel("Proxy URL", { exact: true });
    await proxy.fill("http://synthetic-new-secret@proxy.example:8080");
    const solver = dialog.getByLabel("FlareSolverr URL", { exact: true });
    await solver.fill("http://solver.example:8191");
    revision = 2;
    const reread = page.waitForResponse(response => response.request().postData()?.includes('"operationName":"IndexerNetworkSettings"') ?? false);
    notify();
    await reread;
    await dialog.getByRole("button", { name: "Save", exact: true }).click();
    await expect(dialog.getByRole("alert")).toContainText("Settings changed");
    expect(calls[0].variables.expectedRevision).toBe("config-1");
    await expect(proxy).toHaveValue("http://synthetic-new-secret@proxy.example:8080");
    await expect(solver).toHaveValue("http://solver.example:8191");
  });
  for (const [locale, theme] of [["zh-CN", "light"], ["en", "dark"], ["ar-XB", "dark"]] as const) {
    test(`${locale} ${theme} network controls fit and retain accessible fields`, async ({ page }) => {
      await page.addInitScript(({ locale, theme }) => { localStorage.setItem("moji:locale", locale); localStorage.setItem("moji:theme", theme); }, { locale, theme });
      await page.goto(`/settings/indexers?panel=cache${locale === "ar-XB" ? "&__locale=ar-XB" : ""}`);
      await expect(page.locator("html")).toHaveAttribute("dir", locale === "ar-XB" ? "rtl" : "ltr");
      const card = page.locator("article").filter({ has: page.locator(".indexer-network-profiles") });
      await expect(card.locator(".indexer-network-profiles li")).toHaveCount(3);
      await card.scrollIntoViewIfNeeded();
      await expect(card).toHaveScreenshot(`${locale}-${theme}-network-profiles.png`);
      const spare = card.locator("li").filter({ has: page.getByText("spare", { exact: true }) });
      await spare.getByRole("button").first().click();
      await expect(page.getByRole("dialog")).toHaveCount(1);
      await expect(page.locator("#indexer-proxy-url")).toHaveCSS("direction", "ltr");
      await expect(page.locator("#indexer-solver-url")).toHaveCSS("direction", "ltr");
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    });
  }

});
