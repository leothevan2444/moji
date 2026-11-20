// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskStage, TaskStageStatus, TaskSource, type TaskStashScenesDocumentQuery } from "../../graphql/generated/graphql";
import type { DashboardTask } from "../../utils";
import { TaskSummary } from "./TaskSummary";

type QueryState = {fetching?:boolean;data?:TaskStashScenesDocumentQuery;error?:Error};
const mocks = vi.hoisted(() => ({query:vi.fn(),reload:vi.fn(),copy:vi.fn(),result:{} as QueryState}));
vi.mock("urql",()=>({useQuery:(options:unknown)=>{mocks.query(options);return [mocks.result,mocks.reload];}}));
const film = {endpoint:"https://catalog.test/graphql",sceneId:"source-1",code:"DEMO-001",title:"Source title",date:"2026-10-01",studio:"Source studio",performers:["Source actor"],imageUrl:"/api/images/source",url:"",confirmationDigest:"fixture"};
const task:DashboardTask = {id:"task-1",source:TaskSource.Manual,film,code:film.code,stage:TaskStage.Completed,stageStatus:TaskStageStatus.Done,stageLabel:"",stageStatusLabel:"",progress:1,torrentUrl:"",stashSceneIds:["1","2"],createdAt:"2026-10-18T00:00:00Z",updatedAt:"2026-10-18T01:00:00Z",files:[]};
const scene = {id:"1",title:"Stash title",code:"DEMO-001",date:"2026-10-18",studio:"Stash studio",performers:["Stash actor One","Stash actor Two"],imageUrl:"/api/images/stash",url:"https://stash.test/scenes/1"};
const view = (value=task)=><TaskSummary key={value.id} task={value} onCopy={mocks.copy} />;

beforeEach(async()=>{vi.clearAllMocks();mocks.result={fetching:false};await i18n.changeLanguage("en");});
afterEach(cleanup);

