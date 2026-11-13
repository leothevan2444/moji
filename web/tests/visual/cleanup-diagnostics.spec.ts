import { expect } from "@playwright/test";
import { downloaderSnapshot, test } from "./fixtures";

const labels={en:{title:"Automatic cleanup",IDLE:"Enabled",RUNNING:"Cleaning",WARNING:"Cleanup blocked",ERROR:"Cleanup failed",reason:"Cannot verify file usage"},"zh-CN":{title:"自动清理",IDLE:"已启用",RUNNING:"清理中",WARNING:"清理受阻",ERROR:"清理失败",reason:"无法确认文件占用"}};
for(const locale of ["en","zh-CN"] as const){
 for(const phase of ["IDLE","RUNNING","WARNING","ERROR"] as const){
  for(const theme of ["light","dark"] as const){
   test.describe(`${locale} ${phase} ${theme}`,()=>{
    const reason=phase==="WARNING"?"CONTENT_PATH_UNKNOWN":phase==="ERROR"?"DELETE_REJECTED":phase==="RUNNING"?"CONFIRMING":"";
    test.use({graphqlOverrides:{downloader:{...downloaderSnapshot,cleanup:{enabled:true,phase,reason,hash:"a".repeat(40),name:phase==="IDLE"?"":"EXAMPLE-101",updatedAt:"2026-01-15T12:00:00Z"},freeSpace:21000000000}}});
    test("shows compact cleanup state and accessible details",async({page})=>{
     await page.addInitScript(({locale,theme})=>{localStorage.setItem("moji:locale",locale);localStorage.setItem("moji:theme",theme)},{locale,theme});
     await page.goto("/");
     const section=page.locator(".home-downloader");
     const trigger=section.getByRole("button",{name:`${labels[locale].title}${locale==="en"?": ":"："}${labels[locale][phase]}`,exact:true});
     await expect(trigger).toBeVisible();await expect(trigger).toHaveClass(new RegExp(`home-cleanup--${phase.toLowerCase()}`));
     await page.emulateMedia({reducedMotion:"no-preference"});
     expect(await trigger.locator("svg").evaluate(el=>getComputedStyle(el).animationName)).toBe(phase==="RUNNING"?"cleanup-sweep":"none");
     await page.emulateMedia({reducedMotion:"reduce"});
     expect(await trigger.locator("svg").evaluate(el=>getComputedStyle(el).animationName)).toBe("none");
     await page.evaluate(()=>document.fonts.ready);
     await expect(section).toHaveScreenshot(`${locale}-${theme}-${phase.toLowerCase()}.png`);
     await trigger.focus();await page.keyboard.press("Enter");
     const dialog=page.getByRole("dialog",{name:labels[locale].title,exact:true});await expect(dialog).toBeVisible();
     if(phase==="WARNING")await expect(dialog.getByText(labels[locale].reason,{exact:true})).toBeVisible();
     if(locale==="zh-CN"&&phase==="WARNING"&&theme==="light")await expect(dialog).toHaveScreenshot("cleanup-details.png");
     await page.keyboard.press("Escape");await expect(dialog).toHaveCount(0);await expect(trigger).toBeFocused();
    });
   });
  }
 }
}
test.describe("stale cleanup observation",()=>{
 test.use({graphqlOverrides:{downloader:{...downloaderSnapshot,stale:true,cleanup:{enabled:true,phase:"RUNNING",reason:"CONFIRMING",hash:"a".repeat(40),name:"EXAMPLE-101",updatedAt:""}}}});
 test("stops animation when status cannot be confirmed",async({page})=>{
  await page.goto("/");const control=page.locator(".home-cleanup");await expect(control).toHaveClass(/home-cleanup--warning/);await expect(control).not.toHaveClass(/home-cleanup--running/);
 });
});
