import { expect } from "@playwright/test";
import { test, indexerInstances } from "./fixtures";

test("trusted origin draft tests, saves, restores and revokes without changing credentials", async ({ page }) => {
  let revision = indexerInstances.revision;
  let origins = ["https://download.example"];
  const saved: Record<string, unknown>[] = [];
  const checked: Record<string, unknown>[] = [];
  await page.addInitScript(() => localStorage.setItem("moji:locale", "en"));
  await page.route("**/graphql", async route => {
    const { operationName, variables } = route.request().postDataJSON();
    const snapshot = () => ({ ...indexerInstances, revision, items: indexerInstances.items.map((site, i) => i ? site : { ...site, trustedOrigins: origins }) });
    if (operationName === "IndexerInstances") return route.fulfill({ json: { data: { indexerInstances: snapshot() } } });
    if (operationName === "UpdateBuiltinIndexer") {
      saved.push(variables);
      origins = variables.input.trustedOrigins;
      revision = "config-next";
      return route.fulfill({ json: { data: { updateBuiltinIndexer: { revision, instance: snapshot().items[0] } } } });
    }
    if (operationName === "TestIndexer") checked.push(variables.input);
    return route.fallback();
  });
  await page.goto("/settings/indexers");
  const open = () => page.locator('.indexer-row[data-indexer-id="builtin-one"]').getByRole("button", { name: "Configure", exact: true }).click();
  await open();
  const editor = page.getByRole("dialog", { name: "Configure site", exact: true });
  await expect(editor.getByLabel("Additional trusted origins")).not.toBeVisible();
  await editor.getByText("Advanced settings", { exact: true }).click();
  const field = editor.getByLabel("Additional trusted origins");
  await expect(field).toHaveValue("https://download.example");
  await field.fill("https://download.example/path");
  await expect(editor.getByRole("button", { name: "Save", exact: true })).toBeDisabled();
  await expect(editor.getByRole("button", { name: "Test", exact: true })).toBeDisabled();
  await field.fill("https://auth.example:8443\nhttps://download.example");
  await editor.getByRole("button", { name: "Test", exact: true }).click();
  await expect(editor.getByRole("button", { name: "Connected", exact: true })).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(1);
  expect(checked[0]).toMatchObject({ draft: { trustedOrigins: ["https://auth.example:8443", "https://download.example"] } });
  expect(saved).toHaveLength(0);
  await expect(field).toHaveValue("https://auth.example:8443\nhttps://download.example");
  await editor.getByRole("button", { name: "Save", exact: true }).click();
  await expect(editor).toHaveCount(0);
  expect(saved[0]).toEqual({ id: "builtin-one", expectedRevision: indexerInstances.revision, input: { trustedOrigins: ["https://auth.example:8443", "https://download.example"] } });
  await page.reload(); await open();
  await editor.getByText("Advanced settings", { exact: true }).click();
  await expect(field).toHaveValue("https://auth.example:8443\nhttps://download.example");
  await field.fill("");
  await editor.getByRole("button", { name: "Save", exact: true }).click();
  await expect(editor).toHaveCount(0);
  expect(saved[1]).toEqual({ id: "builtin-one", expectedRevision: "config-next", input: { trustedOrigins: [] } });
});

for (const [locale, theme] of [["zh-CN", "light"], ["en", "dark"], ["ar-XB", "dark"]] as const) {
  test(`${locale} advanced origins are readable with bounded errors`, async ({ page }) => {
    await page.addInitScript(value => { localStorage.setItem("moji:locale", value.locale); localStorage.setItem("moji:theme", value.theme); }, { locale, theme });
    await page.goto("/settings/indexers" + (locale === "ar-XB" ? "?__locale=ar-XB" : ""));
    await expect(page.locator("html")).toHaveAttribute("dir", locale === "ar-XB" ? "rtl" : "ltr");
    await page.locator('.indexer-row[data-indexer-id="builtin-one"]').getByRole("button", { name: locale === "zh-CN" ? "配置" : "Configure", exact: true }).click();
    const editor = page.getByRole("dialog");
    await editor.locator(".indexer-editor__advanced summary").click();
    const field = editor.locator("textarea");
    await field.fill(Array.from({ length: 17 }, (_, i) => `https://download${i}.example`).join("\n"));
    await expect(editor.getByRole("alert")).toBeVisible();
    await expect(field).toHaveCSS("direction", "ltr");
    expect(await editor.evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
    await expect(editor.getByRole("button", { name: locale === "zh-CN" ? "保存" : "Save", exact: true })).toBeInViewport();
    const head = await editor.locator(".drawer__head").boundingBox();
    expect(head?.y).toBeGreaterThanOrEqual(0);
    await expect(editor).toHaveScreenshot(`${locale}-trusted-origins.png`);
  });
}
