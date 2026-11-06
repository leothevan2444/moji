// @vitest-environment jsdom
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StrictMode } from "react";
import i18n from "../../i18n/i18n";
import { TaskIndexerSearch } from "./TaskIndexerSearch";
import type { DashboardTask } from "../../utils";
const state=vi.hoisted(()=>({start:vi.fn(),cancel:vi.fn(),refresh:vi.fn(),resolve:vi.fn(),read:vi.fn(),reread:vi.fn(()=>({toPromise:()=>Promise.resolve({data:null})})),snapshot:null as unknown,error:null as unknown,starting:false}));
vi.mock("urql",()=>({useClient:()=>({query:state.reread}),useMutation:()=>[{fetching:false},state.resolve]}));
vi.mock("../../hooks/useTorrentSearch",()=>({useTorrentSearch:(id:string|null,offset:number,limit:number,sort:string)=>{state.read(id,offset,limit,sort);return {snapshot:state.snapshot,start:state.start,cancel:state.cancel,refresh:state.refresh,error:state.error,starting:state.starting,fetching:false,cancelling:false}}}));
const task={id:"one",code:"ABCD-123",revision:7,stage:"SOURCING",stageStatus:"BLOCKED"} as DashboardTask;
const candidate={id:"candidate",instanceId:"site",instanceName:"Site",kind:"BUILTIN",title:"ABCD-123 synthetic release",detailsUrl:"https://example.test/details/1",size:null,seeders:null,peers:null,publishedAt:null,categories:[]};
const snapshot={id:"run",taskId:"one",state:"COMPLETED",revision:1,total:1,sites:[],candidates:[candidate],cancelRequested:false};
function Location(){const location=useLocation();return <><output data-testid="location">{location.search}</output><output data-testid="background">{JSON.stringify(location.state)}</output></>}
function setup(url="/?sourcingTask=one&sourcingRun=run",currentTask=task){const resolved=vi.fn(),busy=vi.fn();const result=render(<MemoryRouter initialEntries={[url]}><TaskIndexerSearch task={currentTask} disabled={false} onResolved={resolved} onBusyChange={busy}/><Location/></MemoryRouter>);return {...result,resolved,busy}}
beforeEach(async()=>{await i18n.changeLanguage("en");state.start.mockReset().mockResolvedValue("run");state.resolve.mockReset().mockResolvedValue({data:{resolveTaskSourcingCandidate:{id:"one",stage:"SOURCING",stageStatus:"PENDING",revision:8,sourcingNextCheckAt:null}}});state.cancel.mockReset();state.refresh.mockReset();state.read.mockClear();state.reread.mockClear();state.snapshot=snapshot;state.error=null;state.starting=false});afterEach(cleanup);
it("a saved task run only reads and retains unknown observations and safe details",()=>{
 setup();expect(state.start).not.toHaveBeenCalled();expect(state.read).toHaveBeenCalledWith("run",0,10,"RELEVANCE");expect(screen.getByText("Seeders unknown")).toBeTruthy();expect(screen.getByText("Unknown size")).toBeTruthy();expect(screen.getByRole("link",{name:"Indexer details"}).getAttribute("href")).toBe(candidate.detailsUrl);
});
it("summarizes terminal and active sites without treating failures as successful searches",async()=>{
 state.snapshot={...snapshot,state:"RUNNING",sites:[
  {instanceId:"done",name:"Empty site",state:"COMPLETED",results:0,cacheHit:false},
  {instanceId:"failed",name:"Failed site",state:"FAILED",results:0,errorCode:"SYNTHETIC_UNKNOWN_ERROR"},
  {instanceId:"cancelled",name:"Cancelled site",state:"CANCELLED",results:0},
  {instanceId:"running",name:"Running site",state:"RUNNING",results:0},
  {instanceId:"queued",name:"Queued site",state:"QUEUED",results:0}
 ]};
 const view=setup();const summary=view.container.querySelector("summary")!;
 expect(summary.textContent).toContain("3/5 sites finished");expect(summary.textContent).toContain("Issues 1");
 expect(summary.parentElement?.hasAttribute("open")).toBe(false);
 await userEvent.setup().click(summary);
 expect(summary.parentElement?.hasAttribute("open")).toBe(true);
 expect(screen.getByText("0 results")).toBeTruthy();
 expect(screen.getByText("Unable to reach the site. Check the network and site settings.")).toBeTruthy();
 expect(screen.getByRole("button",{name:"Cancel search"})).toBeTruthy();
});
it("opening without a run admits once and saves a URL, including StrictMode",async()=>{
 state.snapshot=null;render(<StrictMode><MemoryRouter initialEntries={["/?section=queue"]}><TaskIndexerSearch task={task} disabled={false} onResolved={vi.fn()}/><Location/></MemoryRouter></StrictMode>);
 await waitFor(()=>expect(screen.getByTestId("location").textContent).toContain("sourcingRun=run"));expect(state.start).toHaveBeenCalledTimes(1);expect(state.start).toHaveBeenCalledWith({keywords:"ABCD-123",taskId:"one",refresh:false});expect(screen.getByTestId("location").textContent).toContain("section=queue");
});
it("choosing only sends task version and server identities, then reports admission",async()=>{
 const view=setup();await userEvent.setup().click(screen.getByRole("button",{name:"Select: ABCD-123 synthetic release"}));expect(state.resolve).toHaveBeenCalledWith({input:{taskId:"one",searchId:"run",candidateId:"candidate",expectedRevision:7}});expect(view.resolved).toHaveBeenCalledWith(expect.objectContaining({stage:"SOURCING",stageStatus:"PENDING"}));expect(view.busy.mock.calls).toEqual([[true],[false]]);
});
it("fresh search is explicit while failed reads keep candidates and manual siblings usable",async()=>{
 state.error=new Error("synthetic read failure");setup();expect(screen.getByRole("alert").textContent).toContain("synthetic read failure");await userEvent.setup().click(screen.getByRole("button",{name:"Search again"}));expect(state.start).toHaveBeenCalledWith({keywords:"ABCD-123",taskId:"one",refresh:true});
});
it("foreign task runs are not shown or selectable",()=>{
 state.snapshot={...snapshot,taskId:"other"};setup();expect(screen.queryByRole("button",{name:"Select: ABCD-123 synthetic release"})).toBeNull();expect(state.resolve).not.toHaveBeenCalled();expect(screen.getByText("This search belongs to another task. Search again for the current task.")).toBeTruthy();
});
it("failed selection retains candidates and does not close the task",async()=>{
 state.resolve.mockResolvedValue({error:new Error("selection failed")});const view=setup();await userEvent.setup().click(screen.getByRole("button",{name:"Select: ABCD-123 synthetic release"}));expect(screen.getByRole("alert").textContent).toContain("selection failed");expect(view.resolved).not.toHaveBeenCalled();expect(screen.getByText(candidate.title)).toBeTruthy();expect(state.reread).toHaveBeenCalledTimes(1);
});
it("releases the parent busy state even when accepted work unmounts this view",async()=>{
 let finish!:(data:unknown)=>void;state.resolve.mockImplementation(()=>new Promise(resolve=>{finish=resolve}));const view=setup();await userEvent.setup().click(screen.getByRole("button",{name:"Select: ABCD-123 synthetic release"}));view.unmount();await act(async()=>{finish({data:{resolveTaskSourcingCandidate:{id:"one",stage:"SOURCING",stageStatus:"PENDING"}}})});expect(view.busy.mock.calls).toEqual([[true],[false]]);expect(view.resolved).not.toHaveBeenCalled();
});

it("saving an automatically admitted run preserves the background route",async()=>{
 state.snapshot=null;render(<MemoryRouter initialEntries={[{pathname:"/tasks/one/resolve",search:"?status=failed",state:{backgroundLocation:{pathname:"/tasks",search:"?status=failed&page=2"}}}]}><TaskIndexerSearch task={task} disabled={false} onResolved={vi.fn()}/><Location/></MemoryRouter>);
 await waitFor(()=>expect(screen.getByTestId("location").textContent).toContain("sourcingRun=run"));expect(screen.getByTestId("background").textContent).toContain("page=2");
});
