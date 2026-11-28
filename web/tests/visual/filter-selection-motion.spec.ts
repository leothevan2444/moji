import {expect, type Locator} from "@playwright/test";
import {test} from "./fixtures";

async function aligned(group:Locator) {
 await expect(group).toHaveAttribute('data-window-ready','true');
 await expect.poll(()=>group.evaluate(node=>{
  const selected=node.querySelector<HTMLButtonElement>('button[aria-pressed=true]')!;
  const a=node.getBoundingClientRect(),b=selected.getBoundingClientRect(),style=getComputedStyle(node),window=getComputedStyle(node,'::before');
  const scale=a.width/parseFloat(style.width),matrix=new DOMMatrixReadOnly(window.transform);
  return Math.max(Math.abs(matrix.m41-(b.left-a.left)/scale+node.clientLeft),Math.abs(matrix.m42-(b.top-a.top)/scale+node.clientTop),Math.abs(parseFloat(window.width)-b.width/scale),Math.abs(parseFloat(window.height)-b.height/scale));
 })).toBeLessThan(.6);
}
for(const path of ['tasks','performers']) for(const motion of ['no-preference','reduce'] as const) {
 test(`${path} selection window follows changes and layout with ${motion}`,async({page},info)=>{
  await page.emulateMedia({reducedMotion:motion});
  await page.goto(`/${path}${path==='performers'?'?__locale=ar-XB':''}`);
  const group=page.locator('.filter-count-heading').getByRole('toolbar');
  await aligned(group);
  await group.evaluate(node=>{
   const positions=[getComputedStyle(node,'::before').transform];
   const observer=new MutationObserver(()=>{positions.push(getComputedStyle(node,'::before').transform);node.setAttribute('data-window-positions',JSON.stringify(positions));});
   observer.observe(node,{attributes:true,attributeFilter:['style']});
   node.addEventListener('test-stop-window-observer',()=>observer.disconnect(),{once:true});
  });
  await group.getByRole('button').nth(2).click();
  await expect(group.getByRole('button').nth(2)).toHaveAttribute('aria-pressed','true');
  await aligned(group);
  if(motion==='no-preference') expect(new Set(JSON.parse(await group.getAttribute('data-window-positions')??'[]')).size).toBeGreaterThan(1);
  else {
   expect(await group.evaluate(node=>getComputedStyle(node,'::before').transitionDuration)).toBe('0s');
   expect(new Set(JSON.parse(await group.getAttribute('data-window-positions')??'[]')).size).toBeLessThanOrEqual(2);
  }
  await group.evaluate(node=>node.dispatchEvent(new Event('test-stop-window-observer')));
  // Retarget an in-flight transition instead of leaving a stale window behind.
  await group.getByRole('button').nth(4).click();
  await group.getByRole('button').nth(1).click();
  await aligned(group);
  for(const width of [600,360,320,1440]) {
   await page.setViewportSize({width,height:1000});
   await aligned(group);
  }
  await page.evaluate(()=>{document.documentElement.style.zoom='1.25';});
  await aligned(group);
  await group.getByRole('button').first().click();
  await aligned(group);
  await group.screenshot({path:info.outputPath('selection-window.png')});
 });
}

