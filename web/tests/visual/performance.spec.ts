import { expect } from "@playwright/test";
import { settingsSnapshot, test } from "./fixtures";

const now = "2026-01-15T12:00:00.000Z";
const pool = { open: 4, inUse: 1, idle: 3, limit: 4, waitCount: 12, waitMilliseconds: 24 };
const metrics = {
  startedAt: "2026-01-15T09:00:00.000Z", sampledAt: now, intervalSeconds: 5, lastError: "",
  databases: ["core", "runtime", "cache"].map((role, index) => ({
    role, databaseBytes: (index + 1) * 4 * 1024 * 1024, walBytes: 1024 * 128, freeBytes: 4096,
    writeAttempts: 200, writeErrors: 0, busyRetries: 2, queueRejected: 0, queueCanceled: 0,
    queued: 1, queueCapacity: 64, writesPerSecond: 12.4, averageWaitMilliseconds: 0.15,
    averageExecutionMilliseconds: 1.25, totalWaitMilliseconds: 30, totalExecutionMilliseconds: 250,
    reader: pool, writer: { ...pool, open: 1, idle: 0, limit: 1 }
  }))
};

test("developer mode is hidden by default", async ({ page }) => {
  await page.goto("/?__locale=en");
  await expect(page.getByRole("button", { name: "Statistics", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Performance monitoring", exact: true })).toHaveCount(0);
});

for (const locale of ["en", "zh-CN"] as const) {
  test.describe(locale, () => {
    test.use({ graphqlOverrides: { settings: { ...settingsSnapshot, system: { ...settingsSnapshot.system, developerMode: true } }, databasePerformance: metrics } });
    test(`${locale} database performance modal`, async ({ page }, info) => {
      await page.addInitScript(value => localStorage.setItem("moji:locale", value), locale);
      await page.goto(`/?__locale=${locale}`);
      const name = locale === "en" ? "Performance monitoring" : "性能监控";
      const opener = page.getByRole("button", { name, exact: true });
      await expect(opener).toBeVisible();
      const order = await page.locator(".utility-group > button").evaluateAll(buttons => buttons.map(button => button.getAttribute("aria-label")));
      expect(order.slice(0, 2)).toEqual([name, locale === "en" ? "Statistics" : "统计"]);
      await opener.click();
      const dialog = page.getByRole("dialog", { name, exact: true });
      await expect(dialog).toBeVisible();
      await expect(dialog.locator(".performance-card")).toHaveCount(3);
      await expect(dialog.getByText(/Live data|实时数据|Updates automatically via subscription|通过订阅自动更新/)).toHaveCount(0);
      await expect(dialog.getByText(/Sampled at|采样时间/)).toHaveCount(0);
      await expect(dialog.getByText("12.4", { exact: false }).first()).toBeVisible();
      await expect(dialog).toHaveScreenshot(`database-performance-${locale}-${info.project.name}.png`);
      const overflow = await dialog.evaluate(element => element.scrollWidth > element.clientWidth + 1);
      expect(overflow).toBe(false);
      await page.keyboard.press("Escape");
      await expect(dialog).toHaveCount(0);
      await expect(opener).toBeFocused();
    });
  });
}

test.describe("developer configuration", () => {
  let developerMode = false;
  test.beforeEach(() => { developerMode = false; });
  test.use({ graphqlOverrides: {
    settings: () => ({ ...settingsSnapshot, system: { ...settingsSnapshot.system, developerMode } }),
    updateSystemSettings: ({ input }: { input: { developerMode: boolean } }) => {
      developerMode = input.developerMode;
      return { ...settingsSnapshot, system: { ...settingsSnapshot.system, developerMode } };
    }
  } });
  test("saved switch controls the header immediately and survives refresh", async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto("/settings/system");
    const toggle = page.getByRole("switch", { name: "Developer mode", exact: true });
    const entry = page.getByRole("button", { name: "Performance monitoring", exact: true });
    const save = page.getByRole("button", { name: "Save system settings", exact: true });
    await expect(toggle).not.toBeChecked();
    await expect(entry).toHaveCount(0);
    await toggle.check();
    await expect(entry).toHaveCount(0);
    await save.click();
    await expect(entry).toBeVisible();
    await page.reload();
    await expect(toggle).toBeChecked();
    await expect(entry).toBeVisible();
    await toggle.uncheck();
    await save.click();
    await expect(entry).toHaveCount(0);
    await page.reload();
    await expect(toggle).not.toBeChecked();
    await expect(entry).toHaveCount(0);
  });
});


test.describe("performance stream", () => {
  test.use({ graphqlOverrides: { settings: { ...settingsSnapshot, system: { ...settingsSnapshot.system, developerMode: true } } } });
  test("renders pushes without HTTP metric queries and sends complete on close", async ({ page }) => {
    let metricsID = "";
    let send = (_value: unknown) => {};
    const completed: string[] = [];
    let metricRequests = 0;
    page.on("request", request => { if (request.url().endsWith("/graphql") && request.postData()?.includes("databasePerformance")) metricRequests++; });
    await page.routeWebSocket("**/graphql", socket => {
      send = value => socket.send(JSON.stringify({ id: metricsID, type: "next", payload: { data: { databasePerformance: value } } }));
      socket.onMessage(data => {
        const message = JSON.parse(String(data));
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription DatabasePerformanceDocument")) { metricsID = message.id; send(metrics); }
        if (message.type === "complete") completed.push(message.id);
      });
    });
    await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
    await page.goto("/");
    await page.getByRole("button", { name: "Performance monitoring", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "Performance monitoring", exact: true });
    await expect(dialog.locator(".performance-card")).toHaveCount(3);
    send({ ...metrics, databases: metrics.databases.map(row => ({ ...row, writesPerSecond: 99.2 })) });
    await expect(dialog.getByText("99.2", { exact: false }).first()).toBeVisible();
    expect(metricRequests).toBe(0);
    await page.keyboard.press("Escape");
    await expect.poll(() => completed.includes(metricsID)).toBe(true);
    expect(metricRequests).toBe(0);
  });
});
