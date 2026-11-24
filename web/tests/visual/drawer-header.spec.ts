import { expect } from "@playwright/test";
import { test } from "./fixtures";

for (const width of [1440, 800, 640, 320]) {
  test(`${width}px drawer keeps Close beside short and long titles`, async ({page}, info) => {
    await page.setViewportSize({width,height:900});
    await page.goto("/");
    const help = page.locator(".masthead").getByRole("button", {name:"Help",exact:true});
    await help.click();
    const dialog = page.getByRole("dialog");
    const head = dialog.locator(".drawer__head");
    const close = head.getByRole("button", {name:"Close",exact:true});
    for (const title of ["Help", "An exceptionally long drawer title ".repeat(6), "VeryLongTitleWithoutAnySpaces".repeat(8)]) {
      await head.locator("h2").evaluate((node, text) => { node.textContent = text; }, title);
      const heading = await head.locator(":scope > div").boundingBox();
      const button = await close.boundingBox();
      const frame = await head.boundingBox();
      expect(button!.x).toBeGreaterThanOrEqual(heading!.x + heading!.width);
      expect(Math.abs(button!.y - heading!.y)).toBeLessThan(2);
      expect(button!.x + button!.width).toBeLessThanOrEqual(frame!.x + frame!.width + 1);
      expect(await head.evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
      await expect(close).toHaveText("Close");
    }
    await head.screenshot({path:info.outputPath("long-title-header.png")});
    await close.focus();
    await page.keyboard.press("Enter");
    await expect(dialog).toBeHidden();
    await expect(help).toBeFocused();
  });
}

for (const hasTouch of [false, true]) {
  test.describe(hasTouch ? "touch" : "mouse", () => {
    test.use({hasTouch});
    test("task title remains 28px at every width", async ({page}, info) => {
      await page.setViewportSize({width:960,height:900});
      await page.goto("/tasks/MOJI-201");
      const dialog = page.getByRole("dialog");
      const head = dialog.locator(".drawer__head");
      const title = head.locator("h2");
      await expect(title).toHaveText("MOJI-201");
      for (const width of [960,640,320]) {
        await page.setViewportSize({width,height:900});
        await expect(title).toHaveCSS("font-size", "28px");
        // Sample both elements in one frame, including while the drawer enters.
        const positions = await head.evaluate(node => {
          const title = node.querySelector("h2")!.getBoundingClientRect();
          const button = node.querySelector("button")!.getBoundingClientRect();
          return {titleRight:title.right,titleTop:title.top,buttonLeft:button.left,buttonTop:button.top};
        });
        expect(positions.buttonLeft).toBeGreaterThanOrEqual(positions.titleRight);
        expect(Math.abs(positions.buttonTop - positions.titleTop)).toBeLessThan(2);
        expect(await head.evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
        await head.screenshot({path:info.outputPath(`task-header-${width}.png`)});
      }
      await title.evaluate(node => { node.textContent = "TASK-0001 Long task title ".repeat(4); });
      await expect(title).toHaveCSS("font-size", "28px");
      expect(await head.evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
      await head.getByRole("button", {name:"Close",exact:true}).click();
      await expect(dialog).toBeHidden();
    });
  });
}