describe("merged task summary",()=>{
  it("keeps source metadata before ingest and pauses the Stash query",()=>{
    const {container}=render(view({...task,stage:TaskStage.Downloading,stashSceneIds:[]}));
    for (const value of [film.title,film.studio,...film.performers]) expect(screen.getByText(value)).toBeTruthy();
    expect(container.querySelector("img")?.getAttribute("src")).toBe(film.imageUrl);
    expect(mocks.query.mock.calls[0][0].pause).toBe(true);
    expect(screen.queryByText(task.code,{exact:true})).toBeNull();
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.queryByText("Movie information appears after ingest")).toBeNull();
  });

  it("reads all linked Stash movies in the summary and uses their metadata",()=>{
    mocks.result={data:{taskStashScenes:[scene,{...scene,id:"2",title:"Second Stash movie",imageUrl:null,url:"https://stash.test/scenes/2"}]}};
    const {container}=render(view());
    expect(screen.getAllByRole("article")).toHaveLength(1);
    expect(screen.getByRole("article",{name:"Task overview"})).toBeTruthy();
    expect(screen.queryByRole("article",{name:"Stash movie information"})).toBeNull();
    expect(screen.getByRole("heading",{name:scene.title})).toBeTruthy();
    expect(screen.getByRole("heading",{name:"Second Stash movie"})).toBeTruthy();
    expect(screen.getAllByText(scene.performers.join(" / "))).toHaveLength(2);
    for (const value of [film.title,film.studio,...film.performers]) expect(screen.queryByText(value)).toBeNull();
    expect(screen.queryByText(task.code,{exact:true})).toBeNull();
    const link=screen.getByRole("link",{name:"View in Stash: Stash title"});
    expect(link.getAttribute("href")).toBe(scene.url);
    expect(link.getAttribute("rel")).toContain("noopener");
    const image=container.querySelector("img")!;
    expect(image.getAttribute("src")).toBe(scene.imageUrl);
    fireEvent.error(image);
    expect(container.querySelector("img")).toBeNull();
  });

  it("uses actual Stash metadata even before task completion",()=>{
    mocks.result={data:{taskStashScenes:[scene]}};
    render(view({...task,stage:TaskStage.Scanning,stashSceneIds:[scene.id]}));
    expect(screen.getByRole("heading",{name:scene.title})).toBeTruthy();
    expect(screen.queryByText(film.title)).toBeNull();
  });

  it("does not fill incomplete Stash metadata from source snapshots",()=>{
    mocks.result={data:{taskStashScenes:[{...scene,title:"",code:"",date:"",studio:"",performers:[],imageUrl:null}]}};
    const {container}=render(view({...task,stage:TaskStage.Scanning,stashSceneIds:[scene.id]}));
    expect(screen.getByRole("heading",{name:"Untitled movie"})).toBeTruthy();
    expect(screen.getByText(/Stash identification is missing: title, code, release date, studio, performers/)).toBeTruthy();
    for (const value of [film.title,film.studio,...film.performers]) expect(screen.queryByText(value)).toBeNull();
    expect(container.querySelector("img")).toBeNull();
  });

  it("offers a retry without exposing connection details or source metadata as Stash",async()=>{
    mocks.result={error:new Error("private connection detail")};
    render(view());
    expect(screen.getByText("Stash movie information is temporarily unavailable")).toBeTruthy();
    expect(screen.queryByText("private connection detail")).toBeNull();
    expect(screen.queryByText(film.title)).toBeNull();
    await userEvent.setup().click(screen.getByRole("button",{name:"Reload"}));
    expect(mocks.reload).toHaveBeenCalledWith({requestPolicy:"network-only"});
  });

  it("retains confirmed Stash data on failure and clears it on confirmed removal",()=>{
    mocks.result={data:{taskStashScenes:[scene]}};
    const current={...task,stashSceneIds:[scene.id]};
    const {rerender}=render(view(current));
    mocks.result={error:new Error("offline")};
    rerender(view(current));
    expect(screen.getByRole("heading",{name:scene.title})).toBeTruthy();
    expect(screen.getByText("Stash movie information is temporarily unavailable")).toBeTruthy();
    mocks.result={data:{taskStashScenes:[]},fetching:false};
    rerender(view(current));
    expect(screen.queryByRole("heading",{name:scene.title})).toBeNull();
    expect(screen.queryByText(film.title)).toBeNull();
    expect(screen.getByText(/Linked movies were not found in Stash/)).toBeTruthy();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("does not carry confirmed metadata into a different task",()=>{
    mocks.result={data:{taskStashScenes:[scene]}};
    const {rerender}=render(view({...task,stashSceneIds:[scene.id]}));
    mocks.result={error:new Error("offline")};
    rerender(view({...task,id:"task-2",stashSceneIds:[scene.id]}));
    expect(screen.queryByRole("heading",{name:scene.title})).toBeNull();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("distinguishes pending reads, missing records and historical tasks without links",()=>{
    mocks.result={fetching:true};
    const {rerender}=render(view());
    expect(screen.getByRole("status").textContent).toBe("Loading movie information…");
    mocks.result={fetching:false,data:{taskStashScenes:[scene]}};
    rerender(view());
    expect(screen.getByText("Unavailable linked movies: 1")).toBeTruthy();
    rerender(view({...task,stashSceneIds:[]}));
    expect(screen.getByRole("heading",{name:film.title})).toBeTruthy();
    expect(screen.getByText("No linked Stash movies found")).toBeTruthy();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("rereads Stash after an ingest transition",()=>{
    mocks.result={data:{taskStashScenes:[scene]}};
    const {rerender}=render(view({...task,stage:TaskStage.Scanning}));
    expect(mocks.reload).not.toHaveBeenCalled();
    rerender(view());
    expect(mocks.reload).toHaveBeenCalledOnce();
    expect(mocks.reload).toHaveBeenCalledWith({requestPolicy:"network-only"});
  });

  it("ignores records outside the persisted association",()=>{
    mocks.result={data:{taskStashScenes:[{...scene,id:"unrelated"}]}};
    render(view());
    expect(screen.queryByRole("heading",{name:scene.title})).toBeNull();
    expect(screen.queryByRole("link")).toBeNull();
  });
});
