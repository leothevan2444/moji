import { expect } from "@playwright/test";
import { test, homeOverview } from "./fixtures";
const film = {endpoint:"https://metadata.example/graphql",sceneId:"film-one",code:"300MIUM-1437",title:"Synthetic film with a long title for confirmation and narrow screen wrapping",date:"2026-01-01",studio:"Synthetic studio",performers:["Synthetic performer"],imageUrl:"",url:"",durationSeconds:7200,confirmationDigest:"confirmed-digest"};
const url="/discover?source=indexers&q=MOJI-102&run=search-one";
test.use({graphqlOverrides:{
 taskFilmSources:()=>[{name:"Preferred source",endpoint:film.endpoint},{name:"Other source",endpoint:"https://other.example/graphql"}],
 taskFilmCandidates:()=>[film], taskFilm:()=>film,
 createFilmTask:()=>({id:"confirmed-task",code:film.code,stage:"DOWNLOADING",stageStatus:"PENDING"}),
 task:({id}:{id:string})=>({...homeOverview.activeTasks[0],id,film,sourcingMode:"USER_SELECTED"}),
}});

test("resource selection waits for film confirmation, preserves the resource, and creates once",async({page},info)=>{
 const creations:unknown[]=[];const pageErrors:string[]=[];
 page.on("pageerror",error=>pageErrors.push(error.message));
 page.on("request",r=>{if(r.url().endsWith("/graphql")&&r.method()==="POST"){const b=r.postDataJSON();if(b.operationName==="CreateFilmTask")creations.push(b.variables.input)}});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url);
 const card=page.locator(".candidate-card--indexer");await expect(card).toHaveCount(1);await card.getByRole("button").last().click();
 const dialog=page.getByRole("dialog",{name:"Confirm film",exact:true});await expect(dialog).toBeVisible();expect(creations).toHaveLength(0);
 await dialog.getByRole("button",{name:/300MIUM-1437/}).click();await expect(dialog.getByRole("heading",{name:/300MIUM-1437/})).toBeVisible();
 const create=dialog.getByRole("button",{name:"Create task",exact:true});await expect(create).toBeDisabled();expect(creations).toHaveLength(0);
 await expect(dialog).toContainText("does not contain the full film code");await dialog.getByRole("checkbox").check();
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.screenshot({path:info.outputPath("film-resource-confirmation.png")});
 await create.click();await expect(page).toHaveURL(/\/tasks\/confirmed-task/);expect(creations).toHaveLength(1);
 expect(creations[0]).toMatchObject({endpoint:film.endpoint,sceneId:film.sceneId,confirmationDigest:film.confirmationDigest,searchId:"search-one",candidateId:"candidate-one",resourceConfirmed:true});
 expect(creations[0]).not.toHaveProperty("code");expect(pageErrors).toEqual([]);
});

test("cancelling a resource confirmation returns to search without creating a task",async({page})=>{
 let creations=0;page.on("request",r=>{if(r.url().endsWith("/graphql")&&r.method()==="POST"&&r.postDataJSON().operationName==="CreateFilmTask")creations++});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto(url);await page.locator(".candidate-card--indexer").getByRole("button").last().click();
 await expect(page.getByRole("dialog",{name:"Confirm film"})).toBeVisible();await page.keyboard.press("Escape");
 await expect(page.getByRole("dialog",{name:"Confirm film"})).toHaveCount(0);await expect(page.locator(".candidate-card--indexer")).toBeVisible();expect(creations).toBe(0);
});

test("manual resource users can change keywords and choose another configured source",async({page})=>{
 const searches:Array<Record<string,unknown>>=[];page.on("request",r=>{if(r.url().endsWith("/graphql")&&r.method()==="POST"){const b=r.postDataJSON();if(b.operationName==="TaskFilmCandidates")searches.push(b.variables.input)}});
 await page.addInitScript(()=>localStorage.setItem("moji:locale","en"));await page.goto("/discover");await page.getByRole("button",{name:"Create from a film or resource"}).click();
 const dialog=page.getByRole("dialog",{name:"Confirm film"});await dialog.getByLabel(/Magnet or torrent URL/).fill("magnet:?xt=urn:btih:0123456789012345678901234567890123456789&dn=Synthetic");
 await dialog.getByRole("button",{name:"Match film",exact:true}).click();await expect(dialog.getByRole("button",{name:/300MIUM-1437/})).toBeVisible();
 await dialog.getByLabel("Film keywords",{exact:true}).fill("alternate title");await dialog.getByRole("combobox").selectOption("https://other.example/graphql");await dialog.getByRole("button",{name:"Search films",exact:true}).click();
 await expect.poll(()=>searches.at(-1)?.keywords).toBe("alternate title");expect(searches.at(-1)?.endpoint).toBe("https://other.example/graphql");
});
