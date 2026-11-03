import { expect } from "@playwright/test";
import { homeOverview, test } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
});

test("indexers and downloader are peers and fixed gateway cards are removed", async ({ page }) => {
  await page.goto("/");
  const indexers = page.getByRole("region", { name: "Indexers", exact: true });
  const downloader = page.getByRole("region", { name: "Downloader", exact: true });
  await expect(indexers.getByText("Jackett", { exact: true })).toBeVisible();
  await expect(indexers.getByText("1 enabled site")).toHaveCount(2);
  await expect(indexers.getByText("1 built-in site has not been verified")).toBeVisible();
  await expect(indexers.locator("..")).toHaveClass("home-sidebar");
  await expect(downloader.locator("..")).toHaveClass("home-sidebar");
  await page.locator(".home-system .home-diagnostics summary").click();
  await expect(page.locator(".home-system").getByText("Jackett", { exact: true })).toHaveCount(0);
  await expect(page.locator(".home-system").getByText("qBittorrent", { exact: true })).toHaveCount(0);
  await downloader.getByRole("link", { name: "Configure", exact: true }).click();
  await expect(page).toHaveURL(/settings\/downloader/);
});

test.describe("unloaded indexer directory", () => {
  test.use({ graphqlOverrides: { homeIndexerSummary: { definitionJob: null, definitionErrorCode: "", revision: "unknown", total: null, builtinEnabled: 0, builtinUnverified: 0, builtinSearchSucceeded:0,healthErrorCode:"", attention: 1, issues: [], jackett: { sites: null, stale: false, updatedAt: null, errorCode: "INDEXER_CATALOG_NOT_LOADED" } } } });
  test("keeps configured Jackett visible without reporting zero sites", async ({ page }) => {
    await page.goto("/");
    const indexers = page.getByRole("region", { name: "Indexers", exact: true });
    await expect(indexers.getByText("Jackett", { exact: true })).toBeVisible();
    await expect(indexers.getByText("Site count unknown", { exact: true })).toBeVisible();
    await expect(indexers.getByText("0 enabled sites", { exact: true })).toHaveCount(0);
    await indexers.getByRole("link", { name: "Manage indexers" }).click();
    await expect(page).toHaveURL(/settings\/indexers/);
  });
});

test.describe("optional external indexer", () => {
  test.use({ graphqlOverrides: { homeIndexerSummary: { definitionJob: null, definitionErrorCode: "", revision: "builtin-only", total: 1, builtinEnabled: 1, builtinUnverified: 1, builtinSearchSucceeded:0,healthErrorCode:"", attention: 0, issues: [], jackett: null } } });
  test("does not reserve a Jackett row when it is disabled", async ({ page }) => {
    await page.goto("/");
    const indexers = page.getByRole("region", { name: "Indexers", exact: true });
    await expect(indexers.getByText("1 enabled site")).toBeVisible();
    await expect(indexers.getByText("Jackett", { exact: true })).toHaveCount(0);
    await expect(page.locator(".home-alert--setup")).toHaveCount(0);
  });
});

