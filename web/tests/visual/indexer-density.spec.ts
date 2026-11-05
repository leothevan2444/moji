import { expect } from "@playwright/test";
import { test, indexerInstances } from "./fixtures";

test.use({ graphqlOverrides: {
  indexerInstances: { ...indexerInstances, items: indexerInstances.items.map(site => ({ ...site, description: "A short site description" })) }
} });

test("compact sites retain links and inline controls", async ({ page, isMobile }) => {
  await page.addInitScript(() => { localStorage.setItem("moji:locale", "en"); localStorage.setItem("moji:theme", "light"); });
  await page.goto("/settings/indexers");
  const rows = page.locator(".indexer-row");
  await expect(rows).toHaveCount(2);
  for (const row of await rows.all()) {
    expect((await row.boundingBox())!.height).toBeLessThanOrEqual(isMobile ? 104 : 80);
  }
  const header = page.locator(".drawer-card__head").filter({ has: page.getByRole("button", { name: "Add site", exact: true }) });
  await expect(header.getByRole("button", { name: "Refresh Jackett sites" })).toBeVisible();
  await expect(rows.first().getByRole("link", { name: "Sukebei" })).toHaveAttribute("href", "https://sukebei.nyaa.si/");
  await expect(page.getByRole("button", { name: /Move .* (up|down)/ })).toHaveCount(0);
  await expect(page).toHaveScreenshot("compact-indexer-sites.png", { fullPage: true });
});

test("privacy follows automatic dark theme and explicit light override", async ({ page }) => {
  await page.addInitScript(() => { localStorage.setItem("moji:locale", "en"); localStorage.setItem("moji:theme", "auto"); });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/settings/indexers");
  await page.getByRole("button", { name: "Add site", exact: true }).click();
  const badge = page.locator(".indexer-catalog .indexer-privacy").first();
  await expect(badge).toHaveCSS("background-color", "rgb(22, 101, 52)");
  await expect(badge).toHaveCSS("border-radius", "4px");
  await expect(page.getByRole("dialog").getByRole("link", { name: "Sukebei" })).toHaveAttribute("href", "https://sukebei.nyaa.si/");
  await expect(page.locator(".indexer-categories")).toContainText("XXX/DVD");
  const edges = await page.locator(".indexer-catalog").evaluate(node => {
    const modal = node.getBoundingClientRect();
    const body = node.querySelector(".drawer-body")!.getBoundingClientRect();
    return Math.min(Math.abs(modal.right - body.right), Math.abs(modal.left - body.left));
  });
  expect(edges).toBeLessThanOrEqual(2);
  await expect(page).toHaveScreenshot("auto-dark-indexer-catalog.png", { fullPage: true });
  await page.emulateMedia({ colorScheme: "light" });
  await expect(badge).toHaveCSS("background-color", "rgb(21, 128, 61)");
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: /Theme:/ }).click();
  await page.getByRole("menuitemradio", { name: "Light", exact: true }).click();
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator(".indexer-privacy").first()).toHaveCSS("background-color", "rgb(21, 128, 61)");
});

test("settings scroll at workspace edge and Jackett has one connection card", async ({ page }) => {
  await page.addInitScript(() => { localStorage.setItem("moji:locale", "en"); localStorage.setItem("moji:theme", "dark"); });
  await page.goto("/settings/indexers?panel=connection");
  await expect(page.getByRole("heading", { name: "Jackett", exact: true })).toBeVisible();
  await expect(page.locator(".settings-subpanel:not([hidden]) .drawer-card")).toHaveCount(1);
  await expect(page.getByRole("switch", { name: "Enable" })).toBeChecked();
  await expect(page.getByRole("link", { name: "Manage sites" })).toHaveAttribute("href", "/settings/indexers?panel=sites");
  const edges = await page.locator(".settings-main").evaluate(node => {
    const main = node.getBoundingClientRect();
    const workspace = node.parentElement!.getBoundingClientRect();
    return Math.abs(main.right - workspace.right);
  });
  expect(edges).toBeLessThanOrEqual(2);
  await expect(page).toHaveScreenshot("external-jackett-connection.png", { fullPage: true });
  const info = page.locator(".indexer-external-toggle .settings-info");
  await expect(page.getByRole("tooltip")).not.toBeVisible();
  await info.focus();
  await expect(page.getByRole("tooltip")).toBeVisible();
  await expect(page.getByRole("tooltip")).toContainText("Turning this off");
  await page.getByRole("link", { name: "Manage sites" }).click();
  await expect(page.getByRole("button", { name: "Add site", exact: true })).toBeVisible();
});
