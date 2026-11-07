import { expect } from "@playwright/test";
import { homeOverview, settingsSnapshot, test } from "./fixtures";

const task = { ...homeOverview.attentionTasks[0], stashSceneId: "101", stashSceneIds: ["101"] };
test.use({ graphqlOverrides: {
  tasks: [task], task: () => task,
  settings: { ...settingsSnapshot, system: { ...settingsSnapshot.system, taskDeletePolicy: "REMOVE_TORRENT_AND_FILES" } }
} });

for (const locale of ["en", "zh-CN"]) for (const entry of ["tasks", "home"]) {
  test(`${locale} ${entry} deletion policy links to its settings`, async ({ page }, info) => {
    await page.addInitScript(locale => localStorage.setItem("moji:locale", locale), locale);
    await page.emulateMedia({ colorScheme: locale === "en" ? "light" : "dark" });
    const mutations: string[] = [];
    page.on("request", request => {
      if (request.url().endsWith("/graphql") && request.method() === "POST") {
        const query = request.postDataJSON().query as string;
        if (query.includes("mutation DeleteTaskDocument")) mutations.push(query);
      }
    });
    await page.goto(entry === "tasks" ? `/tasks/${task.id}` : `/?task=${task.id}`);
    await page.getByRole("dialog").locator(".task-detail-actions__delete").click();
    const policy = page.locator(".confirm-delete__policy");
    if (locale === "zh-CN") await expect(policy).toContainText("将会同时删除 qBittorrent 下载任务和已下载文件，此操作不可撤销。");
    const checkbox = page.locator(".confirm-delete__stash input");
    await expect(checkbox).not.toBeChecked();
    const type = await page.locator(".confirm-delete").evaluate(element => {
      const typography = (selector: string) => {
        const style = getComputedStyle(element.querySelector(selector)!);
        return [style.fontFamily, style.fontSize, style.fontWeight, style.lineHeight];
      };
      const icon = element.querySelector(".confirm-delete__policy > svg")!.getBoundingClientRect();
      const text = element.querySelector(".confirm-delete__policy > span")!.getBoundingClientRect();
      return { policy: typography(".confirm-delete__policy > span"), stash: typography(".confirm-delete__stash > span"), iconOffset: Math.abs(icon.y + icon.height / 2 - text.y - text.height / 2) };
    });
    expect(type.policy).toEqual(type.stash);
    expect(type.iconOffset).toBeLessThan(1);
    const rows = await page.locator(".confirm-delete").evaluate(element => {
      const stash = element.querySelector(".confirm-delete__stash")!;
      const policy = element.querySelector(".confirm-delete__policy")!;
      const lineCount = (selector: string) => {
        const span = element.querySelector(selector)!;
        const range = document.createRange();
        const walker = document.createTreeWalker(span, NodeFilter.SHOW_TEXT);
        const tops = new Set<number>();
        while (walker.nextNode()) {
          range.selectNodeContents(walker.currentNode);
          for (const rect of range.getClientRects()) if (rect.width > 0) tops.add(Math.round(rect.top));
        }
        return tops.size;
      };
      return { stashTop: stash.getBoundingClientRect().top, policyTop: policy.getBoundingClientRect().top,
        stashLines: lineCount(".confirm-delete__stash > span"), policyLines: lineCount(".confirm-delete__policy > span") };
    });
    expect(rows.stashTop).toBeLessThan(rows.policyTop);
    if (info.project.name === "desktop") {
      expect(rows.stashLines).toBe(1);
      expect(rows.policyLines).toBe(1);
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({ path: info.outputPath("delete-settings.png") });
    await page.getByRole("button", { name: locale === "en" ? "Change deletion settings" : "前往修改配置", exact: true }).click();
    await expect(page).toHaveURL(/\/settings\/tasks\?panel=deletion$/);
    await expect(page.locator("#settings-tab-deletion")).toHaveAttribute("aria-selected", "true");
    await expect(page.getByRole("tabpanel").getByRole("combobox")).toHaveValue("REMOVE_TORRENT_AND_FILES");
    expect(mutations).toEqual([]);
  });
}