test.describe("stale confirmed indexer directory", () => {
  test.use({ graphqlOverrides: { homeIndexerSummary: { definitionJob: null, definitionErrorCode: "", revision: "stale", total: 1, builtinEnabled: 0, builtinUnverified: 0, builtinSearchSucceeded:0,healthErrorCode:"", attention: 1, issues: [], jackett: { sites: 1, stale: true, updatedAt: "2026-01-15T11:00:00Z", errorCode: "INDEXER_REQUEST_FAILED" } } } });
  test("keeps the last confirmed count and offers indexer management", async ({ page }) => {
    await page.goto("/");
    const indexers = page.getByRole("region", { name: "Indexers", exact: true });
    await expect(indexers.getByText("1 enabled site")).toBeVisible();
    await expect(indexers.getByText(/directory is out of date/)).toBeVisible();
    await expect(indexers.locator("time")).toHaveAttribute("datetime", "2026-01-15T11:00:00Z");
    await expect(indexers.getByRole("link", { name: "Manage indexers" })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  });
});

test.describe("indexer invalidations", () => {
  let changed = false;
  test.use({ graphqlOverrides: { homeIndexerSummary: () => ({ definitionJob: null, definitionErrorCode: "", revision: changed ? "after" : "before", total: changed ? 3 : 2, builtinEnabled: 1, builtinUnverified: 1, builtinSearchSucceeded:0,healthErrorCode:"", attention: 0, issues: [], jackett: { sites: changed ? 2 : 1, stale: false, updatedAt: "2026-01-15T11:00:00Z", errorCode: "" } }) } });
  test("server changes refresh only the indexer region without reloading Home", async ({ page }) => {
    changed = false;
    let notify: (() => void) | undefined;
    let homeRequests = 0;
    page.on("request", request => { if (request.method() === "POST" && request.postDataJSON()?.operationName === "HomePageDocument") homeRequests++; });
    await page.routeWebSocket("**/graphql", socket => {
      socket.onMessage(data => {
        const message = JSON.parse(String(data)) as { id: string; type: string; payload?: { query?: string } };
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription IndexerEvents")) {
          const send = (sequence: number, kind: string) => socket.send(JSON.stringify({ id: message.id, type: "next", payload: { data: { indexerEvents: { streamId: "live", sequence, kind } } } }));
          send(0, "READY");
          notify = () => send(3, "CATALOG");
        }
      });
    });
    try {
      await page.goto("/");
      const indexers = page.getByRole("region", { name: "Indexers", exact: true });
      await expect(indexers.getByText("1 enabled site")).toHaveCount(2);
      await expect.poll(() => Boolean(notify)).toBe(true);
      const before = homeRequests;
      expect(before).toBeGreaterThan(0);
      changed = true; notify!();
      await expect(indexers.getByText("2 enabled sites")).toBeVisible();
      expect(homeRequests).toBe(before);
      await expect(page).toHaveURL("/");
    } finally { changed = false; }
  });

  test("retry reopens a rejected indexer subscription", async ({ page }) => {
    changed = false;
    let opens = 0;
    await page.routeWebSocket("**/graphql", socket => {
      socket.onMessage(data => {
        const message = JSON.parse(String(data)) as { id: string; type: string; payload?: { query?: string } };
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription IndexerEvents")) {
          opens++;
          if (opens === 1) socket.send(JSON.stringify({ id: message.id, type: "error", payload: [{ message: "Indexer subscription unavailable", extensions: { code: "INDEXER_BUSY", correlationId: "synthetic" } }] }));
          else socket.send(JSON.stringify({ id: message.id, type: "next", payload: { data: { indexerEvents: { streamId: "recovered", sequence: 0, kind: "READY" } } } }));
        }
      });
    });
    try {
      await page.goto("/");
      const indexers = page.getByRole("region", { name: "Indexers", exact: true });
      await expect(indexers.getByRole("button", { name: "Retry", exact: true })).toBeVisible();
      changed = true;
      await indexers.getByRole("button", { name: "Retry", exact: true }).click();
      await expect.poll(() => opens).toBe(2);
      await expect(indexers.getByText("2 enabled sites")).toBeVisible();
      await expect(indexers.getByRole("button", { name: "Retry", exact: true })).toHaveCount(0);
    } finally { changed = false; }
  });
});

test("home opens file confirmation directly and restores focus after Escape", async ({ page }) => {
  await page.goto("/");
  const confirm = page.getByRole("button", { name: "Select movie files: MOJI-101" });
  await confirm.click();
  await expect(page).toHaveURL(/task=MOJI-101&files=1/);
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("checkbox")).toBeVisible();
  await expect(dialog.getByRole("heading", { name: "Task overview" })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(page).toHaveURL("/");
  await expect(confirm).toBeFocused();
});

test("home search preserves query and source when returning", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("combobox").selectOption("indexers");
  await page.getByRole("textbox", { name: "Search movies or resources" }).fill("MOJI-101");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page).toHaveURL(/discover\?q=MOJI-101&source=indexers/);
  await page.goBack();
  await expect(page.getByRole("textbox", { name: "Search movies or resources" })).toHaveValue("MOJI-101");
  await expect(page.getByRole("combobox")).toHaveValue("indexers");
});

test.describe("review queue", () => {
  test.use({ graphqlOverrides: { tasks: homeOverview.attentionTasks } });
  test("waits for tasks before opening the session and returns Home on close", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Review queue" }).click();
    await expect(page.getByRole("dialog", { name: "Task workbench" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "MOJI-101", exact: true })).toBeVisible();
    await expect(page.getByText("0 handled · 3 remaining")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("search", { name: "Search movies or resources" })).toBeVisible();
    await expect(page).toHaveURL("/");
  });
});

for (const locale of ["qps-ploc", "ar-XB"] as const) {
  test(`${locale} home layout`, async ({ page }) => {
    await page.goto(`/?__locale=${locale}`);
    await expect(page.locator("html")).toHaveAttribute("lang", locale);
    await expect(page.locator(".home-metric")).toHaveCount(4);
    await expect(page.locator(".home-loading")).toHaveCount(0);
    await page.evaluate(() => document.fonts.ready);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await expect(page).toHaveScreenshot(`${locale}-home.png`, { fullPage: true });
  });
}

test("dark home layout", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  await expect(page.locator(".home-loading")).toHaveCount(0);
  await expect.poll(() => page.locator("html").evaluate(element => getComputedStyle(element).colorScheme)).toBe("dark");
  await page.evaluate(() => document.fonts.ready);
  await expect(page).toHaveScreenshot("dark-home.png", { fullPage: true });
});
