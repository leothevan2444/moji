import { expect } from "@playwright/test";
import { test } from "./fixtures";

const entry = (sequence: number, level = "Info") => ({
  __typename: "LogEntry", sequence, level, time: "2026-10-31T08:00:00Z",
  message: `Log ${sequence}: ${sequence === 1 ? "历史日志 · " : ""}download observation and task status — ${"long-message/".repeat(8)}`
});
let snapshot = Array.from({ length: 100 }, (_, index) => entry(100 - index));
test.use({ graphqlOverrides: { logs: ({ minLevel }: { minLevel: string }) => snapshot.filter(log => minLevel !== "Warning" || log.level === "Warning") } });

for (const locale of ["en", "zh-CN"]) {
  for (const theme of ["light", "dark"]) {
    test(`${locale} ${theme} logs follow, history, filter and snapshot recovery`, async ({ page }, info) => {
      snapshot = Array.from({ length: 100 }, (_, index) => entry(100 - index));
      let emit: (sequence: number) => void = () => { throw new Error("Log subscription not connected"); };
      await page.routeWebSocket("**/graphql", socket => socket.onMessage(raw => {
        const message = JSON.parse(String(raw));
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "subscribe" && message.payload.query.includes("subscription LogEvents")) {
          emit = sequence => socket.send(JSON.stringify({ id: message.id, type: "next", payload: { data: { logEvents: { __typename: "LogEvent", sequence, entry: entry(sequence) } } } }));
        }
      }));
      await page.addInitScript(({ locale, theme }) => {
        localStorage.setItem("moji:locale", locale);
        localStorage.setItem("moji:theme", theme);
      }, { locale, theme });
      await page.goto("/settings/logs");
      const log = page.getByRole("log");
      const atBottom = () => log.evaluate(node => node.scrollHeight - node.clientHeight - node.scrollTop < 2);
      await expect(log.locator(".log-line")).toHaveCount(100);
      await expect.poll(atBottom).toBe(true);
      emit(101);
      await expect(log.locator(".log-line").last()).toContainText("Log 101:");
      await expect.poll(atBottom).toBe(true);
      await log.evaluate(node => { node.scrollTop = 100; node.dispatchEvent(new Event("scroll")); });
      await expect(page.getByRole("button", { name: locale === "en" ? "Resume following" : "恢复跟随", exact: true })).toBeVisible();
      for (let sequence = 102; sequence <= 310; sequence++) {
        snapshot = [entry(sequence), ...snapshot].slice(0, 200);
        emit(sequence);
      }
      const resume = page.getByRole("button", { name: locale === "en" ? "New logs · Resume following" : "有新日志 · 恢复跟随", exact: true });
      await expect(resume).toBeVisible();
      await expect(log.locator(".log-line")).toHaveCount(101);
      expect(await log.evaluate(node => node.scrollTop)).toBe(100);
      await resume.click();
      await expect(log.locator(".log-line")).toHaveCount(200);
      await expect(log.locator(".log-line").last()).toContainText("Log 310:");
      await expect.poll(atBottom).toBe(true);
      // A sequence gap uses the existing snapshot recovery path.
      snapshot = [entry(401, "Warning"), entry(400)];
      emit(402);
      await expect(log.locator(".log-line")).toContainText(["Log 400:", "Log 401:", "Log 402:"]);
      await page.getByRole("combobox").selectOption("Warning");
      await expect(log.locator(".log-line")).toHaveCount(1);
      await expect(log).toContainText("Log 401:");
      await page.getByRole("combobox").selectOption("Info");
      await expect(log.locator(".log-line")).toHaveCount(2);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      expect(await log.evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
      await page.evaluate(() => document.fonts.ready);
      await page.screenshot({ path: info.outputPath("logs.png"), fullPage: true });
    });
  }
}