for(const path of ['tasks','performers']) for(const motion of ['no-preference','reduce'] as const) {
 test(`${path} button widths transition when selected names change with ${motion}`,async({page})=>{
  await page.emulateMedia({reducedMotion:motion});
  await page.goto('/'+path);
  const heading=page.locator('.filter-count-heading'),group=heading.getByRole('toolbar');
  await heading.evaluate(node=>{
   const probes=node.querySelector('.filter-count__measurement')!,title=probes.querySelector('h2')!;
   const selected=probes.querySelector<HTMLElement>('[data-density=selected]')!.cloneNode(true) as HTMLElement;
   probes.append(selected);
   const buttons=[...selected.children]; let widest=0;
   for(const active of buttons) {
    for(const button of buttons) button.classList.toggle('is-active',button===active);
    widest=Math.max(widest,selected.scrollWidth);
   }
   selected.remove();
   (node as HTMLElement).style.width=`${Math.ceil(title.scrollWidth+widest+parseFloat(getComputedStyle(node).columnGap)+4)}px`;
  });
  await expect(group).toHaveAttribute('data-density','selected');
  await aligned(group);
  await expect.poll(()=>group.evaluate(node=>[...node.querySelectorAll('button')].some(button=>button.getAnimations().some(animation=>'transitionProperty' in animation && animation.transitionProperty==='width' && (animation.playState==='running'||animation.pending))))).toBe(false);
  await group.evaluate(node=>{
   const target=node.querySelectorAll('button')[2],widths:number[]=[];
   const observer=new ResizeObserver(()=>{widths.push(target.getBoundingClientRect().width);node.setAttribute('data-width-samples',JSON.stringify(widths));});
   observer.observe(target);
   node.addEventListener('transitionrun',event=>{
    if((event as TransitionEvent).propertyName==='width' && (event.target as HTMLElement).matches('button')) node.setAttribute('data-button-width-transition','true');
   });
   node.addEventListener('transitionend',event=>{
    if((event as TransitionEvent).propertyName==='width' && event.target===target && target.getAttribute('aria-pressed')==='true') requestAnimationFrame(()=>{
     const a=node.getBoundingClientRect(),b=target.getBoundingClientRect(),style=getComputedStyle(node),marker=getComputedStyle(node,'::before');
     const scale=a.width/parseFloat(style.width),matrix=new DOMMatrixReadOnly(marker.transform);
     node.setAttribute('data-error-at-width-end',String(Math.max(Math.abs(matrix.m41-(b.left-a.left)/scale+node.clientLeft),Math.abs(parseFloat(marker.width)-b.width/scale))));
    });
   });
   node.addEventListener('test-stop-width-observer',()=>observer.disconnect(),{once:true});
  });
  await group.getByRole('button').nth(2).click();
  await expect(group.getByRole('button').nth(2)).toHaveAttribute('aria-pressed','true');
  await aligned(group);
  if(motion==='no-preference') {
   await expect(group).toHaveAttribute('data-button-width-transition','true');
   await expect(group).toHaveAttribute('data-error-at-width-end',/.+/);
   expect(Number(await group.getAttribute('data-error-at-width-end'))).toBeLessThan(.7);
   const samples=await group.getAttribute('data-width-samples');
   expect(new Set((JSON.parse(samples??'[]') as number[]).map(width=>Math.round(width*10))).size).toBeGreaterThan(2);
  } else {
   expect(await group.getByRole('button').nth(2).evaluate(node=>getComputedStyle(node).transitionDuration)).toBe('0s');
   await expect(group).not.toHaveAttribute('data-button-width-transition','true');
  }
  await group.evaluate(node=>node.dispatchEvent(new Event('test-stop-width-observer')));
 });
}

test.describe('quantity width changes',()=>{
 let total=1;
 test.beforeEach(()=>{total=1;});
 test.use({graphqlOverrides:{taskPage:()=>({items:[],page:1,pageSize:24,totalCount:total,totalPages:Math.ceil(total/24),sequence:0,streamId:'visual',filterCounts:{total,active:total,waiting:0,completed:0,attention:0},stats:{total,active:total,waiting:0,completed:0,attention:0,downloading:total,pendingScans:0,failed:0}})}});
 test('numeric updates animate the button width without changing the density',async({page})=>{
  await page.emulateMedia({reducedMotion:'no-preference'});
  await page.setViewportSize({width:1440,height:1000});
  await page.goto('/tasks');
  const group=page.locator('.filter-count-heading').getByRole('toolbar');
  await expect(group).toHaveAttribute('data-density','inline');
  await aligned(group);
  await group.evaluate(node=>node.addEventListener('transitionrun',event=>{
   if((event as TransitionEvent).propertyName==='width' && (event.target as HTMLElement).matches('button')) node.setAttribute('data-quantity-transition','true');
  }));
  await group.evaluate(node=>{
   const target=node.querySelector('button')!,samples:{width:number;error:number}[]=[];
   // Observe the rendered window after its frame update. Sampling from an
   // earlier ResizeObserver would inspect the previous frame before drawing.
   const observer=new MutationObserver(()=>{
    const frame=node.getBoundingClientRect(),scale=frame.width/parseFloat(getComputedStyle(node).width);
    const width=target.getBoundingClientRect().width/scale;
    samples.push({width,error:Math.abs(parseFloat(getComputedStyle(node,'::before').width)-width)});
    node.setAttribute('data-sync-samples',JSON.stringify(samples));
   });
   observer.observe(node,{attributes:true,attributeFilter:['style']});
   node.addEventListener('test-stop-sync-observer',()=>observer.disconnect(),{once:true});
  });
  total=123456;
  await page.getByRole('button',{name:'Refresh',exact:true}).click();
  await expect(group.getByRole('button').first()).toHaveAccessibleName('All tasks 123456');
  await expect(group).toHaveAttribute('data-quantity-transition','true');
  await expect.poll(()=>group.getByRole('button').first().evaluate(node=>Math.abs(parseFloat(getComputedStyle(node).width)-parseFloat((node as HTMLElement).style.width)))).toBeLessThan(.1);
  await aligned(group);
  const samples=JSON.parse(await group.getAttribute('data-sync-samples')??'[]') as {width:number;error:number}[];
  expect(new Set(samples.map(sample=>sample.width)).size).toBeGreaterThan(2);
  expect(Math.max(...samples.map(sample=>sample.error))).toBeLessThan(.7);
  await group.evaluate(node=>node.dispatchEvent(new Event('test-stop-sync-observer')));
  await expect(group).toHaveAttribute('data-density','inline');
  await aligned(group);
 });
});
