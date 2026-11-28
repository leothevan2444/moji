import {expect} from '@playwright/test';
import {test,homeOverview} from './fixtures';

const tasks=Array.from({length:16},(_,i)=>({...homeOverview.activeTasks[0],id:`sort-${i+1}`,code:`ABC-${i+1}`,stage:'DOWNLOADING',stageStatus:'RUNNING',progress:(i+1)/20,createdAt:new Date(Date.UTC(2026,0,i+1)).toISOString(),releaseDate:i<3?null:`2025-01-${String(i+1).padStart(2,'0')}`}));
const actors=[
 {id:'a',name:'Actor 10',created:'2020-01-01',sceneCount:10,checked:'2025-01-01'},
 {id:'b',name:'Actor 2',created:'2025-01-01',sceneCount:2,checked:null},
 {id:'c',name:'Actor 1',created:null,sceneCount:0,checked:'2026-01-01'},
].map(actor=>({...actor,aliasList:[],favorite:false,subscribed:false,imagePath:null}));
const performerWorkspace=({options}:{options?:{sort?:string;direction?:string}})=>{
 const items=[...actors].sort((a,b)=>{
  const field=options?.sort??'NAME';let value=0;
  if(field==='SCENE_COUNT')value=a.sceneCount-b.sceneCount;
  else if(field==='CREATED'||field==='LAST_CHECKED'){
   const x=field==='CREATED'?a.created:a.checked,y=field==='CREATED'?b.created:b.checked;
   if(Boolean(x)!==Boolean(y))return x?-1:1;
   value=(x??'').localeCompare(y??'');
  }else value=a.name.localeCompare(b.name,'en',{numeric:true});
  return options?.direction==='DESC'?-value:value;
 });
 return {activities:[],subscribedPerformers:[],filterCounts:{total:3,subscribed:0,favorite:0,review:0,error:0},performers:{items,page:1,pageSize:24,totalCount:3,totalPages:1,hasPrevPage:false,hasNextPage:false}};
};
test.use({graphqlOverrides:{tasks,performerWorkspace}});

test('task sort field and direction persist independently and grouping stays in the browser',async({page},info)=>{
 const clickAction=async(name:string)=>{
  const button=page.getByRole('button',{name,exact:true});
  if(!await button.isVisible())await page.locator('.workspace-toolbar__panel.is-active button.workspace-toolbar__more').click();
  await button.click();
 };
 const reads:any[]=[];page.on('request',request=>{if(request.url().endsWith('/graphql')&&request.method()==='POST')reads.push(request.postDataJSON());});
 await page.goto('/tasks?q=ABC&status=active&page=2&pageSize=12');
 await expect(page.locator('.task-card')).toHaveCount(4);
 await expect(page.locator('.task-group-section')).toHaveCount(0);
 await page.getByRole('button',{name:'Creation time: Descending; switch to Ascending'}).click();
 await expect(page.locator('.task-card__open').first()).toContainText('ABC-1');
 await expect(page).not.toHaveURL(/page=2/);
 await page.getByRole('combobox',{name:'Task sort order'}).selectOption('title');
 await expect(page.getByRole('button',{name:'Title: Ascending; switch to Descending'})).toBeVisible();
 await expect(page.locator('.task-card__open').nth(1)).toContainText('ABC-2');
 const url=new URL(page.url());expect(url.searchParams.get('q')).toBe('ABC');expect(url.searchParams.get('status')).toBe('active');expect(url.searchParams.get('direction')).toBe('asc');
 const before=reads.filter(read=>read.query.includes('query TasksWorkspace')).length;
 await clickAction('Enable task grouping');
 await expect(page.locator('.task-group-section')).toHaveCount(1);
 expect(reads.filter(read=>read.query.includes('query TasksWorkspace')).length).toBe(before);
 expect(await page.evaluate(()=>localStorage.getItem('moji:tasks-grouped'))).toBe('true');
 await page.reload();await expect(page.locator('.task-group-section')).toHaveCount(1);
 await clickAction('Disable task grouping');
 await page.getByRole('combobox',{name:'Task sort order'}).selectOption('releaseDate');
 await expect(page.getByRole('button',{name:'Film release date: Ascending; switch to Descending'})).toBeVisible();
 await expect(page.locator('.task-card__open').first()).toContainText('ABC-4');
 expect(reads.filter(read=>read.query.includes('query TasksWorkspace')).every(read=>!('grouped' in read.variables.input))).toBe(true);
 await page.locator('.workspace-toolbar').screenshot({path:info.outputPath('task-sort-controls.png')});
});

test('performers use name ascending by default and keep unknown dates last in either direction',async({page},info)=>{
 await page.goto('/performers');const cards=page.locator('.profile-card');
 await expect(cards.first()).toContainText('Actor 1');
 const sort=page.getByRole('combobox',{name:'Sort'});
 await sort.selectOption('created');await expect(cards.first()).toContainText('Actor 10');await expect(cards.last()).toContainText('Actor 1');
 await page.getByRole('button',{name:'Creation time: Ascending; switch to Descending'}).click();
 await expect(cards.first()).toContainText('Actor 2');await expect(cards.last()).toContainText('Actor 1');
 await sort.selectOption('last_checked');await expect(cards.first()).toContainText('Actor 1');await expect(cards.last()).toContainText('Actor 2');
 await page.getByRole('button',{name:'Last check time: Descending; switch to Ascending'}).click();
 await expect(cards.first()).toContainText('Actor 10');await expect(cards.last()).toContainText('Actor 2');
 await page.reload();await expect(sort).toHaveValue('last_checked');await expect(page.getByRole('button',{name:'Last check time: Ascending; switch to Descending'})).toBeVisible();
 await page.locator('.workspace-toolbar').screenshot({path:info.outputPath('performer-sort-controls.png')});
});
